package promote

import (
	"fmt"
	. "github.com/coreeng/corectl/pkg/command"
	"io"
	"strings"
)

type Validator struct {
	FileSystem FileSystem
	Commander  Commander
}

func (v *Validator) validate(opts *promoteOpts) error {
	if err := DepsInstalled(opts.Exec, "gcloud"); err != nil {
		return err
	}

	for _, registry := range []string{opts.SourceRegistry, opts.DestRegistry} {
		if !isRegistrySupported(registry) {
			return fmt.Errorf("only GCP registries are supported, got %s", registry)
		}
	}

	if err := v.validateRegistryAccess(opts.SourceRegistry, opts.SourceAuthOverride, "source"); err != nil {
		return err
	}

	if err := v.validateRegistryAccess(opts.DestRegistry, opts.DestAuthOverride, "destination"); err != nil {
		return err
	}

	return nil
}

func (v *Validator) validateRegistryAccess(registry string, credFile string, target string) error {
	envs := make(map[string]string)
	if credFile != "" {
		if err := v.checkCredFile(credFile); err != nil {
			return fmt.Errorf("error accessing %s credential file: %s", target, err)
		}
		envs["CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE"] = credFile
	}

	_, err := v.Commander.Execute("gcloud",
		WithArgs("artifacts", "docker", "images", "list", registry, "--limit=1"),
		WithEnv(envs),
		WithOverrideStdout(io.Discard),
	)

	if err != nil {
		return fmt.Errorf("error accessing %s registry: %s", target, err)
	}

	return nil
}

func (v *Validator) checkCredFile(credFile string) error {
	if _, err := v.FileSystem.Stat(credFile); err != nil {
		return err
	}
	return nil
}

func isRegistrySupported(registry string) bool {
	for _, suffix := range []string{
		"-docker.pkg.dev",
		".gcr.io",
	} {
		if strings.HasSuffix(basePath(registry), suffix) {
			return true
		}
	}
	return false
}
