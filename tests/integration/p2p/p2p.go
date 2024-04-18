package p2p

import (
	"slices"
	"time"
	"fmt"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/environment"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	"github.com/google/go-github/v59/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/thanhpk/randstr"
	
)

var _ = Describe("p2p", Ordered, func() {
	var (
		homeDir      string
		corectl      *testconfig.CorectlClient
		cfgDetails   testsetup.CorectlConfigDetails
		cfg          *config.Config
		devEnv       environment.Environment
		githubClient *github.Client
	)
	t := GinkgoT()

	BeforeAll(func(ctx SpecContext) {
		homeDir = t.TempDir()
		corectl = testconfig.NewCorectlClient(homeDir)
		cfg, cfgDetails = testsetup.InitCorectl(corectl)
		githubClient = testconfig.NewGitHubClient()
		testsetup.SetupGitGlobalConfigFromCurrentToOtherHomeDir(homeDir)
		envs, err := environment.List(cfg.Repositories.CPlatform.Value)
		Expect(err).NotTo(HaveOccurred())
		devEnvIdx := slices.IndexFunc(envs, func(e environment.Environment) bool {
			return e.Environment == environment.Name(testdata.DevEnvironment())
		})
		Expect(devEnvIdx).To(BeNumerically(">=", 0))
		devEnv = envs[devEnvIdx]
	})

	Context("sync", Ordered, func() {
		var (
			appRepo    string
			tenant     string
		)

		BeforeAll(func(ctx SpecContext) {
			_= cfgDetails
			_= devEnv
			appRepo = "new-test-repo-" + randstr.Hex(6)
			tenant = "default-tenant"
			deleteBranchOnMerge := true
			visibility := "private"
			fmt.Printf("%s", cfg.GitHub.Organization.Value)
			_, _, err := githubClient.Repositories.Create(
				ctx,
				cfg.GitHub.Organization.Value,
				&github.Repository{
					Name: &appRepo,
					DeleteBranchOnMerge: &deleteBranchOnMerge,
					Visibility: &visibility,
				},				
			)
			Expect(err).NotTo(HaveOccurred())

			Expect(corectl.Run(
				"p2p", "env", "sync", 
				appRepo,
				tenant,
			)).To(Succeed())
		}, NodeTimeout(time.Minute))

		It("checks repository variables", func(ctx SpecContext) {
			_, _, err := githubClient.Actions.GetRepoVariable(
				ctx, 
				cfg.GitHub.Organization.Value, 
				appRepo,
				"TENANT_NAME", 
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
