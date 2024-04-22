package p2p

import (
	"context"
	"errors"

	"github.com/coreeng/corectl/pkg/tenant"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/environment"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/utils"
	"github.com/google/go-github/v59/github"
)

func SynchroniseEnvironment(
	githubClient *github.Client,
	repository   *github.Repository,
	cfg          *config.Config,
	tenant       *tenant.Tenant,
) error {
	repoId := git.NewGithubRepoFullId(repository)
	environments, err := environment.List(cfg.Repositories.CPlatform.Value)
	if err != nil {
		return err
	}

	//Remove any existing environments as per #295

	for _, env := range environments {
		if env.Platform.Vendor == "gcp" {

			_, response, err := githubClient.Repositories.GetEnvironment(
				context.Background(),
				repoId.Fullname.Organization,
				repoId.Fullname.Name,
				string(env.Environment),
			)
			if err != nil {
				if response.StatusCode == 404 {
					continue
				} else {
					return err
				}
			}
			_, err = githubClient.Repositories.DeleteEnvironment(
				context.Background(),
				repoId.Fullname.Organization,
				repoId.Fullname.Name,
				string(env.Environment),
			)
			if err != nil {
				return errors.New("Unable to delete existing environments " + string(env.Environment) + " in " + repoId.Fullname.Name)
			}
		}
	}
	for _, env := range environments {
		
		err = CreateUpdateEnvironmentForRepository(
			githubClient,
			&repoId,
			&env,
		)
		if err != nil {
			return errors.New("Unable to create environment - " + err.Error())
		}
	}
	err = CreateTenantVariable(githubClient, &repoId.Fullname, tenant)
	
	if err != nil {
		return err
	}
	
	fastFeedbackEnvs := utils.FilterEnvs(cfg.P2P.FastFeedback.DefaultEnvs.Value, environments)
	extendedTestEnvs := utils.FilterEnvs(cfg.P2P.ExtendedTest.DefaultEnvs.Value, environments)
	prodEnvs := utils.FilterEnvs(cfg.P2P.Prod.DefaultEnvs.Value, environments)
	if err := CreateStageRepositoryConfig(
		githubClient,
		&repoId.Fullname,
		FastFeedbackVar,
		NewStageRepositoryConfig(fastFeedbackEnvs)); err != nil {
		return err
	}

	if err := CreateStageRepositoryConfig(
		githubClient,
		&repoId.Fullname,
		ExtendedTestVar,
		NewStageRepositoryConfig(extendedTestEnvs)); err != nil {
		return err
	}

	if err := CreateStageRepositoryConfig(
		githubClient,
		&repoId.Fullname,
		ProdVar,
		NewStageRepositoryConfig(prodEnvs)); err != nil {
		return err
	}
	return nil
}

func CreateUpdateEnvironmentForRepository(
	githubClient *github.Client,
	repoId *git.GithubRepoFullId,
	env *environment.Environment,
) error {
	if env.Platform.Vendor == "gcp" {
		repoEnv, _, err := githubClient.Repositories.CreateUpdateEnvironment(
			context.Background(),
			repoId.Fullname.Organization,
			repoId.Fullname.Name,
			string(env.Environment),
			&github.CreateUpdateEnvironment{},
		)
		if err != nil {
			return err
		}
		defaultDomain := env.GetDefaultIngressDomain()
		if defaultDomain == nil {
			return errors.New("default ingress domain is not found")
		}
		varsToCreate := []github.ActionsVariable{
			{
				Name:  "BASE_DOMAIN",
				Value: defaultDomain.Domain,
			},
			{
				Name:  "INTERNAL_SERVICES_DOMAIN",
				Value: env.InternalServices.Domain,
			},
			{
				Name:  "DPLATFORM",
				Value: string(env.Environment),
			},
			{
				Name:  "PROJECT_ID",
				Value: env.Platform.ProjectId,
			},
			{
				Name:  "PROJECT_NUMBER",
				Value: env.Platform.ProjectNumber,
			},
		}
		for i := range varsToCreate {
			response, err := githubClient.Actions.CreateEnvVariable(
				context.Background(),
				repoId.Id,
				*repoEnv.Name,
				&varsToCreate[i],
			)
			if err != nil {
				if response.StatusCode == 409 {
					_, err := githubClient.Actions.UpdateEnvVariable(
						context.Background(),
						int(repoId.Id),
						*repoEnv.Name,
						&varsToCreate[i],
					)
					if err != nil {
						return err
					}
				} else if response.StatusCode == 200 {
					return nil
				} else {
					return err
				}
			}
		}
	}
	return nil
}
