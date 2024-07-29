package promote

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/command"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

type promoteOpts struct {
	ImageWithTag       string
	SourceRegistry     string
	SourceStage        string
	SourceAuthOverride string
	DestRegistry       string
	DestStage          string
	DestAuthOverride   string
	Exec               command.Commander
	Streams            userio.IOStreams
}

type imageOpts struct {
	ImageNameWithTag string
	Registry         string
	RepoPath         string
	AuthOverride     string
}

func NewP2PPromoteCmd() (*cobra.Command, error) {
	var opts = promoteOpts{}
	var promoteCommand = &cobra.Command{
		Use:   "promote <image_with_tag>",
		Short: "Promotes image from source to destination registry. Only GCP is supported for now",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ImageWithTag = args[0]
			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
			)
			opts.Exec = command.NewCommand(
				command.WithStdout(cmd.OutOrStdout()),
			)
			return run(&opts)
		},
	}

	requiredFlags := map[string]*string{
		"source-registry": &opts.SourceRegistry,
		"source-stage":    &opts.SourceStage,
		"dest-registry":   &opts.DestRegistry,
		"dest-stage":      &opts.DestStage,
	}

	for name, field := range requiredFlags {
		err := addFlag(promoteCommand, field, name, true)
		if err != nil {
			return nil, err
		}
	}

	optionalFlags := map[string]*string{
		"source-auth-override": &opts.SourceAuthOverride,
		"dest-auth-override":   &opts.DestAuthOverride,
	}
	for name, field := range optionalFlags {
		err := addFlag(promoteCommand, field, name, false)
		if err != nil {
			return nil, err
		}
	}

	return promoteCommand, nil
}

func addFlag(promoteCommand *cobra.Command, field *string, name string, required bool) error {
	envVariableName := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))

	description := fmt.Sprintf("optional, defaults to environment variable: %s", envVariableName)
	if required {
		description = fmt.Sprintf("required, defaults to environment variable: %s", envVariableName)
	}

	promoteCommand.Flags().StringVar(
		field,
		name,
		"",
		description,
	)

	envVariableValue := os.Getenv(envVariableName)

	if envVariableValue != "" {
		*field = envVariableValue
	}

	if envVariableValue == "" && required {
		if err := promoteCommand.MarkFlagRequired(name); err != nil {
			return err
		}
	}
	return nil
}

func run(opts *promoteOpts) error {
	if err := validate(opts); err != nil {
		return err
	}

	sourceImage := &imageOpts{
		ImageNameWithTag: opts.ImageWithTag,
		Registry:         opts.SourceRegistry,
		RepoPath:         opts.SourceStage,
		AuthOverride:     opts.SourceAuthOverride,
	}

	destinationImage := &imageOpts{
		ImageNameWithTag: opts.ImageWithTag,
		Registry:         opts.DestRegistry,
		RepoPath:         opts.DestStage,
		AuthOverride:     opts.DestAuthOverride,
	}

	logInfo := opts.Streams.Info

	for _, registry := range []string{opts.SourceRegistry, opts.DestRegistry} {
		logInfo("Configuring docker for registry: ", registry)
		_, err := configureDockerWithGcloud(basePath(registry), opts.Exec)
		if err != nil {
			return err
		}
	}

	logInfo("Pulling image", imageUri(sourceImage))
	_, err := pullDockerImage(sourceImage, opts.Exec)
	if err != nil {
		return err
	}

	logInfo("Tagging image", imageUri(sourceImage), "with", imageUri(destinationImage))
	_, err = tagDockerImage(sourceImage, destinationImage, opts.Exec)
	if err != nil {
		return err
	}

	logInfo("Pushing image", imageUri(destinationImage))
	_, err = pushDockerImage(destinationImage, opts.Exec)
	if err != nil {
		return err
	}

	return nil
}

func basePath(registry string) string {
	return strings.Split(registry, "/")[0]
}
