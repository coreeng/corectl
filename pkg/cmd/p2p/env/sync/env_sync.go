package sync

import (
	"context"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/environment"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/p2p"
	"github.com/coreeng/corectl/pkg/utils"
	"github.com/google/go-github/v59/github"
	"github.com/spf13/cobra"
)

type EnvCreateOpts struct {
	AppRepo string
	Tenant  string
	Streams userio.IOStreams
}

func NewP2PSyncCmd(cfg *config.Config) (*cobra.Command, error) {

	var opts = EnvCreateOpts{}
	var syncEnvironmentsCmd = &cobra.Command{
		Use:   "sync <app repository> <tenant>",
		Short: "Synchronise Environments",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.AppRepo = args[0]
			opts.Tenant = args[1]

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
	spinnerHandler := opts.Streams.Spinner("Configuring environments...")
		defer spinnerHandler.Done()
	for _, env := range environments {
		
		err = p2p.CreateUpdateEnvironmentForRepository(
			githubClient,
			&repoId,
			&env,
		)
		if err != nil {
			return err
		}
	}

	err = p2p.CreateTenantVariableFromName(
		githubClient,
		&repoId.Fullname,
		opts.Tenant,
	)
	if err != nil {
		return err
	}
	fastFeedbackEnvs := utils.FilterEnvs(cfg.P2P.FastFeedback.DefaultEnvs.Value, environments)
	extendedTestEnvs := utils.FilterEnvs(cfg.P2P.ExtendedTest.DefaultEnvs.Value, environments)
	prodEnvs := utils.FilterEnvs(cfg.P2P.Prod.DefaultEnvs.Value, environments)
	if err := p2p.CreateStageRepositoryConfig(
		githubClient,
		&repoId.Fullname,
		p2p.FastFeedbackVar,
		p2p.NewStageRepositoryConfig(fastFeedbackEnvs)); err != nil {
		return err
	}

	if err := p2p.CreateStageRepositoryConfig(
		githubClient,
		&repoId.Fullname,
		p2p.ExtendedTestVar,
		p2p.NewStageRepositoryConfig(extendedTestEnvs)); err != nil {
		return err
	}

	if err := p2p.CreateStageRepositoryConfig(
		githubClient,
		&repoId.Fullname,
		p2p.ProdVar,
		p2p.NewStageRepositoryConfig(prodEnvs)); err != nil {
		return err
	}
	return nil
}
