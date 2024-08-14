package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

var _ = Describe("config", Ordered, func() {
	var (
		homeDir, baseDirPath, repositoriesPath, configPath, initConfigPath string
		corectl                                                            *testconfig.CorectlClient
	)

	t := GinkgoT()

	BeforeAll(func() {
		homeDir = t.TempDir()
		baseDirPath = filepath.Join(homeDir, ".config", "corectl")
		repositoriesPath = filepath.Join(baseDirPath, "repositories")
		configPath = filepath.Join(baseDirPath, "corectl.yaml")
		corectl = testconfig.NewCorectlClient(homeDir)
		initConfigPath = filepath.Join(homeDir, "corectl-init.yaml")
		testsetup.SetupGitGlobalConfigFromCurrentToOtherHomeDir(homeDir)
	})

	Context("init", Ordered, func() {
		var (
			cfg        *config.Config
			cfgDetails *testsetup.CorectlConfigDetails
		)

		Context("errors", func() {
			AfterEach(func() {
				Expect(os.RemoveAll(baseDirPath)).ToNot(HaveOccurred())
			})
			It("returns meaningful error when cplatform repository already exist", func() {
				cloneOpt := cloneGit(testconfig.Cfg.CPlatformRepoFullId, repositoriesPath)

				_, _, err := testsetup.InitCorectl(corectl)

				Expect(err.Error()).To(ContainSubstring("Error: repoUrl \"%s.git\", target dir \"%s\": failed to clone repository: repository already exists: initialised already? run `corectl config update` to update repositories", cloneOpt.URL, cloneOpt.TargetPath))
			})
			It("returns meaningful error when templates repository already exist", func() {
				cloneOpt := cloneGit(testconfig.Cfg.TemplatesRepoFullId, repositoriesPath)

				_, _, err := testsetup.InitCorectl(corectl)

				Expect(err.Error()).To(ContainSubstring("Error: repoUrl \"%s.git\", target dir \"%s\": failed to clone repository: repository already exists: initialised already? run `corectl config update` to update repositories, alternatively to initialise again delete corectl config dir at \"%s\" and run `corectl config init`", cloneOpt.URL, cloneOpt.TargetPath, baseDirPath))
			})
			It("returns meaningful error when invalid templates remote repository configuration", func() {
				err := testdata.RenderInitFile(
					initConfigPath,
					testconfig.Cfg.CPlatformRepoFullId.RepositoryFullname.HttpUrl(),
					"",
				)
				Expect(err).NotTo(HaveOccurred())

				_, _, err = testsetup.InitCorectlWithFile(corectl, initConfigPath)

				Expect(err.Error()).To(ContainSubstring("Error: init config key \"templates\" invalid, path \"%s\": unexpected url \"\"", initConfigPath))
			})
			It("returns meaningful error when invalid cplatform remote repository configuration", func() {
				err := testdata.RenderInitFile(
					initConfigPath,
					"",
					testconfig.Cfg.TemplatesRepoFullId.RepositoryFullname.HttpUrl(),
				)
				Expect(err).NotTo(HaveOccurred())

				_, _, err = testsetup.InitCorectlWithFile(corectl, initConfigPath)

				Expect(err.Error()).To(ContainSubstring("Error: init config key \"cplatform\" invalid, path \"%s\": unexpected url \"\"", initConfigPath))
			})
		})
		Context("successfully initialise", func() {
			BeforeAll(func() {
				var err error
				cfg, cfgDetails, err = testsetup.InitCorectl(corectl)
				Expect(err).ToNot(HaveOccurred())
			})

			It("created config file", func() {
				Expect(cfg.Path()).To(Equal(configPath))
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.IsPersisted()).To(BeTrue())
				Expect(cfg.Repositories.CPlatform.Value).To(
					Equal(filepath.Join(repositoriesPath, cfgDetails.CPlatformRepoName.Name())))
				Expect(cfg.Repositories.Templates.Value).To(
					Equal(filepath.Join(repositoriesPath, cfgDetails.TemplatesRepoName.Name())))
				Expect(cfg.GitHub.Organization.Value).To(Equal(testconfig.Cfg.GitHubOrg))
				Expect(cfg.GitHub.Token.Value).To(Equal(testconfig.Cfg.GitHubToken))
				Expect(cfg.P2P.FastFeedback.DefaultEnvs.Value).To(ConsistOf(testdata.DevEnvironment()))
				Expect(cfg.P2P.ExtendedTest.DefaultEnvs.Value).To(ConsistOf(testdata.DevEnvironment()))
				Expect(cfg.P2P.Prod.DefaultEnvs.Value).To(ConsistOf(testdata.ProdEnvironment()))
			})
			It("cloned cplatform repository", func() {
				repo, err := git.OpenLocalRepository(cfg.Repositories.CPlatform.Value)
				Expect(repo).NotTo(BeNil())
				Expect(err).NotTo(HaveOccurred())
			})
			It("cloned templates repository", func() {
				repo, err := git.OpenLocalRepository(cfg.Repositories.Templates.Value)
				Expect(repo).NotTo(BeNil())
				Expect(err).NotTo(HaveOccurred())
			})
		})
	},
	)

	Context("update", Ordered, func() {
		var (
			cfg                            *config.Config
			originalCPlatformPullTimestamp time.Time
			originalTemplatesPullTimestamp time.Time
		)
		BeforeAll(func() {
			var err error
			cfg, err = config.ReadConfig(corectl.ConfigPath())
			Expect(err).NotTo(HaveOccurred())

			originalCPlatformPullTimestamp, err = getLastPullTime(cfg.Repositories.CPlatform.Value)
			Expect(err).NotTo(HaveOccurred())

			originalTemplatesPullTimestamp, err = getLastPullTime(cfg.Repositories.Templates.Value)
			Expect(err).NotTo(HaveOccurred())

			_, err = corectl.Run("config", "update")
			Expect(err).ToNot(HaveOccurred())
		})

		It("pulls configuration changes from remote configuration repositories", func() {
			updateCPlatformPullTimestamp, err := getLastPullTime(cfg.Repositories.CPlatform.Value)
			Expect(err).NotTo(HaveOccurred())
			Expect(originalCPlatformPullTimestamp.Before(updateCPlatformPullTimestamp)).To(BeTrue())

			updatedTemplatesPullTimestamp, err := getLastPullTime(cfg.Repositories.Templates.Value)
			Expect(err).NotTo(HaveOccurred())
			Expect(originalTemplatesPullTimestamp.Before(updatedTemplatesPullTimestamp)).To(BeTrue())
		})
	})
})

func getLastPullTime(repoPath string) (time.Time, error) {
	// TODO: do a real pull check instead of this hacky one
	stat, err := os.Stat(filepath.Join(repoPath, ".git", "refs", "heads", "main"))
	if err != nil {
		return time.Time{}, err
	}
	return stat.ModTime(), nil
}

func cloneGit(repoId git.GithubRepoFullId, dstPath string) git.CloneOp {
	cloneOpt := git.CloneOp{
		URL:        repoId.HttpUrl(),
		TargetPath: filepath.Join(dstPath, repoId.RepositoryFullname.Name()),
		Auth:       git.UrlTokenAuthMethod(testconfig.Cfg.GitHubToken),
	}
	_, err := git.CloneToLocalRepository(cloneOpt)
	Expect(err).NotTo(HaveOccurred())
	return cloneOpt
}
