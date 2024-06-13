package env

import (
	"testing"

	"github.com/coreeng/developer-platform/pkg/environment"
)

func TestRenderBorderEnabled(t *testing.T) {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := NewTable("Name", "ID", "Cloud Platform")
			AppendEnv(table, tt.env)
			CompareOutput(t, table.Render(), tt.expected)
		})
	}
}
