package main

import (
	"os"

	"github.com/coreeng/corectl/pkg/cmd"
)

func main() {
	statusCode := cmd.Run()
	os.Exit(statusCode)
}
