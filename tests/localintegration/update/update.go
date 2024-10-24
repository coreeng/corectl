package update

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/coreeng/corectl/tests/integration/testconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/otiai10/copy"
	"github.com/phuslu/log"
)

func runCommand(dir string, name string, args ...string) (string, string, error) {
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
	return stdout.String(), stderr.String(), err
}

var _ = Describe("update", Ordered, func() {
	updateCmd := func(args []string) (string, string, error) {
		tmpFile, err := os.CreateTemp("", "corectl-update-test")
		Expect(err).ShouldNot(HaveOccurred())

		tmpPath := tmpFile.Name()

		err = tmpFile.Close()
		Expect(err).ShouldNot(HaveOccurred())
		parentDir := filepath.Dir(tmpPath)
		fileName := "./" + filepath.Base(tmpPath)
		err = copy.Copy(testconfig.Cfg.CoreCTLBinary, tmpPath)
		Expect(err).ShouldNot(HaveOccurred())
		err = os.Chmod(tmpPath, os.FileMode(0755))
		Expect(err).ShouldNot(HaveOccurred())

		initialVersion, stderr, err := runCommand(parentDir, fileName, "version", "--non-interactive")
		if err != nil {
			Fail(fmt.Sprintf("failed to get initial version: %v stdout: %s, stderr: %s", err, initialVersion, stderr))
		}
		log.Info().Msgf("Initial version: %s", initialVersion)

		updateArgs := []string{"update", "--skip-confirmation"}
		updateArgs = append(updateArgs, args...)
		stdout, stderr, err := runCommand(parentDir, fileName, updateArgs...)
		if err != nil {
			Fail(fmt.Sprintf("failed to run update: %v, args: %v, stdout: %s, stderr: %s", err, updateArgs, stdout, stderr))
		}

		updatedVersion, stderr, err := runCommand(parentDir, fileName, "version")
		if err != nil {
			Fail(fmt.Sprintf("failed to get updated version: %v, stdout: %s, stderr: %s", err, stdout, stderr))
		}
		log.Info().Msgf("Updated version: %s", updatedVersion)
		return initialVersion, updatedVersion, nil
	}

	Context("from local build", func() {
		It("updates the version to latest", func() {
			initialVersion, updatedVersion, err := updateCmd([]string{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(updatedVersion).ShouldNot(Equal(initialVersion))
		})

		It("updates to specified version", func() {
			versionTag := "v0.25.2"
			versionLine := "corectl 0.25.2 (commit: 4da4e686dc5adca21ed579374bca6a4b41f4b092) 2024-09-30T10:21:08Z amd64"
			_, updatedVersion, err := updateCmd([]string{versionTag})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(strings.Trim(updatedVersion, " \n")).Should(Equal(versionLine))
		})
	})
})
