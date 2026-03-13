package tenant

import (
	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
	"time"

	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	"github.com/google/go-github/v60/github"
	"github.com/thanhpk/randstr"

	//nolint:staticcheck
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck
	. "github.com/onsi/gomega"
)

var _ = Describe("tenant", Ordered, func() {
	var (
		homeDir      string
		corectl      *testconfig.CorectlClient
		cfgDetails   *testsetup.CorectlConfigDetails
		githubClient *github.Client
	)
	t := GinkgoT()

	BeforeAll(func() {
		var err error
		homeDir = t.TempDir()
		configpath.SetCorectlHome(homeDir)
		corectl = testconfig.NewCorectlClient(homeDir)
		_, cfgDetails, err = testsetup.InitCorectl(corectl)
		Expect(err).ToNot(HaveOccurred())
		githubClient = testconfig.NewGitHubClient()
		testsetup.SetupGitGlobalConfigFromCurrentToOtherHomeDir(homeDir)
	})

	Context("create org unit", Ordered, func() {
		var (
			newOrgUnitName string
		)

		BeforeAll(func() {
			newOrgUnitName = "new-ou-" + randstr.Hex(6)
			_, err := corectl.Run(
				"tenant", "create",
				"--name", newOrgUnitName,
				"--description", "Some org unit description",
				"--contact-email", "ou@company.com",
				"--environments", "dev,prod",
				"--admin-group", "ag-"+newOrgUnitName,
				"--readonly-group", "rg-"+newOrgUnitName,
				"--prefix", "area/subarea",
				"--non-interactive",
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("created a PR in the CPlatform repository", func(ctx SpecContext) {
			prList, _, err := git.RetryGitHubAPI(
				func() ([]*github.PullRequest, *github.Response, error) {
					return githubClient.PullRequests.List(
						ctx,
						cfgDetails.CPlatformRepoName.Organization(),
						cfgDetails.CPlatformRepoName.Name(),
						&github.PullRequestListOptions{
							Head: cfgDetails.CPlatformRepoName.Organization() + ":" + "new-ou-tenant-" + newOrgUnitName,
							Base: git.MainBranch,
						},
					)
				},
				git.DefaultMaxRetries,
				git.DefaultBaseDelay,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(prList).To(HaveLen(1))
			pr := prList[0]
			Expect(pr.GetTitle()).To(Equal("New org unit: " + newOrgUnitName))
			Expect(pr.GetState()).To(Equal("open"))

			prFiles, _, err := git.RetryGitHubAPI(
				func() ([]*github.CommitFile, *github.Response, error) {
					return githubClient.PullRequests.ListFiles(
						ctx,
						cfgDetails.CPlatformRepoName.Organization(),
						cfgDetails.CPlatformRepoName.Name(),
						pr.GetNumber(),
						&github.ListOptions{},
					)
				},
				git.DefaultMaxRetries,
				git.DefaultBaseDelay,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(prFiles).To(HaveLen(1))
			Expect(prFiles[0].GetStatus()).To(Equal("added"))
			Expect(prFiles[0].GetFilename()).To(Equal("tenants/tenants/" + newOrgUnitName + ".ou.yaml"))
		}, SpecTimeout(time.Minute))
	})
})
