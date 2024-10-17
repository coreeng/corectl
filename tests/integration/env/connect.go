package env

import (
	"os"

	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("env", Ordered, func() {
	t := GinkgoT()
	var (
		corectl *testconfig.CorectlClient
	)

	BeforeAll(func() {
		homeDir := t.TempDir()
		corectl = testconfig.NewCorectlClient(homeDir)
		_, _, err := testsetup.InitCorectl(corectl)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("connect", func() {
		Context("errors when no GCP access", func() {
			It("returns meaningful error when no credentials provided", func() {
				Expect(os.Setenv("CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE", "/tmp/not-exist")).NotTo(HaveOccurred())

				_, err := corectl.Run("env", "connect", "--non-interactive", testdata.DevEnvironment())

				Expect(err.Error()).To(SatisfyAll(
					ContainSubstring("Error: create google cluster client: credentials: could not find default credentials"),
					ContainSubstring("did you run `gcloud auth application-default login`?")))
			})
		})
	})
})
