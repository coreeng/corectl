package userio

import (
	"errors"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/coreeng/corectl/pkg/cmdutil/userio/wizard"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
	"github.com/phuslu/log"
)

var (
	ErrInterrupted = errors.New("input was interrupted")
)

type IOStreams struct {
	stdin          io.Reader
	stdout         *lipgloss.Renderer
	stdoutRaw      io.Writer
	stderr         *lipgloss.Renderer
	stderrRaw      io.Writer
	styles         *styles
	isInteractive  bool
	CurrentHandler wizard.Handler
}

func NewIOStreams(stdin io.Reader, stdout io.Writer, stderr io.Writer) IOStreams {
	return NewIOStreamsWithInteractive(stdin, stdout, stderr, true)
}

func NewIOStreamsWithInteractive(stdin io.Reader, stdout io.Writer, stderr io.Writer, isInteractive bool) IOStreams {
	renderer := lipgloss.NewRenderer(stdout, termenv.WithColorCache(true))
	return IOStreams{
		stdin:          stdin,
		stdout:         renderer,
		stdoutRaw:      stdout,
		stderr:         lipgloss.NewRenderer(stderr, termenv.WithColorCache(true)),
		stderrRaw:      stderr,
		styles:         newStyles(renderer),
		isInteractive:  isInteractive && IsTerminalInteractive(stdin, stdout),
		CurrentHandler: nil,
	}
}

func (s IOStreams) IsInteractive() bool {
	return s.isInteractive
}

func (s IOStreams) GetOutput() io.Writer {
	return s.stdout.Output()
}

func IsTerminalInteractive(in io.Reader, out io.Writer) bool {
	_, inIsFile := in.(*os.File)
	if !inIsFile {
		return false
	}
	outF, outIsFile := out.(*os.File)
	if !outIsFile {
		return false
	}
	return isatty.IsTerminal(outF.Fd()) ||
		isatty.IsCygwinTerminal(outF.Fd())

}

func (s *IOStreams) Execute(model tea.Model, opts ...tea.ProgramOption) (tea.Model, error) {
	if _, isWizard := model.(wizard.Model); isWizard || s.CurrentHandler == nil {
		log.Debug().Msgf("IOStreams.execute: starting new session [%T]", model)

		options := append([]tea.ProgramOption{
			tea.WithInput(s.stdin),
			tea.WithOutput(s.stdoutRaw),
		}, opts...)

		model, err := tea.NewProgram(
			model,
			options...,
		).Run()
		return model, err
	} else {
		log.Debug().Msgf("IOStreams.execute: setting input model inside existing session [%T]", model)
		// Run inside the existing session
		return s.CurrentHandler.SetInputModel(model), nil
	}
}

type InputPrompt[V any] interface {
	GetInput(streams IOStreams) (V, error)
}
