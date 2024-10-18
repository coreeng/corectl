package version

import (
	"fmt"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/version"
	"github.com/spf13/cobra"
)

func VersionCmd(cfg *config.Config) *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "List corectl version",
		Long:  `This command allows you to list the currently running corectl version.`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _args []string) {
			tag := version.Version
			if tag != "untagged" {
				tag = "v" + tag
			}
			fmt.Printf("corectl %s (commit: %s) %s %s\n", tag, version.Commit, version.Date, version.Arch)
		},
	}

	return versionCmd
}
