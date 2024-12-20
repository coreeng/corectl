package userio

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/coreeng/corectl/pkg/cmdutil/userio/wizard"
	"github.com/coreeng/corectl/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func (s IOStreams) MsgE(message string, style lipgloss.Style, stream *lipgloss.Renderer) error {
	_, err := stream.Output().Write([]byte(style.Render(message)))
	if err != nil {
		return fmt.Errorf("couldn't output info message: %v", err)
	}
	if !strings.HasSuffix(message, "\n") {
		if _, err = stream.Output().Write([]byte("\n")); err != nil {
			return fmt.Errorf("couldn't output info message: %v", err)
		}
	}
	return nil
}

func (s IOStreams) Print(messages string) {
	err := s.MsgE(messages, lipgloss.NewStyle(), s.stdout)
	if err != nil {
		panic(err.Error())
	}
	logger.GetFileOnlyLogger().Info(messages)
}

func (s IOStreams) Info(messages string) {
	if logger.LogLevel() <= zapcore.InfoLevel {
		err := s.MsgE(messages, s.styles.info, s.stderr)
		if err != nil {
			panic(err.Error())
		}
	}
	logger.GetFileOnlyLogger().Info(messages)
}

func (s IOStreams) Warn(messages string) {
	if logger.LogLevel() <= zapcore.WarnLevel {
		err := s.MsgE(messages, s.styles.warn, s.stderr)
		if err != nil {
			panic(err.Error())
		}
	}
	logger.GetFileOnlyLogger().Warn(messages)
}

func (s IOStreams) Error(messages string) {
	if logger.LogLevel() <= zapcore.ErrorLevel {
		err := s.MsgE(messages, s.styles.err, s.stderr)
		if err != nil {
			panic(err.Error())
		}
	}
	logger.GetFileOnlyLogger().Error(messages)
}

func (s *IOStreams) Wizard(title string, completedTitle string) wizard.Handler {
	if s.IsInteractive() {
		model, handler, doneSync := wizard.New()
		s.CurrentHandler = handler
		go func() {
			_, err := s.Execute(model, tea.WithFilter(handler.OnQuit))
			if err != nil {
				logger.Error().With(zap.Error(err)).Msgf("Error in Wizard execution")
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
