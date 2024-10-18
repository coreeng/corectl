package wizard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wrap"
	"github.com/phuslu/log"
)

type Model struct {
	messageChannel     <-chan tea.Msg
	inputResultChannel chan<- tea.Model
	spinner            spinner.Model
	quitting           bool
	tasks              []task
	inputModel         tea.Model
	height             int
	width              int
	Styles             Styles
}

type InputCompleted struct {
	model tea.Model
}

type task struct {
	title          string
	completedTitle string
	status         TaskStatus
	completed      bool
	logs           []logMsg
}

func (t task) isAnonymous() bool {
	return t.title == "" && t.completedTitle == ""
}

type TaskStatus uint

const (
	taskStatusUnknown TaskStatus = iota
	TaskStatusSuccess
	TaskStatusError
	TaskStatusSkipped
)

type updateCurrentTaskCompletedTitle struct {
	title  string
	status TaskStatus
}
type logMsg struct {
	message string
	level   log.Level
}
type doneMsg bool

func New() (Model, Handler, chan<- bool) {
	doneChannel := make(chan bool)
	messageChannel := make(chan tea.Msg)
	inputResultChannel := make(chan tea.Model)
	styles := DefaultStyles()
	spinnerModel := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(styles.Spinner),
	)
	model := Model{
		messageChannel:     messageChannel,
		inputResultChannel: inputResultChannel,
		spinner:            spinnerModel,
		Styles:             styles,
		tasks:              []task{},
	}
	handler := asyncHandler{
		messageChannel:     messageChannel,
		inputResultChannel: inputResultChannel,
		doneChannel:        doneChannel,
	}
	return model, handler, doneChannel
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.ReceiveUpdateMessages,
	)
}

func (m Model) ReceiveUpdateMessages() tea.Msg {
	message := <-m.messageChannel
	return message
}

func (m Model) getLatestTask() *task {
	if len(m.tasks) > 0 {
		return &m.tasks[len(m.tasks)-1]
	} else {
		return nil
	}
}

func (m Model) markLatestTaskComplete() *task {
	task := m.getLatestTask()
	if task == nil {
		log.Warn().Msgf("Wizard: Marking task complete, but no tasks found")
	} else {
		log.Debug().Msgf("Wizard: Marking task complete: %s", task.completedTitle)
		task.completed = true
	}

	return task
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updateListener := m.ReceiveUpdateMessages

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		log.Debug().Msgf("Wizard: Received [%T], (w:%d, h:%d)", msg, msg.Width, msg.Height)
		m.width = msg.Width
		m.height = msg.Height
		if m.inputModel != nil {
			newInputModel, inputCmd := m.inputModel.Update(msg)
			m.inputModel = newInputModel
			return m, inputCmd
		}
		return m, nil
	case doneMsg:
		log.Debug().Msgf("Wizard: Received [%T]", msg)
		if messageLen := len(m.messageChannel); messageLen > 0 {
			log.Debug().Msgf("Wizard: Message channel still has %d items, postponing shutdown", messageLen)
			return m, tea.Sequence(updateListener, func() tea.Msg { return doneMsg(true) })
		} else {
			m.markLatestTaskComplete()
			m.quitting = true
			return m, tea.Quit
		}
	case task:
		m.markLatestTaskComplete()
		log.Debug().Msgf("Wizard: New task: %s", msg.title)
		m.tasks = append(m.tasks, msg)
		return m, updateListener
	case logMsg:
		log.Debug().Msgf("Wizard: Log received %s: %s", msg.level, msg.message)
		latestTask := m.getLatestTask()
		if latestTask == nil || latestTask.completed {
			m.tasks = append(m.tasks, task{
				logs: []logMsg{msg},
			})
		} else {
			latestTask.logs = append(latestTask.logs, msg)
		}
		return m, updateListener
	case updateCurrentTaskCompletedTitle:
		log.Debug().Msgf("Wizard: Update current task completed title -> %s", msg.title)
		latestTask := m.getLatestTask()
		if latestTask != nil && !latestTask.isAnonymous() {
			latestTask.completedTitle = msg.title
			latestTask.status = msg.status
			latestTask.completed = true
		} else {
			log.Panic().Bool("isNil", latestTask == nil).Bool("isAnonymous", latestTask.isAnonymous()).
				Msg("Wizard: unable to update task completed title")
		}
		return m, updateListener
	case InputCompleted:
		log.Debug().Msgf("Wizard: Input completed")
		m.inputResultChannel <- m.inputModel
		m.inputModel = nil
		return m, updateListener
	case tea.Model:
		log.Debug().Msgf("Wizard: Input component injected")
		var cmd tea.Cmd
		m.inputModel, cmd = msg.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, tea.Sequence(updateListener, cmd)
	case tea.KeyMsg:
		log.Debug().Msgf("Wizard: Received keystroke [%s]", msg.String())

		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit
		default:
			if m.inputModel != nil {
				newInputModel, inputCmd := m.inputModel.Update(msg)
				m.inputModel = newInputModel
				return m, inputCmd
			}
		}
	}
	var newInputModel tea.Model
	var inputCmd tea.Cmd
	if m.inputModel != nil {
		newInputModel, inputCmd = m.inputModel.Update(msg)
		m.inputModel = newInputModel
	}
	newSpinnerModel, cmd := m.spinner.Update(msg)
	m.spinner = newSpinnerModel
	return m, tea.Batch(cmd, inputCmd)
}

func (m Model) InfoLog(message string) string {
	return fmt.Sprintf("%s %s", m.Styles.InfoLogHeading.Render("INFO:"), m.Styles.InfoLogBody.Render(message))
}

func (m Model) WarnLog(message string) string {
	return fmt.Sprintf("%s %s", m.Styles.WarnLogHeading.Render("WARN:"), m.Styles.WarnLogBody.Render(message))
}

func (m Model) View() string {
	var buffer strings.Builder
	for taskIndex, task := range m.tasks {
		if taskIndex > 0 {
			buffer.WriteString("\n")
		}
		for _, message := range task.logs {
			buffer.WriteString(wrap.String(m.generateLog(message.message, message.level), m.width) + "\n")
		}
		if m.inputModel != nil {
			// show editing icon if an input component has been injected
			buffer.WriteString(fmt.Sprintf(
				"%s%s\n",
				"📝 ",
				m.Styles.Bold.Render(wrap.String(task.title, m.width)),
			))
		} else if task.isAnonymous() {
			continue
		} else if task.completed {
			buffer.WriteString(fmt.Sprintf(
				"%s %s\n",
				m.Styles.Marks.Render(task.status),
				m.Styles.Bold.Render(wrap.String(task.completedTitle, m.width)),
			))
		} else {
			// show spinner for incomplete tasks
			buffer.WriteString(fmt.Sprintf(
				"%s%s\n",
				m.spinner.View(),
				m.Styles.Bold.Render(wrap.String(task.title, m.width)),
			))
		}
	}

	log.Trace().Msgf("sm.tasks: %v", m.tasks)
	if m.inputModel != nil {
		buffer.WriteString(m.inputModel.View())
	}
	return buffer.String()
}

func (m Model) generateLog(message string, level log.Level) string {
	switch level {
	case log.InfoLevel:
		return m.InfoLog(message)
	case log.WarnLevel:
		return m.WarnLog(message)
	default:
		log.Warn().Str("log", message).Str("level", level.String()).Msg("Wizard: log level not set")
		return "[LEVEL NOT SET] " + message
	}
}
