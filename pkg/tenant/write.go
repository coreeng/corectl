package tenant

import (
	"context"
	"fmt"
	"github.com/coreeng/corectl/pkg/git"
	"os"
	"path/filepath"

	"github.com/google/go-github/v59/github"
	"gopkg.in/yaml.v3"
)

type CreateOrUpdateOp struct {
	Tenant            *Tenant
	CplatformRepoPath string
	BranchName        string
	CommitMessage     string
	PRName            string
	GitAuth           git.AuthMethod
}

type CreateOrUpdateResult struct {
	PRUrl string
}

func CreateOrUpdate(
	op *CreateOrUpdateOp,
	githubClient *github.Client,
) (result CreateOrUpdateResult, err error) {
	result = CreateOrUpdateResult{}

	repository, err := git.OpenAndResetRepositoryState(op.CplatformRepoPath)
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

	serializedTenant, err := yaml.Marshal(op.Tenant)
	if err != nil {
		return result, err
	}
	var tenantFilePath string
	if op.Tenant.path == nil {
		tenantFilePath = filepath.Join(tenantsRelativePath, string(op.Tenant.Name)+".yaml")
	} else {
		tenantFilePath = filepath.Join(tenantsRelativePath, *op.Tenant.path)
	}
	newTenantFileAbsPath := filepath.Join(op.CplatformRepoPath, tenantFilePath)
	if err = os.WriteFile(newTenantFileAbsPath, serializedTenant, 0o644); err != nil {
		return result, err
	}

	if err = repository.AddFiles(tenantFilePath); err != nil {
		return result, err
	}
	if err = repository.Commit(&git.CommitOp{Message: op.CommitMessage}); err != nil {
		return result, err
	}
	if err = repository.Push(op.GitAuth); err != nil {
		return result, err
	}

	fullname, err := git.DeriveRepositoryFullname(repository)
	if err != nil {
		return result, err
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
		return result, err
	}

	result.PRUrl = pullRequest.GetHTMLURL()
	return result, nil
}
