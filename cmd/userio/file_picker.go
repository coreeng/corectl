package userio

import (
	"errors"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"path/filepath"
	"strings"
)

type FilePicker struct {
	Prompt         string
	WorkingDir     string
	InitialValue   string
	ValidateAndMap ValidateAndMap[string, string]
}

func (ifp *FilePicker) GetInput(streams IOStreams) (string, error) {
	if ifp.ValidateAndMap == nil {
		panic("ValidateAndMap function is missing")
	}
	initialValue := ifp.InitialValue
	if initialValue == "" {
		initialValue = "./"
	}

	model := textinput.New()
	model.ShowSuggestions = true
	model.Focus()
	model.SetValue(initialValue)

	expandedValue, err := expandPath(ifp.WorkingDir, ifp.InitialValue)
	if err != nil {
		return "", err
	}
	suggestions, err := generateSuggestions(initialValue, expandedValue)
	if err != nil {
		return "", err
	}
	model.SetSuggestions(suggestions)

	ifpModel := inlineFilePickerModel{
		model:          model,
		prompt:         ifp.Prompt,
		workingDir:     ifp.WorkingDir,
		validateAndMap: ifp.ValidateAndMap,
		expandedValue:  expandedValue,
	}
	result, err := streams.execute(ifpModel)
	if err != nil {
		return "", err
	}
	ifpModel = result.(inlineFilePickerModel)
	return ifpModel.expandedValue, ifpModel.err
}

type inlineFilePickerModel struct {
	model          textinput.Model
	prompt         string
	workingDir     string
	validateAndMap ValidateAndMap[string, string]

	expandedValue string
	quitting      bool
	err           error
}

func (m inlineFilePickerModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inlineFilePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	previousValue := m.model.Value()
	m.model, cmd = m.model.Update(msg)
	currentValue := m.model.Value()

	if previousValue != currentValue {
		expandedValue, err := expandPath(m.workingDir, currentValue)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.expandedValue = expandedValue
		suggestions, err := generateSuggestions(currentValue, expandedValue)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.model.SetSuggestions(suggestions)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			v, err := m.validateAndMap(m.expandedValue)
			m.err = err
			if err == nil {
				m.expandedValue = v
			}
			if m.err == nil {
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

func generateSuggestions(userValue string, expandedValue string) ([]string, error) {
	parentDir, _ := filepath.Split(userValue)
	expandedParentDir, _ := filepath.Split(expandedValue)
	dirEntries, err := os.ReadDir(expandedParentDir)
	if err != nil {
		return nil, err
	}
	var autocompleteSuggestions []string
	for _, entry := range dirEntries {
		shouldAddSeparatorInBetween := !strings.HasSuffix(parentDir, string(os.PathSeparator))
		shouldAddSeparatorOnEnd := entry.IsDir()
		var suggestion strings.Builder
		suggestion.WriteString(parentDir)
		if parentDir != "" && shouldAddSeparatorInBetween {
			suggestion.WriteByte(os.PathSeparator)
		}
		suggestion.WriteString(entry.Name())
		if shouldAddSeparatorOnEnd {
			suggestion.WriteByte(os.PathSeparator)
		}
		autocompleteSuggestions = append(autocompleteSuggestions, suggestion.String())
	}
	return autocompleteSuggestions, nil
}

func (m inlineFilePickerModel) View() string {
	var s strings.Builder
	s.WriteString(m.prompt + "\n")
	s.WriteString(m.model.View() + "\n")
	if m.err == nil {
		s.WriteString("\n")
	} else {
		s.WriteString(errorStyle.Render("Error: " + m.err.Error()))
	}
	return s.String()
}

func expandPath(workingDir string, path string) (string, error) {
	expanded := os.ExpandEnv(path)

	if strings.HasPrefix(expanded, "~") {
		// it's not completely correct, but covers the main usecase
		dir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		expanded = dir + expanded[1:]
	}

	if filepath.IsLocal(expanded) {
		expanded = filepath.Join(workingDir, expanded)
	} else if expanded == "" {
		expanded = workingDir
	}

	expandedInfo, err := os.Stat(expanded)
	if err != nil && (os.IsNotExist(err) || os.IsPermission(err)) {
		return expanded, nil
	} else if err != nil {
		return "", nil
	}
	if expandedInfo.IsDir() && !strings.HasSuffix(expanded, string(os.PathSeparator)) {
		expanded += string(os.PathSeparator)
	}
	return expanded, nil
}

type FileValidatorOptions struct {
	ExistingOnly bool
	DirsOnly     bool
	DirIsEmpty   bool
	FilesOnly    bool
	Optional     bool
}

func NewFileValidator(opt FileValidatorOptions) ValidateAndMap[string, string] {
	return func(inp string) (string, error) {
		return inp, ValidateFilePath(inp, opt)
	}
}

func ValidateFilePath(path string, opt FileValidatorOptions) error {
	path = strings.TrimSpace(path)
	if path == "" && !opt.Optional {
		return errors.New("empty input")
	} else if path == "" {
		return nil
	}
	fileInfo, err := os.Stat(path)
	if err != nil && (opt.ExistingOnly || !os.IsNotExist(err)) {
		return err
	} else if err != nil {
		return nil
	}
	if opt.FilesOnly && fileInfo.IsDir() {
		return errors.New("directory is not expected")
	}
	if opt.DirsOnly && !fileInfo.IsDir() {
		return errors.New("directory only is expected")
	}
	if opt.DirIsEmpty && fileInfo.IsDir() {
		dirEntries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		if len(dirEntries) > 0 {
			return errors.New("directory is not empty")
		}
	}
	return nil
}
