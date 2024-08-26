package userio

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"io"
	"os"
	"slices"
)

type testEnviron struct{}

func (testEnviron) Environ() []string {
	return append(slices.Clone(os.Environ()), "NO_COLOR=true")
}

func (testEnviron) Getenv(name string) string {
	if name == "NO_COLOR" {
		return "true"
	}
	return os.Getenv(name)
}

func NewTestIOStreams(in io.Reader, out io.Writer, isInteractive bool) IOStreams {
	renderer := lipgloss.NewRenderer(
		out,
		termenv.WithTTY(isInteractive),
		termenv.WithEnvironment(testEnviron{}),
	)
	return IOStreams{
		in:            in,
		out:           renderer,
		styles:        newStyles(renderer),
		isInteractive: isInteractive,
	}
}
