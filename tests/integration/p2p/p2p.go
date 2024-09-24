package p2p

import (
	"slices"
	"time"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/google/go-github/v59/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/thanhpk/randstr"
)

var _ = Describe("p2p", Ordered, func() {
	var (
		homeDir      string
		corectl      *testconfig.CorectlClient
		cfg          *config.Config
		githubClient *github.Client
		tmpRepo      *github.Repository
		devEnv       environment.Environment
		envVars      = []string{"BASE_DOMAIN", "DPLATFORM", "INTERNAL_SERVICES_DOMAIN", "PROJECT_ID", "PROJECT_NUMBER"}
	)
	t := GinkgoT()

	BeforeAll(func(ctx SpecContext) {
		var err error
		homeDir = t.TempDir()
		corectl = testconfig.NewCorectlClient(homeDir)
		cfg, _, err = testsetup.InitCorectl(corectl)
		Expect(err).ToNot(HaveOccurred())
		githubClient = testconfig.NewGitHubClient()
		testsetup.SetupGitGlobalConfigFromCurrentToOtherHomeDir(homeDir)
		envs, err := environment.List(environment.DirFromCPlatformRepoPath(cfg.Repositories.CPlatform.Value))
		Expect(err).NotTo(HaveOccurred())
		devEnvIdx := slices.IndexFunc(envs, func(e environment.Environment) bool {
			return e.Environment == testdata.DevEnvironment()
		})
		Expect(devEnvIdx).To(BeNumerically(">=", 0))
		devEnv = envs[devEnvIdx]
	})
	Context("sync", Ordered, func() {
		var (
			appRepo string
			tenant  string
			err     error
		)

		BeforeAll(func(ctx SpecContext) {
			appRepo = "new-test-repo-" + randstr.Hex(6)
			tenant = testdata.DefaultTenant()
			deleteBranchOnMerge := true
			visibility := "private"
			tmpRepo, _, err = githubClient.Repositories.Create(
				ctx,
				cfg.GitHub.Organization.Value,
				&github.Repository{
					Name:                &appRepo,
					DeleteBranchOnMerge: &deleteBranchOnMerge,
					Visibility:          &visibility,
				},
			)
			Expect(err).NotTo(HaveOccurred())

			_, err := corectl.Run(
				"p2p", "env", "sync",
				appRepo,
				tenant,
				"--log-level=panic")
			Expect(err).ToNot(HaveOccurred())
		}, NodeTimeout(time.Minute))

		AfterAll(func(ctx SpecContext) {
			Expect(githubClient.Repositories.Delete(
				ctx,
				cfg.GitHub.Organization.Value,
				appRepo,
			)).Error().NotTo(HaveOccurred())
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

		It("checks repository environment", func(ctx SpecContext) {
			for _, envVar := range envVars {
				_, _, err := githubClient.Actions.GetEnvVariable(
					ctx,
					int(tmpRepo.GetID()),
					devEnv.Environment,
					envVar,
				)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})
})
