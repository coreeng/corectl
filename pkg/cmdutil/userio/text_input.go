package userio

import (
	"errors"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
)

type ValidateTextAndMapFn[V interface{}] func(string) (V, error)

func Required(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return "", errors.New("empty input")
	}
	return s, nil
}

type TextInput[V interface{}] struct {
	Prompt         string
	InitialValue   string
	Placeholder    string
	ValidateAndMap ValidateTextAndMapFn[V]
}

func (ti *TextInput[V]) GetInput(streams IOStreams) (V, error) {
	if ti.ValidateAndMap == nil {
		panic("ValidateAndMap is required")
	}
	model := textinput.New()
	if streams.out.ColorProfile() != termenv.Ascii {
		model.Placeholder = ti.Placeholder
		model.PlaceholderStyle = streams.styles.suggestion
		model.CompletionStyle = streams.styles.suggestion
	}
	model.Focus()
	if ti.InitialValue != "" {
		model.SetValue(ti.InitialValue)
	}

	tiModel := textInputModel[V]{
		model:          model,
		prompt:         ti.Prompt,
		validateAndMap: ti.ValidateAndMap,
		styles:         streams.styles,
	}

	// Allow nesting inside other components
	var result tea.Model
	var err error
	if streams.CurrentHandler != nil {
		result = streams.CurrentHandler.SetInputModel(tiModel)
	} else {
		result, err = streams.execute(tiModel, nil)
		if err != nil {
			var noop V
			return noop, err
		}
	}

	tiModel = result.(textInputModel[V])
	return tiModel.result, tiModel.err
}

type textInputModel[V interface{}] struct {
	model          textinput.Model
	prompt         string
	validateAndMap ValidateTextAndMapFn[V]
	styles         *styles

	result   V
	quitting bool
	err      error
}

func (m textInputModel[V]) Init() tea.Cmd {
	return textinput.Blink
}

func (m textInputModel[V]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	m.model, cmd = m.model.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			v, err := m.validateAndMap(m.model.Value())
			m.err = err
			if err == nil {
				m.result = v
				m.model.Cursor.SetMode(cursor.CursorHide)
				m.quitting = true
				return m, tea.Quit
			}
		case tea.KeyCtrlC, tea.KeyEsc:
			m.model.Cursor.SetMode(cursor.CursorHide)
			m.err = ErrInterrupted
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, cmd
}

func (m textInputModel[V]) View() string {
	var s strings.Builder
	s.WriteString(m.prompt + "\n")
	s.WriteString(m.model.View() + "\n")
	if m.err == nil {
		s.WriteString("\n")
	} else {
		s.WriteString(m.styles.err.Render("Error: " + m.err.Error()))
		s.WriteString("\n")
	}
	return s.String()
}
