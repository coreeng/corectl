package p2p

import (
	"slices"

	"github.com/coreeng/corectl/pkg/environment"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/tenant"
	"github.com/google/go-github/v59/github"
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
	err := ConfigureRepository(op, githubClient)
	if err != nil {
		return err
	}

	var createdEnvs []environment.Name
	for _, env := range slices.Concat(op.FastFeedbackEnvs, op.ExtendedTestEnvs, op.ProdEnvs) {
		if slices.Contains(createdEnvs, env.Environment) {
			continue
		}
		createdEnvs = append(createdEnvs, env.Environment)
		if err := CreateUpdateEnvironmentForRepository(githubClient, op.RepositoryId, &env); err != nil {
			return err
		}
	}
	return nil
}

func ConfigureRepository(
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

	return nil
}
