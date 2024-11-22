package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/cmdutil/userio/confirmation"
	"github.com/coreeng/corectl/pkg/logger"
	"github.com/coreeng/corectl/pkg/version"
	"github.com/google/go-github/v59/github"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const CmdName = "update"

type UpdateOpts struct {
	githubToken      string
	streams          userio.IOStreams
	targetVersion    string
	skipConfirmation bool
}

type CoreCtlAsset struct {
	Version   string
	Url       string
	Changelog string
}

// Any failures we recieve will log a warning, we don't want this to cause any command to fail, this is an optional
// check which shouldn't prevent or interrupt any command from running (especially in ci)
func CheckForUpdates(cfg *config.Config, cmd *cobra.Command) {
	updateInterval := 1 * time.Hour
	updateStatusFileName := "corectl-autoupdate"
	logger.Debug("checking for updates")

	nonInteractive, err := cmd.Flags().GetBool("non-interactive")
	if err != nil {
		logger.Warn("could not get non-interactive flag",
			zap.Error(err),
		)
		return
	}

	if !userio.IsTerminalInteractive(os.Stdin, os.Stdout) {
		// Override this setting if the terminal itself is not capable of interactivity
		nonInteractive = true
	}

	if nonInteractive {
		logger.Debug("skipping update check for --non-interactive command")
		return
	}

	tempDir := os.TempDir()
	tempFilePath := filepath.Join(tempDir, updateStatusFileName)
	file, err := os.OpenFile(tempFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		logger.Warn("could not open file to set update status",
			zap.String("file", tempFilePath),
			zap.Error(err),
		)
		return
	}
	defer file.Close()

	data, err := os.ReadFile(tempFilePath)
	if err != nil {
		logger.Warn("could not read timestamp from update status file",
			zap.String("file", tempFilePath),
			zap.Error(err),
		)
		return
	}

	now := time.Now()
	previousTimeString := string(data)
	previousTime, err := time.Parse(time.RFC3339, previousTimeString)
	// This is expected to fail on first run
	if err != nil {
		logger.Debug("could not parse timestamp from update status file",
			zap.String("file", tempFilePath),
			zap.String("previousTimeString", previousTimeString),
			zap.Error(err),
		)
		// go's time.Sub only works with time.Time, not time.Duration
		previousTime = now.Add(-updateInterval)
	}

	timeSince := now.Sub(previousTime)
	if timeSince >= updateInterval {
		githubToken := cfg.GitHub.Token.Value

		// Update the previousTime since we're checking
		_, err := file.WriteString(now.Format(time.RFC3339))
		if err != nil {
			logger.Warn("could not write timestamp to update status file",
				zap.String("file", tempFilePath),
				zap.Error(err),
			)
			return
		}
		err = file.Sync()
		if err != nil {
			logger.Warn("could not sync update status file",
				zap.String("file", tempFilePath),
				zap.Error(err),
			)
			return
		}

		githubClient := github.NewClient(nil)
		if githubToken != "" {
			githubClient = githubClient.WithAuthToken(githubToken)
		}
		available, version, err := updateAvailable(githubClient)
		if err != nil {
			logger.Warn("could not check for updates",
				zap.Error(err),
			)
			return
		}

		if available {
			styles := userio.NewNonInteractiveStyles()

			streams := userio.NewIOStreamsWithInteractive(
				os.Stdin,
				os.Stdout,
				os.Stderr,
				false,
			)

			streams.Info(
				styles.Bold.Render(
					fmt.Sprintf("corectl %s is available, run `corectl update` to install.", version),
				),
			)
		}
	} else {
		timeLeft := (updateInterval - timeSince).Round(time.Second)
		logger.Debug("next update check will be in",
			zap.Duration("timeLeft", timeLeft),
		)
	}
}

func getCurrentExecutablePath() string {
	execPath, err := os.Executable()
	if err != nil {
		logger.Fatal("Failed to get executable path",
			zap.Error(err),
		)
	}

	absolutePath, err := filepath.Abs(execPath)
	if err != nil {
		logger.Fatal("Failed to get absolute executable path",
			zap.Error(err),
		)
	}

	return absolutePath
}

