package command

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Commander interface {
	Execute(string, ...string) ([]byte, error)
	ExecuteWithEnv(string, map[string]string, ...string) ([]byte, error)
}

type Command struct {
	Stdout io.Writer
}

func NewCommand(options ...func(*Command)) Commander {
	cmd := &Command{
		Stdout: os.Stdout, // Default to standard output
	}

	for _, option := range options {
		option(cmd)
	}

	return cmd
}

func WithStdout(w io.Writer) func(*Command) {
	return func(c *Command) {
		c.Stdout = w
	}
}

func (c *Command) Execute(cmd string, args ...string) ([]byte, error) {
	return c.execute(cmd, args, map[string]string{})
}

func (c *Command) ExecuteWithEnv(cmd string, envs map[string]string, args ...string) ([]byte, error) {
	return c.execute(cmd, args, envs)
}

func (c *Command) execute(cmd string, args []string, envs map[string]string) ([]byte, error) {
	command := exec.Command(cmd, args...)
	if c.Stdout != nil {
		command.Stdout = c.Stdout
	}
	for key, value := range envs {
		command.Env = append(command.Env, fmt.Sprintf("%s=%s", key, value))
	}
	var out []byte
	var err error

	if c.Stdout != nil {
		err = command.Run()
	} else {
		out, err = command.Output()
	}

	if err != nil {
		return nil, fmt.Errorf("execute command: %s %s: %w", cmd, args, err)
	}
	return out, nil
}
