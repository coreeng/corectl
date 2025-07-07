package env

import (
	"os"

	"github.com/coreeng/corectl/pkg/cmdutil/configpath"

	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("env", ginkgo.Ordered, func() {
	t := ginkgo.GinkgoT()
	var (
		corectl *testconfig.CorectlClient
	)

	ginkgo.BeforeAll(func() {
		homeDir := t.TempDir()
		configpath.SetCorectlHome(homeDir)
		corectl = testconfig.NewCorectlClient(homeDir)
		_, _, err := testsetup.InitCorectl(corectl)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	})

	ginkgo.Context("connect", func() {
		ginkgo.Context("errors when no GCP access", func() {
			ginkgo.It("returns meaningful error when no credentials provided", func() {
				gomega.Expect(os.Setenv("CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE", "/tmp/not-exist")).NotTo(gomega.HaveOccurred())

				_, err := corectl.Run("env", "connect", "--non-interactive", testdata.DevEnvironment())

				gomega.Expect(err.Error()).To(gomega.SatisfyAll(
					gomega.ContainSubstring("Error: create google cluster client: credentials: could not find default credentials"),
					gomega.ContainSubstring("did you run `gcloud auth application-default login`?")))
			})
		})
	})
})