func UpdateCmd(cfg *config.Config) *cobra.Command {
	opts := UpdateOpts{
		githubToken:   "",
		targetVersion: "",
	}
	updateCmd := &cobra.Command{
		Use:   CmdName,
		Short: "Update corectl",
		Long:  `Update to the latest corectl version.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.targetVersion = args[0]
			}
			opts.githubToken = cfg.GitHub.Token.Value

			nonInteractive, err := cmd.Flags().GetBool("non-interactive")
			if err != nil {
				return fmt.Errorf("could not get non-interactive flag: %+v", err)
			}

			opts.streams = userio.NewIOStreamsWithInteractive(
				os.Stdin,
				os.Stdout,
				os.Stderr,
				!nonInteractive,
			)

			return update(opts)
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

func updateAvailable(githubClient *github.Client) (bool, string, error) {
	release, err := getLatestCorectlRelease(githubClient)
	if err != nil {
		return false, "", err
	}
	asset, err := getLatestCorectlAsset(release)
	if err != nil {
		return false, "", err
	}

	if version.Version == asset.Version {
		return false, "", nil
	} else {
		return true, asset.Version, nil
	}
}

func update(opts UpdateOpts) error {
	if opts.targetVersion != "" {
		logger.Debug("target version set to",
			zap.String("version", opts.targetVersion),
		)
	}

	wizard := opts.streams.Wizard("Checking for updates", "Retrieved version metadata")
	githubClient := github.NewClient(nil)
	if opts.githubToken != "" {
		githubClient = githubClient.WithAuthToken(opts.githubToken)
	}

	var release *github.RepositoryRelease
	var err error
	if opts.targetVersion == "" {
		release, err = getLatestCorectlRelease(githubClient)
	} else {
		release, err = getCorectlReleaseByTag(githubClient, opts.targetVersion)
	}
	if err != nil {
		wizard.Abort(err.Error())
		return err
	}

	asset, err := getLatestCorectlAsset(release)
	if err != nil {
		wizard.Abort(err.Error())
		return err
	}
	logger.Debug("comparing versions",
		zap.String("current_version", version.Version),
		zap.String("remote_version", asset.Version),
	)
	if version.Version == asset.Version {
		wizard.SetCurrentTaskCompletedTitle(fmt.Sprintf("Already running %s release (%v)", opts.targetVersion, version.Version))
		wizard.Done()
		return nil
	} else {
		wizard.SetCurrentTaskCompletedTitle(fmt.Sprintf("Update available: %v", asset.Version))
		wizard.Done()
	}

	out, err := glamour.Render(asset.Changelog, "dark")
	if err == nil {
		_, _ = opts.streams.GetOutput().Write([]byte(out))
	} else {
		logger.Warn("could not render changelog markdown, falling back to plaintext")
		_, _ = opts.streams.GetOutput().Write([]byte(asset.Changelog))
	}

	logger.Debug("checking params",
		zap.Bool("skipConfirmation", opts.skipConfirmation),
	)

	wizard = opts.streams.Wizard("Confirming update", "Confirmation received")
	if opts.skipConfirmation {
		wizard.Info("--skip-confirmation is set, continuing with update")
	} else {
		if opts.streams.IsInteractive() {
			confirmation, err := confirmation.GetInput(opts.streams, fmt.Sprintf("Update to %s now?", asset.Version))
			if err != nil {
				return fmt.Errorf("could not get confirmation from user: %+v", err)
			}

			if confirmation {
				wizard.Info("Update accepted")
			} else {
				err = fmt.Errorf("update cancelled by user")
				wizard.Abort(err.Error())
				return err
			}
		} else {
			err = fmt.Errorf("non interactive terminal, cannot ask for confirmation")
			wizard.Abort(err.Error())
			return err
		}
	}

	wizard.SetTask(fmt.Sprintf("Downloading release %s", asset.Version), fmt.Sprintf("Downloaded release %s", asset.Version))
	data, err := downloadCorectlAsset(asset)
	if err != nil {
		wizard.Abort(err.Error())
		return fmt.Errorf("could not download release %s: %+v", asset.Version, err)
	}

	opts.streams.CurrentHandler.SetTask(fmt.Sprintf("Decompressing release %s", asset.Version), fmt.Sprintf("Decompressed release %s", asset.Version))
	var decompressed *tar.Reader
	decompressed, err = decompressCorectlAssetInMemory(data)
	if err != nil {
		wizard.Abort(err.Error())
		return fmt.Errorf("could not decompress release %s: %+v", asset.Version, err)
	}

	path := getCurrentExecutablePath()

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("corectl-%s-", asset.Version))
	if err != nil {
		err = fmt.Errorf("unable to create temporary file %s: %+v", asset.Version, err)
		return err
	}
	tmpPath := tmpFile.Name()

	opts.streams.CurrentHandler.SetTask(fmt.Sprintf("Installing release %s to path: %s", asset.Version, path), fmt.Sprintf("Release %s installed", asset.Version))
	err = writeCorectlAssetToPath(decompressed, tmpPath, tmpFile)
	if err != nil {
		return fmt.Errorf("could not write release %s to path %s: %+v", asset.Version, path, err)
	}
	err = tmpFile.Close()
	if err != nil {
		return fmt.Errorf("could not close temporary file %s: %+v", tmpPath, err)
	}

	partialPath := path + ".partial"

	// NOTE: os.Rename is the only way to overwrite an existing executable, but this doesn't work across
	// filesystems. Usually /tmp is set up as a separate filesystem, therefore we must copy and then remove to
	// simulate the rename
	err = moveFile(tmpPath, partialPath)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			wizard.Abort(fmt.Sprintf("Could not write to %s, try `sudo corectl update`", path))
			return fmt.Errorf("could not move file to partial path, try `sudo corectl update %s: %+v", path, err)
		} else {
			wizard.Abort(err.Error())
			return fmt.Errorf("could not move file to partial path %s: %+v", path, err)
		}
	}
	err = os.Rename(partialPath, path)
	if err != nil {
		wizard.Abort(err.Error())
		return fmt.Errorf("could not move file to path %s: %+v", path, err)
	}
	logger.Debug("moved",
		zap.String("from", tmpPath),
		zap.String("to", path),
	)
	wizard.Done()
	return nil
}

