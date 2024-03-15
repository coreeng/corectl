package application

import (
	"github.com/coreeng/corectl/pkg/cmd/application/create"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/spf13/cobra"
)

func NewAppCmd(cfg *config.Config) (*cobra.Command, error) {
	appCmd := &cobra.Command{
		Use:     "app",
		Aliases: []string{"apps", "application", "applications"},
		Short:   "Operations with applications",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Help(); err != nil {
				return err
			}
			return nil
		},
	}

	appCreateCmd, err := create.NewAppCreateCmd(cfg)
	if err != nil {
		return nil, err
	}
	appCmd.AddCommand(appCreateCmd)

	return appCmd, nil
}
