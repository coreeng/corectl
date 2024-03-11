package userio

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type SpinnerHandler struct {
	doneChan    chan<- bool
	quittedChan <-chan bool
}

func (sh SpinnerHandler) Done() {
	sh.doneChan <- true
	_ = <-sh.quittedChan
}

func newSpinner(message string, streams IOStreams) SpinnerHandler {
	doneChan := make(chan bool)
	quittedChan := make(chan bool)
	m := spinner.New(spinner.WithSpinner(spinner.Dot))
	sm := spinnerModel{
		message:  message,
		doneChan: doneChan,
		model:    m,
	}
	handler := SpinnerHandler{
		doneChan:    doneChan,
		quittedChan: quittedChan,
	}
	go func() {
		_, _ = streams.execute(sm)
		quittedChan <- true
	}()
	return handler
}

type spinnerModel struct {
	message  string
	doneChan <-chan bool
	done     bool
	model    spinner.Model
	quitting bool
	err      error
}

type doneMsg bool

func (sm spinnerModel) Init() tea.Cmd {
	return tea.Batch(
		sm.model.Tick,
		func() tea.Msg {
			_ = <-sm.doneChan
			sm.done = true
			return doneMsg(sm.done)
		},
	)
}

func (sm spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(doneMsg); ok {
		sm.quitting = true
		return sm, tea.Quit
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			sm.err = ErrInterrupted
			sm.quitting = true
			return sm, tea.Quit
		}
	}
	newSpinnerModel, cmd := sm.model.Update(msg)
	sm.model = newSpinnerModel
	return sm, cmd
}

func (sm spinnerModel) View() string {
	if sm.quitting {
		return ""
	} else {
		return sm.model.View() + " " + sm.message + ""
	}
}
