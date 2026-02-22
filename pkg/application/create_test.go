package application

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
	"github.com/stretchr/testify/assert"

	"github.com/coreeng/core-platform/pkg/environment"
	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/cmd/template/render"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/coreeng/corectl/pkg/testutil/gittest"
	"github.com/coreeng/corectl/pkg/testutil/httpmock"
	"github.com/coreeng/corectl/testdata"
	"github.com/google/go-github/v60/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Create new application", func() {
	t := GinkgoTB()

	var (
		cplatformServerRepo *gittest.BareRepository
		templatesServerRepo *gittest.BareRepository
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
		renderer             *render.StubTemplateRenderer
		service              *Service
	)
	BeforeEach(OncePerOrdered, func() {
		newRepoId = 1234
		newAppName = "new-app-name"
		githubOrg = "github-org-name"

		var err error
		_, err = gittest.CreateTestCorectlConfig(t.TempDir())
		assert.NoError(t, err)
		cplatformServerRepo, _, err = gittest.CreateBareAndLocalRepoFromDir(&gittest.CreateBareAndLocalRepoOp{
			SourceDir:          testdata.CPlatformEnvsPath(),
			TargetBareRepoDir:  t.TempDir(),
			TargetLocalRepoDir: configpath.GetCorectlCPlatformDir(),
		})
		Expect(err).NotTo(HaveOccurred())

		templatesServerRepo, _, err = gittest.CreateBareAndLocalRepoFromDir(&gittest.CreateBareAndLocalRepoOp{
			SourceDir:          testdata.TemplatesPath(),
			TargetBareRepoDir:  t.TempDir(),
			TargetLocalRepoDir: configpath.GetCorectlTemplatesDir(),
		})
		Expect(err).NotTo(HaveOccurred())

		newAppServerRepo, err = gittest.InitBareRepository(t.TempDir())
		Expect(err).NotTo(HaveOccurred())

		defaultTenant, err = coretnt.FindByName(configpath.GetCorectlCPlatformDir("tenants"), testdata.DefaultTenant())
		Expect(err).NotTo(HaveOccurred())

		allEnvs, err := environment.List(configpath.GetCorectlCPlatformDir("environments"))
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
			renderer = &render.StubTemplateRenderer{
				Renderer: &render.FlagsAwareTemplateRenderer{},
			}
			service = NewService(renderer, githubClient, false)
			templateToUse, err := template.FindByName(configpath.GetCorectlTemplatesDir(), testdata.BlankTemplate())
			Expect(err).NotTo(HaveOccurred())

			localAppRepoDir = t.TempDir()
			createResult, err = service.Create(CreateOp{
				Name:             "new-app-name",
				GitHubRepoName:   "",
				OrgName:          "github-org-name",
				LocalPath:        localAppRepoDir,
				Tenant:           defaultTenant,
				FastFeedbackEnvs: []environment.Environment{devEnv},
				ExtendedTestEnvs: []environment.Environment{devEnv},
				ProdEnvs:         []environment.Environment{prodEnv},
				Template:         templateToUse,
			})
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
			Expect(createEnvVarCapture.Requests).To(HaveLen(12))
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
					Satisfy(func(r httpmock.ActionEnvVariableRequest) bool {
						return r.Var.Name == "REGION" &&
							r.Var.Value == gcpVendor.Region
					}),
				))
				Expect(envRelatedRequests).To(HaveEach(Satisfy(func(r httpmock.ActionEnvVariableRequest) bool {
					return r.RepoID == newRepoId
				})))
			}
		})
		It("local repository is present and correct", func() {
			var err error
			newAppLocalRepo, err = git.OpenLocalRepository(localAppRepoDir, false)
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

		It("passes correct arguments to template", func() {
			Expect(renderer.PassedAdditionalArgs).To(HaveLen(1))
			// Args: name, description, tenant, working_directory, version_prefix (config is passed via configJSON)
			Expect(renderer.PassedAdditionalArgs[0]).To(HaveLen(5))

			// Verify key arguments are present
			argNames := make([]string, len(renderer.PassedAdditionalArgs[0]))
			for i, arg := range renderer.PassedAdditionalArgs[0] {
				argNames[i] = arg.Name
			}
			Expect(argNames).To(ContainElements("name", "description", "tenant", "working_directory", "version_prefix"))

			// Check specific values
			for _, arg := range renderer.PassedAdditionalArgs[0] {
				switch arg.Name {
				case "name":
					Expect(arg.Value).To(Equal(newAppName))
				case "description":
					Expect(arg.Value).To(Equal(""))
				case "tenant":
					Expect(arg.Value).To(Equal("default-tenant"))
				case "working_directory":
					Expect(arg.Value).To(Equal(""))
				case "version_prefix":
					Expect(arg.Value).To(Equal("v"))
				}
			}
		})

		It("renders template with passed arguments", func() {
			rootWorkflowsPath := filepath.Join(newAppLocalRepo.Path(), ".github", "workflows")

			content := readFileContent(rootWorkflowsPath, "fast-feedback.yaml")
			Expect(content).To(ContainSubstring("tenant: " + defaultTenant.Name))
			Expect(content).To(ContainSubstring("name: " + newAppName))
			Expect(content).To(ContainSubstring("working_directory: "))
			Expect(content).To(ContainSubstring("version_prefix: v"))

			content = readFileContent(rootWorkflowsPath, "extended-test.yaml")
			Expect(content).To(ContainSubstring("tenant: " + defaultTenant.Name))
			Expect(content).To(ContainSubstring("name: " + newAppName))
			Expect(content).To(ContainSubstring("working_directory: "))
			Expect(content).To(ContainSubstring("version_prefix: v"))
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
							"app.yaml",
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

	Context("from template with public visibility", Ordered, func() {
		var (
			createPublicRepoCapture *httpmock.HttpCaptureHandler[github.Repository]
		)
		BeforeAll(func() {
			renderer = &render.StubTemplateRenderer{
				Renderer: &render.FlagsAwareTemplateRenderer{},
			}

			newAppCloneUrl := newAppServerRepo.LocalCloneUrl()
			createPublicRepoCapture = httpmock.NewCaptureHandler[github.Repository](
				&github.Repository{
					ID:   &newRepoId,
					Name: &newAppName,
					Owner: &github.User{
						Login: &githubOrg,
					},
					CloneURL: &newAppCloneUrl,
				})

			publicGithubClient := github.NewClient(mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.PostOrgsReposByOrg,
					createPublicRepoCapture.Func(),
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

			service = NewService(renderer, publicGithubClient, false)
			templateToUse, err := template.FindByName(configpath.GetCorectlTemplatesDir(), testdata.BlankTemplate())
			Expect(err).NotTo(HaveOccurred())

			localAppRepoDir := t.TempDir()
			_, err = service.Create(CreateOp{
				Name:             "new-app-name",
				OrgName:          "github-org-name",
				LocalPath:        localAppRepoDir,
				Tenant:           defaultTenant,
				FastFeedbackEnvs: []environment.Environment{devEnv},
				ExtendedTestEnvs: []environment.Environment{devEnv},
				ProdEnvs:         []environment.Environment{prodEnv},
				Template:         templateToUse,
				Public:           true,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("created repo with public visibility", func() {
			Expect(createPublicRepoCapture.Requests).To(HaveLen(1))
			newRepoReq := createPublicRepoCapture.Requests[0]
			Expect(*newRepoReq.Visibility).To(Equal("public"))
		})
	})

	Context("from template with config", Ordered, func() {
		var (
			createResult    CreateResult
			localAppRepoDir string
		)
		BeforeAll(func() {
			renderer = &render.StubTemplateRenderer{
				Renderer: &render.FlagsAwareTemplateRenderer{},
			}
			service = NewService(renderer, githubClient, false)
			templateToUse, err := template.FindByName(configpath.GetCorectlTemplatesDir(), testdata.BlankTemplate())
			Expect(err).NotTo(HaveOccurred())

			localAppRepoDir = t.TempDir()
			createResult, err = service.Create(CreateOp{
				Name:             "new-app-name",
				OrgName:          "github-org-name",
				LocalPath:        localAppRepoDir,
				Tenant:           defaultTenant,
				FastFeedbackEnvs: []environment.Environment{devEnv},
				ExtendedTestEnvs: []environment.Environment{devEnv},
				ProdEnvs:         []environment.Environment{prodEnv},
				Template:         templateToUse,
				Config:           `{"resources":{"requests":{"cpu":"500m","memory":"512Mi"},"limits":{"cpu":"1000m","memory":"1024Mi"}}}`,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns correct repository name", func() {
			Expect(createResult.RepositoryFullname.Name()).To(Equal(newAppName))
		})

		It("passes config JSON to template renderer", func() {
			Expect(renderer.PassedAdditionalArgs).To(HaveLen(1))
			// Args: name, description, tenant, working_directory, version_prefix (config is passed via configJSON)
			Expect(renderer.PassedAdditionalArgs[0]).To(HaveLen(5))

			// Verify configJSON was passed correctly
			Expect(renderer.PassedConfigJSON).To(HaveLen(1))
			Expect(renderer.PassedConfigJSON[0]).To(Equal(`{"resources":{"requests":{"cpu":"500m","memory":"512Mi"},"limits":{"cpu":"1000m","memory":"1024Mi"}}}`))
		})

		It("creates app.yaml file with merged config at repo root", func() {
			appYamlPath := filepath.Join(localAppRepoDir, "app.yaml")
			Expect(appYamlPath).To(BeAnExistingFile())

			content, err := os.ReadFile(appYamlPath)
			Expect(err).NotTo(HaveOccurred())

			// Verify starts with --- and fields are in order: name, description, config
			Expect(string(content)).To(HavePrefix("---\nname: new-app-name\n"))
			Expect(string(content)).To(ContainSubstring("description:"))
			Expect(string(content)).To(ContainSubstring("config:"))
			Expect(string(content)).To(ContainSubstring("  resources:"))
			Expect(string(content)).To(ContainSubstring("cpu: 500m"))
			Expect(string(content)).To(ContainSubstring("memory: 512Mi"))
			Expect(string(content)).To(ContainSubstring("cpu: 1000m"))
			Expect(string(content)).To(ContainSubstring("memory: 1024Mi"))
		})
	})

	Context("from template with invalid config JSON", func() {
		It("returns error for invalid JSON", func() {
			renderer = &render.StubTemplateRenderer{
				Renderer: &render.FlagsAwareTemplateRenderer{},
			}
			service = NewService(renderer, githubClient, false)
			templateToUse, err := template.FindByName(configpath.GetCorectlTemplatesDir(), testdata.BlankTemplate())
			Expect(err).NotTo(HaveOccurred())

			localAppRepoDir := t.TempDir()
			_, err = service.Create(CreateOp{
				Name:             "new-app-name",
				OrgName:          "github-org-name",
				LocalPath:        localAppRepoDir,
				Tenant:           defaultTenant,
				FastFeedbackEnvs: []environment.Environment{devEnv},
				ExtendedTestEnvs: []environment.Environment{devEnv},
				ProdEnvs:         []environment.Environment{prodEnv},
				Template:         templateToUse,
				Config:           `{invalid json}`,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid config JSON"))
		})
	})

	Context("from template without --config flag and no config section in template.yaml", func() {
		var localAppRepoDir string

		BeforeEach(func() {
			renderer = &render.StubTemplateRenderer{
				Renderer: &render.FlagsAwareTemplateRenderer{},
			}
			service = NewService(renderer, githubClient, false)
			// Blank template has no config: section in template.yaml
			templateToUse, err := template.FindByName(configpath.GetCorectlTemplatesDir(), testdata.BlankTemplate())
			Expect(err).NotTo(HaveOccurred())

			localAppRepoDir = t.TempDir()
			_, err = service.Create(CreateOp{
				Name:             "new-app-name",
				OrgName:          "github-org-name",
				LocalPath:        localAppRepoDir,
				Tenant:           defaultTenant,
				FastFeedbackEnvs: []environment.Environment{devEnv},
				ExtendedTestEnvs: []environment.Environment{devEnv},
				ProdEnvs:         []environment.Environment{prodEnv},
				Template:         templateToUse,
				Config:           "", // Empty --config flag
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("passes empty config JSON when template has no config section", func() {
			Expect(renderer.PassedAdditionalArgs).To(HaveLen(1))
			// Args: name, description, tenant, working_directory, version_prefix (config is passed via configJSON)
			Expect(renderer.PassedAdditionalArgs[0]).To(HaveLen(5))

			// Verify empty configJSON was passed
			Expect(renderer.PassedConfigJSON).To(HaveLen(1))
			Expect(renderer.PassedConfigJSON[0]).To(BeEmpty())
		})

		It("creates app.yaml with empty config when there is no config", func() {
			appYamlPath := filepath.Join(localAppRepoDir, "app.yaml")
			Expect(appYamlPath).To(BeAnExistingFile())

			content, err := os.ReadFile(appYamlPath)
			Expect(err).NotTo(HaveOccurred())
			// Verify starts with --- and fields are in order: name, description, config
			Expect(string(content)).To(HavePrefix("---\nname: new-app-name\n"))
			Expect(string(content)).To(ContainSubstring("description:"))
			Expect(string(content)).To(ContainSubstring("config: {}"))
		})
	})

	Context("when template has config section and --config provides overrides", func() {
		It("deep merges template config with --config overrides in app.yaml", func() {
			renderer = &render.StubTemplateRenderer{
				Renderer: &render.FlagsAwareTemplateRenderer{},
			}
			service = NewService(renderer, githubClient, false)
			templateToUse, err := template.FindByName(configpath.GetCorectlTemplatesDir(), testdata.BlankTemplate())
			Expect(err).NotTo(HaveOccurred())

			// Simulate template.yaml having a config section
			templateToUse.Config = map[string]any{
				"replicas": 2,
				"resources": map[string]any{
					"limits": map[string]any{
						"cpu":    "1000m",
						"memory": "1024Mi",
					},
				},
			}

			localAppRepoDir := t.TempDir()
			_, err = service.Create(CreateOp{
				Name:             "new-app-name",
				OrgName:          "github-org-name",
				LocalPath:        localAppRepoDir,
				Tenant:           defaultTenant,
				FastFeedbackEnvs: []environment.Environment{devEnv},
				ExtendedTestEnvs: []environment.Environment{devEnv},
				ProdEnvs:         []environment.Environment{prodEnv},
				Template:         templateToUse,
				Config:           `{"resources":{"limits":{"cpu":"2000m"}}}`, // Override only CPU
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify configJSON was passed correctly
			Expect(renderer.PassedConfigJSON).To(HaveLen(1))
			Expect(renderer.PassedConfigJSON[0]).To(Equal(`{"resources":{"limits":{"cpu":"2000m"}}}`))

			// Verify deep merge in generated app.yaml
			appYamlPath := filepath.Join(localAppRepoDir, "app.yaml")
			content, err := os.ReadFile(appYamlPath)
			Expect(err).NotTo(HaveOccurred())

			// Verify merged config values
			Expect(string(content)).To(ContainSubstring("replicas: 2"))    // From template config
			Expect(string(content)).To(ContainSubstring("cpu: 2000m"))     // Overridden by --config
			Expect(string(content)).To(ContainSubstring("memory: 1024Mi")) // From template config
		})
	})

	Context("when template has config section but --config is empty", func() {
		It("uses template config as-is in app.yaml", func() {
			renderer = &render.StubTemplateRenderer{
				Renderer: &render.FlagsAwareTemplateRenderer{},
			}
			service = NewService(renderer, githubClient, false)
			templateToUse, err := template.FindByName(configpath.GetCorectlTemplatesDir(), testdata.BlankTemplate())
			Expect(err).NotTo(HaveOccurred())

			// Simulate template.yaml having a config section
			templateToUse.Config = map[string]any{
				"replicas": 3,
				"resources": map[string]any{
					"limits": map[string]any{
						"cpu":    "500m",
						"memory": "512Mi",
					},
				},
			}

			localAppRepoDir := t.TempDir()
			_, err = service.Create(CreateOp{
				Name:             "new-app-name",
				OrgName:          "github-org-name",
				LocalPath:        localAppRepoDir,
				Tenant:           defaultTenant,
				FastFeedbackEnvs: []environment.Environment{devEnv},
				ExtendedTestEnvs: []environment.Environment{devEnv},
				ProdEnvs:         []environment.Environment{prodEnv},
				Template:         templateToUse,
				Config:           "", // Empty --config
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify empty configJSON was passed
			Expect(renderer.PassedConfigJSON).To(HaveLen(1))
			Expect(renderer.PassedConfigJSON[0]).To(BeEmpty())

			// Verify template config is used in app.yaml
			appYamlPath := filepath.Join(localAppRepoDir, "app.yaml")
			content, err := os.ReadFile(appYamlPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("replicas: 3"))
			Expect(string(content)).To(ContainSubstring("cpu: 500m"))
			Expect(string(content)).To(ContainSubstring("memory: 512Mi"))
		})
	})

	Context("when --config adds new keys not in template config", func() {
		var localAppRepoDir string

		BeforeEach(func() {
			renderer = &render.StubTemplateRenderer{
				Renderer: &render.FlagsAwareTemplateRenderer{},
			}
			service = NewService(renderer, githubClient, false)
			templateToUse, err := template.FindByName(configpath.GetCorectlTemplatesDir(), testdata.BlankTemplate())
			Expect(err).NotTo(HaveOccurred())

			// Simulate template.yaml having a config section
			templateToUse.Config = map[string]any{
				"replicas": 2,
			}

			localAppRepoDir = t.TempDir()
			_, err = service.Create(CreateOp{
				Name:             "new-app-name",
				OrgName:          "github-org-name",
				LocalPath:        localAppRepoDir,
				Tenant:           defaultTenant,
				FastFeedbackEnvs: []environment.Environment{devEnv},
				ExtendedTestEnvs: []environment.Environment{devEnv},
				ProdEnvs:         []environment.Environment{prodEnv},
				Template:         templateToUse,
				Config:           `{"newKey":"newValue","resources":{"limits":{"cpu":"1000m"}}}`,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("passes configJSON to renderer", func() {
			// Verify configJSON was passed correctly
			Expect(renderer.PassedConfigJSON).To(HaveLen(1))
			Expect(renderer.PassedConfigJSON[0]).To(Equal(`{"newKey":"newValue","resources":{"limits":{"cpu":"1000m"}}}`))
		})

		It("creates app.yaml with merged config from template and --config", func() {
			appYamlPath := filepath.Join(localAppRepoDir, "app.yaml")
			Expect(appYamlPath).To(BeAnExistingFile())

			content, err := os.ReadFile(appYamlPath)
			Expect(err).NotTo(HaveOccurred())

			// Verify starts with --- and fields are in order: name, description, config
			Expect(string(content)).To(HavePrefix("---\nname: new-app-name\n"))
			Expect(string(content)).To(ContainSubstring("description:"))
			Expect(string(content)).To(ContainSubstring("config:"))
			// Verify both template config and --config values are present
			Expect(string(content)).To(ContainSubstring("replicas: 2"))
			Expect(string(content)).To(ContainSubstring("newKey: newValue"))
			Expect(string(content)).To(ContainSubstring("cpu: 1000m"))
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

			templateToUse, err := template.FindByName(configpath.GetCorectlTemplatesDir(), testdata.BlankTemplate())
			Expect(err).NotTo(HaveOccurred())
			Expect(templateToUse).NotTo(BeNil())

			newAppLocalPath = filepath.Join(monorepoLocalRepo.Path(), appName)

			createOp = CreateOp{
				Name:             appName,
				GitHubRepoName:   "",
				OrgName:          "github-org-name",
				LocalPath:        newAppLocalPath,
				Tenant:           defaultTenant,
				FastFeedbackEnvs: []environment.Environment{devEnv},
				ExtendedTestEnvs: []environment.Environment{devEnv},
				ProdEnvs:         []environment.Environment{prodEnv},
				Template:         templateToUse,
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

			renderer = &render.StubTemplateRenderer{
				Renderer: &render.FlagsAwareTemplateRenderer{},
			}
			service = NewService(renderer, githubClient, false)
			createResult, err = service.Create(createOp)
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

		It("passes correct arguments to template", func() {
			Expect(renderer.PassedAdditionalArgs).To(HaveLen(1))
			// Args: name, description, tenant, working_directory, version_prefix (config is passed via configJSON)
			Expect(renderer.PassedAdditionalArgs[0]).To(HaveLen(5))

			// Verify key arguments are present
			argNames := make([]string, len(renderer.PassedAdditionalArgs[0]))
			for i, arg := range renderer.PassedAdditionalArgs[0] {
				argNames[i] = arg.Name
			}
			Expect(argNames).To(ContainElements("name", "description", "tenant", "working_directory", "version_prefix"))

			// Check specific values
			for _, arg := range renderer.PassedAdditionalArgs[0] {
				switch arg.Name {
				case "name":
					Expect(arg.Value).To(Equal(appName))
				case "tenant":
					Expect(arg.Value).To(Equal("default-tenant"))
				case "working_directory":
					Expect(arg.Value).To(Equal(appName))
				case "version_prefix":
					Expect(arg.Value).To(Equal(appName + "/v"))
				}
			}
		})

		It("renders template with passed arguments", func() {
			rootWorkflowsPath := filepath.Join(monorepoLocalRepo.Path(), ".github", "workflows")

			content := readFileContent(rootWorkflowsPath, "new-app-name-fast-feedback.yaml")
			Expect(content).To(ContainSubstring("tenant: " + defaultTenant.Name))
			Expect(content).To(ContainSubstring("name: " + appName))
			Expect(content).To(ContainSubstring("working_directory: " + appName))
			Expect(content).To(ContainSubstring("version_prefix: " + appName + "/v"))

			content = readFileContent(rootWorkflowsPath, "new-app-name-extended-test.yaml")
			Expect(content).To(ContainSubstring("tenant: " + defaultTenant.Name))
			Expect(content).To(ContainSubstring("name: " + appName))
			Expect(content).To(ContainSubstring("working_directory: " + appName))
			Expect(content).To(ContainSubstring("version_prefix: " + appName + "/v"))
		})

		It("don't delete monorepo directory", func() {
			Expect(monorepoLocalRepo.Path()).To(BeADirectory())
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
							"new-app-name/app.yaml",
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

			templateToUse, _ := template.FindByName(configpath.GetCorectlTemplatesDir(), testdata.BlankTemplate())

			monorepoLocalPath = monorepoLocalRepo.Path()
			newAppLocalPath = filepath.Join(monorepoLocalRepo.Path(), "app-with-error")
			service = NewService(&panicTemplateRenderer{}, githubClient, false)
			executeCreate := func() {
				createOp := CreateOp{
					Name:             "app-with-error",
					GitHubRepoName:   "",
					OrgName:          "github-org-name",
					LocalPath:        newAppLocalPath,
					Tenant:           defaultTenant,
					FastFeedbackEnvs: []environment.Environment{devEnv},
					ExtendedTestEnvs: []environment.Environment{devEnv},
					ProdEnvs:         []environment.Environment{prodEnv},
					Template:         templateToUse,
				}

				_, _ = service.Create(createOp)
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
	Context("monorepo mode - checkout is done strictly from main", func() {
		var (
			localRepo *git.LocalRepository
		)
		BeforeEach(func() {
			var err error
			localRepo, err = git.InitLocalRepository(GinkgoT().TempDir(), false)
			Expect(err).NotTo(HaveOccurred())
			Expect(localRepo.Commit(&git.CommitOp{
				Message:    "initial commit",
				AllowEmpty: true,
			})).To(Succeed())
		})

		It("checks out normally from main", func() {
			mainHead, err := localRepo.Repository().Head()
			Expect(err).NotTo(HaveOccurred())
			mainHead, err = localRepo.Repository().Reference(mainHead.Name(), true)
			Expect(err).NotTo(HaveOccurred())

			branchName := "new-branch"
			Expect(checkoutNewBranch(localRepo, branchName)).To(Succeed())
			head, err := localRepo.Repository().Head()
			Expect(err).NotTo(HaveOccurred())
			head, err = localRepo.Repository().Reference(head.Name(), true)
			Expect(err).NotTo(HaveOccurred())
			Expect(head.Name().Short()).To(Equal(branchName))
			Expect(head.Hash()).To(Equal(mainHead.Hash()))
		})

		It("if not in main, it checkouts still from main", func() {
			mainHead, err := localRepo.Repository().Head()
			Expect(err).NotTo(HaveOccurred())
			mainHead, err = localRepo.Repository().Reference(mainHead.Name(), true)
			Expect(err).NotTo(HaveOccurred())

			oldBranchName := "old-branch"
			newBranchName := "new-branch"
			Expect(localRepo.CheckoutBranch(&git.CheckoutOp{
				BranchName:      oldBranchName,
				CreateIfMissing: true,
			})).To(Succeed())
			Expect(localRepo.Commit(&git.CommitOp{
				Message:    "Just to have different hash from main",
				AllowEmpty: true,
				DryRun:     false,
			})).To(Succeed())

			oldBranchHead, err := localRepo.Repository().Head()
			Expect(err).NotTo(HaveOccurred())
			oldBranchHead, err = localRepo.Repository().Reference(oldBranchHead.Name(), true)
			Expect(err).NotTo(HaveOccurred())
			Expect(oldBranchHead).NotTo(Equal(mainHead))

			Expect(checkoutNewBranch(localRepo, newBranchName)).To(Succeed())
			head, err := localRepo.Repository().Head()
			Expect(err).NotTo(HaveOccurred())
			head, err = localRepo.Repository().Reference(head.Name(), true)
			Expect(err).NotTo(HaveOccurred())
			Expect(head.Name().Short()).To(Equal(newBranchName))
			Expect(head.Hash()).To(Equal(mainHead.Hash()))
		})
	})

	DescribeTable("monorepo mode - app directory calculation",
		func(appPath string, workingDirRelToMonorepo string, expectedAppRelPath string, success bool) {
			cwd, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				Expect(os.Chdir(cwd)).To(Succeed())
				Expect(os.Getwd()).To(Equal(cwd))
			}()

			monorepoDir, err := filepath.EvalSymlinks(GinkgoT().TempDir())
			Expect(err).NotTo(HaveOccurred())
			monorepoSubDir := filepath.Join(monorepoDir, workingDirRelToMonorepo)
			Expect(os.MkdirAll(monorepoSubDir, 0755)).To(Succeed())
			Expect(os.Chdir(monorepoSubDir)).To(Succeed())

			appRelPath, err := calculateWorkingDirForMonorepo(monorepoDir, appPath)
			if success {
				Expect(err).NotTo(HaveOccurred())
				Expect(appRelPath).To(Equal(expectedAppRelPath))
			} else {
				Expect(err).To(HaveOccurred())
				Expect(appRelPath).To(Equal(""))
			}
		},
		Entry("from repo root", "app-path", "./", "app-path", true),
		Entry("from repo root with dir embedding", "subdir/app-path", "./", "subdir/app-path", true),
		Entry("from repo subdir", "app-path", "./subdir", "subdir/app-path", true),
		Entry("from repo subdir with return", "../app-path", "./subdir", "app-path", true),
		Entry("result is outside of monorepo – error", "../app-path", "./", "", false),
	)
})

func readFileContent(path ...string) string {
	filePath := filepath.Join(path...)
	content, err := os.ReadFile(filePath)
	Expect(err).NotTo(HaveOccurred())
	return string(content)
}

type panicTemplateRenderer struct {
}

func (r *panicTemplateRenderer) Render(_ *template.Spec, _ string, _ bool, _ string, _ ...template.Argument) error {
	panic("Panic for test sake")
}

var _ = Describe("render.DeepMerge", func() {
	It("merges two empty maps", func() {
		base := map[string]any{}
		override := map[string]any{}
		result := render.DeepMerge(base, override)
		Expect(result).To(BeEmpty())
	})

	It("returns override when base is empty", func() {
		base := map[string]any{}
		override := map[string]any{
			"key1": "value1",
			"key2": 42,
		}
		result := render.DeepMerge(base, override)
		Expect(result).To(HaveLen(2))
		Expect(result["key1"]).To(Equal("value1"))
		Expect(result["key2"]).To(Equal(42))
	})

	It("returns base when override is empty", func() {
		base := map[string]any{
			"key1": "value1",
			"key2": 42,
		}
		override := map[string]any{}
		result := render.DeepMerge(base, override)
		Expect(result).To(HaveLen(2))
		Expect(result["key1"]).To(Equal("value1"))
		Expect(result["key2"]).To(Equal(42))
	})

	It("override takes precedence for same keys", func() {
		base := map[string]any{
			"key1": "base-value",
			"key2": "only-in-base",
		}
		override := map[string]any{
			"key1": "override-value",
			"key3": "only-in-override",
		}
		result := render.DeepMerge(base, override)
		Expect(result).To(HaveLen(3))
		Expect(result["key1"]).To(Equal("override-value"))
		Expect(result["key2"]).To(Equal("only-in-base"))
		Expect(result["key3"]).To(Equal("only-in-override"))
	})

	It("deep merges nested maps", func() {
		base := map[string]any{
			"replicas": 2,
			"resources": map[string]any{
				"limits": map[string]any{
					"cpu":    "1000m",
					"memory": "1024Mi",
				},
				"requests": map[string]any{
					"cpu":    "100m",
					"memory": "128Mi",
				},
			},
		}
		override := map[string]any{
			"resources": map[string]any{
				"limits": map[string]any{
					"cpu": "2000m", // Override only CPU
				},
			},
		}
		result := render.DeepMerge(base, override)

		Expect(result["replicas"]).To(Equal(2))

		resources, ok := result["resources"].(map[string]any)
		Expect(ok).To(BeTrue())

		limits, ok := resources["limits"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(limits["cpu"]).To(Equal("2000m"))     // Overridden
		Expect(limits["memory"]).To(Equal("1024Mi")) // From base

		requests, ok := resources["requests"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(requests["cpu"]).To(Equal("100m"))     // From base
		Expect(requests["memory"]).To(Equal("128Mi")) // From base
	})

	It("override replaces non-map with map", func() {
		base := map[string]any{
			"key": "string-value",
		}
		override := map[string]any{
			"key": map[string]any{
				"nested": "value",
			},
		}
		result := render.DeepMerge(base, override)
		nested, ok := result["key"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(nested["nested"]).To(Equal("value"))
	})

	It("override replaces map with non-map", func() {
		base := map[string]any{
			"key": map[string]any{
				"nested": "value",
			},
		}
		override := map[string]any{
			"key": "string-value",
		}
		result := render.DeepMerge(base, override)
		Expect(result["key"]).To(Equal("string-value"))
	})
})
