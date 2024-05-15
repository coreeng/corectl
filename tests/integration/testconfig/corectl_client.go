package testconfig

import (
	"bufio"
	"bytes"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	. "github.com/onsi/gomega"
	"os"
	"os/exec"
)

type CorectlClient struct {
	binaryPath string
	homeDir    string
	env        []string
}

func NewCorectlClient(homeDir string) *CorectlClient {
	env := os.Environ()
	env = append(env, "HOME="+homeDir)
	return &CorectlClient{
		binaryPath: Cfg.CoreCTLBinary,
		homeDir:    homeDir,
		env:        env,
	}
}

func (c *CorectlClient) Run(args ...string) error {
	cmd := exec.Command(c.binaryPath, args...)
	cmd.Env = c.env
	cmd.Dir = c.homeDir

	outBuf := bytes.Buffer{}
	outWriter := bufio.NewWriter(&outBuf)
	cmd.Stdout = outWriter
	cmd.Stderr = outWriter
	if err := cmd.Run(); err != nil {
		println(outBuf.String())
		return err
	}
	return nil
}

func (c *CorectlClient) HomeDir() string {
	return c.homeDir
}

func (c *CorectlClient) ConfigPath() string {
	originalHome := os.Getenv("HOME")
	defer func() {
		Expect(os.Setenv("HOME", originalHome)).To(Succeed())
	}()
	Expect(os.Setenv("HOME", c.homeDir)).To(Succeed())
	configPath, err := config.Path()
	Expect(err).NotTo(HaveOccurred())
	return configPath
}

func (c *CorectlClient) Config() *config.Config {
	cfg, err := config.ReadConfig(c.ConfigPath())
	Expect(err).NotTo(HaveOccurred())
	return cfg
}
