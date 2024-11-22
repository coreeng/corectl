package root

import (
	"fmt"
	"os"

	"github.com/coreeng/corectl/pkg/cmd/application"
	configcmd "github.com/coreeng/corectl/pkg/cmd/config"
	"github.com/coreeng/corectl/pkg/cmd/env"
	"github.com/coreeng/corectl/pkg/cmd/p2p"
	"github.com/coreeng/corectl/pkg/cmd/template"
	"github.com/coreeng/corectl/pkg/cmd/tenant"
	"github.com/coreeng/corectl/pkg/cmd/update"
	"github.com/coreeng/corectl/pkg/cmd/version"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/logger"
	"go.uber.org/zap"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func isCompletion() bool {
	return (len(os.Args) >= 2) && (os.Args[1] == "__complete")
}

func NewRootCmd(cfg *config.Config) *cobra.Command {

	// Initialize the logger
	if err := logger.Init(); err != nil {
		panic(err)
	}
	defer func() {
		logger.Sync()
	}()

	var logLevel string
	var nonInteractive bool

	rootCmd := &cobra.Command{
		Use:   "corectl",
		Short: "CLI interface for the CECG core platform.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

			switch logLevel {
			case "debug":
				logger.AtomicLevel.SetLevel(zap.DebugLevel)
			case "info":
				logger.AtomicLevel.SetLevel(zap.InfoLevel)
			case "warn":
				logger.AtomicLevel.SetLevel(zap.WarnLevel)
			case "error":
				logger.AtomicLevel.SetLevel(zap.ErrorLevel)
			default:
				logger.AtomicLevel.SetLevel(zap.InfoLevel)
			}

			if !isCompletion() {
				cmd.SilenceErrors = true

				// Get command path and remove "corectl"
				fullCmd := cmd.CommandPath()

				// Get only the flags
				flags := []string{}
				cmd.Flags().Visit(func(f *pflag.Flag) {
					if f.Value.Type() == "bool" {
						flags = append(flags, "-"+f.Name)
					} else {
						flags = append(flags, "-"+f.Name, f.Value.String())
					}
				})

				logger.Debug("starting command",
					zap.String("command", fullCmd),
					zap.Strings("args", flags),
					zap.Strings("raw_input", os.Args),
				)

				cmdName := cmd.Name()
				if cmdName != update.CmdName {
					update.CheckForUpdates(cfg, cmd)
				}
				if !cfg.IsPersisted() && !(cmdName == configcmd.CmdName || cmdName == version.CmdName) {
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
		"info",
		fmt.Sprintf("options: debug|info|warn|error, writes to %s", logger.LogFile),
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
		logger.Fatal("unable to set --nonint as deprecated")
	}

	appCmd, err := application.NewAppCmd(cfg)
	if err != nil {
		panic("Unable to execute app command")
	}
	p2pCmd, err := p2p.NewP2PCmd(cfg)
	if err != nil {
		panic("Unable to execute p2p command")
	}

	rootCmd.AddCommand(appCmd)
	rootCmd.AddCommand(p2pCmd)
	rootCmd.AddCommand(configcmd.NewConfigCmd(cfg))
	rootCmd.AddCommand(tenant.NewTenantCmd(cfg))
	rootCmd.AddCommand(template.NewTemplateCmd(cfg))
	rootCmd.AddCommand(env.NewEnvCmd(cfg))
	rootCmd.AddCommand(version.VersionCmd(cfg))
	rootCmd.AddCommand(update.UpdateCmd(cfg))

	return rootCmd
}
