package core

import (
	"github.com/godzilla-s/k3s-installer/pkg/config"
	"github.com/godzilla-s/k3s-installer/pkg/utils"
	"github.com/sirupsen/logrus"
)

type Installer struct {
	cluster *cluster
	msg     *utils.Print
}

func Install(conf *config.Config, log *logrus.Logger) error {
	cluster, err := newCluster(conf, log)
	if err != nil {
		return err
	}

	for _, s := range cluster.steps {
		err := s.install()
		if err != nil {
			return err
		}
	}
	return nil
}
