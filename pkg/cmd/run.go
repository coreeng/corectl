package cmd

import (
	"fmt"
	"os"

	"github.com/coreeng/corectl/pkg/cmd/root"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/version"
	"github.com/phuslu/log"
)

func Run() int {
	log.Debug().
		Str("version", version.Version).
		Str("commit", version.Commit).
		Str("date", version.Commit).
		Msg("starting up")

	cfg, err := config.DiscoverConfig()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		return 10
	}

	rootCmd := root.NewRootCmd(cfg)

	err = rootCmd.Execute()

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		return 1
	}
	return 0
}
