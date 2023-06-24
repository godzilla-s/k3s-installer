package config

const (
	PackageFile          = "file"
	PackageDirectory     = "directory"
	PackageDockerService = "docker"
	PackageRPM           = "rpm"
	PackageKernel        = "kernel"
)

const (
	DefaultK3SConfigPath = "/etc/rancher/k3s"

	DefaultK3SLoadImagePath = "/var/lib/rancher/k3s/agent/images"
)
