package tenant

import (
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/testutil/gittest"
	"github.com/coreeng/corectl/pkg/testutil/httpmock"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/developer-platform/pkg/tenant"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-github/v59/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"path/filepath"
)

var _ = Describe("Create or Update", func() {
	const expectedTenantFileContent = `name: new-tenant
parent: parent
description: Tenant description
contactEmail: abc@abc.com
costCentre: cost-centre
environments:
    - dev
    - prod
adminGroup: admin-group
readonlyGroup: readonly-group
cloudAccess: []
`
	t := GinkgoTB()

	var (
		cplatformServerRepo *gittest.BareRepository
		cplatformLocalRepo  *git.LocalRepository
		mainBranchRefName   plumbing.ReferenceName
		originalMainRef     *plumbing.Reference
		defaultTenant       tenant.Tenant
		parentTenant        *tenant.Tenant
		branchName          string
		commitMsg           string
		newPrName           string
		newPrHtmlUrl        string
		createPrCapture     *httpmock.HttpCaptureHandler[github.NewPullRequest]
		githubClient        *github.Client
	)
	BeforeEach(OncePerOrdered, func() {
		var err error
		cplatformServerRepo, cplatformLocalRepo, err = gittest.CreateBareAndLocalRepoFromDir(&gittest.CreateBareAndLocalRepoOp{
			SourceDir:          testdata.CPlatformEnvsPath(),
			TargetBareRepoDir:  t.TempDir(),
			TargetLocalRepoDir: t.TempDir(),
		})
		Expect(err).NotTo(HaveOccurred())

		parentTenant, err = tenant.FindByName(tenant.DirFromCPlatformPath(cplatformLocalRepo.Path()), "parent")
		Expect(parentTenant).NotTo(BeNil())
		Expect(err).NotTo(HaveOccurred())
		defaultTenant = tenant.Tenant{
			Name:         "new-tenant",
			Parent:       parentTenant.Name,
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

		mainBranchRefName = plumbing.NewBranchReferenceName(git.MainBranch)
		originalMainRef, err = cplatformLocalRepo.Repository().Reference(mainBranchRefName, true)
		Expect(err).NotTo(HaveOccurred())

		branchName = "new-tenant"
		commitMsg = "New tenant create msg"
		newPrName = "New PR"
		newPrHtmlUrl = "https://github.com/org/repo/pull/1"
		createPrCapture = httpmock.NewCaptureHandler[github.NewPullRequest](
			&github.PullRequest{
				HTMLURL: &newPrHtmlUrl,
			},
		)
		githubClient = github.NewClient(mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.PostReposPullsByOwnerByRepo,
				createPrCapture.Func(),
			),
		))
	})

	When("creating a new tenant", Ordered, func() {
		var createResult CreateOrUpdateResult
		BeforeAll(func() {
			var err error
			createResult, err = CreateOrUpdate(
				&CreateOrUpdateOp{
					Tenant:            &defaultTenant,
					ParentTenant:      parentTenant,
					CplatformRepoPath: cplatformLocalRepo.Path(),
					BranchName:        branchName,
					CommitMessage:     commitMsg,
					PRName:            newPrName,
				},
				githubClient,
			)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns correct PR url", func() {
			Expect(createResult.PRUrl).To(Equal(newPrHtmlUrl))
		})
		It("called create PR correctly", func() {
			Expect(createPrCapture.Requests).To(HaveLen(1))
			newPrRequest := createPrCapture.Requests[0]
			Expect(*newPrRequest.Title).To(Equal(newPrName))
			Expect(*newPrRequest.Head).To(Equal(branchName))
			Expect(*newPrRequest.Base).To(Equal(git.MainBranch))
		})
		It("leave local repository clean", func() {
			localChangesPresent, err := cplatformLocalRepo.IsLocalChangesPresent()
			if Expect(err).NotTo(HaveOccurred()) {
				Expect(localChangesPresent).To(BeFalse())
			}
			currentBranch, err := cplatformLocalRepo.CurrentBranch()
			if Expect(err).NotTo(HaveOccurred()) {
				Expect(currentBranch).To(Equal(git.MainBranch))
			}
		})
		It("leave main branch unchanged", func() {
			currentMainRef, err := cplatformLocalRepo.Repository().Reference(mainBranchRefName, true)
			if Expect(err).NotTo(HaveOccurred()) {
				Expect(currentMainRef).To(Equal(originalMainRef))
			}
		})
		It("pushes all the changes to the remote repository", func() {
			cplatformServerRepo.AssertInSyncWith(cplatformLocalRepo)
		})
		It("creates a commit with a new file for the new tenant", func() {
			branchNameRef, err := cplatformLocalRepo.Repository().Reference(plumbing.NewBranchReferenceName(branchName), false)
			Expect(err).NotTo(HaveOccurred())
			fromHash := originalMainRef.Hash()
			cplatformServerRepo.AssertCommits(gittest.AssertCommitOp{
				From: &fromHash,
				To:   branchNameRef.Hash(),
				ExpectedCommits: []gittest.ExpectedCommit{
					{
						Message:      commitMsg,
						ChangedFiles: []string{"./tenants/tenants/parent/new-tenant.yaml"},
					},
				},
			})
		})
		It("creates the file for the new tenant with the correct content", func() {
			Expect(cplatformLocalRepo.CheckoutBranch(&git.CheckoutOp{BranchName: branchName})).To(Succeed())
			newTenantFile, err := os.ReadFile(filepath.Join(*defaultTenant.SavedPath()))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(newTenantFile)).To(MatchYAML(expectedTenantFileContent))
		})
	})
})
