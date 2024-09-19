package application

import (
	"context"
	"errors"
	"fmt"
	"github.com/coreeng/corectl/pkg/cmd/template/render"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/coreeng/corectl/pkg/undo"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/coreeng/developer-platform/pkg/p2p"
	coretnt "github.com/coreeng/developer-platform/pkg/tenant"
	gogit "github.com/go-git/go-git/v5"
	"github.com/google/go-github/v59/github"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Service struct {
	TemplateRenderer render.TemplateRenderer
	GithubClient     *github.Client
}

func NewService(templateRenderer render.TemplateRenderer, githubClient *github.Client) *Service {
	return &Service{
		TemplateRenderer: templateRenderer,
		GithubClient:     githubClient,
	}
}

type CreateOp struct {
	Name             string
	OrgName          string
	LocalPath        string
	Tenant           *coretnt.Tenant
	FastFeedbackEnvs []environment.Environment
	ExtendedTestEnvs []environment.Environment
	ProdEnvs         []environment.Environment
	Template         *template.Spec
	GitAuth          git.AuthMethod
}

type CreateResult struct {
	RepositoryFullname git.RepositoryFullname
	MonorepoMode       bool
	PRUrl              string
}

func checkoutNewBranch(localRepo *git.LocalRepository, branchName string) error {
	currentBranch, err := localRepo.CurrentBranch()
	if err != nil {
		return err
	}
	if currentBranch != git.MainBranch {
		if err := localRepo.CheckoutBranch(&git.CheckoutOp{
			BranchName: git.MainBranch,
		}); err != nil {
			return err
		}
	}
	return localRepo.CheckoutBranch(&git.CheckoutOp{
		BranchName:      branchName,
		CreateIfMissing: true,
	})
}

func (svc *Service) Create(op CreateOp) (result CreateResult, err error) {
	undoSteps := undo.NewSteps()
	defer undoWhenError(&undoSteps)

	if err := prepareLocalPath(op.LocalPath, &undoSteps); err != nil {
		return result, err
	}

	localRepo, isMonorepo, err := setupLocalRepository(op.LocalPath)
	if err != nil {
		return result, err
	}

	if isMonorepo {
		return svc.handleMonorepo(op, localRepo)
	} else {
		return svc.handleSingleRepo(op, localRepo)
	}
}

func (svc *Service) handleSingleRepo(op CreateOp, localRepo *git.LocalRepository) (result CreateResult, err error) {
	additionalArgs := []template.Argument{
		{
			Name:  "working_directory",
			Value: "./",
		},
		{
			Name:  "version_prefix",
			Value: "v",
		},
	}
	if err := svc.renderTemplateMaybe(op, op.LocalPath, additionalArgs...); err != nil {
		return result, err
	}

	if err := commitAllChanges(localRepo, "Initial commit\n[skip ci]", true); err != nil {
		return result, err
	}

	repoFullId, err := svc.createRemoteRepository(op, localRepo)
	if err != nil {
		return result, err
	}
	if err := svc.synchronizeRepository(op, repoFullId); err != nil {
		return result, err
	}

	if err := localRepo.Push(git.PushOp{
		Auth: op.GitAuth,
	}); err != nil {
		return result, err
	}

	return CreateResult{
		MonorepoMode:       false,
		RepositoryFullname: repoFullId.RepositoryFullname,
	}, nil
}

func (svc *Service) handleMonorepo(op CreateOp, localRepo *git.LocalRepository) (result CreateResult, err error) {
	branchName := "add-" + op.Name
	if err := checkoutNewBranch(localRepo, branchName); err != nil {
		return result, err
	}

	appRelPath, err := calculateWorkingDirForMonorepo(localRepo.Path(), op.LocalPath)
	if err != nil {
		return result, err
	}
	additionalArgs := []template.Argument{
		{
			Name:  "working_directory",
			Value: "./" + appRelPath,
		},
		{
			Name:  "version_prefix",
			Value: op.Name + "/v",
		},
	}
	appAbsPath := filepath.Join(localRepo.Path(), appRelPath)
	if err := svc.renderTemplateMaybe(op, appAbsPath, additionalArgs...); err != nil {
		return result, err
	}
	if err := svc.moveGithubWorkflowsToRootMaybe(op); err != nil {
		return result, err
	}

	if err := commitAllChanges(localRepo, fmt.Sprintf("New app: %s\n[skip ci]", op.Name), false); err != nil {
		return result, err
	}

	if err := localRepo.Push(git.PushOp{
		Auth:       op.GitAuth,
		BranchName: branchName,
	}); err != nil {
		return result, err
	}

	repoFullId, err := svc.getRemoteRepositoryFullId(op, localRepo)
	if err != nil {
		return result, err
	}

	pullRequest, err := svc.createPR(op.Name, branchName, repoFullId)
	if err != nil {
		return result, err
	}

	return CreateResult{
		MonorepoMode:       true,
		RepositoryFullname: repoFullId.RepositoryFullname,
		PRUrl:              pullRequest.GetHTMLURL(),
	}, nil
}

func calculateWorkingDirForMonorepo(repoPath string, path string) (string, error) {
	absAppPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	appRelPath, err := filepath.Rel(repoPath, absAppPath)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(appRelPath, "..") {
		return "", fmt.Errorf("app relative path is not inside the monorepo: %s", appRelPath)
	}
	return appRelPath, nil
}

func (svc *Service) createPR(name string, branchName string, repoFullId *git.GithubRepoFullId) (*github.PullRequest, error) {
	pullRequest, _, err := svc.GithubClient.PullRequests.Create(
		context.Background(),
		repoFullId.Organization(),
		repoFullId.Name(),
		&github.NewPullRequest{
			Base:  github.String(git.MainBranch),
			Head:  github.String(branchName),
			Title: github.String("Add " + name + " application"),
			Body:  github.String(fmt.Sprintf("Adding `%s` application", name)),
		})
	return pullRequest, err
}

func (svc *Service) getRemoteRepositoryFullId(op CreateOp, localRepo *git.LocalRepository) (*git.GithubRepoFullId, error) {
	remoteRepoName, err := localRepo.GetRemoteRepoName()
	if err != nil {
		return nil, err
	}

	githubRepo, _, err := svc.GithubClient.Repositories.Get(
		context.Background(),
		op.OrgName,
		remoteRepoName,
	)
	if err != nil {
		return nil, err
	}

	repo := git.NewGithubRepoFullId(githubRepo)
	return &repo, nil
}

func (svc *Service) createRemoteRepository(op CreateOp, localRepo *git.LocalRepository) (git.GithubRepoFullId, error) {
	githubRepo, err := svc.createGithubRepository(op)
	if err != nil {
		return git.GithubRepoFullId{}, err
	}

	repoFullId := git.NewGithubRepoFullId(githubRepo)

	if err := localRepo.SetRemote(githubRepo.GetCloneURL()); err != nil {
		return git.GithubRepoFullId{}, err
	}
	return repoFullId, nil
}

func undoWhenError(undoSteps *undo.Steps) {
	if err := recover(); err != nil {
		errs := undoSteps.Undo()
		panic(undo.FormatError("create new application", fmt.Errorf("%v", err), errs))
	}
}

func prepareLocalPath(localPath string, undoSteps *undo.Steps) error {
	if err := os.MkdirAll(localPath, 0o755); err != nil {
		return err
	}
	undoSteps.Add(func() error {
		return os.RemoveAll(localPath)
	})
	return nil
}

func setupLocalRepository(localPath string) (*git.LocalRepository, bool, error) {
	localRepo, isMonorepo, err := openMonorepoMaybe(localPath)
	if err != nil {
		return nil, false, err
	}

	if localRepo.Repository() == nil {
		var err error
		localRepo, err = git.InitLocalRepository(localPath)
		if err != nil {
			return nil, false, err
		}
	}

	return localRepo, isMonorepo, nil
}

func openMonorepoMaybe(localPath string) (localRepo *git.LocalRepository, isMonorepo bool, err error) {
	localRepo, err = git.OpenLocalRepository(filepath.Dir(localPath))
	if err != nil && !errors.Is(err, gogit.ErrRepositoryNotExists) {
		return nil, false, err
	}
	isMonorepo = localRepo.Repository() != nil
	return localRepo, isMonorepo, nil
}

func (svc *Service) renderTemplateMaybe(op CreateOp, targetDir string, additionalArgs ...template.Argument) error {
	if op.Template == nil {
		return nil
	}
	args := []template.Argument{
		{
			Name:  "name",
			Value: op.Name,
		},
		{
			Name:  "tenant",
			Value: op.Tenant.Name,
		},
	}
	args = append(args, additionalArgs...)
	return svc.TemplateRenderer.Render(op.Template, targetDir, args...)
}

func (svc *Service) moveGithubWorkflowsToRootMaybe(op CreateOp) error {
	exists, err := githubWorkflowsExist(op.LocalPath)
	if err != nil {
		return err
	}
	if exists {
		return svc.moveGithubWorkflowsToRoot(op.LocalPath, op.Name+"-")
	}
	return nil
}

func commitAllChanges(localRepo *git.LocalRepository, message string, allowEmpty bool) error {
	if err := localRepo.AddAll(); err != nil {
		return err
	}
	return localRepo.Commit(&git.CommitOp{
		Message:    message,
		AllowEmpty: allowEmpty,
	})
}

func (svc *Service) createGithubRepository(op CreateOp) (*github.Repository, error) {
	deleteBranchOnMerge := true
	visibility := "private"
	githubRepo, _, err := svc.GithubClient.Repositories.Create(
		context.Background(),
		op.OrgName,
		&github.Repository{
			Name:                &op.Name,
			DeleteBranchOnMerge: &deleteBranchOnMerge,
			Visibility:          &visibility,
		},
	)
	return githubRepo, err
}

func (svc *Service) synchronizeRepository(op CreateOp, repoFullId git.GithubRepoFullId) error {
	return p2p.SynchronizeRepository(&p2p.SynchronizeOp{
		RepositoryId:     &repoFullId,
		Tenant:           op.Tenant,
		FastFeedbackEnvs: op.FastFeedbackEnvs,
		ExtendedTestEnvs: op.ExtendedTestEnvs,
		ProdEnvs:         op.ProdEnvs,
	}, svc.GithubClient)
}

func (svc *Service) moveGithubWorkflowsToRoot(path string, filePrefix string) error {
	githubWorkflowsPath := filepath.Join(path, ".github", "workflows")
	rootWorkflowsPath := filepath.Join(filepath.Dir(path), ".github", "workflows")
	dir, err := os.ReadDir(githubWorkflowsPath)
	if err != nil {
		return err
	}
	err = os.MkdirAll(rootWorkflowsPath, 0o755)
	if err != nil {
		return err
	}

	for _, file := range dir {
		if file.IsDir() {
			continue
		}
		src := filepath.Join(githubWorkflowsPath, file.Name())
		dst := filepath.Join(rootWorkflowsPath, filePrefix+file.Name())

		err = os.Rename(src, dst)
		if err != nil {
			return err
		}
	}
	err = os.RemoveAll(filepath.Join(path, ".github"))
	if err != nil {
		return err
	}
	return nil
}

func githubWorkflowsExist(path string) (bool, error) {
	githubWorkflowsPath := filepath.Join(path, ".github", "workflows")
	dir, err := os.ReadDir(githubWorkflowsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // Directory doesn't exist, but that's not an error for our purposes
		}
		return false, fmt.Errorf("error checking .github/workflows directory: %w", err)
	}
	return len(dir) > 0, nil
}

