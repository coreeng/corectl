package root

import (
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

	log.Trace().Msgf("Log level set to %s", logLevelFlag)
}

func NewRootCmd(cfg *config.Config) *cobra.Command {
	var logLevel string
	rootCmd := &cobra.Command{
		Use:   "corectl",
		Short: "CLI interface for the CECG core platform.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ConfigureGlobalLogger(logLevel)
			cmd.SilenceErrors = true
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

	appCmd, err := application.NewAppCmd(cfg)
	if err != nil {
		panic("Unable to execute app command")
	}
	p2pCmd, err := p2p.NewP2PCmd(cfg)
	if err != nil {
		panic("Unable to execute p2p command")
	}
	rootCmd.AddCommand(appCmd)
	rootCmd.AddCommand(configcmd.NewConfigCmd(cfg))
	rootCmd.AddCommand(p2pCmd)
	rootCmd.AddCommand(tenant.NewTenantCmd(cfg))
	rootCmd.AddCommand(template.NewTemplateCmd(cfg))
	rootCmd.AddCommand(env.NewEnvCmd(cfg))
	rootCmd.AddCommand(version.VersionCmd(cfg))
	rootCmd.AddCommand(update.UpdateCmd(cfg))

	return rootCmd
}
