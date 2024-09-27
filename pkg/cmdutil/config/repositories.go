package config

import (
	"errors"
	"fmt"

	"github.com/coreeng/corectl/pkg/git"
)

func ResetConfigRepositoryState(repositoryParam *Parameter[string], dryRun bool) (*git.LocalRepository, error) {
	if repositoryParam.Value == "" {
		return nil, fmt.Errorf("%s path is not set. consider initializing corectl first:\n  corectl config init", repositoryParam.name)
	}
	repo, err := git.OpenAndResetRepositoryState(repositoryParam.Value, dryRun)
	if errors.Is(err, git.ErrLocalChangesIsPresent) {
		return nil, fmt.Errorf("local changes are present in repo on path %s. consider removing it before using corectl", repositoryParam.Value)
	} else if err != nil {
		return nil, fmt.Errorf("couldn't reset state for %s: %v. path: %s", repositoryParam.name, err, repositoryParam.Value)
	}
	return repo, nil
}
