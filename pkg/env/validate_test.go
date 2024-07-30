package env

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/coreeng/corectl/pkg/gcp"
	gcptest "github.com/coreeng/corectl/pkg/testutil/gcp"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		env  *environment.Environment
		err  error
	}{
		{
			name: "GCP environment",
			env: &environment.Environment{
				Environment: "predev",
				Platform: &environment.GCPVendor{
					ProjectId: "gcp-predev-1234",
				},
			},
			err: nil,
		},
		{
			name: "AWS environment",
			env: &environment.Environment{
				Environment: "production",
				Platform: &environment.AWSVendor{
					AccountId: "aws-production-5678",
				},
			},
			err: errors.New("AWS cloud platform is not supported"),
		},
	}

	clusterSvc, err := gcptest.NewClusterMockClient()
	assert.NoError(t, err)

	ctx := context.Background()
	client, err := gcp.NewClient(clusterSvc)
	assert.NoError(t, err)

	mockCmd := &mockCommand{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(ctx, tt.env, mockCmd, client)
			if tt.err != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.err.Error(), err.Error())
			}
		})
	}
}

type mockCommand struct {
}

func (m *mockCommand) Execute(c string, args ...string) ([]byte, error) {
	cs := []string{"-test.run=TestOutput", "--"}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd.CombinedOutput()
}

func TestOutput(*testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}

	defer os.Exit(0)
	fmt.Printf("binary exists")
}
