package render

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/spf13/cobra"
	"os"
)

type TemplateRenderOpts struct {
	Streams userio.IOStreams

	IgnoreChecks  bool
	ArgsFile      string
	Args          []string
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
			opts.Streams = userio.NewIOStreams(cmd.InOrStdin(), cmd.OutOrStdout())
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
		&opts.ArgsFile,
		"args-file",
		"",
		"Path to YAML file containing template arguments",
	)
	templateRenderCmd.Flags().StringSliceVarP(
		&opts.Args,
		"arg",
		"a",
		[]string{},
		"Template argument in the format: <arg-name>=<arg-value>",
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

	args, err := CollectArgsFromAllSources(
		templ,
		opts.ArgsFile,
		opts.Args,
		opts.Streams,
		[]template.Argument{},
	)
	if err != nil {
		return err
	}

	fulfilledT := template.FulfilledTemplate{
		Spec:      templ,
		Arguments: args,
	}
	if err := template.Render(&fulfilledT, opts.TargetPath); err != nil {
		return err
	}

	return nil
}
