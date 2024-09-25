package userio

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type WizardHandler interface {
	Done()
	Info(string)
	Warn(string)
	SetTask(string, string)
	SetInputModel(tea.Model) tea.Model
}

type asyncWizardHandler struct {
	messageChan     chan<- tea.Msg
	inputResultChan <-chan tea.Model
	done            <-chan bool
}

func (sh asyncWizardHandler) Done() {
	sh.update(doneMsg(true))
	<-sh.done
}

func (sh asyncWizardHandler) SetInputModel(input tea.Model) tea.Model {
	sh.update(input)
	modelResult := <-sh.inputResultChan
	return modelResult
}

func (sh asyncWizardHandler) Info(message string) {
	sh.update(logMsg{
		level:   zerolog.InfoLevel,
		message: message,
	})
}
func (sh asyncWizardHandler) Warn(message string) {
	sh.update(logMsg{
		level:   zerolog.WarnLevel,
		message: message,
	})
}

func (sh asyncWizardHandler) update(message tea.Msg) {
	sh.messageChan <- message
}

func (sh asyncWizardHandler) SetTask(title string, completedTitle string) {
	sh.update(task{title: title, completedTitle: completedTitle, logs: []logMsg{}, completed: false})
}

func newWizard(streams *IOStreams) WizardHandler {
	messageChan := make(chan tea.Msg)
	inputResultChan := make(chan tea.Model)
	done := make(chan bool)

	m := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#0404ff"))), // CECG Blue
	)
	sm := wizardModel{
		messageChan:     messageChan,
		inputResultChan: inputResultChan,
		model:           m,
		tasks:           []task{},
	}
	handler := asyncWizardHandler{
		messageChan:     messageChan,
		inputResultChan: inputResultChan,
		done:            done,
	}

	go func() {
		_, err := streams.execute(sm, handler)
		if err != nil {
			log.Panic().Msgf("Error in Wizard execution: %s", err.Error())
		}
		done <- true
	}()
	return handler
}

type wizardModel struct {
	messageChan     <-chan tea.Msg
	inputResultChan chan<- tea.Model
	model           spinner.Model
	quitting        bool
	err             error
	tasks           []task
	inputModel      tea.Model
	height          int
	width           int
}

type task struct {
	title          string
	completedTitle string
	completed      bool
	logs           []logMsg
}

type logMsg struct {
	message string
	level   zerolog.Level
}
type doneMsg bool

var checkMark = fmt.Sprint(lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓"))

func (sm wizardModel) ReceiveUpdateMessages() tea.Msg {
	message := <-sm.messageChan
	return message
}

func (sm wizardModel) Init() tea.Cmd {
	return tea.Batch(
		sm.model.Tick,
		sm.ReceiveUpdateMessages,
	)
}

func (sm wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(spinner.TickMsg); !ok {
		log.Debug().Msgf("Wizard: Received msg [%T] %s", msg, msg)
	}

	updateListener := sm.ReceiveUpdateMessages

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		sm.width = msg.Width
		sm.height = msg.Height
		if sm.inputModel != nil {
			newInputModel, inputCmd := sm.inputModel.Update(msg)
			sm.inputModel = newInputModel
			return sm, inputCmd
		}
		return sm, nil
	case doneMsg:
		if len(sm.messageChan) > 0 {
			return sm, updateListener
		} else {
			// Mark previous tasks as complete if we are done
			if len(sm.tasks) > 0 {
				sm.tasks[len(sm.tasks)-1].completed = true
			}
			sm.quitting = true
			return sm, tea.Quit
		}
	case task:
		// Mark previous tasks as complete if we add another
		if len(sm.tasks) > 0 {
			sm.tasks[len(sm.tasks)-1].completed = true
		}
		sm.tasks = append(sm.tasks, msg)
		return sm, updateListener
	case logMsg:
		if len(sm.tasks) > 0 {
			// Adds logs as children of the most recent task
			sm.tasks[len(sm.tasks)-1].logs = append(sm.tasks[len(sm.tasks)-1].logs, msg)
		} else {
			log.Warn().Msgf("Could not add log, no active tasks [%s: %s]", msg.level, msg.message)
		}
		return sm, updateListener
	case InputCompleted:
		sm.inputResultChan <- sm.inputModel
		sm.inputModel = nil
		return sm, updateListener
	case tea.Model:
		var cmd tea.Cmd
		sm.inputModel, cmd = msg.Update(tea.WindowSizeMsg{Width: sm.width, Height: sm.height})
		return sm, tea.Sequence(updateListener, cmd)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			sm.err = ErrInterrupted
			sm.quitting = true
			return sm, tea.Quit
		}
	}
	var newInputModel tea.Model
	var inputCmd tea.Cmd
	if sm.inputModel != nil {
		newInputModel, inputCmd = sm.inputModel.Update(msg)
		sm.inputModel = newInputModel
	}
	newSpinnerModel, cmd := sm.model.Update(msg)
	sm.model = newSpinnerModel
	return sm, tea.Batch(cmd, inputCmd)
}

func (sm wizardModel) View() string {
	bold := lipgloss.NewStyle().Bold(true)
	buffer := ""
	for taskIndex, task := range sm.tasks {
		if taskIndex > 0 {
			buffer += "\n"
		}
		for _, message := range task.logs {
			switch message.level {
			case zerolog.InfoLevel:
				buffer += wrap.String(InfoLog(message.message), sm.width) + "\n"
			case zerolog.WarnLevel:
				buffer += wrap.String(WarnLog(message.message), sm.width) + "\n"
			}
		}
		if task.completed {
			buffer += fmt.Sprintf("%s %s\n", checkMark, bold.Render(task.completedTitle))
		} else {
			if sm.inputModel != nil {
				buffer += fmt.Sprintf("%s%s\n", "📝 ", bold.Render(task.title))
			} else {
				// show spinner for incomplete tasks
				buffer += fmt.Sprintf("%s%s\n", sm.model.View(), bold.Render(task.title))
			}
		}
	}

	log.Trace().Msgf("sm.tasks: %v", sm.tasks)
	if sm.inputModel != nil {
		buffer += sm.inputModel.View()
	}
	return buffer
}
