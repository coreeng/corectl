package command

import (
	"fmt"
	"io"
)

func DepsInstalled(c Commander, deps ...string) error {
	for _, dep := range deps {
		if _, err := c.Execute(dep, WithArgs("help"), WithOverrideStdout(io.Discard)); err != nil {
			return fmt.Errorf("%s is not installed: %w", dep, err)
		}
	}
	return nil
}
