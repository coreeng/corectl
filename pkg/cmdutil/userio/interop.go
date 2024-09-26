package userio

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type InputCompleted struct {
	model tea.Model
}

type nonInteractiveHandler struct {
	streams *IOStreams
}

func InfoLog(message string) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("123"))
	return fmt.Sprintf("%s %s", style.Render("INFO:"), message)
}
func WarnLog(message string) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("227"))
	messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("228"))
	return fmt.Sprintf("%s %s", style.Render("WARN:"), messageStyle.Render(message))
}

func (nonInteractiveHandler) Done() {}
func (nih nonInteractiveHandler) Info(message string) {
	_, _ = nih.streams.outRaw.Write([]byte(InfoLog(message) + "\n"))
}
func (nih nonInteractiveHandler) Warn(message string) {
	_, _ = nih.streams.outRaw.Write([]byte(WarnLog(message) + "\n"))
}
func (nih nonInteractiveHandler) SetTask(title string, completedTitle string) {
	nih.Info(fmt.Sprintf("[%s]", lipgloss.NewStyle().Bold(true).Render(title)))
}
func (nih nonInteractiveHandler) SetCurrentTaskCompletedTitle(completedTitle string) {
	nih.Info(fmt.Sprintf("[%s]", lipgloss.NewStyle().Bold(true).Render(completedTitle)))
}
func (nih nonInteractiveHandler) SetInputModel(message tea.Model) tea.Model {
	panic("cannot take input in non-interactive mode")
}
