package core

import (
	"github.com/godzilla-s/k3s-installer/pkg/client/kube"
	"github.com/godzilla-s/k3s-installer/pkg/config"
	"github.com/godzilla-s/k3s-installer/pkg/node"
	"github.com/godzilla-s/k3s-installer/pkg/utils"
	"github.com/sirupsen/logrus"
)

func Uninstall(conf *config.Config, log *logrus.Logger) error {
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
	return installer.uninstall()
}

func (i *Installer) uninstall() error {
	for _, s := range i.steps {
		err := s.uninstall()
		if err != nil {
			return err
		}
	}
	return nil
}
