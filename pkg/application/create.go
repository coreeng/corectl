package application

import (
	"context"
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/coreeng/corectl/pkg/undo"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/coreeng/developer-platform/pkg/p2p"
	coretnt "github.com/coreeng/developer-platform/pkg/tenant"
	"github.com/google/go-github/v59/github"
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
	result = CreateResult{}

	undoSteps := undo.NewSteps()
	defer func() {
		if err != nil {
			errs := undoSteps.Undo()
			err = undo.FormatError("create new application", err, errs)
		}
	}()

	if err = os.MkdirAll(op.LocalPath, 0o755); err != nil {
		return result, err
	}

	localRepo, _ := git.OpenLocalRepository(filepath.Dir(op.LocalPath))
	isMonorepo := localRepo.Repository() != nil
	result.MonorepoMode = isMonorepo

	if localRepo.Repository() == nil {
		localRepo, err = git.InitLocalRepository(op.LocalPath)
		if err != nil {
			return result, err
		}
	}

	undoSteps.Add(func() error {
		return os.RemoveAll(op.LocalPath)
	})
	if op.Template != nil {
		if err = template.Render(op.Template, op.LocalPath); err != nil {
			return result, err
		}
		if isMonorepo && githubWorkflowsExist(op.LocalPath) {
			if err = moveGithubWorkflowsToRoot(op.LocalPath, op.Name+"-"); err != nil {
				return result, err
			}
		}
		if err = localRepo.AddAll(); err != nil {
			return CreateResult{}, err
		}
		message := "Initial commit\n[skip ci]"
		if isMonorepo {
			message = fmt.Sprintf("New app: %s\n[skip ci]", op.Name)
		}
		if err = localRepo.Commit(&git.CommitOp{Message: message}); err != nil {
			return result, err
		}

	}

	if !isMonorepo {
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
		if err != nil {
			return result, err
		}
		repoFullId := git.NewGithubRepoFullId(githubRepo)
		result.RepositoryFullname = repoFullId.RepositoryFullname

		if err = localRepo.SetRemote(githubRepo.GetCloneURL()); err != nil {
			return result, err
		}

		if err = p2p.SynchronizeRepository(&p2p.SynchronizeOp{
			RepositoryId:     &repoFullId,
			Tenant:           op.Tenant,
			FastFeedbackEnvs: op.FastFeedbackEnvs,
			ExtendedTestEnvs: op.ExtendedTestEnvs,
			ProdEnvs:         op.ProdEnvs,
		}, githubClient); err != nil {
			return result, err
		}
	}
	if isMonorepo {
		remoteRepoName, err := localRepo.GetRemoteRepoName()
		if err != nil {
			return result, err
		}
		githubRepo, _, err := githubClient.Repositories.Get(
			context.Background(),
			op.OrgName,
			remoteRepoName,
		)
		if err != nil {
			return result, err
		}

		result.RepositoryFullname = git.NewGithubRepoFullId(githubRepo).RepositoryFullname
	}

	if err = localRepo.Push(op.GitAuth); err != nil {
		return result, err
	}
	return result, nil
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
	//todo lkan; probably we don't need this check
	//_, response, err := githubClient.Repositories.Get(
	//	context.Background(),
	//	op.OrgName,
	//	op.Name,
	//)
	//if err == nil {
	//	return fmt.Errorf("%s/%s repository is already exists", op.OrgName, op.Name)
	//}
	//if err != nil && response.StatusCode != http.StatusNotFound {
	//	return fmt.Errorf("error while checking if %s/%s repository exists", op.OrgName, op.Name)
	//}
	return nil
}
