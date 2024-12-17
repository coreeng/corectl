package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreeng/corectl/pkg/logger"
	"github.com/coreeng/corectl/pkg/shell"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/otiai10/copy"
)

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

		initialVersion, stderr, err := shell.RunCommand(parentDir, fileName, "version", "--non-interactive")
		if err != nil {
			Fail(fmt.Sprintf("failed to get initial version: %v stdout: %s, stderr: %s", err, initialVersion, stderr))
		}
		logger.Info().Msgf("Initial version: %s", initialVersion)

		updateArgs := []string{"update", "--skip-confirmation"}
		updateArgs = append(updateArgs, args...)
		stdout, stderr, err := shell.RunCommand(parentDir, fileName, updateArgs...)
		if err != nil {
			Fail(fmt.Sprintf("failed to run update: %v, args: %v, stdout: %s, stderr: %s", err, updateArgs, stdout, stderr))
		}

		updatedVersion, stderr, err := shell.RunCommand(parentDir, fileName, "version")
		if err != nil {
			Fail(fmt.Sprintf("failed to get updated version: %v, stdout: %s, stderr: %s", err, stdout, stderr))
		}
		logger.Info().Msgf("Updated version: %s", updatedVersion)
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
