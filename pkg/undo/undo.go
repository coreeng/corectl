package undo

import (
	"errors"
	"fmt"
)

type Step func() error

type Steps struct {
	steps []Step
}

func NewSteps() Steps {
	return Steps{}
}

func (s *Steps) Add(step Step) {
	s.steps = append(s.steps, step)
}

func (s *Steps) Undo() []error {
	var errs []error
	for len(s.steps) > 0 {
		lastIndex := len(s.steps) - 1
		err := s.steps[lastIndex]()
		if err != nil {
			errs = append(errs, err)
		}
		s.steps = s.steps[:lastIndex]
	}
	return errs
}

func FormatError(operationDescription string, fnErr error, undoErrs []error) error {
	if fnErr == nil && len(undoErrs) == 0 {
		return nil
	} else if fnErr != nil && len(undoErrs) == 0 {
		return fnErr
	} else if fnErr == nil && len(undoErrs) > 0 {
		joinedUndoErrs := errors.Join(undoErrs...)
		return fmt.Errorf("couldn't undo all the changes:\n%v", joinedUndoErrs)
	} else {
		joinedUndoErrs := errors.Join(undoErrs...)
		return fmt.Errorf("%s: %v\nCouldn't undo all the changes:\n%v", operationDescription, fnErr, joinedUndoErrs)
	}
}
