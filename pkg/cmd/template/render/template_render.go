package render

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/spf13/cobra"
	"os"
)

type TemplateRenderOpt struct {
	Force bool
}

func NewTemplateRenderCmd(cfg *config.Config) *cobra.Command {
	opts := TemplateRenderOpt{}
	templateRenderCmd := &cobra.Command{
		Use:   "render <template-name> <target-path>",
		Short: "Render template locally",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateName := args[0]
			targetPath := args[1]

			stat, err := os.Stat(targetPath)
			if err != nil {
				if !os.IsNotExist(err) {
					pathError := err.(*os.PathError)
					return fmt.Errorf("%s: %v", pathError.Path, pathError.Err)
				}
				if err := os.MkdirAll(targetPath, 0755); err != nil {
					return fmt.Errorf("%s: could not create directory: %v", targetPath, err)
				}
			} else {
				if !stat.IsDir() {
					return fmt.Errorf("%s: not a directory", targetPath)
				}
			}

			if !opts.Force {
				dirEntries, err := os.ReadDir(targetPath)
				if err != nil {
					return fmt.Errorf("%s: couldn't list directory: %v", targetPath, err)
				}
				if len(dirEntries) > 0 {
					return fmt.Errorf("%s: directory is not empty", targetPath)
				}
			}

			if !cfg.Repositories.AllowDirty.Value {
				if _, err = config.ResetConfigRepositoryState(&cfg.Repositories.Templates); err != nil {
					return err
				}
			}

			templatesPath := cfg.Repositories.Templates.Value
			t, err := template.FindByName(templatesPath, templateName)
			if err != nil {
				return err
			}
			if t == nil {
				return fmt.Errorf("%s: unknown template", templateName)
			}

			fulfilledT := template.FulfilledTemplate{
				Spec:      t,
				Arguments: []template.Argument{},
			}
			if err := template.Render(&fulfilledT, targetPath); err != nil {
				return err
			}

			return nil
		},
	}

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.Templates,
		templateRenderCmd.Flags(),
	)
	config.RegisterBoolParameterAsFlag(
		&cfg.Repositories.AllowDirty,
		templateRenderCmd.Flags(),
	)

	templateRenderCmd.Flags().BoolVar(
		&opts.Force,
		"force",
		false,
		"Override non-empty directory",
	)

	return templateRenderCmd
}
