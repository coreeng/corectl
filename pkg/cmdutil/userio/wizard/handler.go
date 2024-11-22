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
	SetCurrentTaskCompleted()
	SetCurrentTaskCompletedTitle(string)
	SetCurrentTaskCompletedTitleWithStatus(string, TaskStatus)
	SetInputModel(tea.Model) tea.Model
	SetTask(string, string)
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
		logger.Panic("Done: handler is already completed")
	}
	handler.update(doneMsg(true))
	handler.completed = true
	<-handler.doneChannel
}

func (handler *asyncHandler) Abort(err string) {
	if handler.completed {
		logger.Panic("Abort: handler is already completed")
	}
	handler.update(errorMsg(err))
	handler.completed = true
	<-handler.doneChannel
}

func (handler asyncHandler) OnQuit(m tea.Model, msg tea.Msg) tea.Msg {
	if _, ok := msg.(tea.QuitMsg); !ok {
		return msg
	}
	logger.Debug("received msg",
		zap.String("model", fmt.Sprintf("%T", m)),
		zap.String("msg", fmt.Sprintf("%T", msg)))

	switch m := m.(type) {
	case Model:
		if m.quitting {
			logger.Debug("received tea.Quit from parent")
			return msg
		}
		// If we didn't send the tea.Quit - assume it is from the inputModel and forward it
		logger.Debug("received tea.Quit from child",
			zap.String("type", fmt.Sprintf("%T", m.inputModel)))
		return InputCompleted{model: m.inputModel}
	}
	return msg
}

func (handler asyncHandler) SetInputModel(input tea.Model) tea.Model {
	handler.update(input)
	modelResult := <-handler.inputResultChannel
	return modelResult
}

func (handler asyncHandler) Print(message string) {
	handler.update(logMsg{
		level:   logger.Log.Level(),
		message: message,
	})
}

func (handler asyncHandler) Info(message string) {
	handler.update(logMsg{
		level:   zapcore.InfoLevel,
		message: message,
	})
}

func (handler asyncHandler) Warn(message string) {
	handler.update(logMsg{
		level:   zapcore.WarnLevel,
		message: message,
	})
}

func (handler asyncHandler) Error(message string) {
	handler.update(logMsg{
		level:   zapcore.ErrorLevel,
		message: message,
	})
}

func (handler asyncHandler) update(message tea.Msg) {
	handler.messageChannel <- message
}

func (handler asyncHandler) SetCurrentTaskCompleted() {
	handler.update(taskComplete(true))
}

func (handler asyncHandler) SetCurrentTaskCompletedTitle(title string) {
	handler.update(updateCurrentTaskCompletedTitle{
		title:  title,
		status: TaskStatusSuccess,
	})
}

func (handler asyncHandler) SetCurrentTaskCompletedTitleWithStatus(title string, status TaskStatus) {
	handler.update(updateCurrentTaskCompletedTitle{
		title:  title,
		status: status,
	})
}

func (handler asyncHandler) SetTask(title string, completedTitle string) {
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
	})
}
