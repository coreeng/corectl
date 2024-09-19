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
	"github.com/coreeng/corectl/pkg/cmd/version"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func ConfigureGlobalLogger(logLevelFlag string) {
	logLevel, logLevelParseError := zerolog.ParseLevel(logLevelFlag)
	if logLevelParseError != nil {
		logLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().
		Timestamp().
		Caller().
		Logger()

	log.Logger = logger
	log.Trace().Msgf("Log level set to %s", logLevelFlag)
}

func NewRootCmd(cfg *config.Config) *cobra.Command {
	var logLevelFlag string

	rootCmd := &cobra.Command{
		Use:   "corectl",
		Short: "CLI interface for the CECG core platform.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ConfigureGlobalLogger(logLevelFlag)
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
		&logLevelFlag,
		"log-level",
		"l",
		"INFO",
		"Log level",
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

	return rootCmd
}
