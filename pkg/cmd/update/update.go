package update

import (
	"archive/tar"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/glamour"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/shell"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/cmdutil/userio/confirmation"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/version"
	"github.com/google/go-github/v59/github"
	"github.com/phuslu/log"
	"github.com/spf13/cobra"
)

type UpdateOpts struct {
	githubToken   string
	interactive   bool
	streams       userio.IOStreams
	targetVersion string
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
			interactive := true
			targetVersion := ""
			if len(args) > 0 {
				targetVersion = args[0]
			}
			githubToken := ""
			if cfg.IsPersisted() {
				githubToken = cfg.GitHub.Token.Value
			}

			opts := UpdateOpts{
				targetVersion: targetVersion,
				interactive:   interactive,
				githubToken:   githubToken,
				streams: userio.NewIOStreamsWithInteractive(
					os.Stdin,
					os.Stdout,
					interactive,
				),
			}

			Update(opts)
		},
	}
}

func Update(opts UpdateOpts) {
	if opts.targetVersion != "" {
		log.Debug().Msgf("target version set to %s", opts.targetVersion)
	}

	wizard := opts.streams.Wizard("Checking for updates", "Retrieved version metadata")
	githubClient := github.NewClient(nil)
	if opts.githubToken != "" {
		githubClient = githubClient.WithAuthToken(opts.githubToken)
	}

	var release *github.RepositoryRelease
	var err error
	if opts.targetVersion == "" {
		release, err = git.GetLatestCorectlRelease(githubClient)
	} else {
		release, err = git.GetCorectlReleaseByTag(githubClient, opts.targetVersion)
	}
	if err != nil {
		wizard.Abort(err)
		log.Panic().Err(err)
	}

	asset, err := git.GetLatestCorectlAsset(release)
	if err != nil {
		wizard.Abort(err)
		log.Panic().Err(err)
	}
	log.Debug().Str("current_version", version.Version).Str("remote_version", asset.Version).Msg("comparing versions")
	if version.Version == asset.Version {
		wizard.SetCurrentTaskCompletedTitle(fmt.Sprintf("Already running %s release (%v)", opts.targetVersion, version.Version))
		wizard.Done()
		return
	} else {
		wizard.SetCurrentTaskCompletedTitle(fmt.Sprintf("Update available: %v", asset.Version))
		wizard.Done()
	}

	out, err := glamour.Render(asset.Changelog, "dark")
	if err != nil {
		wizard.Abort(err)
		log.Warn().Err(err).Msg("Could not render changelog markdown, falling back to plaintext")
		fmt.Print(asset.Changelog)
	}
	fmt.Print(out)

	confirmation, err := confirmation.GetInput(opts.streams, fmt.Sprintf("Update to %s now?", asset.Version))
	if err != nil {
		log.Panic().Err(err).Msg("Could not get confirmation from user")
	}

	if !confirmation {
		log.Info().Msg("Update cancelled, exiting")
		return
	}

	wizard = opts.streams.Wizard(fmt.Sprintf("Downloading release %s", asset.Version), fmt.Sprintf("Downloaded release %s", asset.Version))
	data, err := git.DownloadCorectlAsset(asset)
	if err != nil {
		wizard.Abort(err)
		log.Panic().Err(err).Msgf("Could not download release %s", asset.Version)
	}

	opts.streams.CurrentHandler.SetTask(fmt.Sprintf("Decompressing release %s", asset.Version), fmt.Sprintf("Decompressed release %s", asset.Version))
	var decompressed *tar.Reader
	decompressed, err = git.DecompressCorectlAssetInMemory(data)
	if err != nil {
		wizard.Abort(err)
		log.Panic().Err(err).Msgf("Could not decompress release %s", asset.Version)
	}

	path := getCurrentExecutablePath()

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("corectl-%s-", asset.Version))
	if err != nil {
		log.Warn().Msg("unable to create temporary file")
	}
	tmpPath, err := os.Readlink(fmt.Sprintf("/proc/self/fd/%d", tmpFile.Fd()))
	if err != nil {
		log.Warn().Msgf("unable to read link of file descriptor %d", tmpFile.Fd())
	}

	opts.streams.CurrentHandler.SetTask(fmt.Sprintf("Installing release %s to path: %s", asset.Version, path), fmt.Sprintf("Release %s installed", asset.Version))
	err = git.WriteCorectlAssetToPath(decompressed, tmpPath, tmpFile)
	if err != nil {
		wizard.Abort(err)
		log.Panic().Err(err).Msgf("Could not write release %s to path %s", asset.Version, path)
	}
	err = tmpFile.Close()
	if err != nil {
		log.Warn().Msg("could not close temporary file")
	}

	partialPath := path + ".partial"

	// NOTE: os.Rename is the only way to overwrite an existing executable, but this doesn't work across
	// filesystems. Usually /tmp is set up as a separate filesystem, therefore we must copy and then remove to
	// simulate the rename
	err = shell.MoveFile(tmpPath, partialPath)
	if err != nil {
		wizard.Abort(err)
		log.Panic().Err(err).Msgf("Could not move file to partial path %s", path)
		return
	}
	err = os.Rename(partialPath, path)
	if err != nil {
		wizard.Abort(err)
		log.Panic().Err(err).Msgf("Could not move file to path %s", path)
		return
	}
	log.Debug().Msgf("moved %s -> %s", tmpPath, path)

	wizard.Done()
}
