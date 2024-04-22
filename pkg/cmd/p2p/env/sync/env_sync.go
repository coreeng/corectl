package sync

import (
	"context"
	"errors"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/p2p"
	"github.com/coreeng/corectl/pkg/tenant"
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

	spinnerHandler := opts.Streams.Spinner("Configuring environments...")
	defer spinnerHandler.Done()
	
	if tenant, err := tenant.FindByName(cfg.Repositories.CPlatform.Value, tenant.Name(opts.Tenant)); err == nil {
		if tenant != nil {
			err = p2p.SynchroniseEnvironment(githubClient, repository, cfg, tenant)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Tenant not found!")
		}
	}
	return nil
}
