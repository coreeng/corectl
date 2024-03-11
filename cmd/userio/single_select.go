package userio

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"io"
	"strconv"
)

type SingleSelect struct {
	Prompt string
	Items  []string
}

func (op *SingleSelect) GetInput(streams IOStreams) (string, error) {
	items := make([]list.Item, len(op.Items))
	for i, it := range op.Items {
		items[i] = item(it)
	}

	m := list.New(items, itemDelegate{}, -1, len(items)+2)
	m.SetShowStatusBar(false)
	m.SetShowPagination(true)
	m.SetFilteringEnabled(true)
	m.DisableQuitKeybindings()
	m.Title = op.Prompt
	m.Styles.Title = titleStyle
	m.Styles.PaginationStyle = paginationStyle
	m.Styles.HelpStyle = helpStyle
	m.InfiniteScrolling = true

	model := singleSelectModel{
		prompt: op.Prompt,
		model:  m,
	}
	result, err := streams.execute(model)
	if err != nil {
		return "", err
	}
	sSResult := result.(singleSelectModel)
	choice := ""
	if sSResult.choice != nil {
		choice = string(*sSResult.choice)
	}

	return choice, sSResult.err
}

type singleSelectModel struct {
	prompt string
	model  list.Model

	choice   *item
	err      error
	quitting bool
}

func (m singleSelectModel) Init() tea.Cmd {
	return nil
}

func (m singleSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Height > m.model.Height() {
			m.model.SetHeight(min(msg.Height, len(m.model.Items())))
		} else {
			m.model.SetHeight(msg.Height)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.choice = nil
			m.err = ErrInterrupted
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			if m.model.FilterState() == list.Filtering {
				break
			}
			it, ok := m.model.SelectedItem().(item)
			if !ok {
				return m, nil
			}
			m.err = nil
			m.choice = &it
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.model, cmd = m.model.Update(msg)
	return m, cmd
}

func (m singleSelectModel) View() string {
	if !m.quitting {
		return m.model.View()
	}

	if m.choice == nil {
		return ""
	}
	return m.prompt + "\n" +
		"> " + string(*m.choice) + "\n\n"
}

type item string

func (i item) FilterValue() string {
	return string(i)
}

type itemDelegate struct{}

func (d itemDelegate) Height() int {
	return 1
}

func (d itemDelegate) Spacing() int {
	return 0
}

func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(item)
	if !ok {
		return
	}
	str := strconv.Itoa(index+1) + ". " + string(it)

	if index == m.Index() {
		fmt.Fprint(w, selectedItemStyle.Render(str))
	} else {
		fmt.Fprint(w, itemStyle.Render(str))
	}
}
