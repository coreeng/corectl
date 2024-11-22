package git

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/coreeng/corectl/pkg/logger"
	"github.com/coreeng/corectl/pkg/shell"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"

	"go.uber.org/zap"
)

var (
	ErrLocalChangesIsPresent = errors.New("local changes are present")
)

const OriginRemote = "origin"
const MainBranch = "main"

type LocalRepository struct {
	repo     *git.Repository
	worktree *git.Worktree
	DryRun   bool
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

func InitLocalRepository(path string, dryRun bool) (*LocalRepository, error) {
	logger.Debug("git: init",
		zap.String("repo", path),
		zap.Bool("dry_run", dryRun))
	result := &LocalRepository{
		DryRun: dryRun,
		repo:   &git.Repository{},
	}

	gitOptions := &git.PlainInitOptions{
		InitOptions: git.InitOptions{DefaultBranch: plumbing.Main},
	}
	var repository *git.Repository
	var err error
	if dryRun {
		// Initialize an in-memory repo for dry run
		repository, err = git.InitWithOptions(
			memory.NewStorage(),
			osfs.New(path),
			gitOptions.InitOptions,
		)
	} else {
		repository, err = git.PlainInitWithOptions(
			path,
			gitOptions,
		)
	}
	if err != nil {
		return result, err
	}
	result.repo = repository
	worktree, err := repository.Worktree()
	if err != nil {
		return result, err
	}
	result.worktree = worktree

	return result, nil
}

func OpenLocalRepository(path string, dryRun bool) (*LocalRepository, error) {
	logger.Debug("git: opening repository",
		zap.String("repo", path),
		zap.Bool("dry_run", dryRun))
	localRepository := &LocalRepository{DryRun: dryRun}
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
	logger.Debug("git: cloning repository",
		zap.String("url", op.URL),
		zap.String("target_path", op.TargetPath))
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

func OpenAndResetRepositoryState(path string, dryRun bool) (*LocalRepository, error) {
	localRepo, err := OpenLocalRepository(path, dryRun)
	if err != nil {
		return nil, err
	}
	if dryRun {
		return localRepo, nil
	}
	err = localRepo.ResetState()
	if err != nil {
		return nil, err
	}
	return localRepo, nil
}

func (localRepo *LocalRepository) ResetState() error {
	logger.Debug("git: resetting to branch",
		zap.String("repo", localRepo.Path()),
		zap.Bool("dry_run", localRepo.DryRun),
		zap.String("branch", MainBranch))
	localChangesPresent, err := localRepo.IsLocalChangesPresent()
	if err != nil {
		return err
	}
	if localChangesPresent {
		return ErrLocalChangesIsPresent
	}
	// CheckoutBranch will respect DryRun
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
	logger.Debug("git: adding all files",
		zap.String("repo", localRepo.Path()),
		zap.Bool("dry_run", localRepo.DryRun))
	if !localRepo.DryRun {
		return localRepo.worktree.AddWithOptions(&git.AddOptions{All: true})
	}
	return nil
}

func (localRepo *LocalRepository) AddFiles(paths ...string) error {
	for _, p := range paths {
		logger.Debug("git: adding path",
			zap.String("repo", localRepo.Path()),
			zap.Bool("dry_run", localRepo.DryRun),
			zap.String("path", p))
		if !localRepo.DryRun {
			if _, err := localRepo.worktree.Add(p); err != nil {
				return err
			}
		}
	}
	return nil
}

func (localRepo *LocalRepository) IsLocalChangesPresent() (bool, error) {
	status, err := localRepo.worktree.Status()
	if err != nil {
		return false, err
	}
	logger.Debug("git: status",
		zap.String("repo", localRepo.Path()),
		zap.Bool("dry_run", localRepo.DryRun),
		zap.Bool("is_clean", status.IsClean()))
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

		logger.Debug("git: [branch ref -> HEAD]",
			zap.String("repo", localRepo.Path()),
			zap.Bool("dry_run", localRepo.DryRun),
			zap.String("branch_ref", branchReference.Name().Short()),
			zap.String("branch_hash", branchReference.Hash().String()))
		if !localRepo.DryRun {
			if err = localRepo.repo.Storer.SetReference(branchReference); err != nil {
				return err
			}
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
		logger.Debug("git: create branch",
			zap.String("repo", localRepo.Path()),
			zap.Bool("dry_run", localRepo.DryRun),
			zap.String("branch", branch.Name))
		if !localRepo.DryRun {
			if err = localRepo.repo.CreateBranch(branch); err != nil {
				return err
			}
		}
	} else if err != nil {
		return err
	}

	// TODO: We could use `git.CheckoutOptions{Branch: branch.Merge, Create: true, Hash: branchReference}` to refactor
	logger.Debug("git: checkout branch",
		zap.String("repo", localRepo.Path()),
		zap.Bool("dry_run", localRepo.DryRun),
		zap.String("branch", branch.Name))
	if !localRepo.DryRun {
		if err = localRepo.worktree.Checkout(&git.CheckoutOptions{Branch: branch.Merge}); err != nil {
			return err
		}
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

func (localRepo *LocalRepository) Pull(auth AuthMethod) (*PullResult, error) {
	var (
		gitAuth         transport.AuthMethod
		err             error
		alreadyUpToDate bool = false
	)
	if auth != nil {
		gitAuth = auth.toGitAuthMethod()
	}

	originType := unknownOriginType
	if !localRepo.DryRun {
		originType, err = localRepo.originType()
		if err != nil {
			return nil, err
		}
	}

	goGit := originType == httpsOriginType

	logger.Debug("git: pull",
		zap.String("repo", localRepo.Path()),
		zap.Bool("go_git", goGit),
		zap.Bool("dry_run", localRepo.DryRun))
	if !localRepo.DryRun {
		if goGit {
			err = localRepo.worktree.Pull(&git.PullOptions{
				RemoteName: OriginRemote,
				Auth:       gitAuth,
			})
			if errors.Is(err, git.NoErrAlreadyUpToDate) {
				alreadyUpToDate = true
				err = nil
			}
		}
		if err != nil || !goGit {
			stdout, stderr, gitErr := shell.RunCommand(localRepo.Path(), "git", "pull", OriginRemote)
			if gitErr != nil {
				return nil, fmt.Errorf("git pull failed:\n\tstdout:[%s]\n\tstderr:[%s]\n%v\ntoken attempt: %v", stdout, stderr, gitErr, err)
			}
			alreadyUpToDate = strings.Contains(stdout, "Already up to date.")
		}

	}
	return &PullResult{IsUpdated: !alreadyUpToDate}, nil
}

type PushOp struct {
	Auth       AuthMethod
	BranchName string
}

type originType string

var (
	sshOriginType     originType = "ssh"
	httpsOriginType   originType = "https"
	unknownOriginType originType = "unknown"
)

func (localRepo *LocalRepository) originType() (originType, error) {
	remotes, err := localRepo.repo.Remotes()
	if err != nil {
		return unknownOriginType, fmt.Errorf("failed to list git remotes: %v", err)
	}
	remoteNames := []string{}
	for _, remote := range remotes {
		remoteNames = append(remoteNames, remote.Config().Name)
		if remote.Config().Name == OriginRemote {
			for _, url := range remote.Config().URLs {
				if strings.HasPrefix(url, "ssh://") || strings.HasPrefix(url, "git@") {
					return sshOriginType, nil
				} else if strings.HasPrefix(url, "https://") {
					return httpsOriginType, nil
				} else {
					return unknownOriginType, nil
				}
			}
		}
	}
	return unknownOriginType, fmt.Errorf("failed to find remote %s in %+v", OriginRemote, remoteNames)
}

func (localRepo *LocalRepository) Push(op PushOp) error {
	var (
		gitAuth  transport.AuthMethod
		refSpecs []config.RefSpec
	)

	if op.Auth != nil {
		gitAuth = op.Auth.toGitAuthMethod()
	}
	if op.BranchName != "" {
		refSpecs = append(refSpecs, config.RefSpec(
			fmt.Sprintf("+refs/heads/%s:refs/heads/%s", op.BranchName, op.BranchName),
		))
	}

	var err error
	originType := unknownOriginType
	if !localRepo.DryRun {
		originType, err = localRepo.originType()
		if err != nil {
			return err
		}
	}

	goGit := originType == httpsOriginType
	logger.Debug("git: pushing branch to remote",
		zap.String("repo", localRepo.Path()),
		zap.Bool("dry_run", localRepo.DryRun),
		zap.Bool("go_git", goGit),
		zap.String("branch_name", op.BranchName))

	if !localRepo.DryRun {
		err = nil
		if goGit {
			// Use go-git if origin is set up for https, this will work with the token auth
			err = localRepo.repo.Push(&git.PushOptions{
				Auth:     gitAuth,
				RefSpecs: refSpecs,
			})
			if err != nil {
				if errors.Is(err, git.NoErrAlreadyUpToDate) {
					err = nil
				} else {
					return err
				}
			}
		}

		if err != nil || !goGit {
			// Use git binary for any other configuration, this will work with various git configs
			gitArgs := []string{"push", "--set-upstream", OriginRemote}
			if op.BranchName != "" {
				gitArgs = append(gitArgs, op.BranchName)
			} else {
				currentBranch, err := localRepo.CurrentBranch()
				if err != nil {
					return fmt.Errorf("failed to find current branch: %v", err)
				}
				gitArgs = append(gitArgs, currentBranch)
			}
			if stdout, stderr, err := shell.RunCommand(localRepo.Path(), "git", gitArgs...); err != nil {
				return fmt.Errorf("git push failed:\n\tstdout:[%s]\n\tstderr:[%s]\n%v", stdout, stderr, err)
			}
		}
	}

	return nil
}

type CommitOp struct {
	Message    string
	AllowEmpty bool
	DryRun     bool
}

func (localRepo *LocalRepository) Commit(op *CommitOp) error {
	logger.Debug("git: commit",
		zap.String("repo", localRepo.Path()),
		zap.Bool("dry_run", localRepo.DryRun),
		zap.String("message", op.Message),
		zap.Bool("allow_empty", op.AllowEmpty))
	if !localRepo.DryRun {
		_, err := localRepo.worktree.Commit(op.Message, &git.CommitOptions{
			AllowEmptyCommits: op.AllowEmpty,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (localRepo *LocalRepository) SetRemote(url string) error {
	logger.Debug("git: setting origin",
		zap.String("repo", localRepo.Path()),
		zap.Bool("dry_run", localRepo.DryRun),
		zap.String("url", url))
	if !localRepo.DryRun {
		_, err := localRepo.repo.CreateRemote(&config.RemoteConfig{
			Name: OriginRemote,
			URLs: []string{url},
		})
		return err
	}
	return nil
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
