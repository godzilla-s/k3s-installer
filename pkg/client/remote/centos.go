package remote

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
)

type centosClient struct {
	*Client
}

func (c *centosClient) Install(pkgDir string) error {
	entries, err := c.sftp.ReadDir(pkgDir)
	if err != nil {
		return err
	}

	var rpms []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasSuffix(entry.Name(), ".rpm") {
			rpms = append(rpms, filepath.Join(pkgDir, entry.Name()))
		}
	}

	if len(rpms) == 0 {
		//
		return nil
	}

	return c.install(rpms)
}

func (c *centosClient) Uninstall(pkgs []string) error {
	installedRPMs, err := c.listInstalled(pkgs)
	if err != nil {
		return err
	}
	if len(installedRPMs) == 0 {
		return nil
	}

	return c.uninstall(installedRPMs)
}

func (c *centosClient) StopFirewall() error {
	return nil
}

func (c *centosClient) install(rpms []string) error {
	var installingRPMs []string
	for _, rpm := range rpms {
		if strings.HasSuffix(rpm, ".rpm") {
			installingRPMs = append(installingRPMs, strings.TrimRight(rpm, ".rpm"))
		}
	}
	installedRPM, err := c.listInstalled(rpms)
	if err != nil {
		return err
	}
	if len(installedRPM) == len(installingRPMs) {
		return nil
	}
	cmd := fmt.Sprintf("yum localinstall -y %s", strings.Join(rpms, " "))
	output, err := c.execCommand(cmd)
	if err != nil {
		for _, line := range bytes.Split(output, []byte("\n")) {
			c.log.Errorln(string(line))
		}
		return err
	}
	for _, line := range bytes.Split(output, []byte("\n")) {
		c.log.Println(string(line))
	}
	return nil
}

func (c *centosClient) uninstall(rpms []string) error {
	cmd := fmt.Sprintf("yum remove %s", strings.Join(rpms, " "))
	output, err := c.execCommand(cmd)
	if err != nil {
		for _, line := range bytes.Split(output, []byte("\n")) {
			c.log.Errorln(string(line))
		}
		return err
	}
	for _, line := range bytes.Split(output, []byte("\n")) {
		c.log.Println(string(line))
	}
	return nil
}

func (c *centosClient) update(rpms []string) error {
	cmd := fmt.Sprintf("yum update %s", strings.Join(rpms, " "))
	output, err := c.execCommand(cmd)
	if err != nil {
		for _, line := range bytes.Split(output, []byte("\n")) {
			c.log.Errorln(string(line))
		}
		return err
	}
	for _, line := range bytes.Split(output, []byte("\n")) {
		c.log.Println(string(line))
	}
	return nil
}

func (c *centosClient) listInstalled(rpms []string) ([]string, error) {
	output, err := c.execCommand("rpm -qa")
	if err != nil {
		return nil, err
	}
	installedRPMs := make(map[string]struct{})
	output = bytes.TrimRight(output, "\n")
	for _, line := range bytes.Split(output, []byte("\n")) {
		installedRPMs[string(line)] = struct{}{}
	}
	var givenInstalledRPMs []string
	for _, rpm := range rpms {
		if _, ok := installedRPMs[rpm]; ok {
			givenInstalledRPMs = append(givenInstalledRPMs, rpm)
		}
	}
	return givenInstalledRPMs, nil
}
