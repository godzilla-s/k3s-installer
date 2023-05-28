package node

import (
	"fmt"
	"github.com/godzilla-s/k3s-installer/pkg/client/remote"
	"gopkg.in/yaml.v3"
	"time"

	"github.com/godzilla-s/k3s-installer/pkg/utils"
)

func (n *Node) InstallK3S() error {
	if n.isClusterInit {
		return n.initCluster()
	}

	err := n.remote.IsK3SRunning(n.isMaster)
	if err == nil {
		n.log.Printf("k3s is running, continue to next")
		return nil
	}

	if err == remote.ErrK3SNotRunning {
		err = n.remote.RestartK3S(n.isMaster)
		if err != nil {
			return err
		}
		err = utils.Clock(2*time.Minute, 2*time.Second, func() error {
			return n.remote.IsK3SRunning(n.isMaster)
		})
		if err != nil {
			n.log.Errorf("wait for k3s running timeout, error: %v", err)
			return err
		}
		return nil
	}

	// prepare k3s config
	err = n.writeConfig()
	if err != nil {
		n.log.Errorf("fail to prepare k3s config: %v", err)
		return err
	}

	// prepare k3s private registry
	err = n.writeRegistryConfig()
	if err != nil {
		n.log.Errorf("fail to write registries config: %v", err)
		return err
	}

	err = n.installK3S()
	if err != nil {
		n.log.Errorf("fail to install k3s, error: %v", err)
		return err
	}
	err = utils.Clock(2*time.Minute, 2*time.Second, func() error {
		return n.remote.IsK3SRunning(n.isMaster)
	})
	if err != nil {
		n.log.Errorf("wait for k3s running timeout, error: %v", err)
		return err
	}
	n.log.Printf("install k3s success")
	return nil
}

func (n *Node) initCluster() error {
	err := n.isK3SRunning()
	if err == nil {
		// get token if k3s is running
		token := n.getClusterToken()
		if token == "" {
			return fmt.Errorf("missing cluster token")
		}
		setToken(token)
		return nil
	}

	if err == remote.ErrK3SNotRunning {
		err = n.remote.RestartK3S(n.isMaster)
		if err != nil {
			return err
		}
		err = utils.Clock(2*time.Minute, 2*time.Second, func() error {
			return n.remote.IsK3SRunning(n.isMaster)
		})
		if err != nil {
			n.log.Errorf("wait for k3s running timeout, error: %v", err)
			return err
		}
		return nil
	}

	// prepare k3s config
	err = n.writeConfig()
	if err != nil {
		n.log.Errorf("fail to prepare k3s config: %v", err)
		return err
	}

	// prepare k3s private registry
	err = n.writeRegistryConfig()
	if err != nil {
		n.log.Errorf("fail to write registries config: %v", err)
		return err
	}

	err = n.remote.InstallK3S(n.isMaster)
	if err != nil {
		n.log.Errorf("fail to install k3s")
		return err
	}

	err = utils.Clock(2*time.Minute, 3*time.Second, func() error {
		return n.isK3SRunning()
	})
	if err != nil {
		return fmt.Errorf("timeout for waiting k3s running")
	}
	return nil
}

func (n *Node) UninstallK3S() error {
	err := n.isK3SRunning()
	if err != nil && err != remote.ErrK3SNotRunning {
		return err
	}
	if err == remote.ErrK3SNotRunning {
		n.log.Errorf("k3s server is not running")
		return nil
	}

	err = n.uninstallK3S()
	if err != nil {
		n.log.Errorf("fail to uninstall k3s")
		return err
	}
	return nil
}

func (n *Node) writeConfig() error {
	if n.isClusterInit {
		clusterInit := true
		n.config.ClusterInit = &clusterInit
	}
	data, err := yaml.Marshal(n.config)
	if err != nil {
		return err
	}
	return n.remote.WriteFile("/etc/rancher/k3s/config.yaml", data, true)
}

func (n *Node) writeRegistryConfig() error {
	data, err := yaml.Marshal(n.registries)
	if err != nil {
		return err
	}
	return n.remote.WriteFile("/etc/rancher/k3s/registries.yaml", data, true)
}

func (n *Node) getClusterToken() string {
	data, err := n.remote.ReadFile("/var/lib/rancher/k3s/server/token")
	if err != nil {
		return ""
	}
	return string(data)
}

func (n *Node) installK3S() error {
	return n.remote.InstallK3S(n.isMaster)
}

func (n *Node) uninstallK3S() error {
	return n.remote.UninstallK3S(n.isMaster)
}

func (n *Node) GetKubeConfig() ([]byte, error) {
	return n.remote.ReadFile("/etc/rancher/k3s/k3s.yaml")
}
