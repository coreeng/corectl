package cmd

import (
	"fmt"
	"os"

	"github.com/coreeng/corectl/pkg/cmd/root"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
)

var logFile *os.File

func Run() int {
	cfg, err := config.DiscoverConfig()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		return 10
	}

	rootCmd := root.NewRootCmd(cfg, logFile)

	err = rootCmd.Execute()

	defer func() {
		if logFile != nil {
			logFile.Close()
		}
	}()

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		return 1
	}
	return 0
}
