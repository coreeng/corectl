package testconfig

import (
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/testdata"
	"os"
)

const (
	testGitHubOrg = "coreeng-dev"
)

var Cfg = ConfigType{
	GitHubOrg: testGitHubOrg,
	Tenant:    testdata.DefaultTenant(),
}

type ConfigType struct {
	CoreCTLBinary       string
	GitHubToken         string
	GitHubOrg           string
	Tenant              string
	CPlatformRepoFullId git.GithubRepoFullId
	TemplatesRepoFullId git.GithubRepoFullId
}

func init() {
	Cfg.CoreCTLBinary = os.Getenv("TEST_CORECTL_BINARY")
	Cfg.GitHubToken = os.Getenv("TEST_GITHUB_TOKEN")

	if Cfg.CoreCTLBinary == "" {
		panic("TEST_CORECTL_BINARY env is missing")
	}
	corectlStats, err := os.Stat(Cfg.CoreCTLBinary)
	if err != nil {
		panic("TEST_CORECTL_BINARY env is invalid: " + err.Error())
	}
	if corectlStats.IsDir() {
		panic("TEST_CORECTL_BINARY is not a binary")
	}
	if Cfg.GitHubToken == "" {
		panic("TEST_GITHUB_TOKEN is missing")
	}
}
