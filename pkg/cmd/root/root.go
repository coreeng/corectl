package root

import (
	"github.com/coreeng/corectl/pkg/cmd/application"
	configcmd "github.com/coreeng/corectl/pkg/cmd/config"
	"github.com/coreeng/corectl/pkg/cmd/template"
	"github.com/coreeng/corectl/pkg/cmd/tenant"
	"github.com/coreeng/corectl/pkg/cmdutil/config"

	"github.com/spf13/cobra"
)

func NewRootCmd(cfg *config.Config) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "corectl",
		Short: "CLI interface for the CECG core platform.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
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

	appCmd, err := application.NewAppCmd(cfg)
	if err != nil {
		return nil
	}

	rootCmd.AddCommand(configcmd.NewConfigCmd(cfg))
	rootCmd.AddCommand(tenant.NewTenantCmd(cfg))
	rootCmd.AddCommand(template.NewTemplateCmd(cfg))
	rootCmd.AddCommand(appCmd)
	rootCmd.AddCommand(p2p.NewRepoCmd(cfg))

	return rootCmd
}
