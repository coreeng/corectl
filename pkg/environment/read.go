package environment

import (
	"errors"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
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
