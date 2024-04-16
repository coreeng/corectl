package p2p

import (
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/environment"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("p2p", Ordered, func() {
	var (
		homeDir string
		corectl *testconfig.CorectlClient
		cfg     *config.Config
	)
	t := GinkgoT()

	BeforeAll(func() {
		homeDir = t.TempDir()
		corectl = testconfig.NewCorectlClient(homeDir)
		testsetup.SetupGitGlobalConfigFromCurrentToOtherHomeDir(homeDir)
	})

	Context("sync", Ordered, func() {
		var (
			appRepo string
		)

		BeforeAll(func() {
			appRepo = "idp-reference-app-go-pst"
			Expect(corectl.Run(
				"p2p env sync", appRepo,
			)).To(Succeed())
		})
		It("count environments", func() {
			cfg, _ = testsetup.InitCorectl(corectl)
			envs, err := environment.List(cfg.Repositories.CPlatform.Value)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(envs)).To(BeNumerically(">=", 0))
		})
	})
})
