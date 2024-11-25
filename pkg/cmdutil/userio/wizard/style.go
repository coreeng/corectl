package wizard

import "github.com/charmbracelet/lipgloss"

type Styles struct {
	Bold            lipgloss.Style
	Spinner         lipgloss.Style
	InfoLogHeading  lipgloss.Style
	InfoLogBody     lipgloss.Style
	WarnLogHeading  lipgloss.Style
	WarnLogBody     lipgloss.Style
	ErrorLogHeading lipgloss.Style
	ErrorLogBody    lipgloss.Style
	Marks           TaskStatusStyle
}

type TaskStatusStyle struct {
	Success lipgloss.Style
	Error   lipgloss.Style
	Skipped lipgloss.Style
}

func (s TaskStatusStyle) Render(status TaskStatus) string {
	switch status {
	case taskStatusUnknown:
		panic("unknown task status should not be rendered")
	case TaskStatusSuccess:
		return s.Success.Render()
	case TaskStatusError:
		return s.Error.Render()
	case TaskStatusSkipped:
		return s.Skipped.Render()
	default:
		panic("unknown task status")
	}
}

func DefaultStyles() Styles {
	return Styles{
		Bold:            lipgloss.NewStyle().Bold(true),
		Spinner:         lipgloss.NewStyle().Foreground(lipgloss.Color("#0404ff")), // CECG Blue
		InfoLogHeading:  lipgloss.NewStyle().Foreground(lipgloss.Color("051")),
		InfoLogBody:     lipgloss.NewStyle().Foreground(lipgloss.Color("159")),
		WarnLogHeading:  lipgloss.NewStyle().Foreground(lipgloss.Color("227")),
		WarnLogBody:     lipgloss.NewStyle().Foreground(lipgloss.Color("228")),
		ErrorLogHeading: lipgloss.NewStyle().Foreground(lipgloss.Color("203")),
		ErrorLogBody:    lipgloss.NewStyle().Foreground(lipgloss.Color("210")),
		Marks:           DefaultMarks(),
	}
}

func DefaultMarks() TaskStatusStyle {
	return TaskStatusStyle{
		Success: lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓"),
		Error:   lipgloss.NewStyle().Foreground(lipgloss.Color("9")).SetString("✗"),
		Skipped: lipgloss.NewStyle().Foreground(lipgloss.Color("228")).SetString("⚠"),
	}
}
