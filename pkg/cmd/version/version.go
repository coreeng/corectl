package version

import (
	"fmt"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
	arch    = "unknown"
)

func VersionCmd(cfg *config.Config) *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "List corectl version",
		Long:  `This command allows you to list the currently running corectl version.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("corectl %s (commit: %s) %s %s\n", version, commit, date, arch)
		},
	}

	return versionCmd
}
