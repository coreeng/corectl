package tenant

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
	"path/filepath"

	"github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/logger"
	"github.com/google/go-github/v60/github"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
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

	definition, err := yaml.Marshal(op.Tenant)
	if err != nil {
		return result, err
	}
	logger.Debug().With(
		// TODO: add public method to render Tenant in Core Platform
		//       so we can log it here when dry-running
		zap.String("repo", op.CplatformRepoPath),
		zap.Bool("dry_run", op.DryRun),
		zap.String("definition", string(definition))).
		Msg("writing tenant definition to cplatform repo")
	var relativeFilepath string
	if !op.DryRun {
		if err = tenant.CreateOrUpdate(tenant.CreateOrUpdateOp{
			Tenant:       op.Tenant,
			ParentTenant: op.ParentTenant,
			TenantsDir:   configpath.GetCorectlCPlatformDir("tenants"),
		}); err != nil {
			return result, err
		}
		relativeFilepath, err = filepath.Rel(op.CplatformRepoPath, *op.Tenant.SavedPath())
		if err != nil {
			return result, err
		}
	} else {
		// Approximation for dry-run
		relativeFilepath = fmt.Sprintf("tenants/%s.yml", op.Tenant.Name)
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
