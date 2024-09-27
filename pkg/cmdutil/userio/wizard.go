package userio

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
	"github.com/phuslu/log"
)

type WizardHandler interface {
	Done()
	Info(string)
	SetCurrentTaskCompletedTitle(string)
	SetInputModel(tea.Model) tea.Model
	SetTask(string, string)
	Warn(string)
	OnQuit(tea.Model, tea.Msg) tea.Msg
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

func (sh asyncWizardHandler) OnQuit(m tea.Model, msg tea.Msg) tea.Msg {
	log.Debug().
		Str("model", fmt.Sprintf("%T", m)).
		Str("msg", fmt.Sprintf("%T", msg)).
		Msg("received msg")
	if _, ok := msg.(tea.QuitMsg); !ok {
		return msg
	}

	switch m := m.(type) {
	case wizardModel:
		if m.quitting {
			return msg
		}
		// If we didn't send the tea.Quit - assume it is from the inputModel
		return InputCompleted{model: m.inputModel}
	}
	return msg
}

func (sh asyncWizardHandler) SetInputModel(input tea.Model) tea.Model {
	sh.update(input)
	modelResult := <-sh.inputResultChan
	return modelResult
}

func (sh asyncWizardHandler) Info(message string) {
	sh.update(logMsg{
		level:   log.InfoLevel,
		message: message,
	})
}
func (sh asyncWizardHandler) Warn(message string) {
	sh.update(logMsg{
		level:   log.WarnLevel,
		message: message,
	})
}

func (sh asyncWizardHandler) update(message tea.Msg) {
	sh.messageChan <- message
}

func (sh asyncWizardHandler) SetCurrentTaskCompletedTitle(completedTitle string) {
	sh.update(updateCurrentTaskCompletedTitle(completedTitle))
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
			log.Error().Err(err).Msgf("Error in Wizard execution")
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

type updateCurrentTaskCompletedTitle string
type logMsg struct {
	message string
	level   log.Level
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

func (sm wizardModel) getLatestTask() *task {
	var task *task
	if len(sm.tasks) > 0 {
		task = &sm.tasks[len(sm.tasks)-1]
	}
	return task
}

func (sm wizardModel) markLatestTaskComplete() *task {
	task := sm.getLatestTask()
	if task == nil {
		log.Warn().Msgf("Wizard: Marking task complete, but no tasks found")
	} else {
		log.Debug().Msgf("Wizard: Marking task complete: %s", task.completedTitle)
		task.completed = true
	}

	return task
}

func (sm wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updateListener := sm.ReceiveUpdateMessages

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		log.Debug().Msgf("Wizard: Received [%T], (w:%d, h:%d)", msg, msg.Width, msg.Height)
		sm.width = msg.Width
		sm.height = msg.Height
		if sm.inputModel != nil {
			newInputModel, inputCmd := sm.inputModel.Update(msg)
			sm.inputModel = newInputModel
			return sm, inputCmd
		}
		return sm, nil
	case doneMsg:
		log.Debug().Msgf("Wizard: Received [%T]", msg)
		if messageLen := len(sm.messageChan); messageLen > 0 {
			log.Debug().Msgf("Wizard: Message channel still has %d items, postponing shutdown", messageLen)
			return sm, updateListener
		} else {
			sm.markLatestTaskComplete()
			sm.quitting = true
			return sm, tea.Quit
		}
	case task:
		sm.markLatestTaskComplete()
		log.Debug().Msgf("Wizard: New task: %s", msg.title)
		sm.tasks = append(sm.tasks, msg)
		return sm, updateListener
	case logMsg:
		if len(sm.tasks) > 0 {
			log.Debug().Msgf("Wizard: Log received %s: %s", msg.level, msg.message)
			// Adds logs as children of the most recent task
			latestTask := sm.getLatestTask()
			latestTask.logs = append(latestTask.logs, msg)
		} else {
			log.Warn().Msgf("Wizard: Could not add log, no active tasks [%s: %s]", msg.level, msg.message)
		}
		return sm, updateListener
	case updateCurrentTaskCompletedTitle:
		log.Debug().Msgf("Wizard: Update current task completed title -> %s", msg)
		if taskLen := len(sm.tasks); taskLen > 0 {
			sm.tasks[taskLen-1].completedTitle = string(msg)
		}
		return sm, updateListener
	case InputCompleted:
		log.Debug().Msgf("Wizard: Input completed")
		sm.inputResultChan <- sm.inputModel
		sm.inputModel = nil
		return sm, updateListener
	case tea.Model:
		log.Debug().Msgf("Wizard: Input component injected")
		var cmd tea.Cmd
		sm.inputModel, cmd = msg.Update(tea.WindowSizeMsg{Width: sm.width, Height: sm.height})
		return sm, tea.Sequence(updateListener, cmd)
	case tea.KeyMsg:
		log.Debug().Msgf("Wizard: Received keystroke [%s]", msg.String())

		switch msg.Type {
		case tea.KeyCtrlC:
			sm.err = ErrInterrupted
			sm.quitting = true
			return sm, tea.Quit
		default:
			if sm.inputModel != nil {
				newInputModel, inputCmd := sm.inputModel.Update(msg)
				sm.inputModel = newInputModel
				return sm, inputCmd
			}
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
	var buffer strings.Builder
	for taskIndex, task := range sm.tasks {
		if taskIndex > 0 {
			buffer.WriteString("\n")
		}
		for _, message := range task.logs {
			switch message.level {
			case log.InfoLevel:
				buffer.WriteString(wrap.String(InfoLog(message.message), sm.width) + "\n")
			case log.WarnLevel:
				buffer.WriteString(wrap.String(WarnLog(message.message), sm.width) + "\n")
			default:
				log.Warn().Str("log", message.message).Msg("Wizard: log level not set")
				buffer.WriteString(wrap.String("[LEVEL NOT SET] "+message.message, sm.width) + "\n")
			}
		}
		if task.completed {
			buffer.WriteString(fmt.Sprintf("%s %s\n", checkMark, bold.Render(wrap.String(task.completedTitle, sm.width))))
		} else if sm.inputModel != nil {
			// show editing icon if an input component has been injected
			buffer.WriteString(fmt.Sprintf("%s%s\n", "📝 ", bold.Render(wrap.String(task.title, sm.width))))
		} else {
			// show spinner for incomplete tasks
			buffer.WriteString(fmt.Sprintf("%s%s\n", sm.model.View(), bold.Render(wrap.String(task.title, sm.width))))
		}
	}

	log.Trace().Msgf("sm.tasks: %v", sm.tasks)
	if sm.inputModel != nil {
		buffer.WriteString(sm.inputModel.View())
	}
	return buffer.String()
}
