package application

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"time"

	"github.com/coreeng/core-platform/pkg/environment"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	"github.com/google/go-github/v60/github"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("application", ginkgo.Ordered, func() {
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
	t := ginkgo.GinkgoT()

	ginkgo.BeforeAll(func() {
		var err error
		testRunId = testconfig.GetTestRunId()
		homeDir = t.TempDir()
		configpath.SetCorectlHome(homeDir)
		corectl = testconfig.NewCorectlClient(homeDir)
		cfg, cfgDetails, err = testsetup.InitCorectl(corectl)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		githubClient = testconfig.NewGitHubClient()
		testsetup.SetupGitGlobalConfigFromCurrentToOtherHomeDir(homeDir)

		envs, err := environment.List(configpath.GetCorectlCPlatformDir("environments"))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		devEnvIdx := slices.IndexFunc(envs, func(e environment.Environment) bool {
			return e.Environment == testdata.DevEnvironment()
		})
		prodEnvIdx := slices.IndexFunc(envs, func(e environment.Environment) bool {
			return e.Environment == testdata.ProdEnvironment()
		})
		gomega.Expect(devEnvIdx).To(gomega.BeNumerically(">=", 0))
		gomega.Expect(prodEnvIdx).To(gomega.BeNumerically(">=", 0))
		devEnv = envs[devEnvIdx]
		prodEnv = envs[prodEnvIdx]
	})

	ginkgo.Context("create", ginkgo.Ordered, func() {
		var (
			newAppName   string
			newAppRepoId int64
			appDir       string
		)

		ginkgo.BeforeAll(func(ctx ginkgo.SpecContext) {
			newAppName = "new-test-app-" + testRunId
			appDir = filepath.Join(homeDir, newAppName)
			_, err := corectl.Run(
				"application", "create", newAppName, appDir,
				"-t", testdata.BlankTemplate(),
				"--tenant", testconfig.Cfg.Tenant,
				"--non-interactive")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}, ginkgo.NodeTimeout(time.Minute))

		ginkgo.AfterAll(func(ctx ginkgo.SpecContext) {
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
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}, ginkgo.NodeTimeout(time.Minute))

		ginkgo.It("created a new repository for the new app", func(ctx ginkgo.SpecContext) {
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
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			newAppRepoId = newAppRepo.GetID()
		}, ginkgo.NodeTimeout(time.Minute))

		ginkgo.It("correctly configured action variables for new repository", func(ctx ginkgo.SpecContext) {
			repoVars, _, err := githubClient.Actions.ListRepoVariables(
				ctx,
				cfg.GitHub.Organization.Value,
				newAppName,
				&github.ListOptions{},
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(repoVars.TotalCount).To(gomega.Equal(4))
			gomega.Expect(repoVars.Variables).To(gomega.ConsistOf(
				gomega.Satisfy(func(v *github.ActionsVariable) bool {
					return v.Name == "TENANT_NAME" &&
						v.Value == testconfig.Cfg.Tenant
				}),
				gomega.Satisfy(func(v *github.ActionsVariable) bool {
					return v.Name == "FAST_FEEDBACK" &&
						v.Value == fmt.Sprintf("{\"include\":[{\"deploy_env\":\"%s\"}]}", devEnv.Environment)
				}),
				gomega.Satisfy(func(v *github.ActionsVariable) bool {
					return v.Name == "EXTENDED_TEST" &&
						v.Value == fmt.Sprintf("{\"include\":[{\"deploy_env\":\"%s\"}]}", devEnv.Environment)
				}),
				gomega.Satisfy(func(v *github.ActionsVariable) bool {
					return v.Name == "PROD" &&
						v.Value == fmt.Sprintf("{\"include\":[{\"deploy_env\":\"%s\"}]}", prodEnv.Environment)
				}),
			))
		}, ginkgo.NodeTimeout(time.Minute))

		ginkgo.It("correctly configured environments for the new app repo", func(ctx ginkgo.SpecContext) {
			for _, env := range []environment.Environment{devEnv, prodEnv} {
				envVars, _, err := githubClient.Actions.ListEnvVariables(
					ctx,
					int(newAppRepoId),
					env.Environment,
					&github.ListOptions{},
				)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(envVars.TotalCount).To(gomega.Equal(5))
				gcpVendor := env.Platform.(*environment.GCPVendor)
				gomega.Expect(envVars.Variables).To(gomega.ConsistOf(
					gomega.Satisfy(func(v *github.ActionsVariable) bool {
						return v.Name == "DPLATFORM" &&
							v.Value == env.Environment
					}),
					gomega.Satisfy(func(v *github.ActionsVariable) bool {
						return v.Name == "BASE_DOMAIN" &&
							v.Value == env.GetDefaultIngressDomain().Domain
					}),
					gomega.Satisfy(func(v *github.ActionsVariable) bool {
						return v.Name == "INTERNAL_SERVICES_DOMAIN" &&
							v.Value == env.InternalServices.Domain
					}),
					gomega.Satisfy(func(v *github.ActionsVariable) bool {
						return v.Name == "PROJECT_ID" &&
							v.Value == gcpVendor.ProjectId
					}),
					gomega.Satisfy(func(v *github.ActionsVariable) bool {
						return v.Name == "PROJECT_NUMBER" &&
							v.Value == gcpVendor.ProjectNumber
					}),
				))
			}
		}, ginkgo.NodeTimeout(time.Minute))

		ginkgo.It("created a PR with new app link for the tenant", func(ctx ginkgo.SpecContext) {
			prList, _, err := githubClient.PullRequests.List(
				ctx,
				cfgDetails.CPlatformRepoName.Organization(),
				cfgDetails.CPlatformRepoName.Name(),
				&github.PullRequestListOptions{
					Head: cfgDetails.CPlatformRepoName.Organization() + ":" + testconfig.Cfg.Tenant + "-add-repo-" + newAppName,
					Base: git.MainBranch,
				},
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(prList).To(gomega.HaveLen(1))
			gomega.Expect(prList[0]).NotTo(gomega.BeNil())
			pr := prList[0]

			gomega.Expect(pr.GetTitle()).To(gomega.Equal("Add new repository " + newAppName + " for tenant " + testconfig.Cfg.Tenant))
			gomega.Expect(pr.GetState()).To(gomega.Equal("open"))

			prFiles, _, err := githubClient.PullRequests.ListFiles(
				ctx,
				cfgDetails.CPlatformRepoName.Organization(),
				cfgDetails.CPlatformRepoName.Name(),
				pr.GetNumber(),
				&github.ListOptions{},
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(prFiles).To(gomega.HaveLen(1))
			prFile := prFiles[0]

			gomega.Expect(prFile.GetStatus()).To(gomega.Equal("modified"))
			gomega.Expect(prFile.GetFilename()).To(gomega.Equal("tenants/tenants/" + testconfig.Cfg.Tenant + ".yaml"))
		}, ginkgo.SpecTimeout(time.Minute))
	})

	ginkgo.Context("create with --dry-run", ginkgo.Ordered, func() {
		var (
			newAppName string
			appDir     string
		)

		ginkgo.BeforeAll(func(ctx ginkgo.SpecContext) {
			newAppName = "new-test-app-dryrun-" + testRunId
			appDir = filepath.Join(homeDir, newAppName)
			_, err := corectl.Run(
				"application", "create", newAppName, appDir,
				"-t", testdata.BlankTemplate(),
				"--tenant", testconfig.Cfg.Tenant,
				"--non-interactive",
				"--dry-run")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}, ginkgo.NodeTimeout(time.Minute))

		ginkgo.It("did not create a new repository for the new app", func(ctx ginkgo.SpecContext) {
			_, _, err := githubClient.Repositories.Get(
				ctx,
				cfg.GitHub.Organization.Value,
				newAppName,
			)
			gomega.Expect(err.Error()).To(gomega.Equal(fmt.Sprintf("GET https://api.github.com/repos/%s/%s: 404 Not Found []", cfg.GitHub.Organization.Value, newAppName)))
		}, ginkgo.NodeTimeout(time.Minute))
	})

	ginkgo.Context("create in monorepo mode", ginkgo.Ordered, func() {
		var (
			monorepoName string
			monorepoDir  string
			newAppName   string
			appDir       string
		)

		ginkgo.BeforeAll(func(ctx ginkgo.SpecContext) {
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
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}, ginkgo.NodeTimeout(2*time.Minute))

		ginkgo.AfterAll(func(ctx ginkgo.SpecContext) {
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
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}, ginkgo.NodeTimeout(time.Minute))

		ginkgo.It("created a PR for the new app in the monorepo", func(ctx ginkgo.SpecContext) {
			prList, _, err := githubClient.PullRequests.List(
				ctx,
				cfg.GitHub.Organization.Value,
				monorepoName,
				&github.PullRequestListOptions{
					Head: cfg.GitHub.Organization.Value + ":add-" + newAppName,
					Base: git.MainBranch,
				},
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(prList).To(gomega.HaveLen(1))
			gomega.Expect(prList[0]).NotTo(gomega.BeNil())
			pr := prList[0]

			gomega.Expect(pr.GetTitle()).To(gomega.Equal("Add " + newAppName + " application"))
			gomega.Expect(pr.GetState()).To(gomega.Equal("open"))

			prFiles, _, err := githubClient.PullRequests.ListFiles(
				ctx,
				cfg.GitHub.Organization.Value,
				monorepoName,
				pr.GetNumber(),
				&github.ListOptions{},
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(len(prFiles)).To(gomega.BeNumerically(">", 0))

			// Check for specific files in the PR
			expectedFiles := []string{
				fmt.Sprintf("%s/README.md", newAppName),
				fmt.Sprintf(".github/workflows/%s-fast-feedback.yaml", newAppName),
				fmt.Sprintf(".github/workflows/%s-extended-test.yaml", newAppName),
			}

			actualFiles := make([]string, len(prFiles))
			for i, file := range prFiles {
				actualFiles[i] = file.GetFilename()
			}

			gomega.Expect(actualFiles).To(gomega.ConsistOf(expectedFiles))
		}, ginkgo.SpecTimeout(time.Minute))

		ginkgo.It("did not create a new repository for the app", func(ctx ginkgo.SpecContext) {
			_, _, err := githubClient.Repositories.Get(
				ctx,
				cfg.GitHub.Organization.Value,
				newAppName,
			)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*github.ErrorResponse).Response.StatusCode).To(gomega.Equal(404))
		}, ginkgo.NodeTimeout(time.Minute))

		ginkgo.It("did not create a PR for updating tenant configuration", func(ctx ginkgo.SpecContext) {
			prList, _, err := githubClient.PullRequests.List(
				ctx,
				cfgDetails.CPlatformRepoName.Organization(),
				cfgDetails.CPlatformRepoName.Name(),
				&github.PullRequestListOptions{
					Head: cfgDetails.CPlatformRepoName.Organization() + ":" + testconfig.Cfg.Tenant + "-add-repo-" + newAppName,
					Base: git.MainBranch,
				},
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(prList).To(gomega.HaveLen(0))
		}, ginkgo.NodeTimeout(time.Minute))
	})
})

func createMonorepoRepositoryRemoteAndLocal(githubClient *github.Client, ctx context.Context, cfg *config.Config, monorepoName string, monorepoDir string) {
	// Create remote repository
	isPrivate := true
	tmpRepo, _, err := githubClient.Repositories.Create(
		ctx,
		cfg.GitHub.Organization.Value,
		&github.Repository{
			Name:    &monorepoName,
			Private: &isPrivate,
		},
	)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Create local repository
	localRepo, err := git.InitLocalRepository(monorepoDir, false)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Set up git config
	testsetup.SetupGitRepoConfigFromOtherRepo(".", localRepo.Repository())

	// Add remote
	gomega.Expect(localRepo.SetRemote(tmpRepo.GetCloneURL())).To(gomega.Succeed())

	// Create initial commit
	gomega.Expect(localRepo.AddAll()).To(gomega.Succeed())
	gomega.Expect(localRepo.Commit(&git.CommitOp{Message: "Initial commit"})).To(gomega.Succeed())

	// Push to remote
	gitAuth := git.UrlTokenAuthMethod(testconfig.Cfg.GitHubToken)
	gomega.Expect(localRepo.Push(git.PushOp{Auth: gitAuth})).To(gomega.Succeed())
}
