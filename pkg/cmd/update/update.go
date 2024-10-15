package update

import (
	"archive/tar"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/glamour"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/version"
	"github.com/google/go-github/v59/github"
	"github.com/phuslu/log"
	"github.com/spf13/cobra"
)

type UpdateOpts struct {
	Streams        userio.IOStreams
	NonInteractive bool
}

func getCurrentExecutablePath() string {
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get executable path")
	}

	absolutePath, err := filepath.Abs(execPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get absolute executable path")
	}

	return absolutePath
}

func UpdateCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update corectl",
		Long:  `Update to the latest corectl version.`,
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			targetVersion := "latest"
			if len(args) > 0 {
				targetVersion = args[0]
				log.Debug().Msgf("target version set to %s", targetVersion)
			}

			opts := UpdateOpts{}
			opts.Streams = userio.NewIOStreamsWithInteractive(
				os.Stdin,
				os.Stdout,
				!opts.NonInteractive,
			)
			wizard := opts.Streams.Wizard("Checking for updates", "Retrieved version metadata")
			githubClient := github.NewClient(nil)
			if cfg.IsPersisted() {
				githubClient = githubClient.
					WithAuthToken(cfg.GitHub.Token.Value)
			}

			var release *github.RepositoryRelease
			var err error
			if targetVersion == "latest" {
				release, err = git.GetLatestCorectlRelease(githubClient)
			} else {
				release, err = git.GetCorectlReleaseByTag(githubClient, targetVersion)
			}
			if err != nil {
				log.Panic().Err(err)
			}

			asset, err := git.GetLatestCorectlAsset(release)
			if err != nil {
				log.Panic().Err(err)
			}
			log.Debug().Str("current_version", version.Version).Str("remote_version", asset.Version).Msg("comparing versions")
			if version.Version == asset.Version {
				wizard.SetCurrentTaskCompletedTitle(fmt.Sprintf("Already running %s release (%v)", targetVersion, version.Version))
				wizard.Done()
				return
			} else {
				wizard.SetCurrentTaskCompletedTitle(fmt.Sprintf("Update available: %v", asset.Version))
				wizard.Done()
			}

			out, err := glamour.Render(asset.Changelog, "dark")
			if err != nil {
				wizard.Abort(err)
				log.Panic().Err(err).Msg("Could not render changelog")
			}
			fmt.Print(out)
			wizard = opts.Streams.Wizard(fmt.Sprintf("Downloading release %s", asset.Version), fmt.Sprintf("Downloaded release %s", asset.Version))
			data, err := git.DownloadCorectlAsset(asset)
			if err != nil {
				wizard.Abort(err)
				log.Panic().Err(err).Msgf("Could not download release %s", asset.Version)
			}

			opts.Streams.CurrentHandler.SetTask(fmt.Sprintf("Decompressing release %s", asset.Version), fmt.Sprintf("Decompressed release %s", asset.Version))
			var decompressed *tar.Reader
			decompressed, err = git.DecompressCorectlAssetInMemory(data)
			if err != nil {
				wizard.Abort(err)
				log.Panic().Err(err).Msgf("Could not decompress release %s", asset.Version)
			}

			path := getCurrentExecutablePath()
			tmpPath := fmt.Sprintf("%s.partial", path)
			opts.Streams.CurrentHandler.SetTask(fmt.Sprintf("Installing release %s to path: %s", asset.Version, path), fmt.Sprintf("Release %s installed", asset.Version))
			err = git.WriteCorectlAssetToPath(decompressed, tmpPath)
			if err != nil {
				wizard.Abort(err)
				log.Panic().Err(err).Msgf("Could not write release %s to path %s", asset.Version, path)
			}

			err = os.Rename(tmpPath, path)
			if err != nil {
				wizard.Abort(err)
				log.Panic().Err(err).Msgf("Could not move file to path %s", path)
				return
			}
			log.Debug().Msgf("moved %s -> %s", tmpPath, path)

			wizard.Done()
		},
	}
}
