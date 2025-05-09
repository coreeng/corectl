package list

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
	"os"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
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
			cmd.SilenceUsage = true
			if !opts.IgnoreChecks {
				streams := userio.NewIOStreams(os.Stdin, os.Stdout, os.Stderr)
				repoParams := []config.Parameter[string]{cfg.Repositories.Templates}
				err := config.Update(cfg.GitHub.Token.Value, streams, cfg.Repositories.AllowDirty.Value, repoParams)
				if err != nil {
					return fmt.Errorf("failed to update config repos: %w", err)
				}
			}
			ts, err := template.List(configpath.GetCorectlTemplatesDir())
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

	return templateListCmd
}
