package config

import (
	"errors"
	"fmt"

	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/logger"
)

func Update(githubToken string, streams userio.IOStreams, ignoreDirty bool, repoParams []Parameter[string]) error {
	gitAuth := git.UrlTokenAuthMethod(githubToken)

	for _, repoParam := range repoParams {
		err := updateRepository(&repoParam, gitAuth, streams, ignoreDirty)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateRepository(repoParam *Parameter[string], gitAuth git.AuthMethod, streams userio.IOStreams, ignoreDirty bool) error {
	isUpdated, err := func() (bool, error) {

		logger.Info().Msgf("Updating %s", repoParam.Name())
		defer logger.Info().Msgf("Updated %s", repoParam.Name())

		repo, err := resetConfigRepositoryState(repoParam, ignoreDirty)
		if err != nil {
			return false, err
		}
		pullResult, err := repo.Pull(gitAuth)
		if err != nil {
			return false, fmt.Errorf("couldn't pull changes for %s: %v", repoParam.Name(), err)
		}
		return pullResult.IsUpdated, nil
	}()
	if err != nil {
		return err
	}

	var msg string
	if isUpdated {
		msg = fmt.Sprintf("%s is updated succesfully!", repoParam.Name())
	} else {
		msg = fmt.Sprintf("%s is up to date!", repoParam.Name())
	}
	logger.Info().Msg(msg)
	return nil
}

func resetConfigRepositoryState(repositoryParam *Parameter[string], ignoreDirty bool) (*git.LocalRepository, error) {
	if repositoryParam.Value == "" {
		return nil, fmt.Errorf("%s path is not set. consider initializing corectl first:\n  corectl config init", repositoryParam.name)
	}
	repo, err := git.OpenAndResetRepositoryState(repositoryParam.Value, false)
	if errors.Is(err, git.ErrLocalChangesIsPresent) {
		if !ignoreDirty {
			return nil, fmt.Errorf("local changes are present in repo on path %s. consider removing it before using corectl", repositoryParam.Value)
		}
	} else if err != nil {
		return nil, fmt.Errorf("couldn't reset state for %s: %v. path: %s", repositoryParam.name, err, repositoryParam.Value)
	}
	return repo, nil
}
