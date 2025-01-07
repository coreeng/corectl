package update_repositories

import (
	"fmt"
	"path/filepath"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/git"
	// "github.com/coreeng/corectl/pkg/logger"
	// "go.uber.org/zap"
)

func UpdateRepositories(config *config.Config) error {
	githubToken := config.GitHub.Token.Value
	gitAuth := git.UrlTokenAuthMethod(githubToken)
	
	cplatformRepositoryFullname, err := git.DeriveRepositoryFullnameFromUrl(config.Repositories.CPlatform.Value)
	if err != nil {
		fmt.Println("Error deriving cplatform repository fullname:\n", err)
		return err
	}
	templatesRepositoryFullname, err := git.DeriveRepositoryFullnameFromUrl(config.Repositories.Templates.Value)
	if err != nil {
		fmt.Println("Error deriving templates repository fullname:\n", err)
		return err
	}
	cPlatformRepoPath := filepath.Join(config.ConfigPaths.Directory.Value, cplatformRepositoryFullname.Name())
	templatesRepoPath := filepath.Join(config.ConfigPaths.Directory.Value, templatesRepositoryFullname.Name())
	
	localCplatformRepo, err := git.OpenAndResetRepositoryState(cPlatformRepoPath, false)
	if err != nil {
		fmt.Println("Error resetting cplatform repo:\n", err)
		// maybe it doesn't exist...
		if (err != nil) { // todo: replace with the right kind of error, "local repository does not exist" or something like that
			cplatformCloneOp := git.CloneOp{
				URL: cplatformRepositoryFullname.HttpUrl(),
				TargetPath: cPlatformRepoPath,
				Auth: gitAuth,
			}
			_, err := git.CloneToLocalRepository(cplatformCloneOp)
			if err != nil {
				fmt.Println("Error cloning cplatform repo:\n", err)
				return err
			}
			return nil
		}
	}

	localTemplatesRepo, err := git.OpenAndResetRepositoryState(templatesRepoPath, false)
	if err != nil {
		fmt.Println("Error resetting templates repo:\n", err)
		// maybe it doesn't exist...
		if (err != nil) { // todo: replace with the right kind of error, "local repository does not exist" or something like that
			templatesCloneOp := git.CloneOp{
				URL: templatesRepositoryFullname.HttpUrl(),
				TargetPath: templatesRepoPath,
				Auth: gitAuth,
			}
			_, err := git.CloneToLocalRepository(templatesCloneOp)
			if err != nil {
				fmt.Println("Error cloning templates repo:\n", err)
				return err
			}
			return nil
		}
	}
	fmt.Println("Repositories are up to date.", localCplatformRepo, localTemplatesRepo)
	return nil
}
