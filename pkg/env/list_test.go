package env

import (
	"os"
	"strings"
	"testing"

	"github.com/coreeng/core-platform/pkg/environment"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
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
				Tier:        environment.PreDevEnvironmentTier,
				Platform: &environment.GCPVendor{
					ProjectId: "gcp-predev-1234",
				},
			},
			expected: `
			NAME    TIER     ID               CLOUDPLATFORM 
			 predev  pre-dev  gcp-predev-1234  GCP`,
		},
		{
			name: "AWS environment",
			env: environment.Environment{
				Environment: "production",
				Tier:        environment.ProdEnvironmentTier,
				Platform: &environment.AWSVendor{
					AccountId: "aws-production-5678",
				},
			},
			expected: `
			NAME        TIER  ID                   CLOUDPLATFORM 
			 production  prod  aws-production-5678  AWS`,
		},
	}

	streams := userio.NewIOStreams(
		os.Stdin,
		os.Stdout,
		os.Stderr,
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := NewTable(streams, false)
			table.AppendEnv(tt.env, "", "")
			compareOutput(t, table.Render(), tt.expected)
		})
	}
}

func TestAppendRow(t *testing.T) {
	tests := []struct {
		title    string
		tier     string
		platform string
		name     string
		id       string
		expected string
	}{
		{
			title:    "No rows",
			tier:     "",
			platform: "",
			name:     "",
			id:       "",
			expected: `
			NAME  TIER  ID  CLOUDPLATFORM`,
		},
		{
			title:    "GCP rows",
			tier:     "pre-dev",
			platform: "GCP",
			name:     "gcpdev-1234",
			id:       "1234",
			expected: `
			NAME         TIER     ID    CLOUDPLATFORM 
			 gcpdev-1234  pre-dev  1234  GCP`,
		},
		{
			title:    "AWS rows",
			tier:     "prod",
			platform: "AWS",
			name:     "awsprod-5678",
			id:       "5678",
			expected: `
			NAME          TIER  ID    CLOUDPLATFORM 
			 awsprod-5678  prod  5678  AWS`,
		},
	}

	streams := userio.NewIOStreams(
		os.Stdin,
		os.Stdout,
		os.Stderr,
	)

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			table := NewTable(streams, false)
			table.AppendRow(tt.name, tt.tier, tt.id, tt.platform, "", "")
			compareOutput(t, table.Render(), tt.expected)
		})
	}
}

func compareOutput(t *testing.T, out string, expected string) {
	out = strings.TrimSpace(out)
	expected = strings.ReplaceAll(expected, "\t", "")
	expected = strings.TrimSpace(expected)
	assert.Equal(t, expected, out)
}
