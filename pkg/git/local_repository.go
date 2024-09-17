package git

import (
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"path"
	"strings"
)

var (
	ErrLocalChangesIsPresent = errors.New("local changes are present")
)

const OriginRemote = "origin"
const MainBranch = "main"

type LocalRepository struct {
	repo     *git.Repository
	worktree *git.Worktree
}

type RepositoryErr struct {
	repo, cloneDirPath string
	err                error
}

func (r RepositoryErr) Error() string {
	return fmt.Sprintf("repoUrl %q, target dir %q: %s", r.repo, r.cloneDirPath, r.err)
}

type RepositoryCloneErr struct {
	RepositoryErr
}

func newRepositoryCloneErr(url, targetDir string, err error) RepositoryCloneErr {
	return RepositoryCloneErr{RepositoryErr{
		url,
		targetDir,
		fmt.Errorf("failed to clone repository: %w", err)}}
}

func (localRepo *LocalRepository) Repository() *git.Repository {
	return localRepo.repo
}

func (localRepo *LocalRepository) Worktree() *git.Worktree {
	return localRepo.worktree
}

func InitLocalRepository(path string) (*LocalRepository, error) {
	result := &LocalRepository{}

	repository, err := git.PlainInitWithOptions(
		path,
		&git.PlainInitOptions{
			InitOptions: git.InitOptions{DefaultBranch: plumbing.Main},
		},
	)
	result.repo = repository
	if err != nil {
		return result, err
	}

	worktree, err := repository.Worktree()
	result.worktree = worktree
	if err != nil {
		return result, err
	}

	return result, nil
}

func OpenLocalRepository(path string) (*LocalRepository, error) {
	localRepository := &LocalRepository{}
	repository, err := git.PlainOpen(path)
	localRepository.repo = repository
	if err != nil {
		return localRepository, fmt.Errorf("repository on path %s not found: %w", path, err)
	}
	worktree, err := repository.Worktree()
	localRepository.worktree = worktree
	if err != nil {
		return localRepository, err
	}
	return localRepository, nil
}

type CloneOp struct {
	URL        string
	TargetPath string
	Auth       AuthMethod
}

func CloneToLocalRepository(op CloneOp) (*LocalRepository, error) {
	var gitAuth transport.AuthMethod
	if op.Auth != nil {
		gitAuth = op.Auth.toGitAuthMethod()
	}
	repository, err := git.PlainClone(
		op.TargetPath,
		false,
		&git.CloneOptions{
			URL:  op.URL,
			Auth: gitAuth,
		})
	if err != nil {
		if errors.Is(err, git.ErrRepositoryAlreadyExists) {
			return nil, newRepositoryCloneErr(op.URL, op.TargetPath, err)
		}
		return nil, RepositoryErr{op.URL, op.TargetPath, err}
	}
	worktree, err := repository.Worktree()
	if err != nil {
		return nil, RepositoryErr{op.URL, op.TargetPath, err}
	}
	return &LocalRepository{
		repo:     repository,
		worktree: worktree,
	}, nil
}

func OpenAndResetRepositoryState(path string) (*LocalRepository, error) {
	localRepo, err := OpenLocalRepository(path)
	if err != nil {
		return nil, err
	}
	err = localRepo.ResetState()
	if err != nil {
		return nil, err
	}
	return localRepo, nil
}

func (localRepo *LocalRepository) ResetState() error {
	localChangesPresent, err := localRepo.IsLocalChangesPresent()
	if err != nil {
		return err
	}
	if localChangesPresent {
		return ErrLocalChangesIsPresent
	}
	if err = localRepo.CheckoutBranch(&CheckoutOp{
		BranchName: MainBranch,
	}); err != nil {
		return err
	}
	return nil
}

func (localRepo *LocalRepository) Path() string {
	return localRepo.worktree.Filesystem.Root()
}

func (localRepo *LocalRepository) AddAll() error {
	return localRepo.worktree.AddWithOptions(&git.AddOptions{All: true})
}

func (localRepo *LocalRepository) AddFiles(paths ...string) error {
	for _, path := range paths {
		if _, err := localRepo.worktree.Add(path); err != nil {
			return err
		}
	}
	return nil
}

