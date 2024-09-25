package root

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	m.Run()
}

func TestRootIgnoresInvalidLogLevel(t *testing.T) {
	assert.Equal(t, zerolog.Disabled, zerolog.GlobalLevel())

	ConfigureGlobalLogger("invalid", os.Stdout)
	assert.Equal(t, zerolog.Disabled, zerolog.GlobalLevel())
}

func TestRootSetsValidLogLevel(t *testing.T) {
	assert.Equal(t, zerolog.Disabled, zerolog.GlobalLevel())

	ConfigureGlobalLogger("error", os.Stdout)
	assert.Equal(t, zerolog.ErrorLevel, zerolog.GlobalLevel())
}
