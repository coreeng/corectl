package env

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppendRow(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		cluster  string
		id       string
		expected string
	}{
		{
			name:     "No rows",
			platform: "",
			cluster:  "",
			id:       "",
			expected: `
CLOUD PLATFORM  ID  CLUSTER`,
		},
		{
			name:     "GCP rows",
			platform: "GCP",
			cluster:  "gcpdev-1234",
			id:       "1234",
			expected: `
CLOUD PLATFORM  ID    CLUSTER     
 GCP             1234  gcpdev-1234`,
		},
		{
			name:     "AWS rows",
			platform: "AWS",
			cluster:  "awsprod-5678",
			id:       "5678",
			expected: `
CLOUD PLATFORM  ID    CLUSTER      
 AWS             5678  awsprod-5678`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := NewTable("Cloud Platform", "ID", "Cluster")
			table.AppendRow(tt.platform, tt.id, tt.cluster)
			CompareOutput(t, table.Render(), tt.expected)
		})
	}
}

func CompareOutput(t *testing.T, out string, expected string) {
	out = strings.TrimSpace(out)
	if strings.HasPrefix(expected, "\n") {
		expected = strings.Replace(expected, "\n", "", 1)
	}
	assert.Equal(t, expected, out)
}
