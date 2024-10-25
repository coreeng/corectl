package version

import (
	"fmt"
	"os"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/version"
	"github.com/spf13/cobra"
)

const CmdName = "version"

func VersionCmd(cfg *config.Config) *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   CmdName,
		Short: "List corectl version",
		Long:  `This command allows you to list the currently running corectl version.`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _args []string) {
			streams := userio.NewIOStreamsWithInteractive(
				os.Stdin,
				os.Stdout,
				os.Stderr,
				false,
			)
			streams.Print(fmt.Sprintf("corectl %s (commit: %s) %s %s\n", version.Version, version.Commit, version.Date, version.Arch))
		},
	}

	return versionCmd
}
