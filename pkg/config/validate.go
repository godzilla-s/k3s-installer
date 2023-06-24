package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

func (c *Config) validate() error {
	if err := c.validateSettings(); err != nil {
		return err
	}
	if err := c.validateCharts(); err != nil {
		return err
	}
	if err := c.validateNodes(); err != nil {
		return err
	}
	if err := c.validatePackages(); err != nil {
		return err
	}
	if err := c.validateImages(); err != nil {
		return err
	}
	if err := c.validateSteps(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateSettings() error {
	if c.Settings.HaIP == "" {
		return fmt.Errorf("invalid settings: missing ha IP address")
	}
	if c.Settings.RootPath == "" {
		c.Settings.RootPath = "./"
	}
	for _, name := range c.Settings.Cluster.Master {
		if _, ok := c.Nodes[name]; !ok {
			return fmt.Errorf("invalid settings: missing master node <%s> defined", name)
		}
	}
	for _, name := range c.Settings.Cluster.Worker {
		if _, ok := c.Nodes[name]; !ok {
			return fmt.Errorf("invalid settings: missing worker node <%s> defined", name)
		}
	}

	return nil
}
func (c *Config) validateCharts() error {

	for name, chart := range c.Charts {
		if chart.Namespace == "" {
			chart.Namespace = "default"
		}
		if chart.Version == "" {
			return fmt.Errorf("invalid chart <%s>: missing version", name)
		}
		if chart.Path == "" {
			chart.Path = filepath.Join("charts", name)
		}

		chartPkg := filepath.Join(c.Settings.RootPath, chart.Path, fmt.Sprintf("%s-%s.tgz", name, chart.Version))
		fi, err := os.Stat(chartPkg)
		if err != nil {
			return fmt.Errorf("invalid chart <%s>: %v", name, err)
		}
		if fi.IsDir() {
			return fmt.Errorf("invalid chart <%s>: chart package not exist", name)
		}
		chart.Path = chartPkg
		if chart.ReleaseName == "" {
			chart.ReleaseName = name
		}
	}

	return nil
}

func (c *Config) validatePackages() error {
	for name, pkg := range c.Packages {
		switch pkg.Type {
		case PackageFile:
			if pkg.Path == "" {
				return fmt.Errorf("invalid package <%s>: missing path", name)
			}
			pkg.Path = filepath.Join(c.Settings.RootPath, pkg.Path)
			fi, err := os.Stat(pkg.Path)
			if err != nil {
				return fmt.Errorf("invalid package <%s>: %v", name, err)
			}
			if fi.IsDir() {
				return fmt.Errorf("invalid package <%s>: not a exectable file", name)
			}
		case PackageDirectory:
			if pkg.Path == "" {
				return fmt.Errorf("invalid package <%s>: missing path", name)
			}
			if pkg.TargetPath == "" {
				return fmt.Errorf("invalid package <%s>: missing target path", name)
			}
			pkg.Path = filepath.Join(c.Settings.RootPath, pkg.Path)
			fi, err := os.Stat(pkg.Path)
			if err != nil {
				return err
			}

			if !fi.IsDir() {
				return fmt.Errorf("invalid package <%s>: not a directory", name)
			}
		case PackageRPM:
			if pkg.Path == "" {
				return fmt.Errorf("invalid package <%s>: missing path", name)
			}
			pkg.Path = filepath.Join(c.Settings.RootPath, pkg.Path)
		case PackageDockerService:
			if pkg.Path == "" {
				return fmt.Errorf("invalid package <%s>: missing path", name)
			}
			pkg.Path = filepath.Join(c.Settings.RootPath, pkg.Path)
		default:
			return fmt.Errorf("invalid package <%s>: unknown type '%s'", name, pkg.Type)
		}
	}
	return nil
}

func (c *Config) validateNodes() error {
	for name, node := range c.Nodes {
		if node.Address == "" {
			return fmt.Errorf("invalid node <%s>: missing host", name)
		}
		if node.RootPassword == "" {
			return fmt.Errorf("invalid node <%s>: missing root password", name)
		}
		decPwd, err := base64.StdEncoding.DecodeString(node.RootPassword)
		if err != nil {
			return fmt.Errorf("invalid node <%s>: invalid password", name)
		}
		node.RootPassword = string(decPwd)
		if node.OS == "" {
			node.OS = "centos"
		}

		if node.SSHPort == 0 {
			node.SSHPort = 22
		}
		for _, pkgName := range node.InstallPackages {
			if _, ok := c.Packages[pkgName]; !ok {
				return fmt.Errorf("invalid node <%s>: missing package %s", name, pkgName)
			}
		}
		for _, imageName := range node.PreloadImages {
			if _, ok := c.Images[imageName]; !ok {
				return fmt.Errorf("invalid node <%s>: missing image %s", name, imageName)
			}
		}
	}
	return nil
}

func (c *Config) validateImages() error {
	for name, img := range c.Images {
		imagePath := filepath.Join(c.Settings.RootPath, img.Path)
		fi, err := os.Stat(imagePath)
		if err != nil {
			return fmt.Errorf("invalid image <%s>: %v", name, err)
		}
		if fi.IsDir() {
			return fmt.Errorf("invalid image <%s>: image is directory", name)
		}
		img.Path = imagePath
	}
	return nil
}

func (c *Config) validateSteps() error {
	for _, step := range c.Steps {
		switch step.Type {
		case "k3s":
		case "chart":
			for _, chartName := range step.Charts {
				if _, ok := c.Charts[chartName]; !ok {
					return fmt.Errorf("invalid step <%s>: missing chart <%s>", step.Type, chartName)
				}
			}
		}
	}
	return nil
}
