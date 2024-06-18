package env

import (
	"os"
	"strings"
	"testing"

	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/stretchr/testify/assert"
)

func TestAppendEnv(t *testing.T) {
	tests := []struct {
		name     string
		env      environment.Environment
		expected string
	}{
		{
			name: "GCP environment",
			env: environment.Environment{
				Environment: "predev",
				Platform: &environment.GCPVendor{
					ProjectId: "gcp-predev-1234",
				},
			},
			expected: `
NAME    ID               CLOUD PLATFORM 
 predev  gcp-predev-1234  GCP`,
		},
		{
			name: "AWS environment",
			env: environment.Environment{
				Environment: "production",
				Platform: &environment.AWSVendor{
					AccountId: "aws-production-5678",
				},
			},
			expected: `
NAME        ID                   CLOUD PLATFORM 
 production  aws-production-5678  AWS`,
		},
	}

	streams := userio.NewIOStreams(
		os.Stdin,
		os.Stdout,
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := NewTable(streams, "Name", "ID", "Cloud Platform")
			AppendEnv(table, tt.env)
			compareOutput(t, table.Render(), tt.expected)
		})
	}
}

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
NAME  ID  CLOUD PLATFORM`,
		},
		{
			title:    "GCP rows",
			platform: "GCP",
			name:     "gcpdev-1234",
			id:       "1234",
			expected: `
NAME         ID    CLOUD PLATFORM 
 gcpdev-1234  1234  GCP`,
		},
		{
			title:    "AWS rows",
			platform: "AWS",
			name:     "awsprod-5678",
			id:       "5678",
			expected: `
NAME          ID    CLOUD PLATFORM 
 awsprod-5678  5678  AWS`,
		},
	}

	streams := userio.NewIOStreams(
		os.Stdin,
		os.Stdout,
	)

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			table := NewTable(streams, "Name", "ID", "Cloud Platform")
			table.AppendRow(tt.name, tt.id, tt.platform)
			compareOutput(t, table.Render(), tt.expected)
		})
	}
}

func compareOutput(t *testing.T, out string, expected string) {
	out = strings.TrimSpace(out)
	if strings.HasPrefix(expected, "\n") {
		expected = strings.Replace(expected, "\n", "", 1)
	}
	assert.Equal(t, expected, out)
}
