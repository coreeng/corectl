package promote

import (
	"fmt"
	. "github.com/coreeng/corectl/pkg/command"
)

func configureDockerWithGcloud(registryBasePath string, command Commander) ([]byte, error) {
	return command.Execute("gcloud", WithArgs("auth", "configure-docker", "--quiet", registryBasePath))
}

func pushDockerImage(opts *imageOpts, command Commander) ([]byte, error) {
	return executeDocker("push", opts, command)
}

func executeDocker(op string, opts *imageOpts, command Commander) ([]byte, error) {
	envs := map[string]string{}
	if opts.AuthOverride != "" {
		envs["CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE"] = opts.AuthOverride
	}
	return command.Execute("docker", WithArgs(op, imageUri(opts)), WithEnv(envs))
}

func tagDockerImage(source *imageOpts, newTag *imageOpts, command Commander) ([]byte, error) {
	sourceImageUri := imageUri(source)
	tagImageUri := imageUri(newTag)
	return command.Execute("docker", WithArgs("tag", sourceImageUri, tagImageUri))
}

func pullDockerImage(opts *imageOpts, command Commander) ([]byte, error) {
	return executeDocker("pull", opts, command)
}

func imageUri(opts *imageOpts) string {
	return fmt.Sprintf("%s/%s/%s", opts.Registry, opts.RepoPath, opts.ImageNameWithTag)
}
