package userio

import (
	"errors"
	"fmt"
	"github.com/coreeng/developer-platform/pkg/validator"
)

var (
	ErrValueIsNotSet = errors.New("value is not set")
)

// Zeroable this interface is required to check if a value is not set
// we could use simply "comparable" as generic constraint,
// but slices are not comparable, which makes them impossible to use.
// This interface abstracts helps mitigate the issue
type Zeroable[V any] interface {
	Value() V
	IsZeroValue() bool
}

type ValidateAndMap[V, T any] func(V) (T, error)

type InputSourceSwitch[V, T any] struct {
	DefaultValue        Zeroable[V]
	InteractivePromptFn func() (InputPrompt[V], error)
	ValidateAndMap      ValidateAndMap[V, T]
	ErrMessage          string

	mapped bool
	value  T
	err    error
}

func (iss *InputSourceSwitch[V, T]) Validate(streams IOStreams) error {
	if iss.mapped {
		return iss.err
	}
	if !iss.DefaultValue.IsZeroValue() {
		_, err := iss.tryValidateAndMap(iss.DefaultValue.Value())
		return err
	}
	if !streams.IsInteractive() {
		iss.mapped = true
		iss.setError(ErrValueIsNotSet)
		return iss.err
	}
	return nil
}

func (iss *InputSourceSwitch[V, T]) GetValue(streams IOStreams) (T, error) {
	if iss.mapped {
		return iss.value, iss.err
	}
	if !iss.DefaultValue.IsZeroValue() {
		v, err := iss.tryValidateAndMap(iss.DefaultValue.Value())
		return v, err
	}
	if !streams.IsInteractive() {
		iss.mapped = true
		iss.setError(ErrValueIsNotSet)
		return iss.value, iss.err
	}

	interactivePrompt, err := iss.InteractivePromptFn()
	if err != nil {
		return iss.value, err
	}
	input, err := interactivePrompt.GetInput(streams)
	if err != nil {
		return iss.value, err
	}
	return iss.tryValidateAndMap(input)
}

func (iss *InputSourceSwitch[V, T]) tryValidateAndMap(value V) (T, error) {
	if iss.mapped {
		return iss.value, iss.err
	}
	mappedValue, err := iss.ValidateAndMap(value)
	if err == nil {
		iss.value = mappedValue
	}
	if err != nil {
		iss.setError(err)
	}
	iss.mapped = true
	return iss.value, iss.err
}

func (iss *InputSourceSwitch[V, T]) setError(err error) {
	if iss.ErrMessage != "" {
		var errReason string
		var fieldError *validator.FieldError
		if errors.As(err, &fieldError) {
			errReason = fieldError.Message
		} else {
			errReason = err.Error()
		}
		iss.err = fmt.Errorf("%s: %s", iss.ErrMessage, errReason)
	} else {
		iss.err = err
	}
}

func AsZeroable[V comparable](value V) Zeroable[V] {
	return primitiveZeroable[V]{value}
}

func AsZeroableSlice[V []T, T any](value V) Zeroable[V] {
	return sliceZeroable[V, T]{value}
}

type primitiveZeroable[V comparable] struct {
	value V
}

func (p primitiveZeroable[V]) Value() V {
	return p.value
}

func (p primitiveZeroable[V]) IsZeroValue() bool {
	var zeroValue V
	return p.value == zeroValue
}

type sliceZeroable[V []T, T any] struct {
	value V
}

func (s sliceZeroable[V, T]) Value() V {
	return s.value
}

func (s sliceZeroable[V, T]) IsZeroValue() bool {
	return s.value == nil
}
