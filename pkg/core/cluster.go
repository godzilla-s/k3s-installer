package core

import (
	"fmt"

	"github.com/godzilla-s/k3s-installer/pkg/client/kube"
	"github.com/godzilla-s/k3s-installer/pkg/config"
	"github.com/godzilla-s/k3s-installer/pkg/node"
	"github.com/godzilla-s/k3s-installer/pkg/utils"
	"github.com/sirupsen/logrus"
)

type cluster struct {
	initNode     *node.Node
	clusterNodes []*node.Node
	steps        []step
	clusterIP    string
	kubeClient   *kube.Client
	chartClient  *kube.ChartClient
	log          *logrus.Logger
	msg          *utils.Print
}

type step interface {
	install() error
	uninstall() error
}

func newCluster(conf *config.Config, log *logrus.Logger) (*cluster, error) {
	cluster := &cluster{
		log:       log,
		msg:       utils.NewMessage(),
		clusterIP: conf.Settings.HaIP,
	}

	for _, master := range conf.Settings.Cluster.Master {
		isClusterInit := cluster.initNode == nil
		masterNode, err := node.New(conf.Nodes[master], conf, true, isClusterInit, log)
		if err != nil {
			log.Errorf("fail to init node <%s>, error: %v", conf.Nodes[master].Address, err)
			return nil, err
		}
		if isClusterInit {
			cluster.initNode = masterNode
		}
		cluster.clusterNodes = append(cluster.clusterNodes, masterNode)
	}

	for _, worker := range conf.Settings.Cluster.Worker {
		workerNode, err := node.New(conf.Nodes[worker], conf, false, false, log)
		if err != nil {
			log.Errorf("fail to init node <%s>, error: %v", conf.Nodes[worker].Address, err)
			return nil, err
		}
		cluster.clusterNodes = append(cluster.clusterNodes, workerNode)
	}

	for _, s := range conf.Steps {
		switch s.Type {
		case "k3s":
			cluster.steps = append(cluster.steps, &k3sStep{cluster: cluster, msg: cluster.msg})
		case "chart":
			var charts []*kube.Chart
			for _, cname := range s.Charts {
				chart := kube.ToChart(conf.Charts[cname])
				charts = append(charts, chart)
			}
			cluster.steps = append(cluster.steps, &chartStep{cluster: cluster, charts: charts, msg: cluster.msg})
		}
	}
	return cluster, nil
}

func (c *cluster) installChart(chart *kube.Chart) error {
	c.initNode.Test()
	if c.chartClient == nil {
		err := c.initChartClient()
		if err != nil {
			fmt.Println("===== fail to init")
			return err
		}
	}

	rel, err := c.chartClient.GetRelease(chart.ReleaseName, chart.Namespace)
	if err != nil && err != kube.ErrChartNotRelease {
		c.log.Errorf("get release chart <%s> fail, error: %v", chart.ReleaseName, err)
		return err
	}
	if err == kube.ErrChartNotRelease {
		// TODO:
		return c.chartClient.Install(chart)
	}

	if rel.Status == "deployed" {
		// chart has been released
		c.log.Printf("chart <%s> has been release, namespace: %s", chart.ReleaseName, chart.Namespace)

		err = c.kubeClient.Apply(chart.After, kube.ApplyOption{})
		if err != nil {
			c.log.Printf("apply %s fail , error: %v", chart.After, err)
			return err
		}
		return nil
	}

	// delete and reinstall
	err = c.chartClient.Uninstall(chart)
	if err != nil {
		return err
	}

	err = c.chartClient.Install(chart)
	if err != nil {
		c.log.Errorf("chart <%s> installed failed, namepace: %s, error: %v", chart.ReleaseName, chart.Namespace, err)
	}
	return err
}

func (c *cluster) uninstallChart(chart *kube.Chart) error {
	if c.chartClient == nil {
		err := c.initChartClient()
		if err != nil {
			return err
		}
	}
	rel, err := c.chartClient.GetRelease(chart.ReleaseName, chart.Namespace)
	if err != nil && err != kube.ErrChartNotRelease {
		c.log.Errorf("get release chart <%s> fail, error: %v", chart.ReleaseName, err)
		return err
	}
	if err == kube.ErrChartNotRelease {
		c.log.Warnf("chart <%s> may has been release, namespace: %s, error: %v", chart.ReleaseName, chart.Namespace, err)
		return nil
	}
	_ = rel
	err = c.chartClient.Uninstall(chart)
	if err != nil {
		c.log.Errorf("chart <%s> uninstalled failed, namespace: %s, error: %v", chart.ReleaseName, chart.Namespace, err)
	}
	return err
}

func (c *cluster) initKubeClient() error {
	kubeConfig, err := c.initNode.GetKubeConfig()
	if err != nil {
		fmt.Println("fail to get kube config")
		return err
	}

	url := fmt.Sprintf("https://%s:6443", c.clusterIP)
	kubeClient, err := kube.New(url, kubeConfig, c.log)
	if err != nil {
		return err
	}

	c.kubeClient = kubeClient
	return nil
}

func (c *cluster) initChartClient() error {
	if c.kubeClient == nil {
		err := c.initKubeClient()
		if err != nil {
			fmt.Println("inti kube client error")
			return err
		}
	}
	c.chartClient = c.kubeClient.NewChartClient()
	return nil
}
