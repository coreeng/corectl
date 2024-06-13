package env

import (
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	corectlenv "github.com/coreeng/corectl/pkg/env"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/spf13/cobra"
)

func listCmd(cfg *config.Config) *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg)
		},
	}
	return listCmd
}

func list(cfg *config.Config) error {
	if err := config.NotExist(); err != nil {
		return err
	}
	existing, err := environment.List(environment.DirFromCPlatformRepoPath(cfg.Repositories.CPlatform.Value))
	if err != nil {
		return err
	}

	table := corectlenv.NewTable("Name", "ID", "Cloud Platform")
	for _, env := range existing {
		corectlenv.AppendEnv(table, env)
	}
	table.Render()

	return nil
}
