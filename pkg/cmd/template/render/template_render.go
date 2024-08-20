package render

import (
	"fmt"
	"os"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type TemplateRenderOpts struct {
	IgnoreChecks  bool
	ParamsFile    string
	TemplateName  string
	TargetPath    string
	TemplatesPath string
}

func NewTemplateRenderCmd(cfg *config.Config) *cobra.Command {
	var opts = TemplateRenderOpts{}
	templateRenderCmd := &cobra.Command{
		Use:   "render <template-name> <target-path>",
		Short: "Render template locally",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.TemplateName = args[0]
			opts.TargetPath = args[1]
			opts.TemplatesPath = cfg.Repositories.Templates.Value

			if !opts.IgnoreChecks {
				if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.Templates); err != nil {
					return err
				}
			}

			return run(opts)
		},
	}

	templateRenderCmd.Flags().BoolVar(
		&opts.IgnoreChecks,
		"ignore-checks",
		false,
		"Ignore checks for uncommitted changes and branch status",
	)

	templateRenderCmd.Flags().StringVar(
		&opts.ParamsFile,
		"params-file",
		"",
		"Path to YAML file containing template parameters",
	)

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.Templates,
		templateRenderCmd.Flags(),
	)

	return templateRenderCmd
}

func run(opts TemplateRenderOpts) error {
	stat, err := os.Stat(opts.TargetPath)
	if err != nil {
		pathError := err.(*os.PathError)
		return fmt.Errorf("%s: %v", pathError.Path, pathError.Err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("%s: not a directory", opts.TargetPath)
	}

	templ, err := template.FindByName(opts.TemplatesPath, opts.TemplateName)
	if err != nil {
		return err
	}
	if templ == nil {
		return fmt.Errorf("%s: unknown template", opts.TemplateName)
	}

	arguments, err := parseParamsFileAndConvertToArguments(opts.ParamsFile)
	if err != nil {
		return err
	}

	fulfilledT := template.FulfilledTemplate{
		Spec:      templ,
		Arguments: arguments,
	}

	if err := template.Render(&fulfilledT, opts.TargetPath); err != nil {
		return err
	}

	return nil
}

func parseParamsFileAndConvertToArguments(paramsFile string) ([]template.Argument, error) {
	var params map[string]string
	var arguments []template.Argument

	if paramsFile != "" {
		fileContent, err := os.ReadFile(paramsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read params file: %w", err)
		}
		if err := yaml.Unmarshal(fileContent, &params); err != nil {
			return nil, fmt.Errorf("failed to parse params file: %w", err)
		}
	}

	for key, value := range params {
		arguments = append(arguments, template.Argument{
			Name:  key,
			Value: value,
		})
	}

	return arguments, nil
}
