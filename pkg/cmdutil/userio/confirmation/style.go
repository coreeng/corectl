package confirmation

import (
	"github.com/charmbracelet/lipgloss"
)

type styles struct {
	subtle       lipgloss.AdaptiveColor
	dialogBox    lipgloss.Style
	button       lipgloss.Style
	activeButton lipgloss.Style
	question     lipgloss.Style
}

func defaultStyles() styles {
	subtle := lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	button := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFF7DB")).
		Background(lipgloss.Color("#888B7E")).
		Bold(true).
		Padding(0, 2).
		Margin(2, 2, 1, 1)

	return styles{
		subtle: subtle,
		dialogBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(1, 0, 0, 0).
			BorderTop(true).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true),
		button: button,
		activeButton: button.
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.Color("#0404ff")).
			Bold(true).
			Padding(0, 2).
			Margin(2, 2, 1, 1),
		question: lipgloss.NewStyle().Width(50).Align(lipgloss.Center),
	}
}
