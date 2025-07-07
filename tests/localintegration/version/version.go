package version

import (
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("version", ginkgo.Ordered, func() {
	t := ginkgo.GinkgoT()
	var (
		corectl *testconfig.CorectlClient
	)

	ginkgo.BeforeAll(func() {
		corectl = testconfig.NewCorectlClient(t.TempDir())
	})

	ginkgo.Context("version", func() {
		ginkgo.It("returns sensible defaults", func() {
			output, err := corectl.Run("version", "--non-interactive")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(output).Should(gomega.MatchRegexp("corectl (?P<tag>[a-z0-9\\.]+?) \\(commit: (?P<commit>[0-9a-f]+?)\\) (?P<date>.+?) (?P<arch>.+)"))
		})
	})
})
