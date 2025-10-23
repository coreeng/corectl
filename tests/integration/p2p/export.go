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
	"github.com/thanhpk/randstr"

	//nolint:staticcheck
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck
	. "github.com/onsi/gomega"
)

var _ = Describe("export", Ordered, func() {
	t := GinkgoT()
	var (
		corectl      *testconfig.CorectlClient
		cfg          *config.Config
		githubClient *github.Client
		env          *environment.Environment
		appName      string
		appDir       string
	)

	BeforeAll(func(ctx SpecContext) {
		homeDir := tmpDir(t)
		configpath.SetCorectlHome(homeDir)
		corectl, cfg = initCorectl(homeDir)
		githubClient = testconfig.NewGitHubClient()
		appName = "new-test-app-" + randstr.Hex(6)
		appDir = onboardTestApp(homeDir, appName, corectl)
		env = defaultEnv(cfg.Repositories.CPlatform.Value)
	})

	AfterAll(func(ctx SpecContext) {
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
		Expect(err).NotTo(HaveOccurred())
	}, NodeTimeout(time.Minute))

	Context("export", func() {

		var commitHash = func(repoPath string) string {
			r, err := gogit.PlainOpenWithOptions(repoPath, &gogit.PlainOpenOptions{
				DetectDotGit: true,
			})
			Expect(err).NotTo(HaveOccurred())
			ref, err := r.Head()
			Expect(err).NotTo(HaveOccurred())
			return ref.Hash().String()[0:7]
		}

		var assertExportStatements = func(act string) {
			// Check that all required environment variables are present in the export output
			// Note: shellquote may or may not add quotes depending on the value content
			Expect(act).To(SatisfyAll(
				ContainSubstring("export REGION="),
				ContainSubstring(env.Platform.(*environment.GCPVendor).Region),
				ContainSubstring("export REGISTRY="),
				ContainSubstring(fmt.Sprintf("%s-docker.pkg.dev/%s/tenant/%s", env.Platform.(*environment.GCPVendor).Region, env.Platform.(*environment.GCPVendor).ProjectId, testconfig.Cfg.Tenant)),
				ContainSubstring("export BASE_DOMAIN="),
				ContainSubstring(env.GetDefaultIngressDomain().Domain),
				ContainSubstring("export REPO_PATH="),
				ContainSubstring(appDir),
				ContainSubstring("export TENANT_NAME="),
				ContainSubstring(testconfig.Cfg.Tenant),
				ContainSubstring("export VERSION="),
				ContainSubstring(commitHash(appDir))))
		}

		Context("print out env variables", func() {
			It("as export statements", func() {
				output, err := corectl.Run("p2p", "export", "--non-interactive", "--tenant", testconfig.Cfg.Tenant, "--environment", testdata.DevEnvironment(), "--repoPath", appDir)

				Expect(err).NotTo(HaveOccurred())
				assertExportStatements(output)
			})
			It("with shorthand flags", func() {
				output, err := corectl.Run("p2p", "export", "--non-interactive", "-t", testconfig.Cfg.Tenant, "-e", testdata.DevEnvironment(), "-r", appDir)

				Expect(err).NotTo(HaveOccurred())
				Expect(output).ToNot(BeEmpty())
			})

			It("defaulting to local dir when no repoPath flag passed", func() {
				output, err := corectl.RunInDir(appDir, "p2p", "export", "--non-interactive", "--tenant", testconfig.Cfg.Tenant, "--environment", testdata.DevEnvironment())

				Expect(err).NotTo(HaveOccurred())
				assertExportStatements(output)
			})
		})

		Context("shell format support", func() {
			It("exports for bash shell", func() {
				output, err := corectl.Run("p2p", "export", "--non-interactive", "--tenant", testconfig.Cfg.Tenant, "--environment", testdata.DevEnvironment(), "--repoPath", appDir, "--shell", "bash")

				Expect(err).NotTo(HaveOccurred())
				Expect(output).To(ContainSubstring("export REGION="))
				Expect(output).To(ContainSubstring("export TENANT_NAME="))
			})

			It("exports for zsh shell", func() {
				output, err := corectl.Run("p2p", "export", "--non-interactive", "--tenant", testconfig.Cfg.Tenant, "--environment", testdata.DevEnvironment(), "--repoPath", appDir, "--shell", "zsh")

				Expect(err).NotTo(HaveOccurred())
				Expect(output).To(ContainSubstring("export REGION="))
				Expect(output).To(ContainSubstring("export TENANT_NAME="))
			})

			It("exports for fish shell", func() {
				output, err := corectl.Run("p2p", "export", "--non-interactive", "--tenant", testconfig.Cfg.Tenant, "--environment", testdata.DevEnvironment(), "--repoPath", appDir, "--shell", "fish")

				Expect(err).NotTo(HaveOccurred())
				Expect(output).To(ContainSubstring("set -gx REGION"))
				Expect(output).To(ContainSubstring("set -gx TENANT_NAME"))
			})

			It("exports for powershell", func() {
				output, err := corectl.Run("p2p", "export", "--non-interactive", "--tenant", testconfig.Cfg.Tenant, "--environment", testdata.DevEnvironment(), "--repoPath", appDir, "--shell", "powershell")

				Expect(err).NotTo(HaveOccurred())
				Expect(output).To(ContainSubstring("$Env:REGION"))
				Expect(output).To(ContainSubstring("$Env:TENANT_NAME"))
			})

			It("exports for cmd shell", func() {
				output, err := corectl.Run("p2p", "export", "--non-interactive", "--tenant", testconfig.Cfg.Tenant, "--environment", testdata.DevEnvironment(), "--repoPath", appDir, "--shell", "cmd")

				Expect(err).NotTo(HaveOccurred())
				Expect(output).To(ContainSubstring("set REGION="))
				Expect(output).To(ContainSubstring("set TENANT_NAME="))
			})

			It("rejects unsupported shell", func() {
				_, err := corectl.Run("p2p", "export", "--non-interactive", "--tenant", testconfig.Cfg.Tenant, "--environment", testdata.DevEnvironment(), "--repoPath", appDir, "--shell", "unsupported")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unsupported shell type"))
			})
		})
	})
})

func initCorectl(homeDir string) (*testconfig.CorectlClient, *config.Config) {
	corectl := testconfig.NewCorectlClient(homeDir)
	cfg, _, err := testsetup.InitCorectl(corectl)
	Expect(err).ToNot(HaveOccurred())
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
	Expect(err).ToNot(HaveOccurred())
	return appDir
}

// macOS has a symlink from /var -> private/var causing a wrong app dir path, ensure we're using actual path
func tmpDir(t GinkgoTInterface) string {
	d, err := filepath.EvalSymlinks(t.TempDir())
	Expect(err).ToNot(HaveOccurred())
	return d
}

func defaultEnv(cPlatRepoPath string) *environment.Environment {
	e, err := environment.FindByName(configpath.GetCorectlCPlatformDir("environments"), testdata.DevEnvironment())
	Expect(err).ToNot(HaveOccurred())
	return e
}
