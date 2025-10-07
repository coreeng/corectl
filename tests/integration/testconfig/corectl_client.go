package testconfig

import (
	"os"
	"os/exec"
	"bufio"
	"bytes"
	"fmt"

	"github.com/coreeng/corectl/pkg/cmdutil/config"

	//nolint:staticcheck
	. "github.com/onsi/gomega"
)

type CorectlClient struct {
	binaryPath string
	homeDir    string
	env        []string
}

func NewCorectlClient(homeDir string) *CorectlClient {
	env := os.Environ()
	return &CorectlClient{
		binaryPath: Cfg.CoreCTLBinary,
		homeDir:    homeDir,
		env:        env,
	}
}

func (c *CorectlClient) RunInDir(dir string, args ...string) (string, error) {
	cmd := exec.Command(c.binaryPath, args...)
	cmd.Env = c.env
	cmd.Dir = dir

	outBuf := bytes.Buffer{}
	outWriter := bufio.NewWriter(&outBuf)
	cmd.Stdout = outWriter
	cmd.Stderr = outWriter

	if err := cmd.Run(); err != nil {
		println(outBuf.String())
		return "", fmt.Errorf("%s: %w", outBuf.String(), err)
	}
	return outBuf.String(), nil
}

func (c *CorectlClient) Run(args ...string) (string, error) {
	return c.RunInDir(c.homeDir, args...)
}

func (c *CorectlClient) HomeDir() string {
	return c.homeDir
}

func (c *CorectlClient) ConfigPath() string {
	configPath, err := config.Path()
	Expect(err).NotTo(HaveOccurred())
	return configPath
}

func (c *CorectlClient) Config() *config.Config {
	cfg, err := config.ReadConfig(c.ConfigPath())
	Expect(err).NotTo(HaveOccurred())
	return cfg
}
