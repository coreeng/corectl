package list

import (
	"fmt"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/spf13/cobra"
)

type TemplateListOpts struct {
	IgnoreChecks bool
}

func NewTemplateListCmd(cfg *config.Config) *cobra.Command {
	opts := TemplateListOpts{}
	templateListCmd := &cobra.Command{
		Use:   "list",
		Short: "List templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !opts.IgnoreChecks {
				if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.Templates, false); err != nil {
					return err
				}
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

	templateListCmd.Flags().BoolVar(
		&opts.IgnoreChecks,
		"ignore-checks",
		false,
		"Ignore checks for uncommitted changes and branch status",
	)

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.Templates,
		templateListCmd.Flags(),
	)

	return templateListCmd
}
