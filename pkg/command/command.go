package command

import (
	"fmt"
	"os/exec"
)

type Commander interface {
	Execute(string, ...string) ([]byte, error)
	ExecuteWithEnv(string, map[string]string, ...string) ([]byte, error)
}

type Command struct {
}

func NewCommand() Commander {
	return &Command{}
}

func (c *Command) Execute(cmd string, args ...string) ([]byte, error) {
	return execute(cmd, args, map[string]string{})
}

func (c *Command) ExecuteWithEnv(cmd string, envs map[string]string, args ...string) ([]byte, error) {
	return execute(cmd, args, envs)
}

func execute(cmd string, args []string, envs map[string]string) ([]byte, error) {
	command := exec.Command(cmd, args...)
	for key, value := range envs {
		command.Env = append(command.Env, fmt.Sprintf("%s=%s", key, value))
	}
	out, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("execute command: %s %s: %w", cmd, args, err)
	}
	return out, nil
}
