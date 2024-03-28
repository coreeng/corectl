package tenant

import (
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	"github.com/google/go-github/v59/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/thanhpk/randstr"
	"time"
)

var _ = Describe("tenant", Ordered, func() {
	var (
		homeDir      string
		corectl      *testconfig.CorectlClient
		cfgDetails   testsetup.CorectlConfigDetails
		githubClient *github.Client
	)
	t := GinkgoT()

	BeforeAll(func() {
		homeDir = t.TempDir()
		corectl = testconfig.NewCorectlClient(homeDir)
		_, cfgDetails = testsetup.InitCorectl(corectl)
		githubClient = testconfig.NewGitHubClient()
		testsetup.SetupGitConfigFromCurrentToOtherHomeDir(homeDir)
	})

	Context("create", Ordered, func() {
		var (
			newTenantName string
		)

		BeforeAll(func() {
			newTenantName = "new-tenant-name-" + randstr.Hex(6)
			Expect(corectl.Run(
				"tenant", "create",
				"--name", newTenantName,
				"--parent", "parent",
				"--description", "Some tenant description",
				"--contact-email", "ce@company.com",
				"--cost-centre", "cc",
				"--environments", "dev,prod",
				// Omitting repositories parameter
				"--admin-group", "ag",
				"--readonly-group", "rg",
				"--nonint",
			)).To(Succeed())
		})

		It("created a PR in the CPlatform repository", func(ctx SpecContext) {
			prList, _, err := githubClient.PullRequests.List(
				ctx,
				cfgDetails.CPlatformRepoName.Organization,
				cfgDetails.CPlatformRepoName.Name,
				&github.PullRequestListOptions{
					Head: cfgDetails.CPlatformRepoName.Organization + ":" + "new-tenant-" + newTenantName,
					Base: git.MainBranch,
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(prList).To(HaveLen(1))
			Expect(prList[0]).NotTo(BeNil())
			pr := prList[0]

			Expect(pr.GetTitle()).To(Equal("New tenant: " + newTenantName))
			Expect(pr.GetState()).To(Equal("open"))

			prFiles, _, err := githubClient.PullRequests.ListFiles(
				ctx,
				cfgDetails.CPlatformRepoName.Organization,
				cfgDetails.CPlatformRepoName.Name,
				pr.GetNumber(),
				&github.ListOptions{},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(prFiles).To(HaveLen(1))
			prFile := prFiles[0]

			Expect(prFile.GetStatus()).To(Equal("added"))
			Expect(prFile.GetFilename()).To(Equal("tenants/tenants/" + newTenantName + ".yaml"))
		}, SpecTimeout(time.Minute))
	})
})
