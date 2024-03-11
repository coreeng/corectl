package p2p

import (
	"context"
	"github.com/coreeng/developer-platform/dpctl/git"
	"github.com/coreeng/developer-platform/dpctl/tenant"
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
