package environment

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func List(configRepoPath string) ([]Environment, error) {
	envAbsPath := filepath.Join(configRepoPath, environmentsRelativePath)
	dir, err := os.ReadDir(envAbsPath)
	if err != nil {
		return nil, err
	}
	var envs []Environment
	for _, dirEntry := range dir {
		if !dirEntry.IsDir() {
			continue
		}
		envConfigPath := filepath.Join(envAbsPath, dirEntry.Name(), "config.yaml")
		envBytes, err := os.ReadFile(envConfigPath)
		if err != nil && errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, err
		}
		var env Environment
		if err := yaml.Unmarshal(envBytes, &env); err != nil {
			return nil, err
		}
		envs = append(envs, env)
	}
	return envs, nil
}

func GetEnvironmentByName(configRepoPath string, envName string) (Environment, error) {
	var env Environment
	var environmentFound = false
	environments, err := List(configRepoPath)
	if err != nil {
		return env, err
	}

	for _, env = range environments {
		if string(env.Environment) == envName {
			environmentFound = true
			break
		}
	}
	if !environmentFound {
		return env, fmt.Errorf("unable to find matching environment %s, check your config.yaml", envName)
	}
	return env, nil
}
