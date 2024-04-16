package sync

import (
	"context"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/environment"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/p2p"
	"github.com/coreeng/corectl/pkg/tenant"
	"github.com/google/go-github/v59/github"
	"github.com/spf13/cobra"
)

type EnvCreateOpts struct {
	AppRepo string
	Streams userio.IOStreams
}

func NewP2PSyncCmd(cfg *config.Config) (*cobra.Command, error) {

	var opts = EnvCreateOpts{}
	var syncEnvironmentsCmd = &cobra.Command{
		Use:   "sync <app repository> ",
		Short: "Synchronise Environments",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.AppRepo = args[0]

			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
			)
			return run(&opts, cfg)
		},
	}

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.CPlatform,
		syncEnvironmentsCmd.Flags())

	config.RegisterStringParameterAsFlag(
		&cfg.GitHub.Organization,
		syncEnvironmentsCmd.Flags())

	config.RegisterStringParameterAsFlag(
		&cfg.GitHub.Token,
		syncEnvironmentsCmd.Flags())

	return syncEnvironmentsCmd, nil
}

func run(opts *EnvCreateOpts, cfg *config.Config) error {

	githubClient := github.NewClient(nil).
		WithAuthToken(cfg.GitHub.Token.Value)

	repository, _, err := githubClient.Repositories.Get(
		context.Background(),
		cfg.GitHub.Organization.Value,
		opts.AppRepo)
	if err != nil {
		return err
	}
	repoId := git.NewGithubRepoFullId(repository)
	environments, err := environment.List(cfg.Repositories.CPlatform.Value)
	if err != nil {
		return err
	}
	for _, env := range environments {
		opts.Streams.Info("Updating " + string(env.Environment) + " environment for " + repoId.Fullname.Name)
		err = p2p.CreateUpdateEnvironmentForRepository(
			githubClient,
			&repoId,
			&env,
		)
		if err != nil {
			return err
		}
	}
	tenantList, err := tenant.List(cfg.Repositories.CPlatform.Value)
	if err != nil {
		return err
	}
	for _, tenant := range tenantList {
		err = p2p.CreateTenantVariableFromName(
			githubClient,
			&repoId.Fullname,
			string(tenant.Name),
		)
		if err != nil {
			return err
		}
	}
	return nil
}
