package application

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/google/go-github/v59/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/thanhpk/randstr"
	"path/filepath"
	"slices"

	"time"
)

var _ = Describe("application", Ordered, func() {
	var (
		homeDir      string
		corectl      *testconfig.CorectlClient
		cfg          *config.Config
		cfgDetails   *testsetup.CorectlConfigDetails
		githubClient *github.Client

		prodEnv environment.Environment
		devEnv  environment.Environment
	)
	t := GinkgoT()

	BeforeAll(func() {
		var err error
		homeDir = t.TempDir()
		corectl = testconfig.NewCorectlClient(homeDir)
		cfg, cfgDetails, err = testsetup.InitCorectl(corectl)
		Expect(err).ToNot(HaveOccurred())
		githubClient = testconfig.NewGitHubClient()
		testsetup.SetupGitGlobalConfigFromCurrentToOtherHomeDir(homeDir)

		envs, err := environment.List(environment.DirFromCPlatformRepoPath(cfg.Repositories.CPlatform.Value))
		Expect(err).NotTo(HaveOccurred())
		devEnvIdx := slices.IndexFunc(envs, func(e environment.Environment) bool {
			return e.Environment == testdata.DevEnvironment()
		})
		prodEnvIdx := slices.IndexFunc(envs, func(e environment.Environment) bool {
			return e.Environment == testdata.ProdEnvironment()
		})
		Expect(devEnvIdx).To(BeNumerically(">=", 0))
		Expect(prodEnvIdx).To(BeNumerically(">=", 0))
		devEnv = envs[devEnvIdx]
		prodEnv = envs[prodEnvIdx]
	})

	Context("create", Ordered, func() {
		var (
			newAppName   string
			newAppRepoId int64
			appDir       string
		)

		BeforeAll(func(ctx SpecContext) {
			newAppName = "new-test-app-" + randstr.Hex(6)
			appDir = filepath.Join(homeDir, newAppName)
			Expect(corectl.Run(
				"application", "create", newAppName, appDir,
				"-t", testdata.BlankTemplate(),
				"--tenant", testconfig.Cfg.Tenant,
				"--nonint",
			)).To(Succeed())
		}, NodeTimeout(time.Minute))

		AfterAll(func(ctx SpecContext) {
			Expect(githubClient.Repositories.Delete(
				ctx,
				cfg.GitHub.Organization.Value,
				newAppName,
			)).Error().NotTo(HaveOccurred())
		}, NodeTimeout(time.Minute))

		It("created a new repository for the new app", func(ctx SpecContext) {
			newAppRepo, _, err := githubClient.Repositories.Get(
				ctx,
				cfg.GitHub.Organization.Value,
				newAppName,
			)
			Expect(err).NotTo(HaveOccurred())
			newAppRepoId = newAppRepo.GetID()
		}, NodeTimeout(time.Minute))

		It("correctly configured action variables for new repository", func(ctx SpecContext) {
			repoVars, _, err := githubClient.Actions.ListRepoVariables(
				ctx,
				cfg.GitHub.Organization.Value,
				newAppName,
				&github.ListOptions{},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(repoVars.TotalCount).To(Equal(4))
			Expect(repoVars.Variables).To(ConsistOf(
				Satisfy(func(v *github.ActionsVariable) bool {
					return v.Name == "TENANT_NAME" &&
						v.Value == testconfig.Cfg.Tenant
				}),
				Satisfy(func(v *github.ActionsVariable) bool {
					return v.Name == "FAST_FEEDBACK" &&
						v.Value == fmt.Sprintf("{\"include\":[{\"deploy_env\":\"%s\"}]}", devEnv.Environment)
				}),
				Satisfy(func(v *github.ActionsVariable) bool {
					return v.Name == "EXTENDED_TEST" &&
						v.Value == fmt.Sprintf("{\"include\":[{\"deploy_env\":\"%s\"}]}", devEnv.Environment)
				}),
				Satisfy(func(v *github.ActionsVariable) bool {
					return v.Name == "PROD" &&
						v.Value == fmt.Sprintf("{\"include\":[{\"deploy_env\":\"%s\"}]}", prodEnv.Environment)
				}),
			))
		}, NodeTimeout(time.Minute))

		It("correctly configured environments for the new app repo", func(ctx SpecContext) {
			for _, env := range []environment.Environment{devEnv, prodEnv} {
				envVars, _, err := githubClient.Actions.ListEnvVariables(
					ctx,
					int(newAppRepoId),
					env.Environment,
					&github.ListOptions{},
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(envVars.TotalCount).To(Equal(5))
				gcpVendor := env.Platform.(*environment.GCPVendor)
				Expect(envVars.Variables).To(ConsistOf(
					Satisfy(func(v *github.ActionsVariable) bool {
						return v.Name == "DPLATFORM" &&
							v.Value == env.Environment
					}),
					Satisfy(func(v *github.ActionsVariable) bool {
						return v.Name == "BASE_DOMAIN" &&
							v.Value == env.GetDefaultIngressDomain().Domain
					}),
					Satisfy(func(v *github.ActionsVariable) bool {
						return v.Name == "INTERNAL_SERVICES_DOMAIN" &&
							v.Value == env.InternalServices.Domain
					}),
					Satisfy(func(v *github.ActionsVariable) bool {
						return v.Name == "PROJECT_ID" &&
							v.Value == gcpVendor.ProjectId
					}),
					Satisfy(func(v *github.ActionsVariable) bool {
						return v.Name == "PROJECT_NUMBER" &&
							v.Value == gcpVendor.ProjectNumber
					}),
				))
			}
		}, NodeTimeout(time.Minute))

		It("created a PR with new app link for the tenant", func(ctx SpecContext) {
			prList, _, err := githubClient.PullRequests.List(
				ctx,
				cfgDetails.CPlatformRepoName.Organization(),
				cfgDetails.CPlatformRepoName.Name(),
				&github.PullRequestListOptions{
					Head: cfgDetails.CPlatformRepoName.Organization() + ":" + testconfig.Cfg.Tenant + "-add-repo-" + newAppName,
					Base: git.MainBranch,
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(prList).To(HaveLen(1))
			Expect(prList[0]).NotTo(BeNil())
			pr := prList[0]

			Expect(pr.GetTitle()).To(Equal("Add new repository " + newAppName + " for tenant " + testconfig.Cfg.Tenant))
			Expect(pr.GetState()).To(Equal("open"))

			prFiles, _, err := githubClient.PullRequests.ListFiles(
				ctx,
				cfgDetails.CPlatformRepoName.Organization(),
				cfgDetails.CPlatformRepoName.Name(),
				pr.GetNumber(),
				&github.ListOptions{},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(prFiles).To(HaveLen(1))
			prFile := prFiles[0]

			Expect(prFile.GetStatus()).To(Equal("modified"))
			Expect(prFile.GetFilename()).To(Equal("tenants/tenants/" + testconfig.Cfg.Tenant + ".yaml"))
		}, SpecTimeout(time.Minute))
	})
})
