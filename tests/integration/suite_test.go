package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/corectl/tests/integration/testconfig"
	"github.com/coreeng/corectl/tests/integration/testsetup"
	"github.com/google/go-github/v59/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/otiai10/copy"
	"github.com/phuslu/log"
	"github.com/thanhpk/randstr"

	// Test cases import
	_ "github.com/coreeng/corectl/pkg/cmd/update"
	_ "github.com/coreeng/corectl/tests/integration/application"
	_ "github.com/coreeng/corectl/tests/integration/config"
	_ "github.com/coreeng/corectl/tests/integration/env"
	_ "github.com/coreeng/corectl/tests/integration/p2p"
	_ "github.com/coreeng/corectl/tests/integration/tenant"
	_ "github.com/coreeng/corectl/tests/integration/version"
)

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Tests")
}

var _ = BeforeSuite(func(ctx SpecContext) {
	log.DefaultLogger.SetLevel(log.PanicLevel)
	testRunId := randstr.String(6)
	testconfig.SetTestRunId(testRunId)
	fmt.Println("Test Run ID: ", testRunId)
	tempDir := GinkgoT().TempDir()
	githubClient := testconfig.NewGitHubClient()
	gitAuth := git.UrlTokenAuthMethod(testconfig.Cfg.GitHubToken)
	testconfig.Cfg.CPlatformRepoFullId = prepareTestRepository(
		ctx,
		testdata.CPlatformEnvsPath(),
		filepath.Join(tempDir, "cplatform-envs"),
		"test-cplatform-envs-",
		testRunId,
		githubClient,
		gitAuth,
	)
	testconfig.Cfg.TemplatesRepoFullId = prepareTestRepository(
		ctx,
		testdata.TemplatesPath(),
		filepath.Join(tempDir, "software-templates"),
		"test-software-templates-",
		testRunId,
		githubClient,
		gitAuth,
	)
}, NodeTimeout(time.Minute))

func prepareTestRepository(
	ctx SpecContext,
	src string,
	dest string,
	repoNamePrefix string,
	testRunId string,
	githubClient *github.Client,
	gitAuth git.AuthMethod,
) git.GithubRepoFullId {
	Expect(os.MkdirAll(
		dest,
		0o777,
	)).To(Succeed())
	Expect(
		copy.Copy(src, dest),
	).To(Succeed())
	localRepo, err := git.InitLocalRepository(dest, false)
	Expect(err).NotTo(HaveOccurred())
	testsetup.SetupGitRepoConfigFromOtherRepo(".", localRepo.Repository())
	Expect(localRepo.AddAll()).To(Succeed())
	Expect(localRepo.Commit(&git.CommitOp{Message: "Initial commit\n[skip ci]"})).To(Succeed())

	repoName := repoNamePrefix + testRunId
	repoDescription := "Temporary repository for running integration tests. TestRunId: " + testRunId
	isPrivate := true
	githubRepo, _, err := githubClient.Repositories.Create(
		ctx,
		testconfig.Cfg.GitHubOrg,
		&github.Repository{
			Name:        &repoName,
			Description: &repoDescription,
			Private:     &isPrivate,
		})
	Expect(err).NotTo(HaveOccurred())
	repoFullId := git.NewGithubRepoFullId(githubRepo)
	DeferCleanup(func(ctx SpecContext) {
		_, err := githubClient.Repositories.Delete(
			ctx,
			repoFullId.RepositoryFullname.Organization(),
			repoFullId.RepositoryFullname.Name(),
		)
		Expect(err).NotTo(HaveOccurred())
	}, NodeTimeout(time.Minute))

	Expect(
		localRepo.SetRemote(githubRepo.GetCloneURL()),
	).To(Succeed())
	Expect(localRepo.Push(git.PushOp{
		Auth: gitAuth,
	})).To(Succeed())
	return repoFullId
}
