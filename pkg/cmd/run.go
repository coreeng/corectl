package cmd

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/coreeng/corectl/pkg/cmd/root"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
)

func Run() int {
	cfg, err := config.DiscoverConfig()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 10
	}

	// If CORECTL_DEBUG_STARTUP_DELAY is set, sleep for that amount of seconds before starting
	// This is useful for debugging when you need to attach to the process for testing interactivity
	if os.Getenv("CORECTL_DEBUG_STARTUP_DELAY") != "" {
		_, _ = fmt.Fprintf(os.Stderr, "Sleeping for %s seconds\n", os.Getenv("CORECTL_DEBUG_STARTUP_DELAY"))
		delay, err := strconv.Atoi(os.Getenv("CORECTL_DEBUG_STARTUP_DELAY"))
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error parsing CORECTL_DEBUG_STARTUP_DELAY: %v\n", err)
			return 1
		}
		time.Sleep(time.Duration(delay) * time.Second)
	}

	rootCmd := root.NewRootCmd(cfg)

	err = rootCmd.Execute()

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}
