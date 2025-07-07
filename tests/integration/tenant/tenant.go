package tenant

import (
	"time"

	"github.com/coreeng/corectl/pkg/cmdutil/configpath"

	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	"github.com/google/go-github/v60/github"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/thanhpk/randstr"
)

var _ = ginkgo.Describe("tenant", ginkgo.Ordered, func() {
	var (
		homeDir      string
		corectl      *testconfig.CorectlClient
		cfgDetails   *testsetup.CorectlConfigDetails
		githubClient *github.Client
	)
	t := ginkgo.GinkgoT()

	ginkgo.BeforeAll(func() {
		var err error
		homeDir = t.TempDir()
		configpath.SetCorectlHome(homeDir)
		corectl = testconfig.NewCorectlClient(homeDir)
		_, cfgDetails, err = testsetup.InitCorectl(corectl)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		githubClient = testconfig.NewGitHubClient()
		testsetup.SetupGitGlobalConfigFromCurrentToOtherHomeDir(homeDir)
	})

	ginkgo.Context("create", ginkgo.Ordered, func() {
		var (
			newTenantName string
		)

		ginkgo.BeforeAll(func() {
			newTenantName = "new-tenant-name-" + randstr.Hex(6)
			_, err := corectl.Run(
				"tenant", "create",
				"--name", newTenantName,
				"--parent", "parent",
				"--description", "Some tenant description",
				"--contact-email", "ce@company.com",
				"--environments", "dev,prod",
				// Omitting repositories parameter
				"--admin-group", "ag",
				"--readonly-group", "rg",
				"--non-interactive")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

		ginkgo.It("created a PR in the CPlatform repository", func(ctx ginkgo.SpecContext) {
			prList, _, err := githubClient.PullRequests.List(
				ctx,
				cfgDetails.CPlatformRepoName.Organization(),
				cfgDetails.CPlatformRepoName.Name(),
				&github.PullRequestListOptions{
					Head: cfgDetails.CPlatformRepoName.Organization() + ":" + "new-tenant-" + newTenantName,
					Base: git.MainBranch,
				},
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(prList).To(gomega.HaveLen(1))
			gomega.Expect(prList[0]).NotTo(gomega.BeNil())
			pr := prList[0]

			gomega.Expect(pr.GetTitle()).To(gomega.Equal("New tenant: " + newTenantName))
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

			gomega.Expect(prFile.GetStatus()).To(gomega.Equal("added"))
			gomega.Expect(prFile.GetFilename()).To(gomega.Equal("tenants/tenants/parent/" + newTenantName + ".yaml"))
		}, ginkgo.SpecTimeout(time.Minute))
	})
})
