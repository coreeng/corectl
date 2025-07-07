package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/coreeng/corectl/pkg/cmdutil/configpath"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("config", ginkgo.Ordered, func() {
	var (
		homeDir, configPath, initConfigPath string
		corectl                             *testconfig.CorectlClient
	)

	t := ginkgo.GinkgoT()

	ginkgo.BeforeAll(func() {
		homeDir = t.TempDir()
		configpath.SetCorectlHome(homeDir)
		configPath = filepath.Join(homeDir, "corectl.yaml")
		corectl = testconfig.NewCorectlClient(homeDir)
		initConfigPath = filepath.Join(homeDir, "corectl-init.yaml")
		testsetup.SetupGitGlobalConfigFromCurrentToOtherHomeDir(homeDir)
	})

	ginkgo.Context("init", ginkgo.Ordered, func() {
		var (
			cfg *config.Config
		)

		ginkgo.Context("errors", func() {
			ginkgo.AfterEach(func() {
				gomega.Expect(os.RemoveAll(filepath.Join(homeDir, "repositories"))).ToNot(gomega.HaveOccurred())
			})
			ginkgo.It("returns meaningful error when cplatform repository already exist", func() {
				cloneOpt := cloneGit(testconfig.Cfg.CPlatformRepoFullId, configpath.GetCorectlCPlatformDir())

				_, _, err := testsetup.InitCorectl(corectl)

				gomega.Expect(err.Error()).To(gomega.ContainSubstring("Error: repoUrl \"%s.git\", target dir \"%s\": failed to clone repository: repository already exists: initialised already? run `corectl config update` to update repositories", cloneOpt.URL, cloneOpt.TargetPath))
			})
			ginkgo.It("returns meaningful error when templates repository already exist", func() {
				cloneOpt := cloneGit(testconfig.Cfg.TemplatesRepoFullId, configpath.GetCorectlTemplatesDir())

				_, _, err := testsetup.InitCorectl(corectl)

				gomega.Expect(err.Error()).To(gomega.ContainSubstring("Error: repoUrl \"%s.git\", target dir \"%s\": failed to clone repository: repository already exists: initialised already? run `corectl config update` to update repositories, alternatively to initialise again delete corectl config dir at \"%s\" and run `corectl config init`", cloneOpt.URL, cloneOpt.TargetPath, homeDir))
			})
			ginkgo.It("returns meaningful error when invalid templates remote repository configuration", func() {
				err := testdata.RenderInitFile(
					initConfigPath,
					testconfig.Cfg.CPlatformRepoFullId.HttpUrl(),
					"",
				)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				_, _, err = testsetup.InitCorectlWithFile(corectl, initConfigPath)

				gomega.Expect(err.Error()).To(gomega.ContainSubstring("Error: init config key \"templates\" invalid, path \"%s\": unexpected url \"\"", initConfigPath))
			})
			ginkgo.It("returns meaningful error when invalid cplatform remote repository configuration", func() {
				err := testdata.RenderInitFile(
					initConfigPath,
					"",
					testconfig.Cfg.TemplatesRepoFullId.HttpUrl(),
				)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				_, _, err = testsetup.InitCorectlWithFile(corectl, initConfigPath)

				gomega.Expect(err.Error()).To(gomega.ContainSubstring("Error: init config key \"cplatform\" invalid, path \"%s\": unexpected url \"\"", initConfigPath))
			})
		})
		ginkgo.Context("successfully initialise", func() {
			ginkgo.BeforeAll(func() {
				var err error
				cfg, _, err = testsetup.InitCorectl(corectl)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			})

			ginkgo.It("created config file", func() {
				gomega.Expect(cfg.Path()).To(gomega.Equal(configPath))
				gomega.Expect(cfg).NotTo(gomega.BeNil())
				gomega.Expect(cfg.IsPersisted()).To(gomega.BeTrue())
				gomega.Expect(cfg.GitHub.Organization.Value).To(gomega.Equal(testconfig.Cfg.GitHubOrg))
				gomega.Expect(cfg.GitHub.Token.Value).To(gomega.Equal(testconfig.Cfg.GitHubToken))
				gomega.Expect(cfg.P2P.FastFeedback.DefaultEnvs.Value).To(gomega.ConsistOf(testdata.DevEnvironment()))
				gomega.Expect(cfg.P2P.ExtendedTest.DefaultEnvs.Value).To(gomega.ConsistOf(testdata.DevEnvironment()))
				gomega.Expect(cfg.P2P.Prod.DefaultEnvs.Value).To(gomega.ConsistOf(testdata.ProdEnvironment()))
			})
			ginkgo.It("cloned cplatform repository", func() {
				repo, err := git.OpenLocalRepository(configpath.GetCorectlCPlatformDir(), false)
				gomega.Expect(repo).NotTo(gomega.BeNil())
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			})
			ginkgo.It("cloned templates repository", func() {
				repo, err := git.OpenLocalRepository(configpath.GetCorectlTemplatesDir(), false)
				gomega.Expect(repo).NotTo(gomega.BeNil())
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			})
		})
	},
	)

	ginkgo.Context("update", ginkgo.Ordered, func() {
		var (
			originalCPlatformPullTimestamp time.Time
			originalTemplatesPullTimestamp time.Time
		)
		ginkgo.BeforeAll(func() {
			var err error

			originalCPlatformPullTimestamp, err = getLastPullTime(configpath.GetCorectlCPlatformDir())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			originalTemplatesPullTimestamp, err = getLastPullTime(configpath.GetCorectlTemplatesDir())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			_, err = corectl.Run("config", "update", "--non-interactive")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

		ginkgo.It("pulls configuration changes from remote configuration repositories", func() {
			updateCPlatformPullTimestamp, err := getLastPullTime(configpath.GetCorectlCPlatformDir())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(originalCPlatformPullTimestamp.Before(updateCPlatformPullTimestamp)).To(gomega.BeTrue())

			updatedTemplatesPullTimestamp, err := getLastPullTime(configpath.GetCorectlTemplatesDir())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(originalTemplatesPullTimestamp.Before(updatedTemplatesPullTimestamp)).To(gomega.BeTrue())
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
		TargetPath: filepath.Join(dstPath),
		Auth:       git.UrlTokenAuthMethod(testconfig.Cfg.GitHubToken),
	}
	_, err := git.CloneToLocalRepository(cloneOpt)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return cloneOpt
}
