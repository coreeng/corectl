package wizard

import "github.com/charmbracelet/lipgloss"

type Styles struct {
	Bold           lipgloss.Style
	Spinner        lipgloss.Style
	InfoLogHeading lipgloss.Style
	WarnLogHeading lipgloss.Style
	InfoLogBody    lipgloss.Style
	WarnLogBody    lipgloss.Style
	CheckMark      lipgloss.Style
}

func DefaultStyles() Styles {
	return Styles{
		Bold:           lipgloss.NewStyle().Bold(true),
		Spinner:        lipgloss.NewStyle().Foreground(lipgloss.Color("#0404ff")), // CECG Blue
		InfoLogHeading: lipgloss.NewStyle().Foreground(lipgloss.Color("051")),
		InfoLogBody:    lipgloss.NewStyle().Foreground(lipgloss.Color("159")),
		WarnLogHeading: lipgloss.NewStyle().Foreground(lipgloss.Color("227")),
		WarnLogBody:    lipgloss.NewStyle().Foreground(lipgloss.Color("228")),
		CheckMark:      lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓"),
	}
}
