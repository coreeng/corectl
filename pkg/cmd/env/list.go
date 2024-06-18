package env

import (
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	corectlenv "github.com/coreeng/corectl/pkg/env"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/spf13/cobra"
)

func listCmd(cfg *config.Config) *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			streams := userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
			)

			return list(cfg, streams)
		},
	}

	return listCmd
}

func list(cfg *config.Config, streams userio.IOStreams) error {
	if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.CPlatform); err != nil {
		return err
	}
	existing, err := environment.List(environment.DirFromCPlatformRepoPath(cfg.Repositories.CPlatform.Value))
	if err != nil {
		return err
	}

	table := corectlenv.NewTable(streams, "Name", "ID", "Cloud Platform")
	for _, env := range existing {
		corectlenv.AppendEnv(table, env)
	}
	table.Render()

	return nil
}
