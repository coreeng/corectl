package command

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Options struct {
	Env    map[string]string
	Args   []string
	Stdout io.Writer
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

func WithOverrideStdout(w io.Writer) Option {
	return func(o *Options) {
		o.Stdout = w
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
	Stdout        io.Writer
	Stderr        io.Writer
	VerboseWriter io.Writer
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

func WithStderr(w io.Writer) func(*DefaultCommander) {
	return func(c *DefaultCommander) {
		c.Stderr = w
	}
}

func WithVerboseWriter(w io.Writer) func(*DefaultCommander) {
	return func(c *DefaultCommander) {
		c.VerboseWriter = w
	}
}

func (c *DefaultCommander) Execute(cmd string, opts ...Option) ([]byte, error) {

	options := ApplyOptions(opts)

	command := exec.Command(cmd, options.Args...)

	out := c.Stdout
	if options.Stdout != nil {
		out = options.Stdout
	}

	if out != nil {
		command.Stdout = out
	}

	if c.Stderr != nil {
		command.Stderr = c.Stderr
	}

	command.Env = os.Environ()
	for key, value := range options.Env {
		command.Env = append(command.Env, fmt.Sprintf("%s=%s", key, value))
	}
	var output []byte
	var err error

	if c.VerboseWriter != nil {
		_, err := fmt.Fprintf(c.VerboseWriter, "Executing command: %s\n", FormatCommand(cmd, options))
		if err != nil {
			return nil, err
		}
	}

	if out != nil {
		err = command.Run()
	} else {
		output, err = command.Output()
	}

	if err != nil {
		return nil, fmt.Errorf("execute command `%s` returned %w", FormatCommand(cmd, options), err)
	}
	return output, nil
}

func FormatCommand(cmd string, options *Options) string {
	var envArray []string
	for k, v := range options.Env {
		envArray = append(envArray, fmt.Sprintf("%s=%s", k, v))
	}

	commandParts := make([]string, 0, len(envArray)+1+len(options.Args))
	commandParts = append(commandParts, envArray...)
	commandParts = append(commandParts, cmd)
	commandParts = append(commandParts, options.Args...)

	return strings.Join(commandParts, " ")
}