func (localRepo *LocalRepository) IsLocalChangesPresent() (bool, error) {
	status, err := localRepo.worktree.Status()
	if err != nil {
		return false, err
	}
	return !status.IsClean(), nil
}

type CheckoutOp struct {
	BranchName      string
	CreateIfMissing bool
}

func (localRepo *LocalRepository) CheckoutBranch(op *CheckoutOp) error {
	branchRefName := plumbing.NewBranchReferenceName(op.BranchName)

	_, err := localRepo.repo.Storer.Reference(branchRefName)
	if errors.Is(err, plumbing.ErrReferenceNotFound) && op.CreateIfMissing {
		head, err := localRepo.repo.Head()
		if err != nil {
			return err
		}
		branchReference := plumbing.NewHashReference(branchRefName, head.Hash())
		if err = localRepo.repo.Storer.SetReference(branchReference); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	branch, err := localRepo.repo.Branch(op.BranchName)
	if errors.Is(err, git.ErrBranchNotFound) {
		branch = &config.Branch{
			Name:   op.BranchName,
			Remote: OriginRemote,
			Merge:  branchRefName,
		}
		if err = localRepo.repo.CreateBranch(branch); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if err = localRepo.worktree.Checkout(&git.CheckoutOptions{Branch: branch.Merge}); err != nil {
		return err
	}

	return nil
}

func (localRepo *LocalRepository) CurrentBranch() (string, error) {
	head, err := localRepo.repo.Head()
	if err != nil {
		return "", err
	}
	if head.Type() == plumbing.InvalidReference {
		return "", errors.New("not following any specific branch")
	}
	headTarget := head.Name()
	if head.Type() == plumbing.SymbolicReference {
		headTarget = head.Target()
	}
	if !headTarget.IsBranch() {
		return "", errors.New("HEAD target is not a branch")
	}
	return headTarget.Short(), nil
}

type PullResult struct {
	IsUpdated bool
}

func (localRepo *LocalRepository) Pull(auth AuthMethod) (PullResult, error) {
	var gitAuth transport.AuthMethod
	if auth != nil {
		gitAuth = auth.toGitAuthMethod()
	}
	err := localRepo.worktree.Pull(&git.PullOptions{
		RemoteName: OriginRemote,
		Auth:       gitAuth,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return PullResult{}, err
	}
	return PullResult{IsUpdated: err == nil}, nil
}

type PushOp struct {
	Auth       AuthMethod
	BranchName string
}

func (localRepo *LocalRepository) Push(op PushOp) error {
	var gitAuth transport.AuthMethod
	if op.Auth != nil {
		gitAuth = op.Auth.toGitAuthMethod()
	}
	var refSpecs []config.RefSpec
	if op.BranchName != "" {
		refSpecs = append(refSpecs, config.RefSpec(
			fmt.Sprintf("+refs/heads/%s:refs/heads/%s", op.BranchName, op.BranchName),
		))
	}
	if err := localRepo.repo.Push(&git.PushOptions{
		Auth:     gitAuth,
		RefSpecs: refSpecs,
	}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}
	return nil
}

type CommitOp struct {
	Message     string
	SkipIfEmpty bool
}

func (localRepo *LocalRepository) Commit(op *CommitOp) error {
	if op.SkipIfEmpty {
		status, err := localRepo.worktree.Status()
		if err != nil {
			return err
		}
		if status.IsClean() {
			return nil
		}
	}

	_, err := localRepo.worktree.Commit(op.Message, &git.CommitOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (localRepo *LocalRepository) SetRemote(url string) error {
	_, err := localRepo.repo.CreateRemote(&config.RemoteConfig{
		Name: OriginRemote,
		URLs: []string{url},
	})
	return err
}
func (localRepo *LocalRepository) GetRemoteRepoName() (string, error) {
	remote, err := localRepo.repo.Remote(OriginRemote)
	if err != nil {
		return "", err
	}
	url := remote.Config().URLs[0]
	return path.Base(strings.TrimSuffix(url, ".git")), nil
}

// HeadShortCommitHash returns short commit hash, currently no support for this feature in go-git lib (some discussions: https://github.com/src-d/go-git/issues/602)
func (localRepo *LocalRepository) HeadShortCommitHash() (string, error) {
	ref, err := localRepo.repo.Head()
	if err != nil {
		return "", err
	}
	return ref.Hash().String()[0:7], nil
}
