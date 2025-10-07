package version

import (
	"github.com/coreeng/corectl/tests/integration/testconfig"

	//nolint:staticcheck
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck
	. "github.com/onsi/gomega"
)

var _ = Describe("version", Ordered, func() {
	t := GinkgoT()
	var (
		corectl *testconfig.CorectlClient
	)

	BeforeAll(func() {
		corectl = testconfig.NewCorectlClient(t.TempDir())
	})

	Context("version", func() {
		It("returns sensible defaults", func() {
			output, err := corectl.Run("version", "--non-interactive")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(output).Should(MatchRegexp("corectl (?P<tag>[a-z0-9\\.]+?) \\(commit: (?P<commit>[0-9a-f]+?)\\) (?P<date>.+?) (?P<arch>.+)"))
		})
	})
})
