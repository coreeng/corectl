package confirmation

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
)

type model struct {
	confirmation bool
	quitting     bool
	question     string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		m.confirmation = key == "y" || key == "Y"
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		answer := "no"
		if m.confirmation {
			answer = "yes"
		}
		return fmt.Sprintf("%s -> %s\n", m.question, answer)
	} else {
		return fmt.Sprintf("%s (y/N)\n", m.question)
	}
}

func GetInput(streams userio.IOStreams, question string) (bool, error) {
	modelInstance := model{question: question}
	result, err := streams.Execute(modelInstance)
	if err != nil {
		return false, err
	}
	return result.(model).confirmation, nil
}
