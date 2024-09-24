package tenant

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/developer-platform/pkg/tenant"
	"github.com/google/go-github/v59/github"
	"github.com/rs/zerolog/log"
)

type CreateOrUpdateOp struct {
	Tenant            *tenant.Tenant
	ParentTenant      *tenant.Tenant
	CplatformRepoPath string
	BranchName        string
	CommitMessage     string
	PRName            string
	PRBody            string
	GitAuth           git.AuthMethod
	DryRun            bool
}

type CreateOrUpdateResult struct {
	PRUrl string
}

func CreateOrUpdate(
	op *CreateOrUpdateOp,
	githubClient *github.Client,
) (result CreateOrUpdateResult, err error) {
	result = CreateOrUpdateResult{}

	repository, err := git.OpenAndResetRepositoryState(op.CplatformRepoPath, op.DryRun)
	if err != nil {
		return result, fmt.Errorf("couldn't open cplatform repository: %v", err)
	}

	if err = repository.CheckoutBranch(&git.CheckoutOp{
		BranchName:      op.BranchName,
		CreateIfMissing: true,
		DryRun:          op.DryRun,
	}); err != nil {
		return result, err
	}
	defer func() {
		_ = repository.CheckoutBranch(&git.CheckoutOp{BranchName: git.MainBranch, DryRun: op.DryRun})
	}()

	log.Debug().Msg("writing tenant definition to cplatform repo")
	if !op.DryRun {
		if err = tenant.CreateOrUpdate(tenant.CreateOrUpdateOp{
			Tenant:       op.Tenant,
			ParentTenant: op.ParentTenant,
			TenantsDir:   tenant.DirFromCPlatformPath(op.CplatformRepoPath),
		}); err != nil {
			return CreateOrUpdateResult{}, err
		}

		relativeFilepath, err := filepath.Rel(op.CplatformRepoPath, *op.Tenant.SavedPath())
		if err != nil {
			return result, err
		}
		if err = repository.AddFiles(relativeFilepath); err != nil {
			return result, err
		}
		if err = repository.Commit(&git.CommitOp{Message: op.CommitMessage}); err != nil {
			return result, err
		}
		if err = repository.Push(git.PushOp{
			Auth:       op.GitAuth,
			BranchName: op.BranchName,
		}); err != nil {
			return result, err
		}
	}

	log.Debug().Msg("creating github PR")
	if !op.DryRun {
		fullname, err := git.DeriveRepositoryFullname(repository)
		if err != nil {
			return result, err
		}
		mainBaseBranch := git.MainBranch
		pullRequest, _, err := githubClient.PullRequests.Create(
			context.Background(),
			fullname.Organization(),
			fullname.Name(),
			&github.NewPullRequest{
				Base:  &mainBaseBranch,
				Head:  &op.BranchName,
				Title: &op.PRName,
				Body:  &op.PRBody,
			})
		if err != nil {
			return result, err
		}

		result.PRUrl = pullRequest.GetHTMLURL()
	}
	return result, nil
}
