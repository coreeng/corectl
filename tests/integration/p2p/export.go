package p2p

import (
	"path/filepath"
	"time"

	"github.com/coreeng/corectl/pkg/cmdutil/configpath"

	"github.com/coreeng/core-platform/pkg/environment"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	gogit "github.com/go-git/go-git/v5"
	"github.com/google/go-github/v60/github"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/thanhpk/randstr"
)

var _ = ginkgo.Describe("export", ginkgo.Ordered, func() {
	t := ginkgo.GinkgoT()
	var (
		corectl      *testconfig.CorectlClient
		cfg          *config.Config
		githubClient *github.Client
		env          *environment.Environment
		appName      string
		appDir       string
	)

	ginkgo.BeforeAll(func(ctx ginkgo.SpecContext) {
		homeDir := tmpDir(t)
		configpath.SetCorectlHome(homeDir)
		corectl, cfg = initCorectl(homeDir)
		githubClient = testconfig.NewGitHubClient()
		appName = "new-test-app-" + randstr.Hex(6)
		appDir = onboardTestApp(homeDir, appName, corectl)
		env = defaultEnv(cfg.Repositories.CPlatform.Value)
	})

	ginkgo.AfterAll(func(ctx ginkgo.SpecContext) {
		// Use retry logic for delete operation to handle propagation delays
		err := git.RetryGitHubOperation(
			func() error {
				_, err := githubClient.Repositories.Delete(
					ctx,
					cfg.GitHub.Organization.Value,
					appName,
				)
				return err
			},
			git.DefaultMaxRetries,
			git.DefaultBaseDelay,
		)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}, ginkgo.NodeTimeout(time.Minute))

	ginkgo.Context("export", func() {

		var commitHash = func(repoPath string) string {
			r, err := gogit.PlainOpen(repoPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			ref, err := r.Head()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			return ref.Hash().String()[0:7]
		}

		var assertExportStatements = func(act string) {
			gomega.Expect(act).To(gomega.SatisfyAll(
				gomega.ContainSubstring("export REGION=\"%s\"", env.Platform.(*environment.GCPVendor).Region),
				gomega.ContainSubstring("export REGISTRY=\"%s-docker.pkg.dev/%s/tenant/%s\"", env.Platform.(*environment.GCPVendor).Region, env.Platform.(*environment.GCPVendor).ProjectId, testconfig.Cfg.Tenant),
				gomega.ContainSubstring("export BASE_DOMAIN=\"%s\"", env.GetDefaultIngressDomain().Domain),
				gomega.ContainSubstring("export REPO_PATH=\"%s\"", appDir),
				gomega.ContainSubstring("export TENANT_NAME=\"%s\"", testconfig.Cfg.Tenant),
				gomega.ContainSubstring("export VERSION=\"%s\"", commitHash(appDir))))
		}

		ginkgo.Context("print out env variables", func() {
			ginkgo.It("as export statements", func() {
				output, err := corectl.Run("p2p", "export", "--non-interactive", "--tenant", testconfig.Cfg.Tenant, "--environment", testdata.DevEnvironment(), "--repoPath", appDir)

				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				assertExportStatements(output)
			})
			ginkgo.It("with shorthand flags", func() {
				output, err := corectl.Run("p2p", "export", "--non-interactive", "-t", testconfig.Cfg.Tenant, "-e", testdata.DevEnvironment(), "-r", appDir)

				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(output).ToNot(gomega.BeEmpty())
			})

			ginkgo.It("defaulting to local dir when no repoPath flag passed", func() {
				output, err := corectl.RunInDir(appDir, "p2p", "export", "--non-interactive", "--tenant", testconfig.Cfg.Tenant, "--environment", testdata.DevEnvironment())

				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				assertExportStatements(output)
			})
		})
	})
})

func initCorectl(homeDir string) (*testconfig.CorectlClient, *config.Config) {
	corectl := testconfig.NewCorectlClient(homeDir)
	cfg, _, err := testsetup.InitCorectl(corectl)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return corectl, cfg
}

func onboardTestApp(homeDir string, appName string, corectl *testconfig.CorectlClient) string {
	testsetup.SetupGitGlobalConfigFromCurrentToOtherHomeDir(homeDir)
	appDir := filepath.Join(homeDir, appName)
	_, err := corectl.Run(
		"application", "create", appName, appDir,
		"-t", testdata.BlankTemplate(),
		"--tenant", testconfig.Cfg.Tenant,
		"--non-interactive",
		"--log-level=panic")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return appDir
}

// macOS has a symlink from /var -> private/var causing a wrong app dir path, ensure we're using actual path
func tmpDir(t ginkgo.GinkgoTInterface) string {
	d, err := filepath.EvalSymlinks(t.TempDir())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return d
}

func defaultEnv(cPlatRepoPath string) *environment.Environment {
	e, err := environment.FindByName(configpath.GetCorectlCPlatformDir("environments"), testdata.DevEnvironment())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return e
}
