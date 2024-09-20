package userio

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SpinnerHandler interface {
	Done()
	Info(string)
	SetTitle(string)
}

type asyncSpinnerHandler struct {
	doneChan    chan<- bool
	quittedChan <-chan bool
	titleChan   chan<- string
	messageChan chan<- string
}

func (sh asyncSpinnerHandler) Done() {
	sh.doneChan <- true
	close(sh.doneChan)
	close(sh.messageChan)
	close(sh.titleChan)
	<-sh.quittedChan
}

func (sh asyncSpinnerHandler) Info(message string) {
	sh.messageChan <- message
}

func (sh asyncSpinnerHandler) SetTitle(title string) {
	sh.titleChan <- title
}

func newSpinner(message string, streams IOStreams) SpinnerHandler {
	doneChan := make(chan bool)
	quittedChan := make(chan bool)
	messageChan := make(chan string)
	titleChan := make(chan string)
	m := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#0404ff"))), // CECG Blue
	)
	sm := spinnerModel{
		message:     message,
		messageChan: messageChan,
		titleChan:   titleChan,
		doneChan:    doneChan,
		model:       m,
	}
	handler := asyncSpinnerHandler{
		doneChan:    doneChan,
		quittedChan: quittedChan,
		messageChan: messageChan,
		titleChan:   titleChan,
	}
	go func() {
		_, _ = streams.execute(sm)
		quittedChan <- true
		close(quittedChan)
	}()
	return handler
}

type spinnerModel struct {
	message     string
	doneChan    <-chan bool
	messageChan <-chan string
	titleChan   <-chan string
	done        bool
	model       spinner.Model
	quitting    bool
	err         error
}

type doneMsg bool

type infoMsg string
type titleMsg string

func (sm spinnerModel) ReceiveInfoMessages() tea.Msg {
	message := <-sm.messageChan
	return infoMsg(message)
}

func (sm spinnerModel) ReceiveTitleChanges() tea.Msg {
	title := <-sm.titleChan
	return titleMsg(title)
}

func (sm spinnerModel) Init() tea.Cmd {
	return tea.Batch(
		sm.model.Tick,
		func() tea.Msg {
			<-sm.doneChan
			return doneMsg(sm.done)
		},
		sm.ReceiveInfoMessages,
		sm.ReceiveTitleChanges,
	)
}

func (sm spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(doneMsg); ok || sm.quitting {
		var quitCmd tea.Cmd
		if !sm.quitting {
			quitCmd = tea.Sequence(tea.Printf("hello\nworld\n"), tea.Quit)
			sm.quitting = true
			return sm, quitCmd
		}
	}
	switch msg := msg.(type) {
	case infoMsg:
		return sm, tea.Sequence(tea.Printf(InfoLog(string(msg))), sm.ReceiveInfoMessages)
	case titleMsg:
		sm.message = string(msg)
		return sm, sm.ReceiveTitleChanges
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
		return "Completed\n"
	} else {
		return sm.model.View() + lipgloss.NewStyle().Bold(true).Render(sm.message+"\n")
	}
}
