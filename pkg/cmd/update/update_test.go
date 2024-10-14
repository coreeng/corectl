package update

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/phuslu/log"
)

func TestUpdate(t *testing.T) {
	// log.DefaultLogger.SetLevel(log.PanicLevel)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Corectl update e2e")
}

var _ = Describe("corectl update", func() {
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

func updateCmd(args []string) (string, string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("Error getting current working directory: %v", err)
	}

	var repoRoot string
	repoRoot, err = findRepoRoot(dir)
	if err != nil {
		return "", "", fmt.Errorf("Error getting current repository root directory: %v", err)
	}
	_, err = RunCommand(repoRoot, "make", "build")
	if err != nil {
		return "", "", fmt.Errorf("Failed to compile corectl: %v", err)
	}
	log.Info().Msg("corectl compiled successfully.")

	initialVersion, err := RunCommand(repoRoot, "./corectl", "version")
	if err != nil {
		return "", "", fmt.Errorf("Failed to get initial version: %v", err)
	}
	log.Info().Msgf("Initial version: %s", initialVersion)

	updateArgs := []string{"update"}
	updateArgs = append(updateArgs, args...)
	_, err = RunCommand(repoRoot, "./corectl", updateArgs...)
	if err != nil {
		return "", "", fmt.Errorf("Failed to run update: %v", err)
	}

	updatedVersion, err := RunCommand(repoRoot, "./corectl", "version")
	if err != nil {
		return "", "", fmt.Errorf("Failed to get updated version: %v", err)
	}
	log.Info().Msgf("Updated version: %s", updatedVersion)
	return initialVersion, updatedVersion, nil
}

func findRepoRoot(dir string) (string, error) {
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

func RunCommand(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	log.Info().Msgf("Running %s in %s", name, dir)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Debug().Msgf("err %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	return stdout.String(), err
}
