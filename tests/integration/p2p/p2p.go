package p2p

import (
	"slices"
	"time"

	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
	"github.com/coreeng/corectl/pkg/git"

	"github.com/coreeng/core-platform/pkg/environment"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	"github.com/google/go-github/v60/github"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/thanhpk/randstr"
)

var _ = ginkgo.Describe("p2p", ginkgo.Ordered, func() {
	var (
		homeDir      string
		corectl      *testconfig.CorectlClient
		cfg          *config.Config
		githubClient *github.Client
		tmpRepo      *github.Repository
		devEnv       environment.Environment
		envVars      = []string{"BASE_DOMAIN", "DPLATFORM", "INTERNAL_SERVICES_DOMAIN", "PROJECT_ID", "PROJECT_NUMBER"}
	)
	t := ginkgo.GinkgoT()

	ginkgo.BeforeAll(func(ctx ginkgo.SpecContext) {
		var err error
		homeDir = t.TempDir()
		configpath.SetCorectlHome(homeDir)
		corectl = testconfig.NewCorectlClient(homeDir)
		cfg, _, err = testsetup.InitCorectl(corectl)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		githubClient = testconfig.NewGitHubClient()
		testsetup.SetupGitGlobalConfigFromCurrentToOtherHomeDir(homeDir)
		envs, err := environment.List(configpath.GetCorectlCPlatformDir("environments"))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		devEnvIdx := slices.IndexFunc(envs, func(e environment.Environment) bool {
			return e.Environment == testdata.DevEnvironment()
		})
		gomega.Expect(devEnvIdx).To(gomega.BeNumerically(">=", 0))
		devEnv = envs[devEnvIdx]
	})
	ginkgo.Context("sync", ginkgo.Ordered, func() {
		var (
			appRepo string
			tenant  string
			err     error
		)

		ginkgo.BeforeAll(func(ctx ginkgo.SpecContext) {
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
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			_, err := corectl.Run(
				"p2p", "env", "sync", "--non-interactive",
				appRepo,
				tenant)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}, ginkgo.NodeTimeout(time.Minute))

		ginkgo.AfterAll(func(ctx ginkgo.SpecContext) {
			// Use retry logic for delete operation to handle propagation delays
			err := git.RetryGitHubOperation(
				func() error {
					_, err := githubClient.Repositories.Delete(
						ctx,
						cfg.GitHub.Organization.Value,
						appRepo,
					)
					return err
				},
				git.DefaultMaxRetries,
				git.DefaultBaseDelay,
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}, ginkgo.NodeTimeout(time.Minute))

		ginkgo.It("checks repository variables", func(ctx ginkgo.SpecContext) {
			_, _, err := githubClient.Actions.GetRepoVariable(
				ctx,
				cfg.GitHub.Organization.Value,
				appRepo,
				"TENANT_NAME",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("checks repository environment", func(ctx ginkgo.SpecContext) {
			for _, envVar := range envVars {
				_, _, err := githubClient.Actions.GetEnvVariable(
					ctx,
					int(tmpRepo.GetID()),
					devEnv.Environment,
					envVar,
				)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}
		})
	})
})
