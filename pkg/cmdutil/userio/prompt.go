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
	in             io.Reader
	out            *lipgloss.Renderer
	outRaw         io.Writer
	styles         *styles
	isInteractive  bool
	CurrentHandler wizard.Handler
}

func newWizard(streams *IOStreams) wizard.Handler {
	model, handler := wizard.New()
	go func() {
		_, err := streams.execute(model, handler)
		if err != nil {
			log.Error().Err(err).Msgf("Error in Wizard execution")
		}
		handler.DoneSendChannel <- true
	}()
	return handler
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

func (streams *IOStreams) execute(model tea.Model, handler wizard.Handler) (tea.Model, error) {
	if _, isWizard := model.(wizard.Model); isWizard || streams.CurrentHandler == nil {
		log.Debug().Msgf("IOStreams.execute: starting new session [%T]", model)
		options := []tea.ProgramOption{
			tea.WithInput(streams.in),
			tea.WithOutput(streams.outRaw),
		}

		if handler != nil {
			streams.CurrentHandler = handler
			options = append(options, tea.WithFilter(handler.OnQuit))
		}

		return tea.NewProgram(
			model,
			options...,
		).Run()
	} else {
		log.Debug().Msgf("IOStreams.execute: setting input model inside existing session [%T]", model)
		// Run inside the existing session
		return streams.CurrentHandler.SetInputModel(model), nil
	}
}

type InputPrompt[V any] interface {
	GetInput(streams IOStreams) (V, error)
}
