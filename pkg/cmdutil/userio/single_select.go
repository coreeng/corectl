package userio

import (
	"fmt"
	"io"
	"slices"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/coreeng/corectl/pkg/logger"
)

const blankListHeight = 6

type SingleSelect struct {
	Prompt          string
	Items           []string
	DisplayedItems  []string
	PreselectedItem string
}

func (op *SingleSelect) GetInput(streams IOStreams) (string, error) {
	items := make([]list.Item, len(op.Items))
	for i, it := range op.Items {
		var disp string
		if op.DisplayedItems == nil || len(op.DisplayedItems) <= i {
			disp = it
		} else {
			disp = op.DisplayedItems[i]
		}

		items[i] = item{
			value:          it,
			displayedValue: disp,
		}
	}

	m := list.New(items, itemDelegate{streams.styles}, -1, len(items)+blankListHeight)
	m.SetShowStatusBar(false)
	m.SetShowPagination(true)
	m.SetFilteringEnabled(true)
	m.DisableQuitKeybindings()
	m.SetWidth(80)
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
		prompt: op.Prompt,
		model:  m,
	}

	result, err := streams.Execute(model)
	if err != nil {
		return "", err
	}

	sSResult := result.(singleSelectModel)
	choice := ""
	if sSResult.choice != nil {
		choice = string(sSResult.choice.value)
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

			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			logger.Debug().Msg("SingleSelect: received enter")
			if m.model.FilterState() == list.Filtering {
				break
			}
			it, ok := m.model.SelectedItem().(item)
			logger.Debug().Msgf("SingleSelect: selected item is %s", it)
			if !ok {
				return m, nil
			}
			m.err = nil
			m.choice = &it
			m.quitting = true
			logger.Debug().Msg("SingleSelect: sending tea.Quit")
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
		"> " + string(m.choice.value) + "\n\n"
}

type item struct {
	value          string
	displayedValue string
}

func (i item) Title() string {
	return i.displayedValue
}

func (i item) FilterValue() string {
	return i.value
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

	if index == m.Index() {
		_, _ = fmt.Fprint(w, d.styles.selectedItem.Render("> "+it.displayedValue))
	} else {
		_, _ = fmt.Fprint(w, d.styles.item.Render(it.displayedValue))
	}
}
