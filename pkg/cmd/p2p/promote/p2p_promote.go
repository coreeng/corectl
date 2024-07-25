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
	Streams            userio.IOStreams
	Exec               command.Commander
}

type imageOpts struct {
	ImageNameWithTag string
	Registry         string
	RepoPath         string
	AuthOverride     string
}

func NewP2PPromoteCmd() (*cobra.Command, error) {
	var opts = promoteOpts{
		Exec: command.NewCommand(),
	}
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

			return run(&opts)
		},
	}

	requiredFlagMap := map[string]*string{
		"source-registry": &opts.SourceRegistry,
		"source-stage":    &opts.SourceStage,
		"dest-registry":   &opts.DestRegistry,
		"dest-stage":      &opts.DestStage,
	}

	for name, field := range requiredFlagMap {
		err := addFlag(promoteCommand, field, name, true)
		if err != nil {
			return nil, err
		}
	}

	optionalFlagMap := map[string]*string{
		"source-auth-override": &opts.SourceAuthOverride,
		"dest-auth-override":   &opts.DestAuthOverride,
	}
	for name, field := range optionalFlagMap {
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
		logInfo("Configuring docker with gcloud")
		output, err := configureDockerWithGcloud(basePath(registry))
		if err != nil {
			return err
		}
		logInfo(string(output))
	}

	logInfo("Pulling image", imageUri(sourceImage))
	output, err := pullDockerImage(sourceImage)
	if err != nil {
		return err
	}
	logInfo(string(output))

	logInfo("Tagging image", imageUri(sourceImage), "with", imageUri(destinationImage))
	output, err = tagDockerImage(sourceImage, destinationImage)
	if err != nil {
		return err
	}
	logInfo(string(output))

	logInfo("Pushing image", imageUri(destinationImage))
	output, err = pushDockerImage(destinationImage)
	if err != nil {
		return err
	}
	logInfo(string(output))

	return nil
}

func basePath(registry string) string {
	return strings.Split(registry, "/")[0]
}
