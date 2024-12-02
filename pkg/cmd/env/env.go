package env

import (
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/spf13/cobra"
)

func NewEnvCmd(cfg *config.Config) *cobra.Command {
	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Work with Platform Environments",
		Long:  `This command allows you to manage environment configurations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	envCmd.AddCommand(listCmd(cfg))
	envCmd.AddCommand(connectCmd(cfg))
	envCmd.AddCommand(openResource(cfg))
	envCmd.AddCommand(disconnectCmd(cfg))
	envCmd.AddCommand(activeCmd(cfg))

	return envCmd
}
