package env

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppendRow(t *testing.T) {
	tests := []struct {
		title    string
		platform string
		name     string
		id       string
		expected string
	}{
		{
			title:    "No rows",
			platform: "",
			name:     "",
			id:       "",
			expected: `
CLOUD PLATFORM  ID  NAME`,
		},
		{
			title:    "GCP rows",
			platform: "GCP",
			name:     "gcpdev-1234",
			id:       "1234",
			expected: `
CLOUD PLATFORM  ID           NAME 
 GCP             gcpdev-1234  1234`,
		},
		{
			title:    "AWS rows",
			platform: "AWS",
			name:     "awsprod-5678",
			id:       "5678",
			expected: `
CLOUD PLATFORM  ID            NAME 
 AWS             awsprod-5678  5678`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			table := NewTable("Cloud Platform", "ID", "name")
			table.AppendRow(tt.platform, tt.id, tt.name)
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
