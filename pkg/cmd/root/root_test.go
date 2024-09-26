package root

import (
	"testing"

	"github.com/phuslu/log"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	m.Run()
}

func TestRootIgnoresInvalidLogLevel(t *testing.T) {
	log.DefaultLogger.SetLevel(log.Level(8))
	assert.Equal(t, log.Level(8), log.DefaultLogger.Level)

	ConfigureGlobalLogger("invalid")
	assert.Equal(t, log.PanicLevel, log.DefaultLogger.Level)
}

func TestRootSetsValidLogLevel(t *testing.T) {
	log.DefaultLogger.SetLevel(log.Level(8))
	assert.Equal(t, log.Level(8), log.DefaultLogger.Level)

	ConfigureGlobalLogger("panic")
	assert.Equal(t, log.PanicLevel, log.DefaultLogger.Level)
}
