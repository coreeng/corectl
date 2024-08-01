package promote

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_isRegistrySupported(t *testing.T) {
	tests := []struct {
		registry string
		valid    bool
	}{
		{registry: "us-central1-1234.docker.pkg.dev/some-random/path", valid: false},
		{registry: "us-central1-1234-docker.pkg.dev/some-random/path", valid: true},
		{registry: "asia-gcr.io/some-random/path", valid: false},
		{registry: "asia.gcr.io/some-random/path", valid: true},
		{registry: "asia.gcr.io", valid: true},
	}
	for _, tt := range tests {
		t.Run(tt.registry, func(t *testing.T) {
			assert.Equal(t, tt.valid, isRegistrySupported(tt.registry))
		})
	}
}
