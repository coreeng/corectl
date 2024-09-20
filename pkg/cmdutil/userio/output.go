package userio

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (streams IOStreams) InfoE(messages ...string) error {
	msg := strings.Join(messages, "")
	styledMsg := streams.styles.info.Render(msg)
	_, err := streams.out.Output().Write([]byte(styledMsg))
	if err != nil {
		return fmt.Errorf("couldn't output info message: %v", err)
	}
	if len(messages) > 0 && !strings.HasSuffix(msg, "\n") {
		if _, err = streams.out.Output().Write([]byte("\n")); err != nil {
			return fmt.Errorf("couldn't output info message: %v", err)
		}
	}
	return nil
}

func (streams IOStreams) Info(messages ...string) {
	err := streams.InfoE(messages...)
	if err != nil {
		panic(err.Error())
	}
}

func (streams IOStreams) Spinner(message string) SpinnerHandler {
	if streams.IsInteractive() {
		return newSpinner(message, streams)
	} else {
		streams.Info(message)
		return dummySpinnerHandler{streams: streams}
	}
}

type dummySpinnerHandler struct {
	streams IOStreams
}

func InfoLog(message string) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("123"))
	return fmt.Sprintf("%s %s", style.Render("INFO:"), message)
}

func (dummySpinnerHandler) Done() {}
func (dsh dummySpinnerHandler) Info(message string) {
	dsh.streams.out.Output().Write([]byte(InfoLog(message) + "\n"))
}
func (dummySpinnerHandler) SetTitle(string) {}
