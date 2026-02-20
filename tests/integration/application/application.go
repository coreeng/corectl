package application

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"time"

	"github.com/coreeng/core-platform/pkg/environment"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	"github.com/google/go-github/v60/github"

	//nolint:staticcheck
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck
	. "github.com/onsi/gomega"
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

		testRunId string
	)
	t := GinkgoT()

	BeforeAll(func() {
		var err error
		testRunId = testconfig.GetTestRunId()
		homeDir = t.TempDir()
		configpath.SetCorectlHome(homeDir)
		corectl = testconfig.NewCorectlClient(homeDir)
		cfg, cfgDetails, err = testsetup.InitCorectl(corectl)
		Expect(err).ToNot(HaveOccurred())
		githubClient = testconfig.NewGitHubClient()
		testsetup.SetupGitGlobalConfigFromCurrentToOtherHomeDir(homeDir)

		envs, err := environment.List(configpath.GetCorectlCPlatformDir("environments"))
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
			newAppName = "new-test-app-" + testRunId
			appDir = filepath.Join(homeDir, newAppName)
			_, err := corectl.Run(
				"application", "create", newAppName, appDir,
				"-t", testdata.BlankTemplate(),
				"--tenant", testconfig.Cfg.Tenant,
				"--non-interactive")
			Expect(err).ToNot(HaveOccurred())
		}, NodeTimeout(time.Minute))

		AfterAll(func(ctx SpecContext) {
			// Use retry logic for delete operation to handle propagation delays
			err := git.RetryGitHubOperation(
				func() error {
					_, err := githubClient.Repositories.Delete(
						ctx,
						cfg.GitHub.Organization.Value,
						newAppName,
					)
					return err
				},
				git.DefaultMaxRetries,
				git.DefaultBaseDelay,
			)
			Expect(err).NotTo(HaveOccurred())
		}, NodeTimeout(time.Minute))

		It("created a new repository for the new app", func(ctx SpecContext) {
			// Use retry logic to handle potential propagation delays
			newAppRepo, _, err := git.RetryGitHubAPI(
				func() (*github.Repository, *github.Response, error) {
					return githubClient.Repositories.Get(
						ctx,
						cfg.GitHub.Organization.Value,
						newAppName,
					)
				},
				git.DefaultMaxRetries,
				git.DefaultBaseDelay,
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
				Expect(envVars.TotalCount).To(Equal(6))
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
					Satisfy(func(v *github.ActionsVariable) bool {
						return v.Name == "REGION" &&
							v.Value == gcpVendor.Region
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
			Expect(prFile.GetFilename()).To(Equal("tenants/tenants/" + testconfig.Cfg.Tenant + ".app.yaml"))
		}, SpecTimeout(time.Minute))
	})

	Context("create with --dry-run", Ordered, func() {
		var (
			newAppName string
			appDir     string
		)

		BeforeAll(func(ctx SpecContext) {
			newAppName = "new-test-app-dryrun-" + testRunId
			appDir = filepath.Join(homeDir, newAppName)
			_, err := corectl.Run(
				"application", "create", newAppName, appDir,
				"-t", testdata.BlankTemplate(),
				"--tenant", testconfig.Cfg.Tenant,
				"--non-interactive",
				"--dry-run")
			Expect(err).ToNot(HaveOccurred())
		}, NodeTimeout(time.Minute))

		It("did not create a new repository for the new app", func(ctx SpecContext) {
			_, _, err := githubClient.Repositories.Get(
				ctx,
				cfg.GitHub.Organization.Value,
				newAppName,
			)
			Expect(err.Error()).To(Equal(fmt.Sprintf("GET https://api.github.com/repos/%s/%s: 404 Not Found []", cfg.GitHub.Organization.Value, newAppName)))
		}, NodeTimeout(time.Minute))
	})

	Context("create in monorepo mode", Ordered, func() {
		var (
			monorepoName string
			monorepoDir  string
			newAppName   string
			appDir       string
		)

		BeforeAll(func(ctx SpecContext) {
			monorepoName = "test-monorepo-" + testRunId
			monorepoDir = filepath.Join(homeDir, monorepoName)

			createMonorepoRepositoryRemoteAndLocal(githubClient, ctx, cfg, monorepoName, monorepoDir)

			// Create a new app within the monorepo
			newAppName = "new-monorepo-app-" + testRunId
			appDir = filepath.Join(monorepoDir, newAppName)
			_, err := corectl.Run(
				"application", "create", newAppName, appDir,
				"-t", testdata.BlankTemplate(),
				"--tenant", testconfig.Cfg.Tenant,
				"--non-interactive")
			Expect(err).ToNot(HaveOccurred())
		}, NodeTimeout(2*time.Minute))

		AfterAll(func(ctx SpecContext) {
			// Use retry logic for delete operation to handle propagation delays
			err := git.RetryGitHubOperation(
				func() error {
					_, err := githubClient.Repositories.Delete(
						ctx,
						cfg.GitHub.Organization.Value,
						monorepoName,
					)
					return err
				},
				git.DefaultMaxRetries,
				git.DefaultBaseDelay,
			)
			Expect(err).NotTo(HaveOccurred())
		}, NodeTimeout(time.Minute))

		It("created a PR for the new app in the monorepo", func(ctx SpecContext) {
			prList, _, err := githubClient.PullRequests.List(
				ctx,
				cfg.GitHub.Organization.Value,
				monorepoName,
				&github.PullRequestListOptions{
					Head: cfg.GitHub.Organization.Value + ":add-" + newAppName,
					Base: git.MainBranch,
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(prList).To(HaveLen(1))
			Expect(prList[0]).NotTo(BeNil())
			pr := prList[0]

			Expect(pr.GetTitle()).To(Equal("Add " + newAppName + " application"))
			Expect(pr.GetState()).To(Equal("open"))

			prFiles, _, err := githubClient.PullRequests.ListFiles(
				ctx,
				cfg.GitHub.Organization.Value,
				monorepoName,
				pr.GetNumber(),
				&github.ListOptions{},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(prFiles)).To(BeNumerically(">", 0))

			// Check for specific files in the PR
			expectedFiles := []string{
				fmt.Sprintf("%s/README.md", newAppName),
				fmt.Sprintf("%s/app.yaml", newAppName),
				fmt.Sprintf(".github/workflows/%s-fast-feedback.yaml", newAppName),
				fmt.Sprintf(".github/workflows/%s-extended-test.yaml", newAppName),
			}

			actualFiles := make([]string, len(prFiles))
			for i, file := range prFiles {
				actualFiles[i] = file.GetFilename()
			}

			Expect(actualFiles).To(ConsistOf(expectedFiles))
		}, SpecTimeout(time.Minute))

		It("did not create a new repository for the app", func(ctx SpecContext) {
			_, _, err := githubClient.Repositories.Get(
				ctx,
				cfg.GitHub.Organization.Value,
				newAppName,
			)
			Expect(err).To(HaveOccurred())
			Expect(err.(*github.ErrorResponse).Response.StatusCode).To(Equal(404))
		}, NodeTimeout(time.Minute))

		It("did not create a PR for updating tenant configuration", func(ctx SpecContext) {
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
			Expect(prList).To(BeEmpty())
		}, SpecTimeout(time.Minute))
	})

	Context("create with team tenant", Ordered, func() {
		var (
			teamTenantName string
			newAppName     string
			newAppRepoId   int64
			appDir         string
		)

		BeforeAll(func(ctx SpecContext) {
			// Use existing "parent" team tenant
			teamTenantName = "parent"

			// Create an app with this team tenant
			newAppName = "team-app-" + strings.ToLower(testRunId)
			appDir = filepath.Join(homeDir, newAppName)
			_, err := corectl.Run(
				"application", "create", newAppName, appDir,
				"-t", testdata.BlankTemplate(),
				"--tenant", teamTenantName,
				"--non-interactive")
			Expect(err).ToNot(HaveOccurred())
		}, NodeTimeout(2*time.Minute))

		AfterAll(func(ctx SpecContext) {
			// Clean up the app repository
			err := git.RetryGitHubOperation(
				func() error {
					_, err := githubClient.Repositories.Delete(
						ctx,
						cfg.GitHub.Organization.Value,
						newAppName,
					)
					return err
				},
				git.DefaultMaxRetries,
				git.DefaultBaseDelay,
			)
			Expect(err).NotTo(HaveOccurred())
		}, NodeTimeout(time.Minute))

		It("created a new repository for the new app", func(ctx SpecContext) {
			newAppRepo, _, err := git.RetryGitHubAPI(
				func() (*github.Repository, *github.Response, error) {
					return githubClient.Repositories.Get(
						ctx,
						cfg.GitHub.Organization.Value,
						newAppName,
					)
				},
				git.DefaultMaxRetries,
				git.DefaultBaseDelay,
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
						v.Value == newAppName // App tenant has same name as app
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
				Expect(envVars.TotalCount).To(Equal(6))
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
					Satisfy(func(v *github.ActionsVariable) bool {
						return v.Name == "REGION" &&
							v.Value == gcpVendor.Region
					}),
				))
			}
		}, NodeTimeout(time.Minute))

		It("created a PR with new app tenant and repository", func(ctx SpecContext) {
			prList, _, err := githubClient.PullRequests.List(
				ctx,
				cfgDetails.CPlatformRepoName.Organization(),
				cfgDetails.CPlatformRepoName.Name(),
				&github.PullRequestListOptions{
					Head: cfgDetails.CPlatformRepoName.Organization() + ":new-app-tenant-" + newAppName,
					Base: git.MainBranch,
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(prList).To(HaveLen(1))
			Expect(prList[0]).NotTo(BeNil())
			pr := prList[0]

			Expect(pr.GetTitle()).To(Equal("New app tenant: " + newAppName))
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

			Expect(prFile.GetStatus()).To(Equal("added"))
			Expect(prFile.GetFilename()).To(Equal("tenants/tenants/" + teamTenantName + "/" + newAppName + ".app.yaml"))
		}, SpecTimeout(time.Minute))
	})
})

func createMonorepoRepositoryRemoteAndLocal(githubClient *github.Client, ctx context.Context, cfg *config.Config, monorepoName string, monorepoDir string) {
	_, _, err := githubClient.Repositories.Create(ctx, cfg.GitHub.Organization.Value, &github.Repository{
		Name:       github.String(monorepoName),
		Visibility: github.String("private"),
	})
	Expect(err).NotTo(HaveOccurred())

	// Use retry logic for CreateFile operation to handle propagation delays
	err = git.RetryGitHubOperation(
		func() error {
			_, _, err := githubClient.Repositories.CreateFile(
				ctx,
				cfg.GitHub.Organization.Value,
				monorepoName,
				"README.md",
				&github.RepositoryContentFileOptions{
					Message: github.String("Initial commit"),
					Content: []byte("# Monorepo\n\nThis is a test monorepo."),
				},
			)
			return err
		},
		git.DefaultMaxRetries,
		git.DefaultBaseDelay,
	)
	Expect(err).NotTo(HaveOccurred())

	_, err = git.CloneToLocalRepository(git.CloneOp{
		URL:        fmt.Sprintf("https://github.com/%s/%s.git", cfg.GitHub.Organization.Value, monorepoName),
		TargetPath: monorepoDir,
		Auth:       git.UrlTokenAuthMethod(cfg.GitHub.Token.Value),
	})
	Expect(err).NotTo(HaveOccurred())
}
