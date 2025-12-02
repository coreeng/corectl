package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/coreeng/core-platform/pkg/environment"
	"github.com/coreeng/core-platform/pkg/p2p"
	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/cmd/template/render"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/logger"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/coreeng/corectl/pkg/undo"
	gogit "github.com/go-git/go-git/v5"
	"github.com/google/go-github/v60/github"
	"go.uber.org/zap"
)

type Service struct {
	TemplateRenderer render.TemplateRenderer
	GithubClient     *github.Client
	DryRun           bool
}

func NewService(templateRenderer render.TemplateRenderer, githubClient *github.Client, dryRun bool) *Service {
	return &Service{
		TemplateRenderer: templateRenderer,
		GithubClient:     githubClient,
		DryRun:           dryRun,
	}
}

type CreateOp struct {
	Name             string
	GitHubRepoName   string
	Description      string
	OrgName          string
	LocalPath        string
	Tenant           *coretnt.Tenant
	FastFeedbackEnvs []environment.Environment
	ExtendedTestEnvs []environment.Environment
	ProdEnvs         []environment.Environment
	Template         *template.Spec
	GitAuth          git.AuthMethod
	Config           string
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
	logger.Info().With(zap.String("name", op.Name),
		zap.String("path", op.LocalPath),
		zap.String("tenant", op.Tenant.Name)).Msg("create local repo")

	undoSteps := undo.NewSteps()
	defer undoWhenError(&undoSteps)

	if err := prepareLocalPath(op.LocalPath, &undoSteps); err != nil {
		return result, err
	}

	localRepo, isMonorepo, err := setupLocalRepository(op.LocalPath, svc.DryRun)
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
			Value: "",
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
			Value: appRelPath,
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

	pullRequest, err := git.CreateGitHubPR(
		svc.GithubClient,
		fmt.Sprintf("Add %s application", op.Name),
		fmt.Sprintf("Adding `%s` application", op.Name),
		branchName,
		repoFullId.Name(),
		repoFullId.Organization(),
		svc.DryRun,
	)

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

func (svc *Service) getRemoteRepositoryFullId(op CreateOp, localRepo *git.LocalRepository) (*git.GithubRepoFullId, error) {
	remoteRepoName, err := localRepo.GetRemoteRepoName()
	if err != nil {
		return nil, err
	}

	// Use retry logic to handle potential propagation delays
	githubRepo, _, err := git.RetryGitHubAPI(
		func() (*github.Repository, *github.Response, error) {
			return svc.GithubClient.Repositories.Get(
				context.Background(),
				op.OrgName,
				remoteRepoName,
			)
		},
		git.DefaultMaxRetries,
		git.DefaultBaseDelay,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository after retries: %w", err)
	}

	repo := git.NewGithubRepoFullId(githubRepo)
	return &repo, nil
}

func (svc *Service) createRemoteRepository(op CreateOp, localRepo *git.LocalRepository) (git.GithubRepoFullId, error) {
	repoName := op.GitHubRepoName
	if repoName == "" {
		repoName = op.Name
	}
	logger.Info().With(zap.String("name", op.Name),
		zap.String("github_repo_name", repoName),
		zap.String("org", op.OrgName),
		zap.Bool("dry_run", svc.DryRun)).
		Msgf("creating github repository https://github.com/%s/%s", op.OrgName, repoName)

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

func setupLocalRepository(localPath string, dryRun bool) (*git.LocalRepository, bool, error) {
	localRepo, isMonorepo, err := openMonorepoMaybe(localPath, dryRun)
	if err != nil {
		return nil, false, err
	}

	if localRepo.Repository() == nil {
		var err error
		localRepo, err = git.InitLocalRepository(localPath, dryRun)
		if err != nil {
			return nil, false, err
		}
	}

	return localRepo, isMonorepo, nil
}

func openMonorepoMaybe(localPath string, dryRun bool) (localRepo *git.LocalRepository, isMonorepo bool, err error) {
	localRepo, err = git.OpenLocalRepository(filepath.Dir(localPath), dryRun)
	if err != nil && !errors.Is(err, gogit.ErrRepositoryNotExists) {
		return nil, false, err
	}
	isMonorepo = localRepo.Repository() != nil
	if isMonorepo {
		logger.Debug().Msg("git: repository is monorepo")
	} else {
		logger.Debug().Msg("git: repository is single repo")
	}
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

	mergedConfig := make(map[string]any)

	if op.Template.Config != nil {
		mergedConfig = deepMerge(mergedConfig, op.Template.Config)
	}

	if op.Config != "" {
		var configOverrides map[string]any
		if err := json.Unmarshal([]byte(op.Config), &configOverrides); err != nil {
			return fmt.Errorf("invalid config JSON: %w", err)
		}
		mergedConfig = deepMerge(mergedConfig, configOverrides)
	}

	if len(mergedConfig) > 0 {
		args = append(args, template.Argument{
			Name:  "config",
			Value: mergedConfig,
		})
	}

	args = append(args, additionalArgs...)
	logger.Debug().With(
		zap.String("tenant", op.Tenant.Name),
		zap.String("app", op.Name),
		zap.String("target_dir", targetDir),
		zap.Any("config", mergedConfig),
		zap.Bool("dry_run", svc.DryRun)).
		Msg("calling render template")

	return svc.TemplateRenderer.Render(op.Template, targetDir, svc.DryRun, args...)
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
	repoName := op.GitHubRepoName
	if repoName == "" {
		repoName = op.Name
	}
	logger.Debug().With(
		zap.String("name", op.Name),
		zap.String("github_repo_name", repoName),
		zap.String("org", op.OrgName),
		zap.Bool("dry_run", svc.DryRun)).
		Msg("github: create repository")
	deleteBranchOnMerge := true
	visibility := "private"
	repo := github.Repository{
		ID:                  github.Int64(1234),
		Name:                &repoName,
		DeleteBranchOnMerge: &deleteBranchOnMerge,
		Visibility:          &visibility,
		Owner: &github.User{
			Login: github.String(op.OrgName),
		},
	}
	if op.Description != "" {
		repo.Description = &op.Description
	}
	if !svc.DryRun {
		githubRepo, _, err := svc.GithubClient.Repositories.Create(
			context.Background(),
			op.OrgName,
			&repo,
		)
		return githubRepo, err
	} else {
		repo.CloneURL = github.String(fmt.Sprintf("https://github.com/%s/%s.git", *repo.Owner.Login, *repo.Name))
		return &repo, nil
	}
}

func (svc *Service) synchronizeRepository(op CreateOp, repoFullId git.GithubRepoFullId) error {
	logger.Debug().With(
		zap.String("name", op.Name),
		zap.String("org", op.OrgName),
		zap.String("tenant", op.Tenant.Name),
		zap.Any("fast_feedback_envs", op.FastFeedbackEnvs),
		zap.Any("extended_test_envs", op.ExtendedTestEnvs),
		zap.Any("prod_envs", op.ProdEnvs),
		zap.Bool("dry_run", svc.DryRun)).
		Msg("github: setting repository variables")
	if !svc.DryRun {
		return svc.synchronizeRepositoryWithRetry(op, repoFullId)
	}
	return nil
}

// synchronizeRepositoryWithRetry wraps the p2p.SynchronizeRepository call with retry logic
func (svc *Service) synchronizeRepositoryWithRetry(op CreateOp, repoFullId git.GithubRepoFullId) error {
	return git.RetryGitHubOperation(
		func() error {
			return p2p.SynchronizeRepository(&p2p.SynchronizeOp{
				RepositoryId:     &repoFullId,
				Tenant:           op.Tenant,
				FastFeedbackEnvs: op.FastFeedbackEnvs,
				ExtendedTestEnvs: op.ExtendedTestEnvs,
				ProdEnvs:         op.ProdEnvs,
			}, svc.GithubClient)
		},
		git.DefaultMaxRetries,
		git.DefaultBaseDelay,
	)
}

func (svc *Service) moveGithubWorkflowsToRoot(path string, filePrefix string) error {
	githubWorkflowsPath := filepath.Join(path, ".github", "workflows")
	rootWorkflowsPath := filepath.Join(filepath.Dir(path), ".github", "workflows")
	dir, err := os.ReadDir(githubWorkflowsPath)
	if err != nil {
		return err
	}

	logger.Debug().With(
		zap.String("path", rootWorkflowsPath),
		zap.Bool("dry_run", svc.DryRun)).
		Msg("github: making workflows directory")
	if !svc.DryRun {
		err = os.MkdirAll(rootWorkflowsPath, 0o755)
		if err != nil {
			return err
		}
	}

	for _, file := range dir {
		if file.IsDir() {
			continue
		}
		src := filepath.Join(githubWorkflowsPath, file.Name())
		dst := filepath.Join(rootWorkflowsPath, filePrefix+file.Name())

		logger.Debug().With(
			zap.String("source", src),
			zap.String("destination", dst),
			zap.Bool("dry_run", svc.DryRun)).
			Msg("github: moving file")
		if !svc.DryRun {
			err = os.Rename(src, dst)
			if err != nil {
				return err
			}
		}
	}
	removePath := filepath.Join(path, ".github")
	logger.Debug().With(
		zap.String("path", removePath),
		zap.Bool("dry_run", svc.DryRun)).
		Msg("github: removing path")
	if !svc.DryRun {
		err = os.RemoveAll(removePath)
		if err != nil {
			return err
		}
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

func deepMerge(base, override map[string]any) map[string]any {
	result := make(map[string]any)

	for k, v := range base {
		result[k] = v
	}

	for k, v := range override {
		if baseVal, exists := result[k]; exists {
			baseMap, baseIsMap := baseVal.(map[string]any)
			overrideMap, overrideIsMap := v.(map[string]any)
			if baseIsMap && overrideIsMap {
				result[k] = deepMerge(baseMap, overrideMap)
				continue
			}
		}
		result[k] = v
	}

	return result
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

	_, isMonorepo, err := openMonorepoMaybe(op.LocalPath, svc.DryRun)
	if err != nil {
		return fmt.Errorf("checking for monorepo failed with %v", err)
	}

	if !isMonorepo {
		repoName := op.GitHubRepoName
		if repoName == "" {
			repoName = op.Name
		}
		logger.Info().With(
			zap.String("org", op.OrgName),
			zap.String("name", op.Name),
			zap.String("github_repo_name", repoName)).
			Msgf("checking github repo availability: https://github.com/%s/%s", op.OrgName, repoName)
		_, response, err := svc.GithubClient.Repositories.Get(
			context.Background(),
			op.OrgName,
			repoName,
		)
		if err == nil {
			return fmt.Errorf("%s/%s repository already exists", op.OrgName, repoName)
		}
		if response.StatusCode != http.StatusNotFound {
			return fmt.Errorf("error while checking if https://github.com/%s/%s repository exists: status code %d, error: %v", op.OrgName, repoName, response.StatusCode, err)
		}
	}
	return nil
}
