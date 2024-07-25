package promote

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/command"
	"strings"
)

var validRegistriesSuffixes = []string{
	"-docker.pkg.dev",
	".gcr.io",
}

func validate(opts *promoteOpts) error {
	if err := command.DepsInstalled(opts.Exec, "gcloud"); err != nil {
		return err
	}

	for _, registry := range []string{opts.SourceRegistry, opts.DestRegistry} {
		if !isRegistrySupported(registry) {
			return fmt.Errorf("only GCP registries are supported, got %s", registry)
		}
	}

	return nil
}

func isRegistrySupported(registry string) bool {
	for _, suffix := range validRegistriesSuffixes {
		if strings.HasSuffix(basePath(registry), suffix) {
			return true
		}
	}
	return false
}
