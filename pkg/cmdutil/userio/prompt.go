package userio

import (
	"errors"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
)

var (
	ErrInterrupted = errors.New("input was interrupted")
)

type IOStreams struct {
	in             io.Reader
	out            *lipgloss.Renderer
	outRaw         io.Writer
	styles         *styles
	isInteractive  bool
	CurrentHandler SpinnerHandler
}

func NewIOStreams(in io.Reader, out io.Writer) IOStreams {
	return NewIOStreamsWithInteractive(in, out, true)
}

func NewIOStreamsWithInteractive(in io.Reader, out io.Writer, isInteractive bool) IOStreams {
	renderer := lipgloss.NewRenderer(out, termenv.WithColorCache(true))
	return IOStreams{
		in:             in,
		out:            renderer,
		outRaw:         out,
		styles:         newStyles(renderer),
		isInteractive:  isInteractive && isTerminalInteractive(in, out),
		CurrentHandler: nil,
	}
}

func (streams IOStreams) IsInteractive() bool {
	return streams.isInteractive
}

func (s IOStreams) GetOutput() io.Writer {
	return s.out.Output()
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

func (streams *IOStreams) execute(model tea.Model, handler SpinnerHandler) (tea.Model, error) {
	if handler != nil {
		streams.CurrentHandler = handler
	}
	return tea.NewProgram(model,
		tea.WithInput(streams.in),
		tea.WithOutput(streams.out.Output()),
	).Run()
}

type InputPrompt[V any] interface {
	GetInput(streams IOStreams) (V, error)
}
