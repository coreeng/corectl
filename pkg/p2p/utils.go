package p2p

import (
	"slices"

	"github.com/coreeng/corectl/pkg/environment"
)

func filterEnvs(nameFilter []string, envs []environment.Environment) []environment.Environment {
	var result []environment.Environment
	for _, env := range envs {
		if slices.Contains(nameFilter, string(env.Environment)) {
			result = append(result, env)
		}
	}
	return result
}