func (svc *Service) ValidateCreate(op CreateOp) error {
	if op.Tenant == nil {
		return fmt.Errorf("tenant is missing")
	}
	if errs := op.Tenant.Validate(); len(errs) > 0 {
		return fmt.Errorf("tenant is invalid: %v", errs)
	}

	for _, env := range slices.Concat(op.FastFeedbackEnvs, op.ExtendedTestEnvs, op.ProdEnvs) {
		if err := env.Validate(); len(err) > 0 {
			return fmt.Errorf("%v environment is invalid: %v", env.Environment, err)
		}
	}

	err := userio.ValidateFilePath(op.LocalPath, userio.FileValidatorOptions{
		DirsOnly:   true,
		DirIsEmpty: true,
	})
	if err != nil {
		return fmt.Errorf("%s: %v", op.LocalPath, err)
	}

	_, isMonorepo, err := openMonorepoMaybe(op.LocalPath)
	if err != nil {
		return fmt.Errorf("checking for monorepo failed with %v", err)
	}

	if !isMonorepo {
		_, response, err := svc.GithubClient.Repositories.Get(
			context.Background(),
			op.OrgName,
			op.Name,
		)
		if err == nil {
			return fmt.Errorf("%s/%s repository already exists", op.OrgName, op.Name)
		}
		if response.StatusCode != http.StatusNotFound {
			return fmt.Errorf("error while checking if https://github.com/%s/%s repository exists: status code %d, error: %v", op.OrgName, op.Name, response.StatusCode, err)
		}
	}
	return nil
}
