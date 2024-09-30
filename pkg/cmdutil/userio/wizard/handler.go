package wizard

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/phuslu/log"
)

type Handler interface {
	Done()
	Info(string)
	SetCurrentTaskCompletedTitle(string)
	SetInputModel(tea.Model) tea.Model
	SetTask(string, string)
	Warn(string)
	OnQuit(tea.Model, tea.Msg) tea.Msg
}

type asyncHandler struct {
	messageChannel     chan<- tea.Msg
	inputResultChannel <-chan tea.Model
	doneReceiveChannel <-chan bool
	DoneSendChannel    chan<- bool
}

func (handler asyncHandler) Done() {
	handler.update(doneMsg(true))
	<-handler.doneReceiveChannel
}

func (handler asyncHandler) OnQuit(m tea.Model, msg tea.Msg) tea.Msg {
	if _, ok := msg.(tea.QuitMsg); !ok {
		return msg
	}
	log.Debug().
		Str("model", fmt.Sprintf("%T", m)).
		Str("msg", fmt.Sprintf("%T", msg)).
		Msg("received msg")

	switch m := m.(type) {
	case Model:
		if m.quitting {
			return msg
		}
		// If we didn't send the tea.Quit - assume it is from the inputModel
		return InputCompleted{model: m.inputModel}
	}
	return msg
}

func (handler asyncHandler) SetInputModel(input tea.Model) tea.Model {
	handler.update(input)
	modelResult := <-handler.inputResultChannel
	return modelResult
}

func (handler asyncHandler) Info(message string) {
	handler.update(logMsg{
		level:   log.InfoLevel,
		message: message,
	})
}
func (handler asyncHandler) Warn(message string) {
	handler.update(logMsg{
		level:   log.WarnLevel,
		message: message,
	})
}

func (handler asyncHandler) update(message tea.Msg) {
	handler.messageChannel <- message
}

func (handler asyncHandler) SetCurrentTaskCompletedTitle(completedTitle string) {
	handler.update(updateCurrentTaskCompletedTitle(completedTitle))
}

func (handler asyncHandler) SetTask(title string, completedTitle string) {
	handler.update(task{title: title, completedTitle: completedTitle, logs: []logMsg{}, completed: false})
}
