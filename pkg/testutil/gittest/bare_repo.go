package gittest

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
	"github.com/onsi/gomega"

	"github.com/coreeng/corectl/pkg/git"
	gitcore "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/otiai10/copy"
)

type BareRepository struct {
	repo *gitcore.Repository
	path string
}

type CreateBareAndLocalRepoOp struct {
	SourceDir          string
	TargetBareRepoDir  string
	TargetLocalRepoDir string
}

type CreateTestCorectlConfigOp struct {
	Path      string
	CPlatform CreateBareAndLocalRepoOp
	Templates CreateBareAndLocalRepoOp
}

func CreateTestCorectlConfig(path string) (*CreateTestCorectlConfigOp, error) {
	op := &CreateTestCorectlConfigOp{
		Path: path,
	}
	configpath.SetCorectlHome(path)
	return op, nil
}

func CreateBareAndLocalRepoFromDir(op *CreateBareAndLocalRepoOp) (*BareRepository, *git.LocalRepository, error) {
	if err := os.MkdirAll(op.TargetBareRepoDir, 0o777); err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(op.TargetLocalRepoDir, 0o777); err != nil {
		return nil, nil, err
	}
	bareRepo, err := InitBareRepository(op.TargetBareRepoDir)
	if err != nil {
		return nil, nil, err
	}
	localRepo, err := git.InitLocalRepository(op.TargetLocalRepoDir, false)
	if err != nil {
		return nil, nil, err
	}
	if err = localRepo.SetRemote(bareRepo.LocalCloneUrl()); err != nil {
		return nil, nil, err
	}

	if err = copy.Copy(op.SourceDir, localRepo.Path()); err != nil {
		return nil, nil, err
	}

	if err = localRepo.AddAll(); err != nil {
		return nil, nil, err
	}
	if err = localRepo.Commit(&git.CommitOp{Message: "Initial commit"}); err != nil {
		return nil, nil, err
	}
	if err = localRepo.Push(git.PushOp{}); err != nil {
		return nil, nil, err
	}

	return bareRepo, localRepo, nil
}

func InitBareRepository(path string) (*BareRepository, error) {
	if err := os.MkdirAll(path, 0o777); err != nil {
		return nil, err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	repo, err := gitcore.PlainInitWithOptions(
		path,
		&gitcore.PlainInitOptions{
			InitOptions: gitcore.InitOptions{DefaultBranch: plumbing.Main},
			Bare:        true,
		})
	if err != nil {
		return nil, err
	}
	return &BareRepository{
		repo: repo,
		path: absPath,
	}, nil
}

func (r *BareRepository) Repository() *gitcore.Repository {
	return r.repo
}

func (r *BareRepository) LocalCloneUrl() string {
	return "file://" + r.path
}

func (r *BareRepository) AssertInSyncWith(localRepo *git.LocalRepository) {
	referencesFromBareRepo, err := r.repo.References()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	err = referencesFromBareRepo.ForEach(func(referenceFromBareRepo *plumbing.Reference) error {
		referenceFromLocalRepo, err := localRepo.Repository().Reference(referenceFromBareRepo.Name(), false)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(referenceFromLocalRepo.Hash()).To(gomega.Equal(referenceFromBareRepo.Hash()))
		return nil
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

type ExpectedCommit struct {
	Message      string
	ChangedFiles []string
}

func (ec *ExpectedCommit) AssertCommit(
	c *object.Commit,
) {
	gomega.Expect(c.Message).To(gomega.Equal(ec.Message))
	var cleanedExpectedFilenames []string
	for _, cf := range ec.ChangedFiles {
		cleanedExpectedFilenames = append(cleanedExpectedFilenames, filepath.Clean(cf))
	}
	var fileNames []string
	stats, err := c.Stats()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	for _, fileStat := range stats {
		fileNames = append(fileNames, filepath.Clean(fileStat.Name))
	}
	gomega.Expect(fileNames).To(gomega.ConsistOf(cleanedExpectedFilenames))
}

type AssertCommitOp struct {
	From            *plumbing.Hash
	To              plumbing.Hash
	ExpectedCommits []ExpectedCommit
}

func (r *BareRepository) AssertCommits(
	op AssertCommitOp,
) {
	// reverse commits just for convenience
	expectedCommits := slices.Clone(op.ExpectedCommits)
	slices.Reverse(expectedCommits)

	commit, err := r.repo.CommitObject(op.To)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	for _, expectedCommit := range expectedCommits {
		gomega.Expect(commit).NotTo(gomega.BeNil())
		expectedCommit.AssertCommit(commit)
		if commit.NumParents() == 0 {
			commit = nil
			continue
		} else {
			gomega.Expect(commit.NumParents()).To(gomega.Equal(1))
			commit, err = commit.Parent(0)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}
	}
	if op.From != nil {
		gomega.Expect(*op.From).To(gomega.Equal(commit.Hash))
	}
}
