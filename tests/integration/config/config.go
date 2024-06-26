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
		homeDir string
		corectl *testconfig.CorectlClient
	)
	t := GinkgoT()

	BeforeAll(func() {
		homeDir = t.TempDir()
		corectl = testconfig.NewCorectlClient(homeDir)
		testsetup.SetupGitGlobalConfigFromCurrentToOtherHomeDir(homeDir)
	})

	Context("init", Ordered, func() {
		var (
			cfg        *config.Config
			cfgDetails testsetup.CorectlConfigDetails
		)
		BeforeAll(func() {
			cfg, cfgDetails = testsetup.InitCorectl(corectl)
		})

		It("created config file", func() {
			Expect(cfg.Path()).To(Equal(filepath.Join(homeDir, ".config", "corectl", "corectl.yaml")))
			Expect(cfg).NotTo(BeNil())
			Expect(cfg.IsPersisted()).To(BeTrue())
			Expect(cfg.Repositories.CPlatform.Value).To(
				Equal(filepath.Join(homeDir, ".config", "corectl", "repositories", cfgDetails.CPlatformRepoName.Name())))
			Expect(cfg.Repositories.Templates.Value).To(
				Equal(filepath.Join(homeDir, ".config", "corectl", "repositories", cfgDetails.TemplatesRepoName.Name())))
			Expect(cfg.GitHub.Organization.Value).To(Equal(testconfig.Cfg.GitHubOrg))
			Expect(cfg.GitHub.Token.Value).To(Equal(testconfig.Cfg.GitHubToken))
			Expect(cfg.Tenant.Value).To(Equal(testconfig.Cfg.Tenant))
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

			Expect(corectl.Run(
				"config", "update",
			)).To(Succeed())
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
