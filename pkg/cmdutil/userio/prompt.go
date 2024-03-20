package userio

import (
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
	"io"
	"os"
)

var (
	ErrInterrupted = errors.New("input was interrupted")
)

type IOStreams struct {
	in            io.Reader
	out           *lipgloss.Renderer
	styles        *styles
	isInteractive bool
}

func NewIOStreams(in io.Reader, out io.Writer) IOStreams {
	return NewIOStreamsWithInteractive(in, out, true)
}

func NewIOStreamsWithInteractive(in io.Reader, out io.Writer, isInteractive bool) IOStreams {
	renderer := lipgloss.NewRenderer(out, termenv.WithColorCache(true))
	return IOStreams{
		in:            in,
		out:           renderer,
		styles:        newStyles(renderer),
		isInteractive: isInteractive && isTerminalInteractive(in, out),
	}
}

func (streams IOStreams) IsInteractive() bool {
	return streams.isInteractive
}

func isTerminalInteractive(in io.Reader, out io.Writer) bool {
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

func (streams IOStreams) execute(model tea.Model) (tea.Model, error) {
	return tea.NewProgram(
		model,
		tea.WithInput(streams.in),
		tea.WithOutput(streams.out.Output()),
	).Run()
}

type InputPrompt[V any] interface {
	GetInput(streams IOStreams) (V, error)
}
