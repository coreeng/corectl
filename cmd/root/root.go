package root

import (
	"github.com/coreeng/developer-platform/dpctl/cmd/application"
	"github.com/coreeng/developer-platform/dpctl/cmd/config"
	initcmd "github.com/coreeng/developer-platform/dpctl/cmd/init"
	"github.com/coreeng/developer-platform/dpctl/cmd/template"
	"github.com/coreeng/developer-platform/dpctl/cmd/tenant"

	"github.com/spf13/cobra"
)

func NewRootCmd(cfg *config.Config) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "dpctl",
		Short: "CLI interface for the CECG developer platform.",
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

	rootCmd.AddCommand(initcmd.NewInitCmd(cfg))
	rootCmd.AddCommand(tenant.NewTenantCmd(cfg))
	rootCmd.AddCommand(template.NewTemplateCmd(cfg))
	rootCmd.AddCommand(appCmd)

	return rootCmd
}
