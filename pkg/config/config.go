package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Chart struct {
	Path        string        `yaml:"path,omitempty"`
	ReleaseName string        `yaml:"releaseName"`
	Version     string        `yaml:"version"`
	Namespace   string        `yaml:"namespace"`
	Timeout     time.Duration `yaml:"timeout"`
	SetValues   []string      `yaml:"setValues"`
}

type Package struct {
	Type       string `yaml:"type"`
	Path       string `yaml:"path"`
	TargetPath string `yaml:"targetPath"`
}

type Image struct {
	Path string `yaml:"path"`
}

type Settings struct {
	RootPath   string `yaml:"rootPath"`
	Config     K3SConfig
	Cluster    Cluster
	HaIP       string      `yaml:"haIP"`
	Registries []*Registry `yaml:"registries"`
}

type K3SConfig struct {
	DisableFlannel   bool `yaml:"disableFlannel"`
	DisableServiceLB bool `yaml:"disableServiceLB"`
	DisableTraefik   bool `yaml:"disableTraefik"`
	DisableLocalPath bool `yaml:"disableLocalPath"`
}

type Registry struct {
	Address  string `yaml:"address"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	CaCert   string `yaml:"cacert"`
}

type Node struct {
	Address         string      `yaml:"address"`
	SSHPort         int         `yaml:"sshPort"`
	RootPassword    string      `yaml:"rootPassword"`
	Hostname        string      `yaml:"hostname"`
	Role            string      `yaml:"role"`
	OS              string      `yaml:"os"`
	Requirement     Requirement `yaml:"requirement"`
	InstallPackages []string    `yaml:"installPackages"`
	PreloadImages   []string    `yaml:"preloadImages"`
}

type Requirement struct {
	KernelVersion string `yaml:"kernelVersion"`
	CPU           int    `yaml:"cpu"`
	Memory        string `yaml:"memory"`
	Storage       string `yaml:"storage"`
}

type Step struct {
	Type      string   `yaml:"type"`
	Charts    []string `yaml:"charts"`
	Manifests []string `yaml:"manifest"`
}

type Config struct {
	Charts   map[string]*Chart   `yaml:"charts"`
	Packages map[string]*Package `yaml:"packages"`
	Images   map[string]*Image   `yaml:"images"`
	Nodes    map[string]*Node    `yaml:"nodes"`
	Settings Settings            `yaml:"settings"`
	Steps    []*Step             `yaml:"steps"`
}

type Cluster struct {
	Master []string `yaml:"master"`
	Worker []string `yaml:"worker"`
}

func Parse(configFile string) (*Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	var config Config

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	err = config.validate()
	if err != nil {
		return nil, err
	}
	return &config, nil
}
