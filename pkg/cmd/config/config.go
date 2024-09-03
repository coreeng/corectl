package config

import (
	initcmd "github.com/coreeng/corectl/pkg/cmd/config/init"
	"github.com/coreeng/corectl/pkg/cmd/config/set"
	"github.com/coreeng/corectl/pkg/cmd/config/update"
	"github.com/coreeng/corectl/pkg/cmd/config/view"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/spf13/cobra"
)

func NewConfigCmd(cfg *config.Config) *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "corectl configuration management operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Help(); err != nil {
				return err
			}
			return nil
		},
	}

	configCmd.AddCommand(initcmd.NewConfigInitCmd(cfg))
	configCmd.AddCommand(update.NewConfigUpdateCmd(cfg))
	configCmd.AddCommand(view.NewConfigViewCmd(cfg))
	configCmd.AddCommand(set.NewConfigSetCmd(cfg))

	return configCmd
}
