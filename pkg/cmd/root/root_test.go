package root

import (
	"testing"
)

func TestMain(m *testing.M) {
	m.Run()
}

// TODO

// func TestRootIgnoresInvalidLogLevel(t *testing.T) {
// 	log.DefaultLogger.SetLevel(log.Level(8))
// 	assert.Equal(t, log.Level(8), log.DefaultLogger.Level)

// 	ConfigureGlobalLogger("invalid")
// 	assert.Equal(t, log.PanicLevel, log.DefaultLogger.Level)
// }

// func TestRootSetsValidLogLevel(t *testing.T) {
// 	log.DefaultLogger.SetLevel(log.Level(8))
// 	assert.Equal(t, log.Level(8), log.DefaultLogger.Level)

// 	ConfigureGlobalLogger("panic")
// 	assert.Equal(t, log.PanicLevel, log.DefaultLogger.Level)
// }
