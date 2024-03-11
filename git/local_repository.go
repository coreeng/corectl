package git

import (
	"errors"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

const originRemote = "origin"
const MainBranch = "main"

type LocalRepository struct {
	repo     *git.Repository
	worktree *git.Worktree
	path     string
}

func InitLocalRepository(path string) (*LocalRepository, error) {
	result := &LocalRepository{
		path: path,
	}

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
	localRepository := &LocalRepository{
		path: path,
	}
	repository, err := git.PlainOpen(path)
	localRepository.repo = repository
	if err != nil {
		return localRepository, err
	}
	worktree, err := repository.Worktree()
	localRepository.worktree = worktree
	if err != nil {
		return localRepository, err
	}
	return localRepository, nil
}

func CloneToLocalRepository(url string, targetPath string) (*LocalRepository, error) {
	repository, err := git.PlainClone(
		targetPath,
		false,
		&git.CloneOptions{
			URL: url,
		})
	if err != nil {
		return nil, err
	}
	worktree, err := repository.Worktree()
	if err != nil {
		return nil, err
	}
	return &LocalRepository{
		repo:     repository,
		worktree: worktree,
		path:     targetPath,
	}, nil
}

func (localRepo *LocalRepository) Path() string {
	return localRepo.path
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

type CheckoutOp struct {
	BranchName      string
	CreateIfMissing bool
}

func (localRepo *LocalRepository) CheckoutBranch(op *CheckoutOp) error {
	branchRefName := plumbing.NewBranchReferenceName(op.BranchName)

	branchReference, err := localRepo.repo.Storer.Reference(branchRefName)
	if errors.Is(err, plumbing.ErrReferenceNotFound) && op.CreateIfMissing {
		head, err := localRepo.repo.Head()
		if err != nil {
			return err
		}
		branchReference = plumbing.NewHashReference(branchRefName, head.Hash())
		if err = localRepo.repo.Storer.SetReference(branchReference); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	branch, err := localRepo.repo.Branch(op.BranchName)
	if errors.Is(err, git.ErrBranchNotFound) && op.CreateIfMissing {
		branch = &config.Branch{
			Name:   op.BranchName,
			Remote: originRemote,
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

func (localRepo *LocalRepository) Pull() error {
	if err := localRepo.worktree.Pull(&git.PullOptions{RemoteName: originRemote}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}
	return nil
}

func (localRepo *LocalRepository) Push() error {
	if err := localRepo.repo.Push(&git.PushOptions{}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
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
		Name: originRemote,
		URLs: []string{url},
	})
	return err
}
