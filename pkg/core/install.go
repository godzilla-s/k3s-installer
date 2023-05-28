package core

import (
	"fmt"
	"github.com/godzilla-s/k3s-installer/pkg/client/kube"
	"github.com/godzilla-s/k3s-installer/pkg/config"
	"github.com/godzilla-s/k3s-installer/pkg/node"
	"github.com/godzilla-s/k3s-installer/pkg/utils"
	"github.com/sirupsen/logrus"
)

type Installer struct {
	initNode     *node.Node
	clusterNodes []*node.Node
	steps        []step
	clusterIP    string
	kubeClient   *kube.Client
	chartClient  *kube.ChartClient
	msg          *utils.Print
	log          *logrus.Logger
}

type step interface {
	install() error
	uninstall() error
}

func Install(conf *config.Config, log *logrus.Logger) error {
	installer := &Installer{
		log:       log,
		msg:       utils.NewMessage(),
		clusterIP: conf.Settings.HaIP,
	}

	for _, n := range conf.Nodes {
		clusterNode, err := node.New(n, conf, log)
		if err != nil {
			log.Errorf("fail to create node <%s>", n.Address)
			return err
		}
		if installer.initNode == nil {
			if n.Role == "master" {
				clusterNode.SetClusterInit()
				installer.initNode = clusterNode
			}
		}
		log.Infof("cluster node <%s> init success", n.Address)
		installer.clusterNodes = append(installer.clusterNodes, clusterNode)
	}

	for _, s := range conf.Steps {
		switch s.Type {
		case "k3s":
			installer.steps = append(installer.steps, &k3sStep{Installer: installer})
		case "chart":
			var charts []*kube.Chart
			for _, name := range s.Charts {
				chart := kube.ToChart(conf.Charts[name])
				charts = append(charts, chart)
			}
			installer.steps = append(installer.steps, &chartsStep{Installer: installer, charts: charts})
		}
	}

	return installer.install()
}

func (i *Installer) install() error {
	for _, s := range i.steps {
		err := s.install()
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *Installer) installChart(c *kube.Chart) error {
	if i.chartClient == nil {
		err := i.initChartClient()
		if err != nil {
			return err
		}
	}
	rel, err := i.chartClient.GetRelease(c.ReleaseName, c.Namespace)
	if err != nil && err != kube.ErrChartNotRelease {
		i.log.Errorf("get  release chart <%s> fail, error: %v", c.ReleaseName, err)
		return err
	}
	if err == kube.ErrChartNotRelease {
		// TODO:
		i.msg.Message("install chart")
		return i.chartClient.Install(c)
	}

	if rel.Status == "" {

	}
	return nil
}

func (i *Installer) initKubeClient() error {
	kubeConfig, err := i.initNode.GetKubeConfig()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://%s:6443", i.clusterIP)
	kubeClient, err := kube.New(url, kubeConfig, i.log)
	if err != nil {
		return err
	}

	i.kubeClient = kubeClient
	return nil
}

func (i *Installer) initChartClient() error {
	if i.kubeClient == nil {
		err := i.initKubeClient()
		if err != nil {
			return err
		}
	}
	i.chartClient = i.kubeClient.NewChartClient()
	return nil
}
