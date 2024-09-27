package userio

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	blueColor = lipgloss.Color("51")
	redColor  = lipgloss.Color("124")
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
}

func newStyles(renderer *lipgloss.Renderer) *styles {
	return &styles{
		title:        renderer.NewStyle().MarginLeft(2),
		item:         renderer.NewStyle().PaddingLeft(2),
		selectedItem: renderer.NewStyle().Foreground(blueColor),
		pagination:   renderer.NewStyle().PaddingLeft(4),
		help:         renderer.NewStyle().Padding(1, 0, 0, 4),
		suggestion:   renderer.NewStyle().Faint(true),
		bold:         renderer.NewStyle().Bold(true),

		err: renderer.NewStyle().
			MarginLeft(2).
			Foreground(redColor),
		info: renderer.NewStyle().
			Foreground(blueColor),
	}
}
