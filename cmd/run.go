package cmd

import (
	"fmt"
	"github.com/coreeng/developer-platform/dpctl/cmd/config"
	"github.com/coreeng/developer-platform/dpctl/cmd/root"
	"os"
)

func Run() int {
	cfg, err := config.DiscoverConfig()
	if err != nil {
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
