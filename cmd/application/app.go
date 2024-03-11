package application

import (
	"github.com/coreeng/developer-platform/dpctl/cmd/application/create"
	"github.com/coreeng/developer-platform/dpctl/cmd/config"
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
