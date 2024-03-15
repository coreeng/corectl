package cmd

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmd/root"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"os"
)

func Run() int {
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
