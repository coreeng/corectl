package main

import (
	"github.com/coreeng/corectl/pkg/cmd"
	"os"
)

func main() {
	statusCode := cmd.Run()
	os.Exit(statusCode)
}
