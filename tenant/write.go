package tenant

import (
	"context"
	"github.com/coreeng/developer-platform/dpctl/git"
	"os"
	"path/filepath"

	"github.com/google/go-github/v59/github"
	"gopkg.in/yaml.v3"
)

type CreateOrUpdateOp struct {
	Tenant            *Tenant
	DPlatformRepoPath string
	BranchName        string
	CommitMessage     string
	PRName            string
}

type CreateOrUpdateResult struct {
	PRUrl string
}

func CreateOrUpdate(
	op *CreateOrUpdateOp,
	githubClient *github.Client,
) (CreateOrUpdateResult, error) {
	repository, err := git.OpenLocalRepository(op.DPlatformRepoPath)
	if err != nil {
		return CreateOrUpdateResult{}, err
	}
	if err := repository.CheckoutBranch(&git.CheckoutOp{BranchName: git.MainBranch}); err != nil {
		return CreateOrUpdateResult{}, err
	}
	if err := repository.Pull(); err != nil {
		return CreateOrUpdateResult{}, err
	}

	if err := repository.CheckoutBranch(&git.CheckoutOp{
		BranchName:      op.BranchName,
		CreateIfMissing: true,
	}); err != nil {
		return CreateOrUpdateResult{}, err
	}
	defer func() {
		_ = repository.CheckoutBranch(&git.CheckoutOp{BranchName: git.MainBranch})
	}()

	serializedTenant, err := yaml.Marshal(op.Tenant)
	if err != nil {
		return CreateOrUpdateResult{}, err
	}
	var tenantFilePath string
	if op.Tenant.path == nil {
		tenantFilePath = filepath.Join(tenantsRelativePath, string(op.Tenant.Name)+".yaml")
	} else {
		tenantFilePath = filepath.Join(tenantsRelativePath, *op.Tenant.path)
	}
	newTenantFileAbsPath := filepath.Join(op.DPlatformRepoPath, tenantFilePath)
	if err = os.WriteFile(newTenantFileAbsPath, serializedTenant, 0o644); err != nil {
		return CreateOrUpdateResult{}, err
	}

	if err = repository.AddFiles(tenantFilePath); err != nil {
		return CreateOrUpdateResult{}, err
	}
	if err = repository.Commit(&git.CommitOp{Message: op.CommitMessage}); err != nil {
		return CreateOrUpdateResult{}, err
	}
	if err = repository.Push(); err != nil {
		return CreateOrUpdateResult{}, err
	}

	fullname, err := git.DeriveRepositoryFullname(repository)
	if err != nil {
		return CreateOrUpdateResult{}, err
	}

	mainBaseBranch := git.MainBranch
	pullRequest, _, err := githubClient.PullRequests.Create(
		context.Background(),
		fullname.Organization,
		fullname.Name,
		&github.NewPullRequest{
			Title: &op.PRName,
			Head:  &op.BranchName,
			Base:  &mainBaseBranch,
		})
	if err != nil {
		return CreateOrUpdateResult{}, err
	}

	return CreateOrUpdateResult{PRUrl: pullRequest.GetHTMLURL()}, nil
}
