package node

import (
	"fmt"
	"github.com/godzilla-s/k3s-installer/pkg/config"
)

type k3sConfig struct {
	ClusterInit            *bool    `yaml:"cluster-init,omitempty"`
	WriteKubeConfigMode    int      `yaml:"write-kubeconfig-mode,omitempty"`
	Token                  string   `yaml:"token,omitempty"`
	FlannelBackend         string   `yaml:"flannel-backend,omitempty"`
	Server                 string   `yaml:"server,omitempty"`
	TlsSAN                 []string `yaml:"tls-san,omitempty"`
	DisableCloudController *bool    `yaml:"disable-cloud-controller,omitempty"`
	DisableKubeProxy       *bool    `yaml:"disable-kube-proxy,omitempty"`
	DisableNetworkPolicy   *bool    `yaml:"disable-network-policy,omitempty"`
	EnableDocker           *bool    `yaml:"docker,omitempty"`
	Disable                []string `yaml:"disable,omitempty"`
}

type registryConfig struct {
	Mirrors map[string]mirror `yaml:"mirrors"`
	Configs map[string]conf   `yaml:"configs"`
}

type mirror struct {
	Endpoints []string `yaml:"endpoint"`
}

type conf struct {
	ConfigTLS configTls `yaml:"tls"`
}

type configTls struct {
	CertFile string `yaml:"cert_file"`
}

func toConfig(isMaster bool, conf *config.Config) *k3sConfig {
	if !isMaster {
		return &k3sConfig{Server: fmt.Sprintf("https://%s:6443", conf.Settings.HaIP)}
	}

	kc := &k3sConfig{
		WriteKubeConfigMode: 644,
		FlannelBackend:      "vxlan",
		TlsSAN:              []string{conf.Settings.HaIP},
	}
	if conf.Settings.Config.DisableFlannel {
		kc.FlannelBackend = "none"
	}
	if conf.Settings.Config.DisableServiceLB {
		kc.Disable = append(kc.Disable, "servicelb")
	}
	if conf.Settings.Config.DisableTraefik {
		kc.Disable = append(kc.Disable, "traefik")
	}
	if conf.Settings.Config.DisableLocalPath {
		kc.Disable = append(kc.Disable, "local-storage")
	}
	return kc
}

func toRegistriesConfig() *registryConfig {
	rc := &registryConfig{
		Mirrors: map[string]mirror{
			"docker.io": {
				Endpoints: []string{"https://dockerhub.io"},
			},
		},
		Configs: make(map[string]conf),
	}

	return rc
}
