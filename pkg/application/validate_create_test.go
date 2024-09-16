package application_test

import (
	"github.com/coreeng/corectl/pkg/application"
	"github.com/coreeng/corectl/pkg/testutil/gittest"
	"github.com/coreeng/corectl/pkg/testutil/httpmock"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/developer-platform/pkg/environment"
	coretnt "github.com/coreeng/developer-platform/pkg/tenant"
	"github.com/google/go-github/v59/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"net/http"
	"path/filepath"
)

var _ = Describe("ValidateCreate", Ordered, func() {

	var (
		t   = GinkgoT()
		svc *application.Service

		validTenant = &coretnt.Tenant{
			Name:         "new-tenant",
			Parent:       "parent",
			Description:  "Tenant description",
			ContactEmail: "abc@abc.com",
			CostCentre:   "cost-centre",
			Environments: []string{
				testdata.DevEnvironment(),
				testdata.ProdEnvironment(),
			},
			AdminGroup:    "admin-group",
			ReadOnlyGroup: "readonly-group",
		}
		existingRepositoryResponse = &github.Repository{
			ID:   github.Int64(1234),
			Name: github.String("new-app"),
			Owner: &github.User{
				Login: github.String("test-org"),
			},
			CloneURL: github.String("https://github.com/test-org/new-app.git"),
		}
	)

	DescribeTable("single app:",
	func(op application.CreateOp, expectError bool, errorMsg string, setupMockedClient func() *http.Client) {
        mockedClient := setupMockedClient()
        svc = &application.Service{
            GithubClient: github.NewClient(mockedClient),
        }

        err := svc.ValidateCreate(op)

        if expectError {
            Expect(err).To(HaveOccurred())
            Expect(err.Error()).To(MatchRegexp(errorMsg)) // Change to `MatchRegexp` here
        } else {
            Expect(err).NotTo(HaveOccurred())
        }
    },
		Entry("Valid operation",
			application.CreateOp{
				Tenant:    validTenant,
				LocalPath: "/valid/path",
				Name:      "new-app",
			},
			false,
			"",
			func() *http.Client { return mock.NewMockedHTTPClient() },
		),
		Entry("Missing tenant",
			application.CreateOp{
				LocalPath: "/valid/path",
			},
			true,
			"tenant is missing",
			func() *http.Client { return mock.NewMockedHTTPClient() },
		),
		Entry("Invalid tenant",
			application.CreateOp{
				Tenant:    &coretnt.Tenant{},
				LocalPath: "/valid/path",
			},
			true,
			"tenant is invalid",
			func() *http.Client { return mock.NewMockedHTTPClient() },
		),
		Entry("Invalid environment",
			application.CreateOp{
				Tenant:           validTenant,
				LocalPath:        "/valid/path",
				FastFeedbackEnvs: []environment.Environment{{Environment: "invalid"}},
			},
			true,
			"invalid environment is invalid",
			func() *http.Client { return mock.NewMockedHTTPClient() },
		),
		Entry("Remote repository already exists",
			application.CreateOp{
				Tenant:    validTenant,
				LocalPath: "/valid/path",
				Name:      "new-app",
				OrgName:   "test-org",
			},
			true,
			"test-org/new-app repository already exists",
			func() *http.Client {
				return mock.NewMockedHTTPClient(
					mock.WithRequestMatchHandler(
						mock.GetReposByOwnerByRepo,
						httpmock.NewCaptureHandler[any](existingRepositoryResponse).Func(),
					),
				)
			},
		),
		Entry("Error while checking repository existence",
    application.CreateOp{
        Tenant:    validTenant,
        LocalPath: "/valid/path",
        Name:      "new-app",
        OrgName:   "test-org",
    },
    true,
    `status code 500.*internal server error`, // Using regular expression to match the error
    func() *http.Client {
        return mock.NewMockedHTTPClient(
            mock.WithRequestMatchHandler(
                mock.GetReposByOwnerByRepo,
                http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    w.WriteHeader(http.StatusInternalServerError) // Simulate a 500 error
                    w.Header().Set("Content-Type", "application/json")
                    if _, err := w.Write([]byte(`{"message": "internal server error"}`)); err != nil {
						
						fmt.Println("Error writing response:", err)
					} 
                }),
            ),
        )
    },
),
	)

	Describe("monorepo", func() {
		var (
			monorepoPath string
		)

		BeforeAll(func() {
			_, monorepoLocalRepo, err := gittest.CreateBareAndLocalRepoFromDir(&gittest.CreateBareAndLocalRepoOp{
				SourceDir:          filepath.Join(testdata.TemplatesPath(), testdata.Monorepo()),
				TargetBareRepoDir:  t.TempDir(),
				TargetLocalRepoDir: t.TempDir(),
			})
			Expect(err).NotTo(HaveOccurred())
			monorepoPath = monorepoLocalRepo.Path()
		})

		DescribeTable("should validate correctly",
			func(repoExists bool) {
				op := application.CreateOp{
					Tenant:    validTenant,
					LocalPath: filepath.Join(monorepoPath, "new-repo"),
					OrgName:   "test-org",
					Name:      "new-repo",
				}

				clientMock := mock.NewMockedHTTPClient()
				if repoExists {
					clientMock = mock.NewMockedHTTPClient(
						mock.WithRequestMatchHandler(
							mock.GetReposByOwnerByRepo,
							httpmock.NewCaptureHandler[any](existingRepositoryResponse).Func(),
						),
					)
				}

				svc = &application.Service{
					GithubClient: github.NewClient(clientMock),
				}

				err := svc.ValidateCreate(op)
				Expect(err).NotTo(HaveOccurred())
			},
			Entry("when repository doesn't exist", false),
			Entry("when repository exists", true),
		)
	})
})
