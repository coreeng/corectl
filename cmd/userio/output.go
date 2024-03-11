package userio

import (
	"strings"
)

func (streams IOStreams) Info(messages ...string) error {
	msg := strings.Join(messages, "")
	styledMsg := infoStyle.Render(msg)
	_, err := streams.out.Write([]byte(styledMsg))
	if err != nil {
		return err
	}
	if len(messages) > 0 && !strings.HasSuffix(msg, "\n") {
		if _, err := streams.out.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}

func (streams IOStreams) Spinner(message string) SpinnerHandler {
	return newSpinner(message, streams)
}
