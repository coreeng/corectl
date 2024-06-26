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
	"net/http"
	"os"
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
	localRepo, err := git.InitLocalRepository(op.LocalPath)
	if err != nil {
		return result, err
	}
	undoSteps.Add(func() error {
		return os.RemoveAll(op.LocalPath)
	})
	if op.Template != nil {
		if err = template.Render(op.Template, op.LocalPath); err != nil {
			return result, err
		}
		if err = localRepo.AddAll(); err != nil {
			return CreateResult{}, err
		}
		if err = localRepo.Commit(&git.CommitOp{Message: "Initial commit\n[skip ci]"}); err != nil {
			return result, err
		}
	}

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
	if err = localRepo.Push(op.GitAuth); err != nil {
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
	return result, nil
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
	_, response, err := githubClient.Repositories.Get(
		context.Background(),
		op.OrgName,
		op.Name,
	)
	if err == nil {
		return fmt.Errorf("%s/%s repository is already exists", op.OrgName, op.Name)
	}
	if err != nil && response.StatusCode != http.StatusNotFound {
		return fmt.Errorf("error while checking if %s/%s repository exists", op.OrgName, op.Name)
	}
	return nil
}
