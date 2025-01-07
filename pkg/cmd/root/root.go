package root

import (
	"fmt"
	"os"
	"strings"

	"github.com/coreeng/corectl/pkg/cmd/env"
	"github.com/coreeng/corectl/pkg/logger"

	"github.com/coreeng/corectl/pkg/cmd/application"
	configcmd "github.com/coreeng/corectl/pkg/cmd/config"
	"github.com/coreeng/corectl/pkg/cmd/p2p"
	"github.com/coreeng/corectl/pkg/cmd/template"
	"github.com/coreeng/corectl/pkg/cmd/tenant"
	update "github.com/coreeng/corectl/pkg/cmd/update-cli"
	"github.com/coreeng/corectl/pkg/cmd/version"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/spf13/cobra"
)

func ConfigureGlobalLogger(logLevelFlag string) {
	logger.Init(logLevelFlag)

}

func isCompletion() bool {
	return (len(os.Args) >= 2) && (os.Args[1] == "__complete")
}

func NewRootCmd(cfg *config.Config) *cobra.Command {
	var logLevel string
	var nonInteractive bool

	rootCmd := &cobra.Command{
		Use:   "corectl",
		Short: "CLI interface for the CECG core platform.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !isCompletion() {
				logger.Init(logLevel)
				defer logger.Sync()

				cmd.SilenceErrors = true

				cmdName := cmd.Name()
				if cmdName != update.CmdName {
					update.CheckForUpdates(cfg, cmd)
				}
				cmdPath := cmd.CommandPath()
				if !cfg.IsPersisted() && !strings.HasPrefix(cmdPath, "corectl config") && cmdName != version.CmdName {
					styles := userio.NewNonInteractiveStyles()
					streams := userio.NewIOStreamsWithInteractive(
						os.Stdin,
						os.Stdout,
						os.Stderr,
						false,
					)
					streams.Warn(
						fmt.Sprintf(
							"Config not initialised, please run %s.",
							styles.Bold.Inherit(styles.WarnMessageStyle).Render("corectl config init"),
						),
					)
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Help(); err != nil {
				return err
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVarP(
		&logLevel,
		"log-level",
		"l",
		"warn",
		"Log level - set up console log level. Default: warn. Accepted values: debug|info|warn|error",
	)

	rootCmd.PersistentFlags().BoolVar(
		&nonInteractive,
		"nonint",
		false,
		"Run in non-interactive mode - the command will error if it needs to ask for user input",
	)
	rootCmd.PersistentFlags().BoolVar(
		&nonInteractive,
		"non-interactive",
		false,
		"Run in non-interactive mode - the command will error if it needs to ask for user input",
	)

	// --non-interactive is the standard used by other clis
	err := rootCmd.PersistentFlags().MarkDeprecated("nonint", "please use --non-interactive instead.")
	if err != nil {
		logger.Panic().Msg("unable to set --nonint as deprecated")
	}

	appCmd, err := application.NewAppCmd(cfg)
	if err != nil {
		logger.Panic().Msg("Unable to execute app command")
	}
	p2pCmd, err := p2p.NewP2PCmd(cfg)
	if err != nil {
		logger.Panic().Msg("Unable to execute p2p command")
	}
	rootCmd.AddCommand(appCmd)
	rootCmd.AddCommand(p2pCmd)

	rootCmd.AddCommand(configcmd.NewConfigCmd(cfg))
	rootCmd.AddCommand(env.NewEnvCmd(cfg))
	rootCmd.AddCommand(tenant.NewTenantCmd(cfg))
	rootCmd.AddCommand(template.NewTemplateCmd(cfg))
	rootCmd.AddCommand(update.UpdateCmd(cfg))
	rootCmd.AddCommand(version.VersionCmd(cfg))

	return rootCmd
}
