package list

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/spf13/cobra"
)

func NewTemplateListCmd(cfg *config.Config) *cobra.Command {
	templateListCmd := &cobra.Command{
		Use:   "list",
		Short: "List templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.Templates); err != nil {
				return err
			}
			ts, err := template.List(cfg.Repositories.Templates.Value)
			if err != nil {
				return err
			}
			for _, t := range ts {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), t.Name); err != nil {
					return err
				}
			}
			return nil
		},
	}

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.Templates,
		templateListCmd.Flags(),
	)

	return templateListCmd
}
