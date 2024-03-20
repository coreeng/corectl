package p2p

import (
	"context"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/tenant"
	"github.com/google/go-github/v59/github"
)

func CreateTenantVariable(
	githubClient *github.Client,
	repoFullname *git.RepositoryFullname,
	tenant *tenant.Tenant,
) error {
	_, err := githubClient.Actions.CreateRepoVariable(
		context.Background(),
		repoFullname.Organization,
		repoFullname.Name,
		&github.ActionsVariable{
			Name:  "TENANT_NAME",
			Value: string(tenant.Name),
		})
	return err

}
