package update

import (
	"archive/tar"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
	githubToken      string
	streams          userio.IOStreams
	targetVersion    string
	skipConfirmation bool
}

// Any failures we recieve will log a warning, we don't want this to cause any command to fail, this is an optional
// check which shouldn't prevent or interrupt any command from running (especially in ci)
func CheckForUpdates(cfg *config.Config, cmd *cobra.Command) {
	updateInterval := 1 * time.Hour
	updateStatusFileName := "corectl-autoupdate"
	log.Debug().Msg("checking for updates")
	nonInteractive, err := cmd.Flags().GetBool("non-interactive")
	if err != nil {
		log.Warn().Err(err).Msg("could not get non-interactive flag")
		return
	}

	if nonInteractive {
		log.Debug().Msg("skipping update check for --non-interactive command")
		return
	}

	tempDir := os.TempDir()
	tempFilePath := filepath.Join(tempDir, updateStatusFileName)
	file, err := os.OpenFile(tempFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Warn().Err(err).Msgf("could not open file to set update status %s", tempFilePath)
		return
	}
	defer file.Close()

	data, err := os.ReadFile(tempFilePath)
	if err != nil {
		log.Warn().Err(err).Msgf("could not read timestamp from update status file: %s", tempFilePath)
		return
	}

	now := time.Now()
	previousTimeString := string(data)
	previousTime, err := time.Parse(time.RFC3339, previousTimeString)
	// This is expected to fail on first run
	if err != nil {
		log.Debug().Err(err).Msgf("could not parse timestamp from update status file: %s previousTimeString: \"%s\"", tempFilePath, previousTimeString)
		// go's time.Sub only works with time.Time, not time.Duration
		previousTime = now.Add(-updateInterval)
	}

	timeSince := now.Sub(previousTime)
	if timeSince >= updateInterval {
		githubToken := ""
		if cfg.IsPersisted() {
			githubToken = cfg.GitHub.Token.Value
		}

		// Update the previousTime since we're checking
		file.WriteString(now.Format(time.RFC3339))
		file.Sync()

		update(UpdateOpts{
			githubToken:      githubToken,
			skipConfirmation: false,
			targetVersion:    "",
			streams: userio.NewIOStreamsWithInteractive(
				os.Stdin,
				os.Stdout,
				!nonInteractive,
			),
		})
	} else {
		timeLeft := (updateInterval - timeSince).Round(time.Second)
		log.Debug().Msgf("next update check will be in %s", timeLeft)
	}
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
	opts := UpdateOpts{
		githubToken:      "",
		skipConfirmation: false,
		targetVersion:    "",
	}
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update corectl",
		Long:  `Update to the latest corectl version.`,
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				opts.targetVersion = args[0]
			}
			if cfg.IsPersisted() {
				opts.githubToken = cfg.GitHub.Token.Value
			}

			nonInteractive, err := cmd.Flags().GetBool("non-interactive")
			if err != nil {
				log.Panic().Err(err).Msg("could not get non-interactive flag")
			}

			opts.streams = userio.NewIOStreamsWithInteractive(
				os.Stdin,
				os.Stdout,
				!nonInteractive,
			)

			update(opts)
		},
	}

	updateCmd.Flags().BoolVar(
		&opts.skipConfirmation,
		"skip-confirmation",
		false,
		"Auto approve confirmation prompt for update.",
	)

	return updateCmd
}

func update(opts UpdateOpts) {
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
		log.Warn().Err(err).Msg("could not render changelog markdown, falling back to plaintext")
		fmt.Print(asset.Changelog)
	}
	fmt.Print(out)

	wizard = opts.streams.Wizard("Confirming update", "Confirmation received")
	defer wizard.Done()

	if opts.skipConfirmation {
		wizard.Info("--skip-confirmation is set, continuing with update")
	} else {
		confirmation, err := confirmation.GetInput(opts.streams, fmt.Sprintf("Update to %s now?", asset.Version))
		if err != nil {
			log.Panic().Err(err).Msg("could not get confirmation from user")
		}

		if confirmation {
			wizard.Info("Update accepted")
		} else {
			wizard.Abort(fmt.Errorf("Update cancelled by user"))
			wizard.Info("Update cancelled")
			return
		}
	}

	wizard.SetTask(fmt.Sprintf("Downloading release %s", asset.Version), fmt.Sprintf("Downloaded release %s", asset.Version))
	data, err := git.DownloadCorectlAsset(asset)
	if err != nil {
		wizard.Abort(err)
		log.Panic().Err(err).Msgf("could not download release %s", asset.Version)
	}

	opts.streams.CurrentHandler.SetTask(fmt.Sprintf("Decompressing release %s", asset.Version), fmt.Sprintf("Decompressed release %s", asset.Version))
	var decompressed *tar.Reader
	decompressed, err = git.DecompressCorectlAssetInMemory(data)
	if err != nil {
		wizard.Abort(err)
		log.Panic().Err(err).Msgf("could not decompress release %s", asset.Version)
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
		log.Panic().Err(err).Msgf("could not write release %s to path %s", asset.Version, path)
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
		log.Panic().Err(err).Msgf("could not move file to partial path %s", path)
		return
	}
	err = os.Rename(partialPath, path)
	if err != nil {
		wizard.Abort(err)
		log.Panic().Err(err).Msgf("could not move file to path %s", path)
		return
	}
	log.Debug().Msgf("moved %s -> %s", tmpPath, path)
}
