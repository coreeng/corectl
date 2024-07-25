package promote

import (
	"fmt"
	"os"
	"os/exec"
)

func configureDockerWithGcloud(registryBasePath string) ([]byte, error) {
	return exec.Command("gcloud", "auth", "configure-docker", "--quiet", registryBasePath).Output()
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
