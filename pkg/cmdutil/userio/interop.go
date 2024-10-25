package userio

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/coreeng/corectl/pkg/cmdutil/userio/wizard"
)

type nonInteractiveHandler struct {
	streams *IOStreams
	styles  NonInteractiveStyles
}

func (nih nonInteractiveHandler) InfoLog(message string) string {
	return fmt.Sprintf("%s %s", nih.styles.InfoStyle.Render("INFO:"), message)
}
func (nih nonInteractiveHandler) WarnLog(message string) string {
	return fmt.Sprintf("%s %s", nih.styles.WarnHeadingStyle.Render("WARN:"), nih.styles.WarnMessageStyle.Render(message))
}

func (*nonInteractiveHandler) Done()            {}
func (*nonInteractiveHandler) Abort(err string) {}
func (nonInteractiveHandler) OnQuit(model tea.Model, msg tea.Msg) tea.Msg {
	panic("cannot take input in non-interactive mode")
}

func (nih nonInteractiveHandler) Info(message string) {
	_, _ = nih.streams.stdoutRaw.Write([]byte(nih.InfoLog(message) + "\n"))
}
func (nih nonInteractiveHandler) Warn(message string) {
	_, _ = nih.streams.stdoutRaw.Write([]byte(nih.WarnLog(message) + "\n"))
}
func (nih nonInteractiveHandler) SetTask(title string, _ string) {
	nih.Info(fmt.Sprintf("[%s]", nih.styles.Bold.Render(title)))
}
func (nih nonInteractiveHandler) SetCurrentTaskCompletedTitle(completedTitle string) {
	nih.Info(fmt.Sprintf("[%s %s]", nih.styles.Status.Render(wizard.TaskStatusSuccess), nih.styles.Bold.Render(completedTitle)))
}
func (nih nonInteractiveHandler) SetCurrentTaskCompletedTitleWithStatus(completedTitle string, status wizard.TaskStatus) {
	nih.Info(fmt.Sprintf("[%s %s]", nih.styles.Status.Render(status), nih.styles.Bold.Render(completedTitle)))
}
func (nih nonInteractiveHandler) SetInputModel(message tea.Model) tea.Model {
	panic("cannot take input in non-interactive mode")
}
