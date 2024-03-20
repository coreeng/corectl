package p2p

import (
	"github.com/coreeng/corectl/pkg/environment"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/tenant"
	"github.com/google/go-github/v59/github"
	"slices"
)

type InitializeOp struct {
	RepositoryId     *git.GithubRepoFullId
	Tenant           *tenant.Tenant
	FastFeedbackEnvs []environment.Environment
	ExtendedTestEnvs []environment.Environment
	ProdEnvs         []environment.Environment
}

func InitializeRepository(
	op *InitializeOp,
	githubClient *github.Client,
) error {
	if err := CreateTenantVariable(
		githubClient,
		&op.RepositoryId.Fullname,
		op.Tenant); err != nil {
		return err
	}
	if err := CreateStageRepositoryConfig(
		githubClient,
		&op.RepositoryId.Fullname,
		FastFeedbackVar,
		NewStageRepositoryConfig(op.FastFeedbackEnvs)); err != nil {
		return err
	}
	if err := CreateStageRepositoryConfig(
		githubClient,
		&op.RepositoryId.Fullname,
		ExtendedTestVar,
		NewStageRepositoryConfig(op.ExtendedTestEnvs)); err != nil {
		return err
	}
	if err := CreateStageRepositoryConfig(
		githubClient,
		&op.RepositoryId.Fullname,
		ProdVar,
		NewStageRepositoryConfig(op.ProdEnvs)); err != nil {
		return err
	}

	var createdEnvs []environment.Name
	for _, env := range slices.Concat(op.FastFeedbackEnvs, op.ExtendedTestEnvs, op.ProdEnvs) {
		if slices.Contains(createdEnvs, env.Environment) {
			continue
		}
		createdEnvs = append(createdEnvs, env.Environment)
		if err := CreateEnvironmentForRepository(githubClient, op.RepositoryId, &env); err != nil {
			return err
		}
	}
	return nil
}
