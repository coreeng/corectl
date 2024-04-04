package testsetup

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	. "github.com/onsi/gomega"
	"os"
	"path/filepath"
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
	Expect(err).NotTo(HaveOccurred())
	saveAsNewConfig(destHomeDir, existingConfig)
}

func SetupGitRepoConfigFromOtherRepo(sourceRepoDir string, destRepo *git.Repository) {
	sourceRepo, err := git.PlainOpenWithOptions(sourceRepoDir, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	Expect(err).NotTo(HaveOccurred())
	sourceCfg, err := sourceRepo.Config()
	Expect(err).NotTo(HaveOccurred())
	destCfg, err := destRepo.Config()
	Expect(err).NotTo(HaveOccurred())
	copyConfigData(sourceCfg, destCfg)
	Expect(destRepo.SetConfig(destCfg)).To(Succeed())
}

func saveAsNewConfig(destHomeDir string, existingConfig *config.Config) {
	newConfig := config.NewConfig()
	copyConfigData(existingConfig, newConfig)
	marshalledCfg, err := newConfig.Marshal()
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(
		filepath.Join(destHomeDir, ".gitconfig"),
		marshalledCfg,
		0o644,
	)).To(Succeed())
}

func copyConfigData(sourceCfg *config.Config, destCfg *config.Config) {
	destCfg.User = sourceCfg.User
	destCfg.Author = sourceCfg.Author
	destCfg.Committer = sourceCfg.Committer
}
