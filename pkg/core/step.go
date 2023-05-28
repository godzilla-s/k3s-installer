package core

import (
	"fmt"
	"github.com/godzilla-s/k3s-installer/pkg/client/kube"
	"github.com/godzilla-s/k3s-installer/pkg/node"
	"sync"
	"sync/atomic"
)

type k3sStep struct {
	waitGroup sync.WaitGroup
	*Installer
}

func (k *k3sStep) install() error {
	k.msg.Step("install k3s")
	var failed atomic.Int32
	for _, clusterNode := range k.clusterNodes {
		k.waitGroup.Add(1)
		go func(n *node.Node) {
			defer k.waitGroup.Done()
			k.log.Printf("install k3s on <%s>", n.Name())
			if err := n.Prepare(); err != nil {
				k.log.Printf("cluster node <%s> fail to prepared, error: %v", n.Name(), err)
				failed.Add(1)
				return
			}
			if err := n.InstallK3S(); err != nil {
				k.log.Printf("cluster node <%s> install failed, error: %v", n.Name(), err)
				failed.Add(1)
				return
			}
		}(clusterNode)
	}

	k.waitGroup.Wait()

	if failed.Load() > 0 {
		k.log.Errorf("k3s cluster install failed")
		return fmt.Errorf("cluster installed fail")
	}

	return nil
}

func (k *k3sStep) uninstall() error {
	k.msg.Step("uninstall k3s")
	var failed atomic.Int32
	for _, clusterNode := range k.clusterNodes {
		k.waitGroup.Add(1)
		go func(n *node.Node) {
			defer k.waitGroup.Done()
			if err := n.UninstallK3S(); err != nil {
				k.log.Errorf("fail to uninstall k3s: %v", err)
				failed.Add(1)
				return
			}
			if err := n.Cleanup(); err != nil {
				return
			}
			k.log.Printf("cluster node <%s> uninstall success", n.Name())
		}(clusterNode)
	}
	k.waitGroup.Wait()
	if failed.Load() > 0 {
		k.log.Errorf("k3s cluster install failed")
		return fmt.Errorf("cluster installed fail")
	}
	return nil
}

type chartsStep struct {
	charts []*kube.Chart
	*Installer
}

func (c *chartsStep) install() error {
	c.msg.Step("install charts")
	for _, chart := range c.charts {
		c.log.Infof("install chart <%s>, namespace: %s", chart.ReleaseName, chart.Namespace)
		err := c.installChart(chart)
		if err != nil {
			c.log.Errorf("fail to install chart <%s>, namespace: %s, error: %v", chart.ReleaseName, chart.Namespace, err)
			return err
		}
	}
	return nil
}

func (c *chartsStep) uninstall() error {
	return nil
}

type manifestStep struct {
	manifests []string
	*Installer
}

func (m *manifestStep) install() error {
	m.msg.Step("Install manifests")
	if m.kubeClient == nil {
		err := m.initKubeClient()
		if err != nil {
			return err
		}
	}
	for _, yamlFile := range m.manifests {
		m.msg.Message("install <%s>", yamlFile)
		err := m.kubeClient.Apply(yamlFile, kube.ApplyOption{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *manifestStep) uninstall() error {
	m.msg.Step("Uninstall manifests")
	if m.kubeClient == nil {
		err := m.initKubeClient()
		if err != nil {
			return err
		}
	}
	for _, yamlFile := range m.manifests {
		m.msg.Message("uninstall <%s>", yamlFile)
		err := m.kubeClient.Delete(yamlFile, kube.DeleteOption{})
		if err != nil {
			return err
		}
	}
	return nil
}
