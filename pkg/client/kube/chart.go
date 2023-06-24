package kube

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/godzilla-s/k3s-installer/pkg/config"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"

	"helm.sh/helm/pkg/chartutil"
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/disk"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var (
	ErrChartNotRelease = errors.New("chart not release")
)

type Chart struct {
	PkgPath         string
	ReleaseName     string
	Namespace       string
	CreateNamespace bool
	ValuesFile      string
	After           string
	Before          string
	Timeout         time.Duration
	SetValues       string
}

type ReleaseChart struct {
	Status      string
	ReleaseName string
	Namespace   string
}

type ChartClient struct {
	kube          *Client
	clientGetters map[string]*restClientGetter
	tempCache     string
}

type restClientGetter struct {
	namespace  string
	restConfig *rest.Config
	apiConfig  clientcmdapi.Config
	debugLog   action.DebugLog
}

func (r *restClientGetter) ToRESTConfig() (*rest.Config, error) {
	return r.restConfig, nil
}

func (r *restClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return disk.NewCachedDiscoveryClientForConfig(r.restConfig, ".tmp/cached", ".tmp/cached/http", 10*time.Second)
}

func (r *restClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	discoveryClient, err := r.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	expander := restmapper.NewShortcutExpander(mapper, discoveryClient)
	return expander, nil
}

func (r *restClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	override := &clientcmd.ConfigOverrides{}
	override.Context.Namespace = r.namespace
	return clientcmd.NewDefaultClientConfig(r.apiConfig, override)
}

func ToChart(c *config.Chart) *Chart {
	baseDir := filepath.Dir(c.Path)
	ch := &Chart{
		PkgPath:     c.Path,
		ReleaseName: c.ReleaseName,
		Namespace:   c.Namespace,
		ValuesFile:  filepath.Join(baseDir, "values.yaml"),
		Timeout:     c.Timeout,
	}
	if ch.Timeout == 0 {
		ch.Timeout = 1 * time.Minute
	}
	_, err := os.Stat(filepath.Join(baseDir, "after"))
	if err == nil {
		ch.After = filepath.Join(baseDir, "after")
	}
	_, err = os.Stat(filepath.Join(baseDir, "before"))
	if err == nil {
		ch.Before = filepath.Join(baseDir, "before")
	}
	fmt.Println("====", ch)
	return ch
}

func (cli *ChartClient) newActionConfig(namespace string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	restGetter := &restClientGetter{
		namespace:  namespace,
		restConfig: cli.kube.restConfig,
		debugLog:   cli.kube.log.Printf,
		apiConfig:  cli.kube.apiConfig,
	}
	err := actionConfig.Init(restGetter, namespace, "secret", cli.kube.log.Infof)
	if err != nil {
		return nil, err
	}
	return actionConfig, nil
}

func (cli *ChartClient) Install(c *Chart) error {
	actionConfig, err := cli.newActionConfig(c.Namespace)
	if err != nil {
		return err
	}
	install := action.NewInstall(actionConfig)
	install.ReleaseName = c.ReleaseName
	install.Namespace = c.Namespace
	install.Wait = true
	install.Timeout = c.Timeout
	install.CreateNamespace = true

	values, err := chartutil.ReadValuesFile(c.ValuesFile)
	if err != nil {
		return err
	}

	chart, err := loader.LoadFile(c.PkgPath)
	if err != nil {
		return err
	}

	if c.Before != "" {
		err = cli.kube.Apply(c.Before, ApplyOption{})
		if err != nil {
			return err
		}
	}

	rel, err := install.Run(chart, values)
	if err != nil {
		return err
	}

	if rel.Info.Status != release.StatusDeployed {
		return fmt.Errorf("install failed")
	}

	if c.After != "" {
		fmt.Println("apply after config")
		err = cli.kube.Apply(c.After, ApplyOption{})
		if err != nil {
			cli.kube.log.Errorln("fail to apply after config, error:", err)
			return err
		}
	}
	return nil
}

func (cli *ChartClient) Uninstall(c *Chart) error {
	actionConfig, err := cli.newActionConfig(c.Namespace)
	if err != nil {
		return err
	}
	uninstall := action.NewUninstall(actionConfig)
	uninstall.Wait = true
	uninstall.Timeout = 2 * time.Minute
	uninstall.KeepHistory = false

	if c.After != "" {
		err = cli.kube.Delete(c.After, DeleteOption{})
		if err != nil {
			return err
		}
	}
	rel, err := uninstall.Run(c.ReleaseName)
	if err != nil {
		return err
	}

	fmt.Println("------ :", rel)

	if c.Before != "" {
		err = cli.kube.Delete(c.Before, DeleteOption{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (cli *ChartClient) GetRelease(name, namespace string) (*ReleaseChart, error) {
	actionConfig, err := cli.newActionConfig(namespace)
	if err != nil {
		return nil, err
	}
	get := action.NewGet(actionConfig)
	rel, err := get.Run(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrChartNotRelease
		}
		return nil, err
	}

	releaseChart := &ReleaseChart{
		Status:      rel.Info.Status.String(),
		ReleaseName: rel.Name,
		Namespace:   rel.Namespace,
	}
	return releaseChart, nil
}
