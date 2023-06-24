package remote

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/godzilla-s/k3s-installer/pkg/utils"
	"github.com/pkg/sftp"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type Client struct {
	address string
	ssh     *ssh.Client
	sftp    *sftp.Client
	auth    *ssh.ClientConfig
	log     *logrus.Entry
	SystemAction
}

type SystemInfo struct {
	NumberCPU     int
	Memory        utils.Capacity // Gi
	Storage       utils.Capacity
	Hostname      string
	KernelVersion utils.KernelVersion
}

type Config struct {
	Address  string
	User     string
	Password string
	Timeout  time.Duration
}
type SystemAction interface {
	Install(object string) error
	Uninstall(objects []string) error
	StopFirewall() error
}

type CommandOption func(*ssh.Session)

func New(conf *Config, log *logrus.Entry) (*Client, error) {
	auth := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password(conf.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         conf.Timeout,
	}
	if auth.Timeout == 0 {
		auth.Timeout = 15 * time.Second
	}

	client := &Client{
		address: conf.Address,
		auth:    auth,
		log:     log,
	}

	err := client.connect()
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) connect() error {
	sshClient, err := ssh.Dial("tcp", c.address, c.auth)
	if err != nil {
		return err
	}
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return err
	}
	c.ssh = sshClient
	c.sftp = sftpClient
	c.SystemAction = &centosClient{Client: c}
	return nil
}

func (c *Client) execCommand(cmd string, options ...CommandOption) ([]byte, error) {
	if c.ssh == nil {
		return nil, fmt.Errorf("ssh client not init")
	}
	sess, err := c.ssh.NewSession()
	if err != nil {
		return nil, err
	}

	return sess.CombinedOutput(cmd)
}

func (c *Client) GetSystemInfo() (*SystemInfo, error) {
	// get cpu core number
	cmd := "cat /proc/cpuinfo |grep processor |wc -l"
	output, err := c.execCommand(cmd)
	if err != nil {
		return nil, err
	}
	cpuNumber, err := strconv.ParseInt(string(bytes.TrimRight(output, "\n")), 0, 10)
	if err != nil {
		return nil, err
	}

	output, err = c.execCommand(`free -h | awk 'NR==2{print $2}'`)
	if err != nil {
		return nil, err
	}
	memorySize, err := utils.ParseCapacity(string(bytes.TrimRight(output, "\n")))
	if err != nil {
		return nil, err
	}

	output, err = c.execCommand("hostname")
	if err != nil {
		return nil, err
	}
	hostname := string(bytes.TrimRight(output, "\n"))
	return &SystemInfo{
		NumberCPU: int(cpuNumber),
		Memory:    memorySize,
		Hostname:  hostname,
	}, nil
}

func (c *Client) WriteFile(file string, data []byte, override bool) error {
	if !override {
		_, err := c.sftp.Stat(file)
		if err == nil {
			return ErrFileDoesExist
		}
	}

	baseDir := filepath.Dir(file)
	fi, err := c.sftp.Stat(baseDir)
	if err != nil {
		err = c.sftp.MkdirAll(baseDir)
		if err != nil {
			return err
		}
	} else {
		if !fi.IsDir() {
			return fmt.Errorf("cannot create, basedir is a file")
		}
	}

	f, err := c.sftp.Create(file)
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	return err
}

