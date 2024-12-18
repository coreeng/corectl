package wizard

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/coreeng/corectl/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Handler interface {
	Abort(string)
	Done()
	Error(string)
	Info(string)
	OnQuit(tea.Model, tea.Msg) tea.Msg
	Print(string)
	SetCurrentTaskCompleted(zapcore.Level)
	SetCurrentTaskCompletedTitle(string, zapcore.Level)
	SetCurrentTaskCompletedTitleWithStatus(string, TaskStatus, zapcore.Level)
	SetInputModel(tea.Model) tea.Model
	SetTask(string, string, zapcore.Level)
	Warn(string)
}

type asyncHandler struct {
	messageChannel     chan<- tea.Msg
	inputResultChannel <-chan tea.Model
	doneChannel        chan bool
	completed          bool
}

func (handler *asyncHandler) Done() {
	if handler.completed {
		logger.Panic().Msgf("Done: handler is already completed")
	}
	handler.update(doneMsg(true), zapcore.WarnLevel)
	handler.completed = true
	<-handler.doneChannel
}

func (handler *asyncHandler) Abort(err string) {
	if handler.completed {
		logger.Panic().Msgf("Abort: handler is already completed")
	}
	handler.update(errorMsg(err), zapcore.ErrorLevel)
	handler.completed = true
	<-handler.doneChannel
}

func (handler asyncHandler) OnQuit(m tea.Model, msg tea.Msg) tea.Msg {
	if _, ok := msg.(tea.QuitMsg); !ok {
		return msg
	}
	logger.Debug().With(
		zap.String("model", fmt.Sprintf("%T", m)),
		zap.String("msg", fmt.Sprintf("%T", msg))).
		Msg("received msg")

	switch m := m.(type) {
	case Model:
		if m.quitting {
			logger.Debug().Msg("received tea.Quit from parent")
			return msg
		}
		// If we didn't send the tea.Quit - assume it is from the inputModel and forward it
		logger.Debug().Msgf("received tea.Quit from child %T", m.inputModel)
		return InputCompleted{model: m.inputModel}
	}
	return msg
}

func (handler asyncHandler) SetInputModel(input tea.Model) tea.Model {
	handler.update(input, zapcore.WarnLevel)
	modelResult := <-handler.inputResultChannel
	return modelResult
}

func (handler asyncHandler) Print(message string) {
	handler.update(logMsg{
		level:   zapcore.DebugLevel,
		message: message,
	}, zapcore.WarnLevel)
}

func (handler asyncHandler) Info(message string) {
	handler.update(logMsg{
		level:   zapcore.InfoLevel,
		message: message,
	}, zapcore.InfoLevel)
}

func (handler asyncHandler) Warn(message string) {
	handler.update(logMsg{
		level:   zapcore.WarnLevel,
		message: message,
	}, zapcore.WarnLevel)
}

func (handler asyncHandler) Error(message string) {
	handler.update(logMsg{
		level:   zapcore.ErrorLevel,
		message: message,
	}, zapcore.ErrorLevel)
}

func (handler asyncHandler) update(message tea.Msg, messageLevel zapcore.Level) {
	if logger.LogLevel() <= messageLevel {
		handler.messageChannel <- message
	}
	logger.GetFileOnlyLogger().Log(messageLevel, fmt.Sprintf("%v", message))
}

func (handler asyncHandler) SetCurrentTaskCompleted(messageLevel zapcore.Level) {
	handler.update(taskComplete(true), messageLevel)
}

func (handler asyncHandler) SetCurrentTaskCompletedTitle(title string, messageLevel zapcore.Level) {
	handler.update(updateCurrentTaskCompletedTitle{
		title:  title,
		status: TaskStatusSuccess,
	}, messageLevel)
}

func (handler asyncHandler) SetCurrentTaskCompletedTitleWithStatus(title string, status TaskStatus, messageLevel zapcore.Level) {
	handler.update(updateCurrentTaskCompletedTitle{
		title:  title,
		status: status,
	}, messageLevel)
}

func (handler asyncHandler) SetTask(title string, completedTitle string, messageLevel zapcore.Level) {
	status := taskStatusUnknown
	if completedTitle != "" {
		status = TaskStatusSuccess
	}
	handler.update(task{
		title:          title,
		completedTitle: completedTitle,
		status:         status,
		logs:           []logMsg{},
		completed:      false,
	}, messageLevel)

}
