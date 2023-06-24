package node

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/godzilla-s/k3s-installer/pkg/client/remote"
	"github.com/godzilla-s/k3s-installer/pkg/config"
)

type Package interface {
	install() error
	uninstall() error
}

type loadImage struct {
	name string
	path string
}

func toPackage(name string, pkg *config.Package, node *Node) Package {
	switch pkg.Type {
	case config.PackageFile:
		target := filepath.Join("/usr/local/bin", filepath.Base(pkg.Path))
		return &file{name: name, localPath: pkg.Path, target: target, Node: node}
	case config.PackageDirectory:
		return &directory{name: name, localPath: pkg.Path, targetPath: pkg.TargetPath, Node: node}
	case config.PackageRPM:
		return &rpm{name: name, localPath: pkg.Path, Node: node}
	case config.PackageKernel:
		return &kernel{name: name, localPath: pkg.Path, Node: node}
	default:
		panic("invalid package type")
	}
}

type file struct {
	name      string
	localPath string
	target    string
	*Node
}

func (b *file) install() error {
	b.log.Printf("install binary <%s>", b.name)
	err := b.remote.CopyFile(b.localPath, b.target, true)
	if err != nil && err != remote.ErrFileDoesExist {
		b.log.Errorf("install binary <%s> fail", b.name)
		return err
	}
	b.log.Printf("install binary <%s> success", b.name)
	return nil
}

func (b *file) uninstall() error {
	b.log.Println("uninstall binary <%s>", b.name)
	return b.remote.Remove(b.localPath)
}

type directory struct {
	name       string
	localPath  string
	targetPath string
	*Node
}

func (d *directory) install() error {
	d.log.Printf("install directory <%s>", d.name)
	err := d.remote.Copy(d.localPath, d.targetPath, true)
	if err != nil {
		return err
	}
	return nil
}

func (d *directory) uninstall() error {
	d.log.Printf("uninstall directory <%s>", d.name)
	err := d.remote.Remove(d.targetPath)
	if err != nil {
		return err
	}
	return nil
}

type rpm struct {
	name      string
	localPath string
	*Node
}

func (r *rpm) install() error {
	r.log.Printf("install rpm <%s>", r.name)
	targetPath := filepath.Join("/tmp", r.name)
	err := r.remote.Copy(r.localPath, targetPath, true)
	if err != nil {
		return err
	}

	// defer r.remote.Remove(targetPath)

	err = r.remote.Install(targetPath)
	if err != nil {
		return err
	}
	return nil
}

func (r *rpm) uninstall() error {
	r.log.Printf("uninstall rpm <%s>", r.name)
	dirEntries, err := os.ReadDir(r.localPath)
	if err != nil {
		return err
	}

	var rpms []string
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}
		if strings.HasSuffix(dirEntry.Name(), ".rpm") {
			rpms = append(rpms, strings.TrimRight(dirEntry.Name(), ".rpm"))
		}
	}
	err = r.remote.Uninstall(rpms)
	if err != nil {
		return err
	}
	return nil
}

type deb struct{}

type kernel struct {
	name      string
	localPath string
	*Node
}

func (k *kernel) install() error {
	targetPath := filepath.Join("/tmp", k.name)
	err := k.remote.Copy(k.localPath, targetPath, true)
	if err != nil {
		return err
	}

	err = k.remote.Install(targetPath)
	if err != nil {
		return err
	}

	return nil
}

func (k *kernel) uninstall() error {
	k.log.Println("unsupported kernel uninstallation")
	return nil
}
