package tenant

import (
	"fmt"
	"path/filepath"

	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/developer-platform/pkg/tenant"
	"github.com/google/go-github/v59/github"
	"github.com/phuslu/log"
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
	}); err != nil {
		return result, err
	}
	defer func() {
		_ = repository.CheckoutBranch(&git.CheckoutOp{BranchName: git.MainBranch})
	}()

	log.Debug().Msg("writing tenant definition to cplatform repo")
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

	fullname, err := git.DeriveRepositoryFullname(repository)
	if err != nil {
		return result, err
	}

	pullRequest, err := git.CreateGitHubPR(
		githubClient,
		op.PRName,
		op.PRName,
		op.BranchName,
		fullname.Name(),
		fullname.Organization(),
		op.DryRun,
	)

	if err != nil {
		return result, err
	}

	result.PRUrl = pullRequest.GetHTMLURL()
	return result, nil
}
