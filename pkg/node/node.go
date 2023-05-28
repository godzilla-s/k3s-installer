package node

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"path/filepath"

	"github.com/godzilla-s/k3s-installer/pkg/client/remote"
	"github.com/godzilla-s/k3s-installer/pkg/config"
)

type Node struct {
	remote             *remote.Client
	address            string
	isMaster           bool
	isClusterInit      bool
	installingPackages []Package
	preloadImages      []loadImage
	log                *logrus.Entry
	config             *k3sConfig
	registries         *registryConfig
}

func New(n *config.Node, conf *config.Config, log *logrus.Logger) (*Node, error) {
	logEntry := logrus.NewEntry(log).WithFields(map[string]interface{}{
		"host": n.Address,
		"role": n.Role,
	})
	remoteCli, err := remote.New(&remote.Config{
		Address:  fmt.Sprintf("%s:%d", n.Address, n.SSHPort),
		User:     "root",
		Password: n.RootPassword,
	}, logEntry)
	if err != nil {
		return nil, err
	}
	isMaster := n.Role == "master"
	node := &Node{
		address:    n.Address,
		remote:     remoteCli,
		log:        logEntry,
		config:     toConfig(isMaster, conf),
		isMaster:   isMaster,
		registries: toRegistriesConfig(),
	}
	for _, imgName := range n.PreloadImages {
		img := conf.Images[imgName]
		node.preloadImages = append(node.preloadImages, loadImage{path: img.Path})
	}

	for _, pkgName := range n.InstallPackages {
		pkg := toPackage(pkgName, conf.Packages[pkgName], node)
		node.installingPackages = append(node.installingPackages, pkg)
	}

	return node, nil
}

func (n *Node) Name() string {
	return n.address
}

func (n *Node) SetClusterInit() {
	n.isClusterInit = true
}

func (n *Node) Prepare() error {
	if err := n.installPackages(); err != nil {
		return err
	}
	if err := n.loadImages(); err != nil {
		return err
	}
	return nil
}

func (n *Node) Cleanup() error {
	return nil
}

func (n *Node) checkSystem() error {
	return nil
}

func (n *Node) isK3SRunning() error {
	return n.remote.IsK3SRunning(n.isMaster)
}

func (n *Node) installPackages() error {
	for _, pkg := range n.installingPackages {
		if err := pkg.install(); err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) loadImages() error {
	for _, img := range n.preloadImages {
		target := filepath.Join("/var/lib/rancher/k3s/agent/images", filepath.Base(img.path))
		err := n.remote.CopyFile(img.path, target, false)
		if err != nil {
			n.log.Errorf("fail to upload images, path: %s, error: %v", img.path, err)
			return err
		}
	}
	return nil
}
func (n *Node) uninstallPackage() error {
	for _, pkg := range n.installingPackages {
		if err := pkg.uninstall(); err != nil {
			return err
		}
	}
	return nil
}
