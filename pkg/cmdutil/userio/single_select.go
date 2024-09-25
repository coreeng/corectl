package userio

import (
	"fmt"
	"io"
	"slices"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

const blankListHeight = 6

type SingleSelect struct {
	Prompt          string
	Items           []string
	PreselectedItem string
}

func (op *SingleSelect) GetInput(streams IOStreams) (string, error) {
	items := make([]list.Item, len(op.Items))
	for i, it := range op.Items {
		items[i] = item(it)
	}

	m := list.New(items, itemDelegate{streams.styles}, -1, len(items)+blankListHeight)
	m.SetShowStatusBar(false)
	m.SetShowPagination(true)
	m.SetFilteringEnabled(true)
	m.DisableQuitKeybindings()
	m.Title = op.Prompt
	m.Styles.Title = streams.styles.title
	m.Styles.PaginationStyle = streams.styles.pagination
	m.Styles.HelpStyle = streams.styles.help
	m.InfiniteScrolling = true

	if op.PreselectedItem != "" {
		preselectedIndex := slices.Index(op.Items, op.PreselectedItem)
		if preselectedIndex >= 0 {
			m.Select(preselectedIndex)
		}
	}

	model := singleSelectModel{
		prompt:   op.Prompt,
		model:    m,
		embedded: false,
	}

	// Allow nesting inside other components
	var result tea.Model
	var err error
	if streams.CurrentHandler != nil {
		model.embedded = true
		result = streams.CurrentHandler.SetInputModel(model)
	} else {
		result, err = streams.execute(model, nil)
		if err != nil {
			return "", err
		}
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
	embedded bool
}

func (m singleSelectModel) Init() tea.Cmd {
	return nil
}

func (m singleSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Height > m.model.Height() {
			m.model.SetHeight(min(msg.Height, len(m.model.Items())+blankListHeight))
		} else {
			m.model.SetHeight(msg.Height)
		}
		newListModel, cmd := m.model.Update(msg)
		m.model = newListModel
		return m, cmd

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.choice = nil
			m.err = ErrInterrupted
			if m.embedded {
				return m, func() tea.Msg {
					return InputCompleted{model: m}
				}
			} else {
				m.quitting = true
				return m, tea.Quit
			}
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
			if m.embedded {
				return m, func() tea.Msg {
					return InputCompleted{model: m}
				}
			} else {
				return m, tea.Quit
			}
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

type itemDelegate struct {
	styles *styles
}

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
		_, _ = fmt.Fprint(w, d.styles.selectedItem.Render("> "+str))
	} else {
		_, _ = fmt.Fprint(w, d.styles.item.Render(str))
	}
}
