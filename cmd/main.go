package main

import (
	"github.com/godzilla-s/k3s-installer/pkg/config"
	"github.com/godzilla-s/k3s-installer/pkg/core"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var (
	configFile string
)

var rootCmd = &cobra.Command{}

var installCmd = &cobra.Command{
	Short: "install",
	Use:   "install",
	Run: func(cmd *cobra.Command, args []string) {
		logger := newLogger("")
		conf, err := config.Parse(configFile)
		if err != nil {
			logger.Errorf("fail to parse config, error: %v", err)
			return
		}

		err = core.Install(conf, logger)
		if err != nil {
			logger.Errorf("install fail, error: %v", err)
			os.Exit(1)
		}
	},
}

var uninstallCmd = &cobra.Command{
	Short: "uninstall",
	Use:   "uninstall",
	Run: func(cmd *cobra.Command, args []string) {
		logger := newLogger("")
		conf, err := config.Parse(configFile)
		if err != nil {
			logger.Errorf("fail to parse config, error: %v", err)
			return
		}

		err = core.Uninstall(conf, logger)
		if err != nil {
			logger.Errorf("install fail, error: %v", err)
			os.Exit(1)
		}
	},
}

func newLogger(prefix string) *logrus.Logger {
	// os.Mkdir(".log", )
	return logrus.New()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "f", "", "config file")
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
