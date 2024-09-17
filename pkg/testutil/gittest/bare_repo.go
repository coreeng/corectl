package gittest

import (
	"github.com/coreeng/corectl/pkg/git"
	gitcore "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	. "github.com/onsi/gomega"
	"github.com/otiai10/copy"
	"os"
	"path/filepath"
	"slices"
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
	localRepo, err := git.InitLocalRepository(op.TargetLocalRepoDir)
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
	Expect(err).NotTo(HaveOccurred())
	err = referencesFromBareRepo.ForEach(func(referenceFromBareRepo *plumbing.Reference) error {
		referenceFromLocalRepo, err := localRepo.Repository().Reference(referenceFromBareRepo.Name(), false)
		Expect(err).NotTo(HaveOccurred())
		Expect(referenceFromLocalRepo.Hash()).To(Equal(referenceFromBareRepo.Hash()))
		return nil
	})
	Expect(err).NotTo(HaveOccurred())
}

type ExpectedCommit struct {
	Message      string
	ChangedFiles []string
}

func (ec *ExpectedCommit) AssertCommit(
	c *object.Commit,
) {
	Expect(c.Message).To(Equal(ec.Message))
	var cleanedExpectedFilenames []string
	for _, cf := range ec.ChangedFiles {
		cleanedExpectedFilenames = append(cleanedExpectedFilenames, filepath.Clean(cf))
	}
	var fileNames []string
	stats, err := c.Stats()
	Expect(err).NotTo(HaveOccurred())
	for _, fileStat := range stats {
		fileNames = append(fileNames, filepath.Clean(fileStat.Name))
	}
	Expect(fileNames).To(ConsistOf(cleanedExpectedFilenames))
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
	Expect(err).NotTo(HaveOccurred())
	for _, expectedCommit := range expectedCommits {
		Expect(commit).NotTo(BeNil())
		expectedCommit.AssertCommit(commit)
		if commit.NumParents() == 0 {
			commit = nil
			continue
		} else {
			Expect(commit.NumParents()).To(Equal(1))
			commit, err = commit.Parent(0)
			Expect(err).NotTo(HaveOccurred())
		}
	}
	if op.From != nil {
		Expect(*op.From).To(Equal(commit.Hash))
	}
}
