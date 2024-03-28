package testsetup

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	. "github.com/onsi/gomega"
	"os"
	"path/filepath"
)

func SetupGitConfigFromCurrentToOtherHomeDir(destHomeDir string) {
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

func saveAsNewConfig(destHomeDir string, existingConfig *config.Config) {
	newConfig := config.NewConfig()
	newConfig.User = existingConfig.User
	newConfig.Author = existingConfig.Author
	newConfig.Committer = existingConfig.Committer
	marshalledCfg, err := newConfig.Marshal()
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(
		filepath.Join(destHomeDir, ".gitconfig"),
		marshalledCfg,
		0o644,
	)).To(Succeed())
}
