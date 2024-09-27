package testsetup

import (
	"path/filepath"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	. "github.com/onsi/gomega"
)

type CorectlConfigDetails struct {
	CPlatformRepoName git.RepositoryFullname
	TemplatesRepoName git.RepositoryFullname
}

func InitCorectl(corectl *testconfig.CorectlClient) (*config.Config, *CorectlConfigDetails, error) {
	initFilePath := filepath.Join(corectl.HomeDir(), "corectl-init.yaml")
	err := testdata.RenderInitFile(
		initFilePath,
		testconfig.Cfg.CPlatformRepoFullId.RepositoryFullname.HttpUrl(),
		testconfig.Cfg.TemplatesRepoFullId.RepositoryFullname.HttpUrl(),
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
		"--nonint",
	)
	if err != nil {
		return nil, nil, err
	}

	cfg := corectl.Config()
	cplatformRepo, err := git.OpenLocalRepository(cfg.Repositories.CPlatform.Value, false)
	if err != nil {
		return nil, nil, err
	}
	cplatformFullname, err := git.DeriveRepositoryFullname(cplatformRepo)
	if err != nil {
		return nil, nil, err
	}
	templatesRepo, err := git.OpenLocalRepository(cfg.Repositories.Templates.Value, false)
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
