package testsetup

import (
	"path/filepath"

	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/corectl/tests/integration/testconfig"

	//nolint:staticcheck
	. "github.com/onsi/gomega"
)

type CorectlConfigDetails struct {
	CPlatformRepoName git.RepositoryFullname
	TemplatesRepoName git.RepositoryFullname
}

func InitCorectl(corectl *testconfig.CorectlClient) (*config.Config, *CorectlConfigDetails, error) {
	initFilePath := filepath.Join(configpath.GetCorectlHomeDir(), "corectl-init.yaml")
	err := testdata.RenderInitFile(
		initFilePath,
		testconfig.Cfg.CPlatformRepoFullId.HttpUrl(),
		testconfig.Cfg.TemplatesRepoFullId.HttpUrl(),
	)
	Expect(err).NotTo(HaveOccurred())
	return InitCorectlWithFile(corectl, initFilePath)
}

func InitCorectlWithFile(corectl *testconfig.CorectlClient, initFilePath string) (*config.Config, *CorectlConfigDetails, error) {
	_, err := corectl.Run(
		"config", "init",
		"--file", initFilePath,
		"--github-token", testconfig.Cfg.GitHubToken,
		"--github-organization", testconfig.Cfg.GitHubOrg,
		"--non-interactive",
	)
	if err != nil {
		return nil, nil, err
	}

	cfg := corectl.Config()
	cplatformRepo, err := git.OpenLocalRepository(configpath.GetCorectlCPlatformDir(), false)
	if err != nil {
		return nil, nil, err
	}
	cplatformFullname, err := git.DeriveRepositoryFullname(cplatformRepo)
	if err != nil {
		return nil, nil, err
	}
	templatesRepo, err := git.OpenLocalRepository(configpath.GetCorectlTemplatesDir(), false)
	if err != nil {
		return nil, nil, err
	}
	templateFullname, err := git.DeriveRepositoryFullname(templatesRepo)
	if err != nil {
		return nil, nil, err
	}
	return cfg, &CorectlConfigDetails{
		CPlatformRepoName: cplatformFullname,
		TemplatesRepoName: templateFullname,
	}, nil
}
