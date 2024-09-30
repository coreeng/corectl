package userio

import (
	"fmt"
	"strings"

	"github.com/coreeng/corectl/pkg/cmdutil/userio/wizard"
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

func (streams *IOStreams) Wizard(title string, completedTitle string) wizard.Handler {
	if streams.IsInteractive() {
		streams.CurrentHandler = newWizard(streams)
	} else {
		streams.CurrentHandler = nonInteractiveHandler{streams: streams}
	}
	streams.CurrentHandler.SetTask(title, completedTitle)
	return streams.CurrentHandler
}
