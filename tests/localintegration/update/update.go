package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreeng/corectl/tests/localintegration/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/phuslu/log"
)

var _ = Describe("update", Ordered, func() {
	findRepoRoot := func(dir string) (string, error) {
		for {
			gitPath := filepath.Join(dir, ".git")
			if _, err := os.Stat(gitPath); err == nil {
				return dir, nil
			}

			parentDir := filepath.Dir(dir)
			if parentDir == dir {
				return "", fmt.Errorf(".git directory not found")
			}
			dir = parentDir
		}
	}

	updateCmd := func(args []string) (string, string, error) {
		dir, err := os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("error getting current working directory: %v", err)
		}

		var repoRoot string
		repoRoot, err = findRepoRoot(dir)
		if err != nil {
			return "", "", fmt.Errorf("error getting current repository root directory: %v", err)
		}
		_, err = utils.RunCommand(repoRoot, "make", "build")
		if err != nil {
			return "", "", fmt.Errorf("failed to compile corectl: %v", err)
		}
		log.Info().Msg("corectl compiled successfully.")

		initialVersion, err := utils.RunCommand(repoRoot, "./corectl", "version")
		if err != nil {
			return "", "", fmt.Errorf("failed to get initial version: %v", err)
		}
		log.Info().Msgf("Initial version: %s", initialVersion)

		updateArgs := []string{"update"}
		updateArgs = append(updateArgs, args...)
		_, err = utils.RunCommand(repoRoot, "./corectl", updateArgs...)
		if err != nil {
			return "", "", fmt.Errorf("failed to run update: %v", err)
		}

		updatedVersion, err := utils.RunCommand(repoRoot, "./corectl", "version")
		if err != nil {
			return "", "", fmt.Errorf("failed to get updated version: %v", err)
		}
		log.Info().Msgf("Updated version: %s", updatedVersion)
		return initialVersion, updatedVersion, nil
	}

	Context("from local build", func() {
		It("updates the version to latest", func() {
			initialVersion, updatedVersion, err := updateCmd([]string{})
			if err != nil {
				Fail(err.Error())
			}
			Expect(updatedVersion).ShouldNot(Equal(initialVersion))
		})

		It("updates to specified version", func() {
			versionTag := "v0.25.2"
			versionLine := "corectl 0.25.2 (commit: 4da4e686dc5adca21ed579374bca6a4b41f4b092) 2024-09-30T10:21:08Z amd64"
			_, updatedVersion, err := updateCmd([]string{versionTag})
			if err != nil {
				Fail(err.Error())
			}
			Expect(strings.Trim(updatedVersion, " \n")).Should(Equal(versionLine))
		})
	})
})
