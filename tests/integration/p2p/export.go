package p2p

import (
	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
	"path/filepath"

	"github.com/coreeng/core-platform/pkg/environment"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	"github.com/go-git/go-git/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/thanhpk/randstr"
)

var _ = Describe("export", Ordered, func() {
	t := GinkgoT()
	var (
		corectl *testconfig.CorectlClient
		env     *environment.Environment
		appDir  string
	)

	BeforeAll(func() {
		var cfg *config.Config
		homeDir := tmpDir(t)
		configpath.SetCorectlHome(homeDir)
		corectl, cfg = initCorectl(homeDir)
		appDir = onboardTestApp(homeDir, corectl)
		env = defaultEnv(cfg.Repositories.CPlatform.Value)
	})

	Context("export", func() {

		var commitHash = func(repoPath string) string {
			r, err := git.PlainOpen(repoPath)
			Expect(err).NotTo(HaveOccurred())
			ref, err := r.Head()
			Expect(err).NotTo(HaveOccurred())
			return ref.Hash().String()[0:7]
		}

		var assertExportStatements = func(act string) {
			Expect(act).To(SatisfyAll(
				ContainSubstring("export REGION=\"%s\"", env.Platform.(*environment.GCPVendor).Region),
				ContainSubstring("export REGISTRY=\"%s-docker.pkg.dev/%s/tenant/%s\"", env.Platform.(*environment.GCPVendor).Region, env.Platform.(*environment.GCPVendor).ProjectId, testconfig.Cfg.Tenant),
				ContainSubstring("export BASE_DOMAIN=\"%s\"", env.GetDefaultIngressDomain().Domain),
				ContainSubstring("export REPO_PATH=\"%s\"", appDir),
				ContainSubstring("export TENANT_NAME=\"%s\"", testconfig.Cfg.Tenant),
				ContainSubstring("export VERSION=\"%s\"", commitHash(appDir))))
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
	})
})

func initCorectl(homeDir string) (*testconfig.CorectlClient, *config.Config) {
	corectl := testconfig.NewCorectlClient(homeDir)
	cfg, _, err := testsetup.InitCorectl(corectl)
	Expect(err).ToNot(HaveOccurred())
	return corectl, cfg
}

func onboardTestApp(homeDir string, corectl *testconfig.CorectlClient) string {
	testsetup.SetupGitGlobalConfigFromCurrentToOtherHomeDir(homeDir)
	newAppName := "new-test-app-" + randstr.Hex(6)
	appDir := filepath.Join(homeDir, newAppName)
	_, err := corectl.Run(
		"application", "create", newAppName, appDir,
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
