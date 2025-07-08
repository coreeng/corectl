package logout

import (
	"github.com/coreeng/corectl/pkg/auth"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/spf13/cobra"
)

func NewLogoutCmd(cfg *config.Config) *cobra.Command {
	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout and clear stored authentication tokens",
		Long: `Logout and clear stored authentication tokens.
		
This command will remove all stored OAuth 2.0 tokens from the configuration,
requiring you to run 'corectl auth login' again to authenticate.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			streams := userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				cmd.OutOrStderr(),
			)

			return run(cfg, streams)
		},
	}

	return logoutCmd
}

func run(cfg *config.Config, streams userio.IOStreams) error {
	if !auth.IsAuthenticated(cfg) {
		streams.Info("You are not currently logged in.")
		return nil
	}

	// Get user info before logout
	userInfo, err := auth.GetUserInfo(cfg)
	if err != nil {
		streams.Warn("Could not retrieve user information, but proceeding with logout")
	}

	// Clear tokens
	if err := auth.Logout(cfg); err != nil {
		return err
	}

	if userInfo != nil {
		streams.Info("Successfully logged out " + userInfo.Email)
	} else {
		streams.Info("Successfully logged out")
	}

	return nil
}
