package userio

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/coreeng/corectl/pkg/cmdutil/userio/wizard"
)

type nonInteractiveHandler struct {
	streams *IOStreams
	styles  nonInteractiveStyles
}

func InfoLog(message string) string {
	style := newNonInteractiveStyles()
	return fmt.Sprintf("%s %s", style.infoStyle.Render("INFO:"), message)
}
func WarnLog(message string) string {
	style := newNonInteractiveStyles()
	return fmt.Sprintf("%s %s", style.warnHeadingStyle.Render("WARN:"), style.warnMessageStyle.Render(message))
}

func (nonInteractiveHandler) Done() {}
func (nonInteractiveHandler) OnQuit(model tea.Model, msg tea.Msg) tea.Msg {
	panic("cannot take input in non-interactive mode")
}

func (nih nonInteractiveHandler) Info(message string) {
	_, _ = nih.streams.outRaw.Write([]byte(InfoLog(message) + "\n"))
}
func (nih nonInteractiveHandler) Warn(message string) {
	_, _ = nih.streams.outRaw.Write([]byte(WarnLog(message) + "\n"))
}
func (nih nonInteractiveHandler) SetTask(title string, _ string) {
	nih.Info(fmt.Sprintf("[%s]", nih.styles.bold.Render(title)))
}
func (nih nonInteractiveHandler) SetCurrentTaskCompletedTitle(completedTitle string) {
	nih.Info(fmt.Sprintf("[%s %s]", nih.styles.status.Render(wizard.TaskStatusSuccess), nih.styles.bold.Render(completedTitle)))
}
func (nih nonInteractiveHandler) SetCurrentTaskCompletedTitleWithStatus(completedTitle string, status wizard.TaskStatus) {
	nih.Info(fmt.Sprintf("[%s %s]", nih.styles.status.Render(status), nih.styles.bold.Render(completedTitle)))
}
func (nih nonInteractiveHandler) SetInputModel(message tea.Model) tea.Model {
	panic("cannot take input in non-interactive mode")
}
