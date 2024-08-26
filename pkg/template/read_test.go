package template

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFillDefaultSpecValues(t *testing.T) {
	t.Run("no parameters", func(t *testing.T) {
		s := Spec{
			Name: "name",
		}
		fillDefaultSpecValues(&s)
		assert.Equal(t, ImplicitParameters, s.Parameters)
	})
	t.Run("with parameters", func(t *testing.T) {
		s := Spec{
			Name: "name",
			Parameters: []Parameter{
				{
					Name: "param1",
				},
				{
					Name: "param2",
					Type: IntParamType,
				},
			},
		}
		fillDefaultSpecValues(&s)

		expectedParams := make([]Parameter, 0, len(ImplicitParameters)+2)
		expectedParams = append(expectedParams, ImplicitParameters...)
		expectedParams = append(expectedParams, Parameter{
			Name: "param1",
			Type: StringParamType,
		})
		expectedParams = append(expectedParams, Parameter{
			Name: "param2",
			Type: IntParamType,
		})
		assert.Exactly(t, expectedParams, s.Parameters)
	})
}
