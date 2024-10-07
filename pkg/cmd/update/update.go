package update

import (
	"archive/tar"
	"fmt"
	"os"

	"github.com/charmbracelet/glamour"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/google/go-github/v59/github"
	"github.com/phuslu/log"
	"github.com/spf13/cobra"
)

type UpdateOpts struct {
	Streams        userio.IOStreams
	NonInteractive bool
}

func UpdateCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update corectl",
		Long:  `Update to the latest corectl version.`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _args []string) {
			opts := UpdateOpts{}
			opts.Streams = userio.NewIOStreamsWithInteractive(
				os.Stdin,
				os.Stdout,
				!opts.NonInteractive,
			)
			wizard := opts.Streams.Wizard("Checking for updates", "")
			githubClient := github.NewClient(nil).
				WithAuthToken(cfg.GitHub.Token.Value)

			release, err := git.GetLatestCorectlRelease(githubClient)
			if err != nil {
				log.Panic().Err(err)
			}
			asset, err := git.GetLatestCorectlAsset(release)
			if err != nil {
				log.Panic().Err(err)
			}
			wizard.SetCurrentTaskCompletedTitle(fmt.Sprintf("Update available: %v", asset.Version))
			wizard.Done()
			out, err := glamour.Render(asset.Changelog, "dark")
			if err != nil {
				log.Panic().Err(err)
			}
			fmt.Print(out)
			wizard = opts.Streams.Wizard("Updating", "")
			opts.Streams.CurrentHandler.Info(fmt.Sprintf("Downloading release %s", asset.Version))
			data, err := git.DownloadCorectlAsset(asset)
			if err != nil {
				log.Panic().Err(err)
			}

			opts.Streams.CurrentHandler.Info(fmt.Sprintf("Decompressing release %s", asset.Version))
			var decompressed *tar.Reader
			decompressed, err = git.DecompressCorectlAssetInMemory(data)
			if err != nil {
				log.Panic().Err(err)
			}

			opts.Streams.CurrentHandler.Info(fmt.Sprintf("Installing release %s to path", asset.Version))
			git.WriteCorectlAssetToPath(decompressed, "/usr/local/bin/corectl")
			if err != nil {
				log.Panic().Err(err)
			}

			wizard.SetCurrentTaskCompletedTitle(fmt.Sprintf("Release %s installed", asset.Version))
			wizard.Done()
		},
	}
}
