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
	model              spinner.Model
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
	completed      bool
	logs           []logMsg
}

type updateCurrentTaskCompletedTitle string
type logMsg struct {
	message string
	level   log.Level
}
type doneMsg bool

func New() (Model, asyncHandler) {
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
		model:              spinnerModel,
		Styles:             styles,
		tasks:              []task{},
	}
	handler := asyncHandler{
		messageChannel:     messageChannel,
		inputResultChannel: inputResultChannel,
		doneReceiveChannel: doneChannel,
		DoneSendChannel:    doneChannel,
	}
	return model, handler
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.model.Tick,
		m.ReceiveUpdateMessages,
	)
}

func (m Model) ReceiveUpdateMessages() tea.Msg {
	message := <-m.messageChannel
	return message
}

func (m Model) getLatestTask() *task {
	var task *task
	if len(m.tasks) > 0 {
		task = &m.tasks[len(m.tasks)-1]
	}
	return task
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
			return m, updateListener
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
		if len(m.tasks) > 0 {
			log.Debug().Msgf("Wizard: Log received %s: %s", msg.level, msg.message)
			// Adds logs as children of the most recent task
			latestTask := m.getLatestTask()
			latestTask.logs = append(latestTask.logs, msg)
		} else {
			log.Warn().Msgf("Wizard: Could not add log, no active tasks [%s: %s]", msg.level, msg.message)
		}
		return m, updateListener
	case updateCurrentTaskCompletedTitle:
		log.Debug().Msgf("Wizard: Update current task completed title -> %s", msg)
		if taskLen := len(m.tasks); taskLen > 0 {
			m.tasks[taskLen-1].completedTitle = string(msg)
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
	newSpinnerModel, cmd := m.model.Update(msg)
	m.model = newSpinnerModel
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
			switch message.level {
			case log.InfoLevel:
				buffer.WriteString(wrap.String(m.InfoLog(message.message), m.width) + "\n")
			case log.WarnLevel:
				buffer.WriteString(wrap.String(m.WarnLog(message.message), m.width) + "\n")
			default:
				log.Warn().Str("log", message.message).Msg("Wizard: log level not set")
				buffer.WriteString(wrap.String("[LEVEL NOT SET] "+message.message, m.width) + "\n")
			}
		}
		if task.completed {
			buffer.WriteString(fmt.Sprintf("%s %s\n", m.Styles.CheckMark, m.Styles.Bold.Render(wrap.String(task.completedTitle, m.width))))
		} else if m.inputModel != nil {
			// show editing icon if an input component has been injected
			buffer.WriteString(fmt.Sprintf("%s%s\n", "📝 ", m.Styles.Bold.Render(wrap.String(task.title, m.width))))
		} else {
			// show spinner for incomplete tasks
			buffer.WriteString(fmt.Sprintf("%s%s\n", m.model.View(), m.Styles.Bold.Render(wrap.String(task.title, m.width))))
		}
	}

	log.Trace().Msgf("sm.tasks: %v", m.tasks)
	if m.inputModel != nil {
		buffer.WriteString(m.inputModel.View())
	}
	return buffer.String()
}
