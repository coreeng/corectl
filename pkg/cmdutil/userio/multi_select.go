package userio

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type MultiSelect struct {
	Prompt         string
	Items          []string
	ValidateAndMap ValidateAndMap[[]string, []string]
}

func (op *MultiSelect) GetInput(streams IOStreams) ([]string, error) {
	items := make([]list.Item, len(op.Items))
	for i, it := range op.Items {
		items[i] = &multiSelectItem{
			value: it,
		}
	}

	m := list.New(items, multiSelectItemDelegate{streams.styles}, -1, len(items)+2)
	m.SetShowStatusBar(false)
	m.SetShowPagination(false)
	m.SetFilteringEnabled(true)
	m.SetShowHelp(false)
	m.DisableQuitKeybindings()
	m.Title = op.Prompt
	m.Styles.Title = streams.styles.title
	m.Styles.PaginationStyle = streams.styles.pagination
	m.Styles.HelpStyle = streams.styles.help
	m.InfiniteScrolling = true

	model := multiSelectModel{
		prompt:         op.Prompt,
		model:          m,
		validateAndMap: op.ValidateAndMap,
	}

	result, err := streams.execute(model, nil)
	if err != nil {
		return nil, err
	}

	mSResult := result.(multiSelectModel)

	return mSResult.choice, mSResult.err
}

type multiSelectModel struct {
	prompt         string
	model          list.Model
	validateAndMap ValidateAndMap[[]string, []string]
	styles         *styles

	choice   []string
	err      error
	quitting bool
}

func (m multiSelectModel) Init() tea.Cmd {
	return nil
}

func (m multiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Height > m.model.Height() {
			m.model.SetHeight(min(msg.Height, len(m.model.Items())+2))
		} else {
			m.model.SetHeight(msg.Height)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			if m.model.FilterState() == list.Filtering {
				break
			}
			m.err = ErrInterrupted
			m.quitting = true
			return m, tea.Quit
		case tea.KeySpace:
			if m.model.FilterState() == list.Filtering {
				break
			}
			it, ok := m.model.SelectedItem().(*multiSelectItem)
			if !ok {
				return m, nil
			}
			it.checked = !it.checked
			m.validateIfNeeded()
			return m, nil
		case tea.KeyEnter:
			m.validateIfNeeded()
			if m.err != nil {
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.model, cmd = m.model.Update(msg)
	return m, cmd
}

func (m *multiSelectModel) extractChoice() []string {
	var choice []string
	for _, i := range m.model.Items() {
		it, ok := i.(*multiSelectItem)
		if !ok {
			panic("Should never happen")
		}
		if it.checked {
			choice = append(choice, it.value)
		}
	}
	return choice
}

func (m *multiSelectModel) validateIfNeeded() {
	choice := m.extractChoice()
	if m.validateAndMap != nil {
		mappedChoice, err := m.validateAndMap(choice)
		choice = mappedChoice
		m.err = err
	} else {
		m.err = nil
	}
	m.choice = choice
}

func (m multiSelectModel) View() string {
	if !m.quitting {
		var s strings.Builder
		s.WriteString(m.model.View())
		s.WriteString("\n")
		if m.err != nil {
			s.WriteString(m.styles.err.Render(m.err.Error()))
			s.WriteString("\n")
		}
		return s.String()
	}

	choiceStr := strings.Join(m.choice, ", ")
	if choiceStr == "" {
		choiceStr = "no items selected"
	}
	return m.prompt + "\n" +
		"> " + choiceStr + "\n\n"
}

type multiSelectItem struct {
	value   string
	checked bool
}

func (i multiSelectItem) FilterValue() string {
	return i.value
}

type multiSelectItemDelegate struct {
	styles *styles
}

func (d multiSelectItemDelegate) Height() int {
	return 1
}

func (d multiSelectItemDelegate) Spacing() int {
	return 0
}

func (d multiSelectItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d multiSelectItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(*multiSelectItem)
	if !ok {
		return
	}

	var checkedSign string
	if it.checked {
		checkedSign = "[X] "
	} else {
		checkedSign = "[ ] "
	}
	str := checkedSign + strconv.Itoa(index+1) + ". " + it.value

	if index == m.Index() {
		_, _ = fmt.Fprint(w, d.styles.selectedItem.Render(str))
	} else {
		_, _ = fmt.Fprint(w, d.styles.item.Render(str))
	}
}
