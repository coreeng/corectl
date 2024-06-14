package env

import "os/exec"

type Commander interface {
	Execute(string, ...string) ([]byte, error)
}

type Command struct {
}

func NewCommand() Commander {
	return &Command{}
}

func (c *Command) Execute(cmd string, args ...string) ([]byte, error) {
	out, err := exec.Command(cmd, args...).Output()
	if err != nil {
		return nil, err
	}

	return out, nil
}
