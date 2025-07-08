package auth

import (
	"github.com/coreeng/corectl/pkg/cmd/auth/login"
	"github.com/coreeng/corectl/pkg/cmd/auth/logout"
	"github.com/coreeng/corectl/pkg/cmd/auth/test"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/spf13/cobra"
)

func NewAuthCmd(cfg *config.Config) *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication operations",
		Long:  "Manage authentication with Google IAP using OAuth 2.0",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Help(); err != nil {
				return err
			}
			return nil
		},
	}

	authCmd.AddCommand(login.NewLoginCmd(cfg))
	authCmd.AddCommand(logout.NewLogoutCmd(cfg))
	authCmd.AddCommand(test.NewTestCmd(cfg))

	return authCmd
}
