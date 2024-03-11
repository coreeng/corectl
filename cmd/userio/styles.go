package userio

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

var (
	blueColor = lipgloss.Color("51")
	redColor  = lipgloss.Color("124")
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(4).Foreground(blueColor)
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4)
	errorStyle        = lipgloss.NewStyle().
				MarginLeft(2).
				Foreground(redColor)

	infoStyle = lipgloss.NewStyle().
			MarginLeft(2).
			Foreground(blueColor)
)
