package env

import (
	"fmt"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	corectlenv "github.com/coreeng/corectl/pkg/env"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/spf13/cobra"
)

type ListOpt struct {
	RepositoryLocation string
	Streams            userio.IOStreams
}

func listCmd(cfg *config.Config) *cobra.Command {
	var opts = ListOpt{}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
			)

			return list(opts, cfg)
		},
	}

	listCmd.Flags().StringVarP(
		&opts.RepositoryLocation,
		"repository",
		"r",
		cfg.Repositories.CPlatform.Value,
		"Repository to source environments from",
	)

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.CPlatform,
		listCmd.Flags(),
	)

	return listCmd
}

func list(opts ListOpt, cfg *config.Config) error {
	if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.CPlatform); err != nil {
		return err
	}
	existing, err := environment.List(environment.DirFromCPlatformRepoPath(opts.RepositoryLocation))
	if err != nil {
		return fmt.Errorf("could not find repository location %q: %w", opts.RepositoryLocation, err)
	}

	table := corectlenv.NewTable(opts.Streams, "Name", "ID", "Cloud Platform")
	for _, env := range existing {
		table.AppendEnv(env)
	}
	table.Render()

	return nil
}
