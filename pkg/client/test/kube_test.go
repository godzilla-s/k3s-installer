package test

import (
	"testing"

	"github.com/godzilla-s/k3s-installer/pkg/client/kube"
	"github.com/godzilla-s/k3s-installer/pkg/client/remote"
)

func TestKubeClient_(t *testing.T) {
	cli, err := remote.New(&remote.Config{
		User:     "root",
		Password: "zwj2023",
		Address:  "192.168.122.62:22",
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := cli.ReadFile("/etc/rancher/k3s/k3s.yaml")
	if err != nil {
		t.Fatal(err)
	}

	kc, err := kube.New("https://192.168.122.62:6443", data)
	if err != nil {
		t.Fatal(err)
	}

	err = kc.GetIngress()
	if err != nil {
		t.Fatal(err)
	}
}
