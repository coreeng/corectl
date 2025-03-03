package env

import (
	"fmt"
	"github.com/coreeng/core-platform/pkg/environment"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	corectlenv "github.com/coreeng/corectl/pkg/env"
	"github.com/coreeng/corectl/pkg/logger"
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
			logger.Info().Msgf("Invoked with args: %+v", opts)
			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				cmd.OutOrStderr(),
			)

			return list(opts, cfg)
		},
	}

	config.RegisterBoolParameterAsFlag(
		&cfg.Repositories.AllowDirty,
		listCmd.Flags(),
	)

	return listCmd
}

func list(opts ListOpt, cfg *config.Config) error {
	repoParams := []config.Parameter[string]{cfg.Repositories.CPlatform}
	err := config.Update(cfg.GitHub.Token.Value, opts.Streams, cfg.Repositories.AllowDirty.Value, repoParams)
	if err != nil {
		return fmt.Errorf("failed to update config repos: %w", err)
	}

	existing, err := environment.List(configpath.GetCorectlCPlatformDir("environments"))
	if err != nil {
		return fmt.Errorf("could not find repository location: %w", err)
	}

	table := corectlenv.NewTable(opts.Streams, false)
	for _, env := range existing {
		table.AppendEnv(env, "-", "-")
	}
	table.Render()

	return nil
}
