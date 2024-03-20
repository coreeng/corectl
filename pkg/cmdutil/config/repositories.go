package config

import (
	"errors"
	"fmt"
	"github.com/coreeng/corectl/pkg/git"
)

func ResetConfigRepositoryState(repositoryParam *Parameter[string]) (*git.LocalRepository, error) {
	if repositoryParam.Value == "" {
		return nil, fmt.Errorf("%s path is not set. consider initializing corectl first:\n  corectl config init", repositoryParam.name)
	}
	repo, err := git.OpenAndResetRepositoryState(repositoryParam.Value)
	if errors.Is(err, git.ErrLocalChangesIsPresent) {
		return nil, fmt.Errorf("local changes are present in %s. consider removing it before using corectl", repositoryParam.name)
	} else if err != nil {
		return nil, fmt.Errorf("couldn't reset state for %s: %v", repositoryParam.name, err)
	}
	return repo, nil
}
