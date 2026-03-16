package testconfig

import (
	"github.com/coreeng/corectl/pkg/git"
	"os"
)

const (
	testGitHubOrg = "coreeng-dev"
)

var Cfg = ConfigType{
	GitHubOrg: testGitHubOrg,
	// Integration tests assume ADR-65 OU/DU model.
	// Tenant is the OrgUnit used as the owner when creating applications.
	// DeliveryUnit is an existing DU in the base cplatform repo used for p2p export tests.
	Tenant:       "parent",
	DeliveryUnit: "default-tenant",
}

type ConfigType struct {
	CoreCTLBinary       string
	GitHubToken         string
	GitHubOrg           string
	Tenant              string
	DeliveryUnit        string
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
