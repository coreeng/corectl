package promote

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"strings"
)

type promoteOpts struct {
	ImageWithTag       string
	SourceRegistry     string
	SourceRepoPath     string
	SourceAuthOverride string
	DestRegistry       string
	DestRepoPath       string
	DestAuthOverride   string
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
		Short: "Promotes image",
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
		"source-registry":  &opts.SourceRegistry,
		"source-repo-path": &opts.SourceRepoPath,
		"dest-registry":    &opts.DestRegistry,
		"dest-repo-path":   &opts.DestRepoPath,
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

	sourceImage := &imageOpts{
		ImageNameWithTag: opts.ImageWithTag,
		Registry:         opts.SourceRegistry,
		RepoPath:         opts.SourceRepoPath,
		AuthOverride:     opts.SourceAuthOverride,
	}

	destinationImage := &imageOpts{
		ImageNameWithTag: opts.ImageWithTag,
		Registry:         opts.DestRegistry,
		RepoPath:         opts.DestRepoPath,
		AuthOverride:     opts.DestAuthOverride,
	}

	logInfo := opts.Streams.Info
	logInfo("Configuring docker with gcloud")
	output, err := configureDockerWithGcloud()
	if err != nil {
		return err
	}
	logInfo(string(output))

	logInfo("Pulling image", imageUri(sourceImage))
	output, err = pullDockerImage(sourceImage)
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

func configureDockerWithGcloud() ([]byte, error) {
	return exec.Command("gcloud", "auth", "configure-docker", "--quiet", "europe-west2-docker.pkg.dev").Output()
}

func pushDockerImage(opts *imageOpts) ([]byte, error) {
	imageUri := imageUri(opts)
	command := exec.Command("docker", "push", imageUri)
	if opts.AuthOverride != "" {
		command.Env = append(os.Environ(), fmt.Sprintf("CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=%s", opts.AuthOverride))
	}
	return command.Output()
}

func tagDockerImage(source *imageOpts, newTag *imageOpts) ([]byte, error) {
	sourceImageUri := imageUri(source)
	tagImageUri := imageUri(newTag)
	return exec.Command("docker", "tag", sourceImageUri, tagImageUri).Output()
}

func pullDockerImage(opts *imageOpts) ([]byte, error) {
	imageUri := imageUri(opts)
	command := exec.Command("docker", "pull", imageUri)
	if opts.AuthOverride != "" {
		command.Env = append(os.Environ(), fmt.Sprintf("CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=%s", opts.AuthOverride))
	}
	return command.Output()
}

func imageUri(opts *imageOpts) string {
	return fmt.Sprintf("%s/%s/%s", opts.Registry, opts.RepoPath, opts.ImageNameWithTag)
}
