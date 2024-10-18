package userio

import (
	"fmt"
	"strings"

	"github.com/coreeng/corectl/pkg/cmdutil/userio/wizard"
)

func (s IOStreams) InfoE(messages ...string) error {
	msg := strings.Join(messages, "")
	styledMsg := s.styles.info.Render(msg)
	_, err := s.out.Output().Write([]byte(styledMsg))
	if err != nil {
		return fmt.Errorf("couldn't output info message: %v", err)
	}
	if len(messages) > 0 && !strings.HasSuffix(msg, "\n") {
		if _, err = s.out.Output().Write([]byte("\n")); err != nil {
			return fmt.Errorf("couldn't output info message: %v", err)
		}
	}
	return nil
}

func (s IOStreams) Info(messages ...string) {
	err := s.InfoE(messages...)
	if err != nil {
		panic(err.Error())
	}
}

func (s *IOStreams) Wizard(title string, completedTitle string) wizard.Handler {
	if s.IsInteractive() {
		s.CurrentHandler = newWizard(s)
	} else {
		s.CurrentHandler = nonInteractiveHandler{
			streams: s,
			styles:  NewNonInteractiveStyles(),
		}
	}
	s.CurrentHandler.SetTask(title, completedTitle)
	return s.CurrentHandler
}
