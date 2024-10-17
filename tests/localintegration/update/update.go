package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreeng/corectl/pkg/cmdutil/shell"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/phuslu/log"
)

var _ = Describe("update", Ordered, func() {
	updateCmd := func(args []string) (string, string, error) {
		tmpFile, err := os.CreateTemp("", "corectl-update-test")

		Expect(err).ShouldNot(HaveOccurred())
		tmpPath, err := os.Readlink(fmt.Sprintf("/proc/self/fd/%d", tmpFile.Fd()))
		tmpFile.Close()
		Expect(err).ShouldNot(HaveOccurred())
		parentDir := filepath.Dir(tmpPath)
		fileName := "./" + filepath.Base(tmpPath)
		err = shell.CopyFile(testconfig.Cfg.CoreCTLBinary, tmpPath)
		Expect(err).ShouldNot(HaveOccurred())
		err = os.Chmod(tmpPath, os.FileMode(0755))
		Expect(err).ShouldNot(HaveOccurred())

		initialVersion, _, err := shell.RunCommand(parentDir, fileName, "version")
		if err != nil {
			Fail(fmt.Sprintf("failed to get initial version: %v", err))
		}
		log.Info().Msgf("Initial version: %s", initialVersion)

		updateArgs := []string{"update", "--non-interactive"}
		updateArgs = append(updateArgs, args...)
		output, _, err := shell.RunCommand(parentDir, fileName, updateArgs...)
		if err != nil {
			Fail(fmt.Sprintf("failed to run update: %v, %s", err, output))
		}

		updatedVersion, _, err := shell.RunCommand(parentDir, fileName, "version")
		if err != nil {
			Fail(fmt.Sprintf("failed to get updated version: %v", err))
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
