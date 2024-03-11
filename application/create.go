package application

import (
	"context"
	"github.com/coreeng/developer-platform/dpctl/environment"
	"github.com/coreeng/developer-platform/dpctl/git"
	"github.com/coreeng/developer-platform/dpctl/p2p"
	"github.com/coreeng/developer-platform/dpctl/template"
	"github.com/coreeng/developer-platform/dpctl/tenant"
	"github.com/google/go-github/v59/github"
	"os"
)

type CreateOp struct {
	Name             string
	OrgName          string
	LocalPath        string
	Tenant           *tenant.Tenant
	FastFeedbackEnvs []environment.Environment
	ExtendedTestEnvs []environment.Environment
	ProdEnvs         []environment.Environment
	TemplatesPath    string
	Template         *template.FulfilledTemplate
}

type CreateResult struct {
	RepositoryFullname git.RepositoryFullname
}

func Create(op CreateOp, githubClient *github.Client) (CreateResult, error) {
	result := CreateResult{}
	if err := os.MkdirAll(op.LocalPath, 0o755); err != nil {
		return result, err
	}
	localRepo, err := git.InitLocalRepository(op.LocalPath)
	if err != nil {
		return result, err
	}
	if op.Template != nil {
		if err = template.Render(op.Template, op.TemplatesPath, op.LocalPath); err != nil {
			return result, err
		}
		if err = localRepo.AddAll(); err != nil {
			return CreateResult{}, err
		}
		if err = localRepo.Commit(&git.CommitOp{Message: "Initial commit\n[skip ci]"}); err != nil {
			return CreateResult{}, err
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
	result.RepositoryFullname = git.NewGithubRepoFullId(githubRepo).Fullname

	if err = localRepo.SetRemote(githubRepo.GetSSHURL()); err != nil {
		return CreateResult{}, err
	}
	if err = localRepo.Push(); err != nil {
		return result, err
	}

	repoFullId := git.NewGithubRepoFullId(githubRepo)
	if err = p2p.InitializeRepository(&p2p.InitializeOp{
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
