package application

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/environment"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/coreeng/corectl/pkg/tenant"
	"github.com/coreeng/corectl/pkg/testutil/gittest"
	"github.com/coreeng/corectl/pkg/testutil/httpmock"
	"github.com/coreeng/corectl/testdata"
	"github.com/google/go-github/v59/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

		defaultTenant *tenant.Tenant
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

		defaultTenant, err = tenant.FindByName(cplatformLocalRepo.Path(), tenant.Name(testdata.DefaultTenant()))
		Expect(err).NotTo(HaveOccurred())

		allEnvs, err := environment.List(cplatformLocalRepo.Path())
		Expect(err).NotTo(HaveOccurred())
		devEnvIdx := slices.IndexFunc(allEnvs, func(e environment.Environment) bool {
			return e.Environment == environment.Name(testdata.DevEnvironment())
		})
		prodEnvIdx := slices.IndexFunc(allEnvs, func(e environment.Environment) bool {
			return e.Environment == environment.Name(testdata.ProdEnvironment())
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
					Spec:      templateToUse,
					Arguments: nil,
				},
			}, githubClient)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns correct repository name", func() {
			Expect(createResult.RepositoryFullname.Name).To(Equal(newAppName))
			Expect(createResult.RepositoryFullname.Organization).To(Equal(githubOrg))
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
						v.Var.Value == string(defaultTenant.Name)
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
					if r.EnvName == string(env.Environment) {
						envRelatedRequests = append(envRelatedRequests, r)
					}
				}
				Expect(envRelatedRequests).To(ConsistOf(
					Satisfy(func(r httpmock.ActionEnvVariableRequest) bool {
						return r.Var.Name == "DPLATFORM" &&
							r.Var.Value == string(env.Environment)
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
							r.Var.Value == env.Platform.ProjectId
					}),
					Satisfy(func(r httpmock.ActionEnvVariableRequest) bool {
						return r.Var.Name == "PROJECT_NUMBER" &&
							r.Var.Value == env.Platform.ProjectNumber
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
						Message:      "Initial commit\n[skip ci]",
						ChangedFiles: []string{"./README.md"},
					},
				},
			})
		})
		It("pushes all the changes to the remote repository", func() {
			newAppServerRepo.AssertInSyncWith(newAppLocalRepo)
		})
	})
})