func (c *Client) ReadFile(file string) ([]byte, error) {
	err := c.connect()
	if err != nil {
		fmt.Println("connect fail:", err)
		return nil, err
	}
	f, err := c.sftp.Open(file)
	if err != nil {
		return nil, err
	}
	defer c.sftp.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) Copy(local, target string, override bool) error {
	fi, err := os.Stat(local)
	if err != nil {
		return err
	}
	// file
	if !fi.IsDir() {
		err = c.CopyFile(local, target, override)
		if err != nil {
			return err
		}
		return nil
	}

	// directory
	entries, err := os.ReadDir(local)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			err = c.Copy(filepath.Join(local, entry.Name()), filepath.Join(target, entry.Name()), override)
			if err != nil {
				return err
			}
		} else {
			localFile := filepath.Join(local, entry.Name())
			targetFile := filepath.Join(target, entry.Name())
			fmt.Println(localFile, targetFile)
			err = c.CopyFile(localFile, targetFile, override)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Client) CopyFile(local, target string, override bool) error {
	fr, err := os.Open(local)
	if err != nil {
		c.log.Printf("fail to read local file, file: %s, error: %v", local, err)
		return err
	}
	fri, _ := fr.Stat()
	defer fr.Close()
	fi, err := c.sftp.Stat(target)
	if err == nil {
		if fi.IsDir() {
			return fmt.Errorf("target is a directory")
		}
		if !override {
			return ErrFileDoesExist
		}
		fw, err := c.sftp.OpenFile(target, os.O_CREATE|os.O_RDWR)
		if err != nil {
			return fmt.Errorf("open or create file fail: %v", err)
		}
		defer fw.Close()
		_, err = io.Copy(fw, fr)
		if err != nil {
			return fmt.Errorf("copy fail: %v", err)
		}
		c.sftp.Chmod(target, fri.Mode())
		return nil
	}
	baseDir := filepath.Dir(target)
	// fmt.Println("baseDir:", baseDir)
	fi, err = c.sftp.Stat(baseDir)
	if err != nil {
		err = c.sftp.MkdirAll(baseDir)
		if err != nil {
			c.log.Errorf("fail to create base directory, dir: %s", baseDir)
			return err
		}
	}

	fw, err := c.sftp.OpenFile(target, os.O_CREATE|os.O_RDWR)
	if err != nil {
		c.log.Errorf("fail to create or open fail, file: %s, err: %v", target, err)
		return err
	}
	defer fw.Close()
	_, err = io.Copy(fw, fr)
	if err != nil {
		c.log.Errorf("copy file fail, local: %s, target:%s, error: %v", local, target, err)
		return err
	}
	c.sftp.Chmod(target, fri.Mode())
	c.log.Printf("copy file success, local: %s, target: %s", local, target)
	return nil
}

func (c *Client) Remove(target string) error {
	fi, err := c.sftp.Stat(target)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		output, err := c.execCommand(fmt.Sprintf("rm -rf %s", target))
		if err != nil {
			c.log.Errorf("remove direcotry fail, dir: %s, error: %v, message: %s", target, err, output)
			return err
		}
		c.log.Panicf("remove directory successful")
	}
	output, err := c.execCommand(fmt.Sprintf("rm -f %s", target))
	if err != nil {
		c.log.Errorf("remove file fail, file: %s, error: %v, message: %s", target, err, output)
		return err
	}
	c.log.Panicf("remove file successful")
	return nil
}

func (c *Client) StartK3S(isMaster bool) error {
	cmd := "install.sh"
	if !isMaster {
		cmd = "install.sh"
	}

	output, err := c.execCommand(cmd)
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

func (c *Client) RestartK3S(isMaster bool) error {
	cmd := "systemctl restart k3s"
	if !isMaster {
		cmd = "systemctl restart k3s-agent"
	}
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

func (c *Client) IsK3SRunning(isMaster bool) error {
	cmd := "systemctl is-active k3s"
	if !isMaster {
		cmd = "systemctl is-active k3s-agent"
	}
	output, _ := c.execCommand(cmd)
	status := string(bytes.TrimRight(output, "\n"))
	c.log.Printf("k3s status: %s", status)
	switch status {
	case "active":
		return nil
	case "inactive":
		return ErrK3SNotRunning
	default:
		return fmt.Errorf("unknown status %s", status)
	}
}

func (c *Client) InstallK3S(isMaster bool) error {
	cmd := "INSTALL_K3S_SKIP_DOWNLOAD=true install.sh"
	if !isMaster {
		cmd = "INSTALL_K3S_SKIP_DOWNLOAD=true INSTALL_K3S_EXEC=agent install.sh"
	}
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

func (c *Client) UninstallK3S(isMaster bool) error {
	cmd := "k3s-uninstall.sh"
	if !isMaster {
		cmd = "k3s-agent-uninstall.sh"
	}

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
