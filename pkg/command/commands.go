package command

import "fmt"

func DepsInstalled(c Commander, deps ...string) error {
	for _, cmd := range deps {
		if _, err := c.Execute(cmd, "help"); err != nil {
			return fmt.Errorf("%s is not installed: %w", cmd, err)
		}
	}

	return nil
}
