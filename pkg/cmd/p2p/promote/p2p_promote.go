package promote

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
)

type PromoteOpts struct {
	ImageWithTag       string
	SourceRegistry     string
	SourceRepoPath     string
	SourceAuthOverride string
	DestRegistry       string
	DestRepoPath       string
	DestAuthOverride   string
	Streams            userio.IOStreams
}

func NewP2PPromoteCmd(cfg *config.Config) (*cobra.Command, error) {
	var opts = PromoteOpts{}
	var promoteCommand = &cobra.Command{
		Use:   "promote <image_with_tag>",
		Short: "Promote image",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ImageWithTag = args[0]

			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
			)

			fmt.Printf("promote %v\n", opts)
			return run(&opts, cfg)
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
	promoteCommand.Flags().StringVar(
		field,
		name,
		"",
		name,
	)
	if required {
		if err := promoteCommand.MarkFlagRequired(name); err != nil {
			return err
		}
	}
	return nil
}

type ImageOpts struct {
	ImageNameWithTag string
	Registry         string
	RepoPath         string
	AuthOverride     string
}

func run(opts *PromoteOpts, cfg *config.Config) error {

	sourceImage := &ImageOpts{
		ImageNameWithTag: opts.ImageWithTag,
		Registry:         opts.SourceRegistry,
		RepoPath:         opts.SourceRepoPath,
		AuthOverride:     opts.SourceAuthOverride,
	}

	destinationImage := &ImageOpts{
		ImageNameWithTag: opts.ImageWithTag,
		Registry:         opts.DestRegistry,
		RepoPath:         opts.DestRepoPath,
		AuthOverride:     opts.DestAuthOverride,
	}
	_, err := ConfigureDockerWithGcloud()
	if err != nil {
		return err
	}

	_, err = PullDockerImage(sourceImage)
	if err != nil {
		return err
	}

	_, err = TagDockerImage(sourceImage, destinationImage)
	if err != nil {
		return err
	}

	_, err = PushDockerImage(destinationImage)
	if err != nil {
		return err
	}

	return nil
}

func ConfigureDockerWithGcloud() ([]byte, error) {
	fmt.Printf("Configuring docker with gcloud\n")
	output, err := exec.Command("gcloud", "auth", "configure-docker", "--quiet", "europe-west2-docker.pkg.dev").Output()
	fmt.Printf("%s", output)
	return output, err
}

func PushDockerImage(opts *ImageOpts) ([]byte, error) {
	imageUri := imageUri(opts)
	fmt.Printf("Pushing image %s\n", imageUri)
	command := exec.Command("docker", "push", imageUri)
	if opts.AuthOverride != "" {
		command.Env = append(os.Environ(), fmt.Sprintf("CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=%s", opts.AuthOverride))
	}
	output, err := command.Output()
	fmt.Printf("%s", output)
	return output, err
}

func GetGCPProjectId(stage string) string {
	//todo lkan; implement
	return "core-platform-efb3c84c"
}

func TagDockerImage(source *ImageOpts, newTag *ImageOpts) (interface{}, error) {
	sourceImageUri := imageUri(source)
	tagImageUri := imageUri(newTag)
	fmt.Printf("tagging image %s with %s\n ", sourceImageUri, tagImageUri)
	output, err := exec.Command("docker", "tag", sourceImageUri, tagImageUri).Output()
	fmt.Printf("%s", output)
	return output, err
}

func GetNextStage(stage string) (string, error) {
	switch stage {
	case "fast-feedback":
		return "extended", nil
	case "extended":
		return "prod", nil
	default:
		return "", fmt.Errorf("missing next stage for: %s", stage)
	}
}

func PullDockerImage(opts *ImageOpts) ([]byte, error) {
	imageUri := imageUri(opts)
	fmt.Printf("pulling image %s\n", imageUri)
	command := exec.Command("docker", "pull", imageUri)
	if opts.AuthOverride != "" {
		command.Env = append(os.Environ(), fmt.Sprintf("CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=%s", opts.AuthOverride))
	}
	output, err := command.Output()
	fmt.Printf("%s", output)
	return output, err
}

func imageUri(opts *ImageOpts) string {
	//return fmt.Sprintf("%s-docker.pkg.dev/%s/tenant/%s/%s/%s", opts.GCPRegion, opts.GCPProjectId, opts.Tenant, opts.Stage, opts.ImageNameWithTag)
	return fmt.Sprintf("%s/%s/%s", opts.Registry, opts.RepoPath, opts.ImageNameWithTag)
}
