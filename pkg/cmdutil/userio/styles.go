package userio

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/coreeng/corectl/pkg/cmdutil/userio/wizard"
)

var (
	blueColor = lipgloss.Color("51")
)

type styles struct {
	title        lipgloss.Style
	item         lipgloss.Style
	selectedItem lipgloss.Style
	pagination   lipgloss.Style
	help         lipgloss.Style
	suggestion   lipgloss.Style
	bold         lipgloss.Style

	err  lipgloss.Style
	info lipgloss.Style
	warn lipgloss.Style
}

func newStyles(renderer *lipgloss.Renderer) *styles {
	niStyles := NewNonInteractiveStyles()

	return &styles{
		title:        renderer.NewStyle().MarginLeft(2),
		item:         renderer.NewStyle().PaddingLeft(2),
		selectedItem: renderer.NewStyle().Foreground(blueColor),
		pagination:   renderer.NewStyle().PaddingLeft(4),
		help:         renderer.NewStyle().Padding(1, 0, 0, 4),
		suggestion:   renderer.NewStyle().Faint(true),
		bold:         renderer.NewStyle().Bold(true),

		err:  niStyles.WarnMessageStyle,
		info: niStyles.InfoStyle,
		warn: niStyles.WarnMessageStyle,
	}
}

type NonInteractiveStyles struct {
	InfoStyle         lipgloss.Style
	ErrorHeadingStyle lipgloss.Style
	ErrorMessageStyle lipgloss.Style
	WarnHeadingStyle  lipgloss.Style
	WarnMessageStyle  lipgloss.Style
	Bold              lipgloss.Style
	Status            wizard.TaskStatusStyle
}

func NewNonInteractiveStyles() NonInteractiveStyles {
	styles := wizard.DefaultStyles()
	return NonInteractiveStyles{
		InfoStyle:         styles.InfoLogBody,
		ErrorHeadingStyle: styles.ErrorLogHeading,
		ErrorMessageStyle: styles.ErrorLogBody,
		WarnHeadingStyle:  styles.WarnLogHeading,
		WarnMessageStyle:  styles.WarnLogBody,
		Bold:              styles.Bold,
		Status:            styles.Marks,
	}
}
