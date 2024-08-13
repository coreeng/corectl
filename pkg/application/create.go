package application

import (
	"context"
	"errors"
	"fmt"
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
)

type CreateOp struct {
	Name             string
	OrgName          string
	LocalPath        string
	Tenant           *coretnt.Tenant
	FastFeedbackEnvs []environment.Environment
	ExtendedTestEnvs []environment.Environment
	ProdEnvs         []environment.Environment
	Template         *template.FulfilledTemplate
	GitAuth          git.AuthMethod
}

type CreateResult struct {
	RepositoryFullname git.RepositoryFullname
	MonorepoMode       bool
}

func Create(op CreateOp, githubClient *github.Client) (result CreateResult, err error) {
	undoSteps := undo.NewSteps()
	defer undoWhenError(&undoSteps)

	if err := prepareLocalPath(op.LocalPath, &undoSteps); err != nil {
		return result, err
	}

	localRepo, isMonorepo, err := setupLocalRepository(op.LocalPath)
	if err != nil {
		return result, err
	}

	if err := renderTemplateMaybe(op); err != nil {
		return result, err
	}

	if isMonorepo {
		if err := moveGithubWorkflowsToRootMaybe(op); err != nil {
			return result, err
		}
		if err := commitAllChanges(localRepo, fmt.Sprintf("New app: %s\n[skip ci]", op.Name)); err != nil {
			return result, err
		}
		repoFullId, err := getRemoteRepositoryFullId(op, githubClient, localRepo)
		if err != nil {
			return result, err
		}
		result = CreateResult{
			MonorepoMode:       true,
			RepositoryFullname: repoFullId.RepositoryFullname,
		}
	} else {
		if err := commitAllChanges(localRepo, "Initial commit\n[skip ci]"); err != nil {
			return result, err
		}
		repoFullId, err := createRemoteRepository(op, githubClient, localRepo)
		if err != nil {
			return result, err
		}
		if err := synchronizeRepository(op, repoFullId, githubClient); err != nil {
			return result, err
		}
		result = CreateResult{
			MonorepoMode:       false,
			RepositoryFullname: repoFullId.RepositoryFullname,
		}

	}

	if err := localRepo.Push(op.GitAuth); err != nil {
		return result, err
	}

	return result, nil
}

func getRemoteRepositoryFullId(op CreateOp, githubClient *github.Client, localRepo *git.LocalRepository) (git.GithubRepoFullId, error) {
	remoteRepoName, err := localRepo.GetRemoteRepoName()
	if err != nil {
		return git.GithubRepoFullId{}, err
	}

	githubRepo, _, err := githubClient.Repositories.Get(
		context.Background(),
		op.OrgName,
		remoteRepoName,
	)
	if err != nil {
		return git.GithubRepoFullId{}, err
	}

	return git.NewGithubRepoFullId(githubRepo), nil
}

func createRemoteRepository(op CreateOp, githubClient *github.Client, localRepo *git.LocalRepository) (git.GithubRepoFullId, error) {
	githubRepo, err := createGithubRepository(op, githubClient)
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

func renderTemplateMaybe(op CreateOp) error {
	if op.Template == nil {
		return nil
	}
	return template.Render(op.Template, op.LocalPath)
}

func moveGithubWorkflowsToRootMaybe(op CreateOp) error {
	if githubWorkflowsExist(op.LocalPath) {
		return moveGithubWorkflowsToRoot(op.LocalPath, op.Name+"-")
	}
	return nil
}

func commitAllChanges(localRepo *git.LocalRepository, message string) error {
	if err := localRepo.AddAll(); err != nil {
		return err
	}
	return localRepo.Commit(&git.CommitOp{Message: message})
}

func createGithubRepository(op CreateOp, githubClient *github.Client) (*github.Repository, error) {
	deleteBranchOnMerge := true
	visibility := "private"
	githubRepo, _, err := githubClient.Repositories.Create(
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

func synchronizeRepository(op CreateOp, repoFullId git.GithubRepoFullId, githubClient *github.Client) error {
	return p2p.SynchronizeRepository(&p2p.SynchronizeOp{
		RepositoryId:     &repoFullId,
		Tenant:           op.Tenant,
		FastFeedbackEnvs: op.FastFeedbackEnvs,
		ExtendedTestEnvs: op.ExtendedTestEnvs,
		ProdEnvs:         op.ProdEnvs,
	}, githubClient)
}
func moveGithubWorkflowsToRoot(path string, filePrefix string) error {
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

func githubWorkflowsExist(path string) bool {
	githubWorkflowsPath := filepath.Join(path, ".github", "workflows")
	dir, err := os.ReadDir(githubWorkflowsPath)
	return err == nil && len(dir) > 0
}

func ValidateCreate(op CreateOp, githubClient *github.Client) error {
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

	if isMonorepo {
		_, response, err := githubClient.Repositories.Get(
			context.Background(),
			op.OrgName,
			op.Name,
		)
		if err == nil {
			return fmt.Errorf("%s/%s repository already exists", op.OrgName, op.Name)
		}
		if err != nil && response.StatusCode != http.StatusNotFound {
			return fmt.Errorf("error while checking if %s/%s repository exists", op.OrgName, op.Name)
		}
	}
	return nil
}
