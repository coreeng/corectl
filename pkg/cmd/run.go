package cmd

import (
	"fmt"
	"os"

	"github.com/coreeng/corectl/pkg/cmd/root"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
)

func Run() int {
	cfg, err := config.DiscoverConfig()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 10
	}

	rootCmd := root.NewRootCmd(cfg)

	err = rootCmd.Execute()

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}
