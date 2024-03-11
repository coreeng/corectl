package userio

import (
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"io"
	"os"
)

var (
	ErrInterrupted = errors.New("input was interrupted")
)

type IOStreams struct {
	in            io.Reader
	out           io.Writer
	isInteractive bool
}

func NewIOStreams(in io.Reader, out io.Writer) IOStreams {
	return IOStreams{
		in:            in,
		out:           out,
		isInteractive: isTerminalInteractive(in, out),
	}
}

func NewIOStreamsWithInteractive(in io.Reader, out io.Writer, isInteractive bool) IOStreams {
	return IOStreams{
		in:            in,
		out:           out,
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
		tea.WithOutput(streams.out),
	).Run()
}

type InputPrompt[V any] interface {
	GetInput(streams IOStreams) (V, error)
}
