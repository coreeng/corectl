package main

import (
	"github.com/coreeng/developer-platform/dpctl/cmd"
	"os"
)

func main() {
	statusCode := cmd.Run()
	os.Exit(statusCode)
}
