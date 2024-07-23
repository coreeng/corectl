package promote

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/spf13/cobra"
	"os/exec"
)

type PromoteOpts struct {
	Tenant       string
	ImageWithTag string
	FromStage    string
	Streams      userio.IOStreams
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
	promoteCommand.Flags().StringVar(
		&opts.Tenant,
		"tenant",
		"",
		"Tenant name",
	)

	if err := promoteCommand.MarkFlagRequired("tenant"); err != nil {
		return nil, err
	}

	promoteCommand.Flags().StringVar(
		&opts.FromStage,
		"fromStage",
		"",
		"Stage to promote from",
	)

	if err := promoteCommand.MarkFlagRequired("fromStage"); err != nil {
		return nil, err
	}

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.CPlatform,
		promoteCommand.Flags())

	return promoteCommand, nil
}

type ImageOpts struct {
	ImageNameWithTag string
	GCPRegion        string
	GCPProjectId     string
	Tenant           string
	Stage            string
}

func run(opts *PromoteOpts, cfg *config.Config) error {
	//todo lkan; implement actual promote
	//spinnerHandler := opts.Streams.Spinner("Spinning up promotion...")
	//defer spinnerHandler.Done()

	//stage, err := GetNextStage(opts.FromStage)
	//if err != nil {
	//	return err
	//}

	sourceImage := &ImageOpts{
		ImageNameWithTag: opts.ImageWithTag,
		GCPRegion:        "europe-west2",
		GCPProjectId:     "core-platform-efb3c84c",
		Tenant:           opts.Tenant,
		Stage:            opts.FromStage,
	}

	destinationStage, err := GetNextStage(opts.FromStage)
	if err != nil {
		return err
	}

	destinationGCPProjectId := GetGCPProjectId(destinationStage)

	destinationImage := &ImageOpts{
		ImageNameWithTag: opts.ImageWithTag,
		GCPRegion:        "europe-west2",
		GCPProjectId:     destinationGCPProjectId,
		Tenant:           opts.Tenant,
		Stage:            destinationStage,
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

func PushDockerImage(opts *ImageOpts) ([]byte, error) {
	imageUri := imageUri(opts)
	fmt.Printf("Pushing image %s\n", imageUri)
	output, err := exec.Command("docker", "push", imageUri).Output()
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
	output, err := exec.Command("docker", "pull", imageUri).Output()
	fmt.Printf("%s", output)
	return output, err
}

func imageUri(opts *ImageOpts) string {
	return fmt.Sprintf("%s-docker.pkg.dev/%s/tenant/%s/%s/%s", opts.GCPRegion, opts.GCPProjectId, opts.Tenant, opts.Stage, opts.ImageNameWithTag)
}
