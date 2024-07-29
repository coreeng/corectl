package command

import "fmt"

func DepsInstalled(c Commander, deps ...string) error {
	for _, dep := range deps {
		if _, err := c.Execute("which", "-s", dep); err != nil {
			return fmt.Errorf("%s is not installed: %w", dep, err)
		}
	}
	return nil
}
