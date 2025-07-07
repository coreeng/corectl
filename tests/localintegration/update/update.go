package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreeng/corectl/pkg/logger"
	"github.com/coreeng/corectl/pkg/shell"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/otiai10/copy"
)

var _ = ginkgo.Describe("update", ginkgo.Ordered, func() {
	updateCmd := func(args []string) (string, string, error) {
		tmpFile, err := os.CreateTemp("", "corectl-update-test")
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		tmpPath := tmpFile.Name()

		err = tmpFile.Close()
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		parentDir := filepath.Dir(tmpPath)
		fileName := "./" + filepath.Base(tmpPath)
		err = copy.Copy(testconfig.Cfg.CoreCTLBinary, tmpPath)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = os.Chmod(tmpPath, os.FileMode(0755))
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		initialVersion, stderr, err := shell.RunCommand(parentDir, fileName, "version", "--non-interactive")
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("failed to get initial version: %v stdout: %s, stderr: %s", err, initialVersion, stderr))
		}
		logger.Info().Msgf("Initial version: %s", initialVersion)

		updateArgs := []string{"update", "--skip-confirmation"}
		updateArgs = append(updateArgs, args...)
		stdout, stderr, err := shell.RunCommand(parentDir, fileName, updateArgs...)
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("failed to run update: %v, args: %v, stdout: %s, stderr: %s", err, updateArgs, stdout, stderr))
		}

		updatedVersion, stderr, err := shell.RunCommand(parentDir, fileName, "version")
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("failed to get updated version: %v, stdout: %s, stderr: %s", err, stdout, stderr))
		}
		logger.Info().Msgf("Updated version: %s", updatedVersion)
		return initialVersion, updatedVersion, nil
	}

	ginkgo.Context("from local build", func() {
		ginkgo.It("updates the version to latest", func() {
			initialVersion, updatedVersion, err := updateCmd([]string{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(updatedVersion).ShouldNot(gomega.Equal(initialVersion))
		})

		ginkgo.It("updates to specified version", func() {
			versionTag := "v0.25.2"
			versionLine := "corectl 0.25.2 (commit: 4da4e686dc5adca21ed579374bca6a4b41f4b092) 2024-09-30T10:21:08Z amd64"
			_, updatedVersion, err := updateCmd([]string{versionTag})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(strings.Trim(updatedVersion, " \n")).Should(gomega.Equal(versionLine))
		})
	})
})
