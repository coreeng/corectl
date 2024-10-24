package root

import (
	"os"
	"time"

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
	versionInfo "github.com/coreeng/corectl/pkg/version"
	"github.com/phuslu/log"
	"github.com/spf13/cobra"
)

func ConfigureGlobalLogger(logLevelFlag string) {
	logLevel := log.ParseLevel(logLevelFlag)
	log.DefaultLogger.SetLevel(logLevel)
	if !(logLevel == log.Level(8)) {
		log.DefaultLogger = log.Logger{
			Level:      logLevel,
			Caller:     1,
			TimeField:  "time",
			TimeFormat: time.TimeOnly,
			Writer: &log.FileWriter{
				Filename: "corectl.log",
			},
		}
	} else {
		log.DefaultLogger = log.Logger{
			Level: log.PanicLevel,
		}
	}
	log.Debug().
		Str("version", versionInfo.Version).
		Str("commit", versionInfo.Commit).
		Str("date", versionInfo.Commit).
		Msgf("starting up, args: %+v", os.Args[1:])
	log.Trace().Msgf("Log level set to %s", logLevelFlag)
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
				ConfigureGlobalLogger(logLevel)
				cmd.SilenceErrors = true

				if cmd.Name() != update.CmdName {
					update.CheckForUpdates(cfg, cmd)
				}
				if !cfg.IsPersisted() && !(cmd.Name() == configcmd.CmdName) {
					styles := userio.NewNonInteractiveStyles()
					streams := userio.NewIOStreamsWithInteractive(
						os.Stdin,
						os.Stdout,
						false,
					)
					streams.GetOutput().Write([]byte(
						styles.WarnMessageStyle.Render("Config not initialised, please run ") +
							styles.Bold.Inherit(styles.WarnMessageStyle).Render("corectl config init") +
							styles.WarnMessageStyle.Render(". Most commands will not be available.")))
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
		"disabled",
		"Log level - writes to ./corectl.log if set",
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
		log.Panic().Msg("unable to set --nonint as deprecated")
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
	rootCmd.AddCommand(configcmd.NewConfigCmd(cfg))
	rootCmd.AddCommand(version.VersionCmd(cfg))
	rootCmd.AddCommand(update.UpdateCmd(cfg))

	return rootCmd
}
