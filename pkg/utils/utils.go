package utils

import (
	"slices"

	"github.com/coreeng/corectl/pkg/environment"
)

func FilterEnvs(nameFilter []string, envs []environment.Environment) []environment.Environment {
	var result []environment.Environment
	for _, env := range envs {
		if slices.Contains(nameFilter, string(env.Environment)) {
			result = append(result, env)
		}
	}
	return result
}
