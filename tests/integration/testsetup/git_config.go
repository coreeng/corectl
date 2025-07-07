package testsetup

import (
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/onsi/gomega"
)

func SetupGitGlobalConfigFromCurrentToOtherHomeDir(destHomeDir string) {
	localRepository, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err == nil {
		existingConfig, err := localRepository.ConfigScoped(config.SystemScope)
		if err == nil {
			saveAsNewConfig(destHomeDir, existingConfig)
			return
		}
	}
	existingConfig, err := config.LoadConfig(config.GlobalScope)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	saveAsNewConfig(destHomeDir, existingConfig)
}

func SetupGitRepoConfigFromOtherRepo(sourceRepoDir string, destRepo *git.Repository) {
	sourceRepo, err := git.PlainOpenWithOptions(sourceRepoDir, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	sourceCfg, err := sourceRepo.Config()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	destCfg, err := destRepo.Config()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	copyConfigData(sourceCfg, destCfg)
	gomega.Expect(destRepo.SetConfig(destCfg)).To(gomega.Succeed())
}

func saveAsNewConfig(destHomeDir string, existingConfig *config.Config) {
	newConfig := config.NewConfig()
	copyConfigData(existingConfig, newConfig)
	marshalledCfg, err := newConfig.Marshal()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(os.WriteFile(
		filepath.Join(destHomeDir, ".gitconfig"),
		marshalledCfg,
		0o644,
	)).To(gomega.Succeed())
}

func copyConfigData(sourceCfg *config.Config, destCfg *config.Config) {
	destCfg.User = sourceCfg.User
	destCfg.Author = sourceCfg.Author
	destCfg.Committer = sourceCfg.Committer
}
