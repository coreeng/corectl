package userio

import (
	"fmt"
	"strings"
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

func (streams IOStreams) Spinner(message string) WizardHandler {
	if streams.IsInteractive() {
		return newWizard(message, streams)
	} else {
		nih := nonInteractiveHandler{streams: streams}
		nih.SetTask(message)
		return nih
	}
}