func getLatestCorectlAsset(release *github.RepositoryRelease) (*CoreCtlAsset, error) {
	if release.Assets == nil {
		return nil, errors.New("no assets found for the latest release")
	}

	architecture := runtime.GOARCH

	// Required due to the goreleaser config
	if architecture == "amd64" {
		architecture = "x86_64"
	}
	targetAssetName := fmt.Sprintf("corectl_%s_%s.tar.gz", runtime.GOOS, architecture)
	for _, asset := range release.Assets {
		assetName := strings.ToLower(asset.GetName())
		if assetName == targetAssetName {
			logger.Debug("github: found release asset with matching architecture & os",
				zap.String("asset", assetName),
			)
			return &CoreCtlAsset{
				Url:       *asset.BrowserDownloadURL,
				Version:   *release.TagName,
				Changelog: *release.Body,
			}, nil
		}
	}

	return nil, errors.New("no asset found for the current architecture and OS")

}

func getLatestCorectlRelease(client *github.Client) (*github.RepositoryRelease, error) {
	dummyRelease := github.RepositoryRelease{}
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), "coreeng", "corectl")
	if err != nil {
		return &dummyRelease, err
	}
	return release, nil
}
func getCorectlReleaseByTag(client *github.Client, version string) (*github.RepositoryRelease, error) {
	dummyRelease := github.RepositoryRelease{}
	release, _, err := client.Repositories.GetReleaseByTag(context.Background(), "coreeng", "corectl", version)
	if err != nil {
		return &dummyRelease, err
	}
	return release, nil
}

func downloadCorectlAsset(asset *CoreCtlAsset) (io.ReadCloser, error) {
	logger.Debug("starting download",
		zap.String("url", asset.Url),
	)
	resp, err := http.Get(asset.Url)

	if err != nil {
		return nil, fmt.Errorf("failed to download corectl release: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download corectl release: status code %v", resp.StatusCode)
	}
	logger.Debug("downloaded",
		zap.String("url", asset.Url),
		zap.Any("response", resp),
	)

	return resp.Body, err
}

func decompressCorectlAssetInMemory(tarData io.ReadCloser) (*tar.Reader, error) {
	logger.Debug("decompressing asset")

	gzr, err := gzip.NewReader(tarData)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzr.Close()
	tarReader := tar.NewReader(gzr)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar archive: %v", err)
		}

		if filepath.Base(header.Name) == "corectl" && header.Typeflag == tar.TypeReg {
			logger.Debug("found corectl in tar")
			return tarReader, nil
		}
	}
	return nil, fmt.Errorf("corectl binary not found in the release")
}

func writeCorectlAssetToPath(tarReader *tar.Reader, tmpPath string, outFile *os.File) error {
	binaryName := "corectl"

	written, err := io.Copy(outFile, tarReader)
	if err != nil {
		return fmt.Errorf("failed to copy %s binary: %v", binaryName, err)
	}

	logger.Debug("copied",
		zap.Int64("bytes", written),
		zap.String("file", tmpPath),
	)

	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions on %s binary: %v", binaryName, err)
	}

	logger.Debug("set executable permissions",
		zap.String("file", tmpPath),
	)

	logger.Debug("installed",
		zap.String("file", tmpPath),
		zap.String("binary", binaryName),
	)
	return nil
}

func moveFile(source string, destination string) error {
	logger.Debug("moving file from",
		zap.String("from", source),
		zap.String("to", destination),
	)
	err := copy.Copy(source, destination)
	if err != nil {
		return err
	}

	err = os.Remove(source)
	if err != nil {
		return err
	}

	logger.Debug("moved file from",
		zap.String("from", source),
		zap.String("to", destination),
	)

	return nil
}
