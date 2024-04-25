package testsetup

import (
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	. "github.com/onsi/gomega"
	"path/filepath"
)

type CorectlConfigDetails struct {
	CPlatformRepoName git.RepositoryFullname
	TemplatesRepoName git.RepositoryFullname
}

func InitCorectl(corectl *testconfig.CorectlClient) (*config.Config, CorectlConfigDetails) {
	initFilePath := filepath.Join(corectl.HomeDir(), "corectl-init.yaml")
	err := testdata.RenderInitFile(
		initFilePath,
		testconfig.Cfg.CPlatformRepoFullId.RepositoryFullname.HttpUrl(),
		testconfig.Cfg.TemplatesRepoFullId.RepositoryFullname.HttpUrl(),
	)
	Expect(err).NotTo(HaveOccurred())
	err = corectl.Run(
		"config", "init",
		"--file", initFilePath,
		"--github-token", testconfig.Cfg.GitHubToken,
		"--github-organization", testconfig.Cfg.GitHubOrg,
		"--tenant", testconfig.Cfg.Tenant,
		"--nonint",
	)
	Expect(err).NotTo(HaveOccurred())

	cfg := corectl.Config()
	cplatformRepo, err := git.OpenLocalRepository(cfg.Repositories.CPlatform.Value)
	Expect(err).NotTo(HaveOccurred())
	cplatformFullname, err := git.DeriveRepositoryFullname(cplatformRepo)
	Expect(err).NotTo(HaveOccurred())

	templatesRepo, err := git.OpenLocalRepository(cfg.Repositories.Templates.Value)
	Expect(err).NotTo(HaveOccurred())
	templateFullname, err := git.DeriveRepositoryFullname(templatesRepo)
	Expect(err).NotTo(HaveOccurred())

	return cfg, CorectlConfigDetails{
		CPlatformRepoName: cplatformFullname,
		TemplatesRepoName: templateFullname,
	}
}
