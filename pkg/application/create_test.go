package application

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/coreeng/corectl/pkg/testutil/gittest"
	"github.com/coreeng/corectl/pkg/testutil/httpmock"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/developer-platform/pkg/environment"
	coretnt "github.com/coreeng/developer-platform/pkg/tenant"
	"github.com/google/go-github/v59/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"path/filepath"
	"slices"
)

var _ = Describe("Create new application", func() {
	t := GinkgoTB()

	var (
		cplatformServerRepo *gittest.BareRepository
		cplatformLocalRepo  *git.LocalRepository
		templatesServerRepo *gittest.BareRepository
		templatesLocalRepo  *git.LocalRepository
		newAppServerRepo    *gittest.BareRepository

		newRepoId  int64
		newAppName string
		githubOrg  string

		defaultTenant *coretnt.Tenant
		devEnv        environment.Environment
		prodEnv       environment.Environment

		createRepoCapture    *httpmock.HttpCaptureHandler[github.Repository]
		createRepoVarCapture *httpmock.HttpCaptureHandler[httpmock.ActionVariableRequest]
		createEnvCapture     *httpmock.HttpCaptureHandler[httpmock.CreateUpdateEnvRequest]
		createEnvVarCapture  *httpmock.HttpCaptureHandler[httpmock.ActionEnvVariableRequest]
		githubClient         *github.Client
	)
	BeforeEach(OncePerOrdered, func() {
		newRepoId = 1234
		newAppName = "new-app-name"
		githubOrg = "github-org-name"

		var err error
		cplatformServerRepo, cplatformLocalRepo, err = gittest.CreateBareAndLocalRepoFromDir(&gittest.CreateBareAndLocalRepoOp{
			SourceDir:          testdata.CPlatformEnvsPath(),
			TargetBareRepoDir:  t.TempDir(),
			TargetLocalRepoDir: t.TempDir(),
		})
		Expect(err).NotTo(HaveOccurred())

		templatesServerRepo, templatesLocalRepo, err = gittest.CreateBareAndLocalRepoFromDir(&gittest.CreateBareAndLocalRepoOp{
			SourceDir:          testdata.TemplatesPath(),
			TargetBareRepoDir:  t.TempDir(),
			TargetLocalRepoDir: t.TempDir(),
		})
		Expect(err).NotTo(HaveOccurred())

		newAppServerRepo, err = gittest.InitBareRepository(t.TempDir())
		Expect(err).NotTo(HaveOccurred())

		defaultTenant, err = coretnt.FindByName(coretnt.DirFromCPlatformPath(cplatformLocalRepo.Path()), testdata.DefaultTenant())
		Expect(err).NotTo(HaveOccurred())

		allEnvs, err := environment.List(environment.DirFromCPlatformRepoPath(cplatformLocalRepo.Path()))
		Expect(err).NotTo(HaveOccurred())
		devEnvIdx := slices.IndexFunc(allEnvs, func(e environment.Environment) bool {
			return e.Environment == testdata.DevEnvironment()
		})
		prodEnvIdx := slices.IndexFunc(allEnvs, func(e environment.Environment) bool {
			return e.Environment == testdata.ProdEnvironment()
		})
		Expect(devEnvIdx).To(BeNumerically(">=", 0))
		Expect(prodEnvIdx).To(BeNumerically(">=", 0))
		devEnv = allEnvs[devEnvIdx]
		prodEnv = allEnvs[prodEnvIdx]

		newAppCloneUrl := newAppServerRepo.LocalCloneUrl()
		createRepoCapture = httpmock.NewCaptureHandler[github.Repository](
			&github.Repository{
				ID:   &newRepoId,
				Name: &newAppName,
				Owner: &github.User{
					Login: &githubOrg,
				},
				CloneURL: &newAppCloneUrl,
			})
		createRepoVarCapture = httpmock.NewCreateActionVariablesCapture()
		createEnvCapture = httpmock.NewCreateUpdateEnvCapture()
		createEnvVarCapture = httpmock.NewCreateActionEnvVariablesCapture()

		githubClient = github.NewClient(mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.PostOrgsReposByOrg,
				createRepoCapture.Func(),
			),
			mock.WithRequestMatchHandler(
				mock.PostReposActionsVariablesByOwnerByRepo,
				createRepoVarCapture.Func(),
			),
			mock.WithRequestMatchHandler(
				mock.PutReposEnvironmentsByOwnerByRepoByEnvironmentName,
				createEnvCapture.Func(),
			),
			mock.WithRequestMatchHandler(
				mock.PostRepositoriesEnvironmentsVariablesByRepositoryIdByEnvironmentName,
				createEnvVarCapture.Func(),
			),
		))
		Expect(cplatformServerRepo)
		Expect(templatesServerRepo)
	})

	Context("from template", Ordered, func() {
		var (
			createResult    CreateResult
			newAppLocalRepo *git.LocalRepository
			localAppRepoDir string
		)
		BeforeAll(func() {
			templateToUse, err := template.FindByName(templatesLocalRepo.Path(), testdata.BlankTemplate())
			Expect(err).NotTo(HaveOccurred())

			localAppRepoDir = t.TempDir()
			createResult, err = Create(CreateOp{
				Name:             "new-app-name",
				OrgName:          "github-org-name",
				LocalPath:        localAppRepoDir,
				Tenant:           defaultTenant,
				FastFeedbackEnvs: []environment.Environment{devEnv},
				ExtendedTestEnvs: []environment.Environment{devEnv},
				ProdEnvs:         []environment.Environment{prodEnv},
				Template: &template.FulfilledTemplate{
					Spec: templateToUse,
					Arguments: []template.Argument{
						{
							Name:  "name",
							Value: "new-app-name",
						},
						{
							Name:  "tenant",
							Value: defaultTenant.Name,
						},
					},
				},
			}, githubClient)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns correct repository name", func() {
			Expect(createResult.RepositoryFullname.Name()).To(Equal(newAppName))
			Expect(createResult.RepositoryFullname.Organization()).To(Equal(githubOrg))
		})
		It("created new repo", func() {
			Expect(createRepoCapture.Requests).To(HaveLen(1))
			newRepoReq := createRepoCapture.Requests[0]
			Expect(*newRepoReq.Name).To(Equal(newAppName))
			Expect(*newRepoReq.DeleteBranchOnMerge).To(BeTrue())
			Expect(*newRepoReq.Visibility).To(Equal("private"))
		})
		It("created variables in new repo", func() {
			Expect(createRepoVarCapture.Requests).To(ConsistOf(
				Satisfy(func(v httpmock.ActionVariableRequest) bool {
					return v.Var.Name == "TENANT_NAME" &&
						v.Var.Value == defaultTenant.Name
				}),
				Satisfy(func(v httpmock.ActionVariableRequest) bool {
					return v.Var.Name == "FAST_FEEDBACK" &&
						v.Var.Value == fmt.Sprintf("{\"include\":[{\"deploy_env\":\"%s\"}]}", devEnv.Environment)
				}),
				Satisfy(func(v httpmock.ActionVariableRequest) bool {
					return v.Var.Name == "EXTENDED_TEST" &&
						v.Var.Value == fmt.Sprintf("{\"include\":[{\"deploy_env\":\"%s\"}]}", devEnv.Environment)
				}),
				Satisfy(func(v httpmock.ActionVariableRequest) bool {
					return v.Var.Name == "PROD" &&
						v.Var.Value == fmt.Sprintf("{\"include\":[{\"deploy_env\":\"%s\"}]}", prodEnv.Environment)
				}),
			))
			Expect(createRepoVarCapture.Requests).To(HaveEach(Satisfy(func(v httpmock.ActionVariableRequest) bool {
				return v.Org == githubOrg &&
					v.RepoName == newAppName
			})))
		})
		It("created environments in new repo", func() {
			Expect(createEnvCapture.Requests).To(ConsistOf(
				Satisfy(func(r httpmock.CreateUpdateEnvRequest) bool {
					return r.EnvName == testdata.DevEnvironment()
				}),
				Satisfy(func(r httpmock.CreateUpdateEnvRequest) bool {
					return r.EnvName == testdata.ProdEnvironment()
				}),
			))
			Expect(createEnvCapture.Requests).To(HaveEach(Satisfy(func(r httpmock.CreateUpdateEnvRequest) bool {
				return r.Org == githubOrg &&
					r.RepoName == newAppName
			})))
		})
		It("configured environments with variables", func() {
			Expect(createEnvVarCapture.Requests).To(HaveLen(10))
			for _, env := range []environment.Environment{devEnv, prodEnv} {
				var envRelatedRequests []httpmock.ActionEnvVariableRequest
				for _, r := range createEnvVarCapture.Requests {
					if r.EnvName == env.Environment {
						envRelatedRequests = append(envRelatedRequests, r)
					}
				}
				gcpVendor := env.Platform.(*environment.GCPVendor)
				Expect(envRelatedRequests).To(ConsistOf(
					Satisfy(func(r httpmock.ActionEnvVariableRequest) bool {
						return r.Var.Name == "DPLATFORM" &&
							r.Var.Value == env.Environment
					}),
					Satisfy(func(r httpmock.ActionEnvVariableRequest) bool {
						return r.Var.Name == "BASE_DOMAIN" &&
							r.Var.Value == env.GetDefaultIngressDomain().Domain
					}),
					Satisfy(func(r httpmock.ActionEnvVariableRequest) bool {
						return r.Var.Name == "INTERNAL_SERVICES_DOMAIN" &&
							r.Var.Value == env.InternalServices.Domain
					}),
					Satisfy(func(r httpmock.ActionEnvVariableRequest) bool {
						return r.Var.Name == "PROJECT_ID" &&
							r.Var.Value == gcpVendor.ProjectId
					}),
					Satisfy(func(r httpmock.ActionEnvVariableRequest) bool {
						return r.Var.Name == "PROJECT_NUMBER" &&
							r.Var.Value == gcpVendor.ProjectNumber
					}),
				))
				Expect(envRelatedRequests).To(HaveEach(Satisfy(func(r httpmock.ActionEnvVariableRequest) bool {
					return r.RepoID == newRepoId
				})))
			}
		})
		It("local repository is present and correct", func() {
			var err error
			newAppLocalRepo, err = git.OpenLocalRepository(localAppRepoDir)
			Expect(err).NotTo(HaveOccurred())

			remote, err := newAppLocalRepo.Repository().Remote(git.OriginRemote)
			Expect(err).NotTo(HaveOccurred())
			Expect(remote.Config().URLs).To(ConsistOf(newAppServerRepo.LocalCloneUrl()))

			currentBranch, err := newAppLocalRepo.CurrentBranch()
			Expect(err).NotTo(HaveOccurred())
			Expect(currentBranch).To(Equal(git.MainBranch))

			localChangesPresent, err := newAppLocalRepo.IsLocalChangesPresent()
			Expect(err).NotTo(HaveOccurred())
			Expect(localChangesPresent).To(BeFalse())

			Expect(filepath.Join(localAppRepoDir, "README.md")).To(BeARegularFile())
		})
		It("creates a commit with a rendered template", func() {
			head, err := newAppLocalRepo.Repository().Head()
			Expect(err).NotTo(HaveOccurred())
			newAppServerRepo.AssertCommits(gittest.AssertCommitOp{
				To: head.Hash(),
				ExpectedCommits: []gittest.ExpectedCommit{
					{
						Message: "Initial commit\n[skip ci]",
						ChangedFiles: []string{
							"README.md",
							".github/workflows/fast-feedback.yaml",
							".github/workflows/extended-test.yaml",
						},
					},
				},
			})
		})
		It("pushes all the changes to the remote repository", func() {
			newAppServerRepo.AssertInSyncWith(newAppLocalRepo)
		})

		It("returns empty PR url", func() {
			Expect(createResult.PRUrl).To(Equal(""))
		})

		It("returns false for monorepo mode", func() {
			Expect(createResult.MonorepoMode).To(BeFalse())
		})

	})

	Context("monorepo mode", Ordered, func() {
		var (
			monorepoServerRepo *gittest.BareRepository
			monorepoLocalRepo  *git.LocalRepository
			newAppLocalPath    string
			createResult       CreateResult
			createOp           CreateOp
			getRepoCapture     *httpmock.HttpCaptureHandler[any]
			createPrCapture    *httpmock.HttpCaptureHandler[github.NewPullRequest]
			appName            = "new-app-name"
			newPrHtmlUrl       = "https://github.com/org/repo/pull/1"
		)

		BeforeAll(func() {
			var err error
			monorepoServerRepo, monorepoLocalRepo, err = gittest.CreateBareAndLocalRepoFromDir(&gittest.CreateBareAndLocalRepoOp{
				SourceDir:          filepath.Join(testdata.TemplatesPath(), testdata.Monorepo()),
				TargetBareRepoDir:  t.TempDir(),
				TargetLocalRepoDir: t.TempDir(),
			})
			Expect(err).NotTo(HaveOccurred())

			templateToUse, err := template.FindByName(templatesLocalRepo.Path(), testdata.BlankTemplate())
			Expect(err).NotTo(HaveOccurred())

			newAppLocalPath = filepath.Join(monorepoLocalRepo.Path(), appName)

			createOp = CreateOp{
				Name:             appName,
				OrgName:          "github-org-name",
				LocalPath:        newAppLocalPath,
				Tenant:           defaultTenant,
				FastFeedbackEnvs: []environment.Environment{devEnv},
				ExtendedTestEnvs: []environment.Environment{devEnv},
				ProdEnvs:         []environment.Environment{prodEnv},
				Template: &template.FulfilledTemplate{
					Spec: templateToUse,
					Arguments: []template.Argument{
						{
							Name:  "name",
							Value: appName,
						},
						{
							Name:  "tenant",
							Value: defaultTenant.Name,
						},
					},
				},
			}

			url := monorepoServerRepo.LocalCloneUrl()
			response := &github.Repository{
				ID:   &newRepoId,
				Name: &newAppName,
				Owner: &github.User{
					Login: &githubOrg,
				},
				CloneURL: &url,
			}
			fmt.Println(response)
			getRepoCapture = httpmock.NewCaptureHandler[any](response)

			createPrCapture = httpmock.NewCaptureHandler[github.NewPullRequest](
				&github.PullRequest{
					HTMLURL: &newPrHtmlUrl,
				},
			)

			githubClient = github.NewClient(mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.GetReposByOwnerByRepo,
					getRepoCapture.Func(),
				),
				mock.WithRequestMatchHandler(
					mock.PostReposPullsByOwnerByRepo,
					createPrCapture.Func(),
				),
			))

			// Execute Create function
			createResult, err = Create(createOp, githubClient)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns true for monorepo mode", func() {
			Expect(createResult.MonorepoMode).To(BeTrue())
		})

		It("creates the new application directory within the monorepo", func() {
			Expect(newAppLocalPath).To(BeADirectory())
		})

		It("moves GitHub workflows to the root .github/workflows directory", func() {
			rootWorkflowsPath := filepath.Join(monorepoLocalRepo.Path(), ".github", "workflows")
			Expect(filepath.Join(rootWorkflowsPath, "new-app-name-fast-feedback.yaml")).To(BeAnExistingFile())
			Expect(filepath.Join(rootWorkflowsPath, "new-app-name-extended-test.yaml")).To(BeAnExistingFile())
		})

		It("renders template with passed arguments", func() {
			rootWorkflowsPath := filepath.Join(monorepoLocalRepo.Path(), ".github", "workflows")

			content := readFileContent(rootWorkflowsPath, "new-app-name-fast-feedback.yaml")
			Expect(content).To(ContainSubstring(defaultTenant.Name))
			Expect(content).To(ContainSubstring(appName))

			content = readFileContent(rootWorkflowsPath, "new-app-name-extended-test.yaml")
			Expect(content).To(ContainSubstring(defaultTenant.Name))
			Expect(content).To(ContainSubstring(appName))
		})

		It("does not leave .github directory in the new application directory", func() {
			Expect(filepath.Join(newAppLocalPath, ".github")).NotTo(BeADirectory())
		})

		It("commits changes to the monorepo", func() {
			head, err := monorepoLocalRepo.Repository().Head()
			Expect(err).NotTo(HaveOccurred())
			monorepoServerRepo.AssertCommits(gittest.AssertCommitOp{
				To: head.Hash(),
				ExpectedCommits: []gittest.ExpectedCommit{
					{
						Message: "New app: new-app-name\n[skip ci]",
						ChangedFiles: []string{
							"new-app-name/README.md",
							".github/workflows/new-app-name-fast-feedback.yaml",
							".github/workflows/new-app-name-extended-test.yaml",
						},
					},
				},
			})
		})

		It("pushes all the changes to the remote repository", func() {
			monorepoServerRepo.AssertInSyncWith(monorepoLocalRepo)
		})

		It("returns correct PR url", func() {
			Expect(createResult.PRUrl).To(Equal(newPrHtmlUrl))
		})

		It("called create PR correctly", func() {
			Expect(createPrCapture.Requests).To(HaveLen(1))
			newPrRequest := createPrCapture.Requests[0]
			Expect(*newPrRequest.Title).To(Equal("Add new-app-name application"))
			Expect(*newPrRequest.Body).To(Equal("Adding `new-app-name` application"))
			Expect(*newPrRequest.Head).To(Equal("add-" + appName))
			Expect(*newPrRequest.Base).To(Equal(git.MainBranch))
		})
	})
	Context("monorepo mode - when in error", Ordered, func() {
		var (
			newAppLocalPath   string
			monorepoLocalPath string
		)
		BeforeAll(func() {
			_, monorepoLocalRepo, err := gittest.CreateBareAndLocalRepoFromDir(&gittest.CreateBareAndLocalRepoOp{
				SourceDir:          filepath.Join(testdata.TemplatesPath(), testdata.Monorepo()),
				TargetBareRepoDir:  t.TempDir(),
				TargetLocalRepoDir: t.TempDir(),
			})
			Expect(err).NotTo(HaveOccurred())

			templateToUse, _ := template.FindByName(templatesLocalRepo.Path(), "incorrect-template")

			monorepoLocalPath = monorepoLocalRepo.Path()
			newAppLocalPath = filepath.Join(monorepoLocalRepo.Path(), "app-with-error")

			executeCreate := func() {
				createOp := CreateOp{
					Name:             "app-with-error",
					OrgName:          "github-org-name",
					LocalPath:        newAppLocalPath,
					Tenant:           defaultTenant,
					FastFeedbackEnvs: []environment.Environment{devEnv},
					ExtendedTestEnvs: []environment.Environment{devEnv},
					ProdEnvs:         []environment.Environment{prodEnv},
					Template: &template.FulfilledTemplate{
						Spec:      templateToUse,
						Arguments: []template.Argument{},
					},
				}

				_, _ = Create(createOp, githubClient)
			}

			Expect(executeCreate).To(Panic())
		})

		It("deletes newly created app directory", func() {
			Expect(newAppLocalPath).NotTo(BeADirectory())
		})

		It("don't delete monorepo directory", func() {
			Expect(monorepoLocalPath).To(BeADirectory())
		})

	})

})

func readFileContent(path ...string) string {
	filePath := filepath.Join(path...)
	content, err := os.ReadFile(filePath)
	Expect(err).NotTo(HaveOccurred())
	return string(content)
}
