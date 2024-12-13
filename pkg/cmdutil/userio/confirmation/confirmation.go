package confirmation

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/logger"
)

type model struct {
	confirmation bool
	quitting     bool
	question     string
	help         help.Model
	styles       styles
}

func newModel(question string) model {
	return model{
		confirmation: true,
		quitting:     false,
		question:     question,
		help:         help.New(),
		styles:       defaultStyles(),
	}
}

type keyMap struct {
	Yes    key.Binding
	No     key.Binding
	Select key.Binding
	Quit   key.Binding
	Help   key.Binding
}

var (
	yesKeys        = key.WithKeys("left", "up")
	yesConfirmKeys = key.WithKeys("y")
	noKeys         = key.WithKeys("right", "down")
	noConfirmKeys  = key.WithKeys("n")
	selectKeys     = key.WithKeys("enter")
	helpKeys       = key.WithKeys("?")
)

var keys = keyMap{
	Yes: key.NewBinding(
		yesConfirmKeys,
		key.WithHelp("y", "yes"),
	),
	No: key.NewBinding(
		noConfirmKeys,
		key.WithHelp("n", "no"),
	),
	Select: key.NewBinding(
		selectKeys,
		key.WithHelp("enter", "select option"),
	),
	Help: key.NewBinding(
		helpKeys,
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "exit"),
	),
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Select, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Yes, k.No, k.Select}, // first column
		{k.Help, k.Quit},        // second column
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(yesKeys)):
			m.confirmation = true
		case key.Matches(msg, key.NewBinding(noKeys)):
			m.confirmation = false
		case key.Matches(msg, key.NewBinding(yesConfirmKeys)):
			m.confirmation = true
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(noConfirmKeys)):
			m.confirmation = false
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(selectKeys)):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(helpKeys)):
			m.help.ShowAll = !m.help.ShowAll
		}
		switch msg.Type {
		case tea.KeyCtrlC:
			logger.Debug().Msg("ConfirmationPrompt: exiting")
			m.quitting = true
			m.confirmation = false
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	var (
		yes, no, dialog string
	)

	question := m.styles.question.Render(m.question)
	activeButtonStyle := m.styles.activeButton

	if m.quitting {
		activeButtonStyle = activeButtonStyle.Border(lipgloss.RoundedBorder()).Margin(1, 1, 0, 1)
	}
	if m.confirmation {
		yes = activeButtonStyle.Render("Yes")
		no = m.styles.button.Render("No")
	} else {
		yes = m.styles.button.Render("Yes")
		no = activeButtonStyle.Render("No")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Top, yes, no)
	ui := lipgloss.JoinVertical(lipgloss.Center, question, buttons)
	dialog = m.styles.dialogBox.Render(ui)

	if m.quitting {
		return dialog + "\n"
	}
	return lipgloss.JoinVertical(lipgloss.Left, dialog, m.help.View(keys))
}

func GetInput(streams userio.IOStreams, question string) (bool, error) {
	modelInstance := newModel(question)

	result, err := streams.Execute(modelInstance)
	if err != nil {
		return false, err
	}
	return result.(model).confirmation, nil
}
