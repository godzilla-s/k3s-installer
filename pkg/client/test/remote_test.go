package test

import (
	"fmt"
	"testing"

	"github.com/godzilla-s/k3s-installer/pkg/client/remote"
	"github.com/sirupsen/logrus"
)

func TestRemoteClient(t *testing.T) {
	cli, err := remote.New(&remote.Config{
		User:     "root",
		Password: "zwj2023",
		Address:  "192.168.122.62:22",
	}, logrus.NewEntry(logrus.New()))
	if err != nil {
		t.Fatal(err)
	}
	data, err := cli.ReadFile("/etc/rancher/k3s/k3s.yaml")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(data))
}
