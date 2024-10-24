package userio

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/coreeng/corectl/pkg/cmdutil/userio/wizard"
	"github.com/phuslu/log"
)

func (s IOStreams) MsgE(message string, style lipgloss.Style) error {
	_, err := s.out.Output().Write([]byte(style.Render(message)))
	if err != nil {
		return fmt.Errorf("couldn't output info message: %v", err)
	}
	if !strings.HasSuffix(message, "\n") {
		if _, err = s.out.Output().Write([]byte("\n")); err != nil {
			return fmt.Errorf("couldn't output info message: %v", err)
		}
	}
	return nil
}

func (s IOStreams) Print(messages string) {
	err := s.MsgE(messages, lipgloss.NewStyle())
	if err != nil {
		panic(err.Error())
	}
}

func (s IOStreams) Info(messages string) {
	err := s.MsgE(messages, s.styles.info)
	if err != nil {
		panic(err.Error())
	}
}

func (s IOStreams) Warn(messages string) {
	err := s.MsgE(messages, s.styles.warn)
	if err != nil {
		panic(err.Error())
	}
}

func (s IOStreams) Error(messages string) {
	err := s.MsgE(messages, s.styles.err)
	if err != nil {
		panic(err.Error())
	}
}

func (s *IOStreams) Wizard(title string, completedTitle string) wizard.Handler {
	if s.IsInteractive() {
		model, handler, doneSync := wizard.New()
		s.CurrentHandler = handler
		go func() {
			_, err := s.Execute(model, tea.WithFilter(handler.OnQuit))
			if err != nil {
				log.Error().Err(err).Msgf("Error in Wizard execution")
			}
			doneSync <- true
			s.CurrentHandler = nil
		}()
	} else {
		s.CurrentHandler = &nonInteractiveHandler{
			streams: s,
			styles:  NewNonInteractiveStyles(),
		}
	}
	s.CurrentHandler.SetTask(title, completedTitle)
	return s.CurrentHandler
}
