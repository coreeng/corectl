package command

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Options struct {
	Env  map[string]string
	Args []string
}

type Option func(*Options)

func WithEnv(env map[string]string) Option {
	return func(o *Options) {
		o.Env = env
	}
}

func WithArgs(args ...string) Option {
	return func(o *Options) {
		o.Args = args
	}
}

func ApplyOptions(opts []Option) *Options {
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

type Commander interface {
	Execute(cmd string, opts ...Option) ([]byte, error)
}

type DefaultCommander struct {
	Stdout io.Writer
}

func NewCommander(options ...func(*DefaultCommander)) Commander {
	cmd := &DefaultCommander{
		Stdout: os.Stdout, // Default to standard output
	}

	for _, option := range options {
		option(cmd)
	}

	return cmd
}

func WithStdout(w io.Writer) func(*DefaultCommander) {
	return func(c *DefaultCommander) {
		c.Stdout = w
	}
}

func (c *DefaultCommander) Execute(cmd string, opts ...Option) ([]byte, error) {

	options := ApplyOptions(opts)

	command := exec.Command(cmd, options.Args...)

	if c.Stdout != nil {
		command.Stdout = c.Stdout
	}
	for key, value := range options.Env {
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
		return nil, fmt.Errorf("execute command: %s: %w", cmd, err)
	}
	return out, nil
}
