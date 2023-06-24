package core

import (
	"github.com/godzilla-s/k3s-installer/pkg/config"
	"github.com/sirupsen/logrus"
)

func Uninstall(conf *config.Config, log *logrus.Logger) error {
	cluster, err := newCluster(conf, log)
	if err != nil {
		return err
	}

	for i := len(cluster.steps) - 1; i >= 0; i-- {
		err := cluster.steps[i].uninstall()
		if err != nil {
			return err
		}
	}
	return nil
}
