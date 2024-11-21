package wizard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/coreeng/corectl/pkg/logger"
	"github.com/muesli/reflow/wrap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

type TaskStatus string

const (
	taskStatusUnknown TaskStatus = "unknown"
	TaskStatusSuccess TaskStatus = "success"
	TaskStatusError   TaskStatus = "error"
	TaskStatusSkipped TaskStatus = "skipped"
)

type updateCurrentTaskCompletedTitle struct {
	title  string
	status TaskStatus
}
type logMsg struct {
	message string
	level   zapcore.Level
}
type quittingMsg bool
type taskComplete bool
type doneMsg bool
type errorMsg string

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
		completed:          false,
	}
	return model, &handler, doneChannel
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
		logger.Warn("Wizard: Marking task complete, but no tasks found")
	} else {
		logger.Debug("Wizard: Marking task complete",
			zap.String("title", task.completedTitle))
		task.status = TaskStatusSuccess
		task.completed = true
	}

	return task
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updateListener := m.ReceiveUpdateMessages

	var newInputModel tea.Model
	var inputCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		logger.Debug("Wizard: Received window size",
			zap.String("type", fmt.Sprintf("%T", msg)),
			zap.Int("width", msg.Width),
			zap.Int("height", msg.Height))
		m.width = msg.Width
		m.height = msg.Height
		if m.inputModel != nil {
			newInputModel, inputCmd := m.inputModel.Update(msg)
			m.inputModel = newInputModel
			return m, inputCmd
		}
		return m, nil
	case quittingMsg:
		m.quitting = true
		logger.Debug("Wizard: Received message",
			zap.String("type", fmt.Sprintf("%T", msg)))
		if messageLen := len(m.messageChannel); messageLen > 0 {
			logger.Debug("Wizard: postponing shutdown",
				zap.Int("remaining_messages", messageLen))
			return m, tea.Sequence(updateListener, func() tea.Msg { return doneMsg(true) })
		}
		return m, tea.Quit
	case taskComplete:
		m.markLatestTaskComplete()
		return m, updateListener
	case doneMsg:
		m.markLatestTaskComplete()
		return m, func() tea.Msg { return quittingMsg(true) }
	case errorMsg:
		logger.Debug("Wizard: Received message",
			zap.String("type", fmt.Sprintf("%T", msg)))
		if messageLen := len(m.messageChannel); messageLen > 0 {
			logger.Debug("Wizard: postponing shutdown",
				zap.Int("remaining_messages", messageLen))
			return m, updateListener
		} else {
			m.quitting = true
			return m, tea.Sequence(
				func() tea.Msg {
					return updateCurrentTaskCompletedTitle{
						title:  string(msg),
						status: TaskStatusError,
					}
				},
				func() tea.Msg { return quittingMsg(true) },
			)
		}
	case task:
		m.markLatestTaskComplete()
		logger.Debug("Wizard: New task",
			zap.String("title", msg.title))
		m.tasks = append(m.tasks, msg)
		return m, updateListener
	case logMsg:
		logger.Debug("Wizard: Log received",
			zap.String("level", msg.level.String()),
			zap.String("message", msg.message))
		latestTask := m.getLatestTask()
		if latestTask == nil || latestTask.completed {
			m.tasks = append(m.tasks, task{
				logs: []logMsg{msg},
			})
		} else {
			latestTask.logs = append(latestTask.logs, msg)
		}
		if msg.level == zapcore.DebugLevel {
			return m, tea.Sequence(tea.Println(msg.message), updateListener)
		}
		return m, updateListener
	case updateCurrentTaskCompletedTitle:
		logger.Debug("Wizard: Update current task completed title",
			zap.String("title", msg.title),
			zap.Any("status", msg.status))
		latestTask := m.getLatestTask()
		if latestTask != nil && !latestTask.isAnonymous() {
			latestTask.completedTitle = msg.title
			latestTask.status = msg.status
			latestTask.completed = true
		} else {
			logger.Panic("Wizard: unable to update task completed title",
				zap.Bool("isNil", latestTask == nil),
				zap.Bool("isAnonymous", latestTask.isAnonymous()))
		}
		return m, updateListener
	case InputCompleted:
		logger.Debug("Wizard: Input completed")
		m.inputResultChannel <- m.inputModel
		m.inputModel = nil
		return m, updateListener
	case tea.Model:
		logger.Debug("Wizard: Input component injected")
		var cmd tea.Cmd
		m.inputModel, cmd = msg.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, tea.Sequence(updateListener, cmd)
	case tea.KeyMsg:
		logger.Debug("Wizard: Received keystroke",
			zap.String("key", msg.String()))
		switch msg.Type {
		case tea.KeyCtrlC:
			if m.inputModel != nil {
				newInputModel, inputCmd = m.inputModel.Update(msg)
				m.inputModel = newInputModel
				return m, inputCmd
			}
			m.quitting = true
			return m, tea.Quit
		}
	}

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

func (m Model) ErrorLog(message string) string {
	return fmt.Sprintf("%s %s", m.Styles.ErrorLogHeading.Render("ERROR:"), m.Styles.ErrorLogBody.Render(message))
}

func (m Model) View() string {
	var buffer strings.Builder
	for _, task := range m.tasks {
		for _, message := range task.logs {
			buffer.WriteString(wrap.String(m.generateLog(message.message, message.level), m.width) + "\n")
		}
		if m.inputModel != nil {
			// show editing icon if an input component has been injected
			buffer.WriteString(fmt.Sprintf("%s%s\n", "📝 ", m.Styles.Bold.Render(wrap.String(task.title, m.width))))
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

	logger.Debug("sm.tasks",
		zap.Any("tasks", m.tasks))
	if m.inputModel != nil {
		buffer.WriteString(m.inputModel.View())
	}
	return buffer.String()
}

func (m Model) generateLog(message string, level zapcore.Level) string {
	switch level {
	case zapcore.InfoLevel:
		return m.InfoLog(message)
	case zapcore.WarnLevel:
		return m.WarnLog(message)
	case zapcore.ErrorLevel:
		return m.ErrorLog(message)
	default:
		logger.Warn("Wizard: log level not set",
			zap.String("log", message),
			zap.String("level", level.String()))
		return "[LEVEL NOT SET] " + message
	}
}
