package env

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/stretchr/testify/assert"
)

func TestConnectSuccess(t *testing.T) {
	var (
		projectID = "gcp-project-1"
		location  = "europe-west-2"
		proxy     = 1234
	)

	env := &environment.Environment{
		Environment: "predev",
		Platform: &environment.GCPVendor{
			ProjectId: "gcp-predev-1234",
		},
	}
	streams := userio.NewIOStreams(
		os.Stdin,
		os.Stdout,
	)
	err := Connect(streams, env, mockCommandSuccess{}, projectID, location, proxy)
	assert.NoError(t, err)
}

func TestConnectFail(t *testing.T) {
	var (
		projectID = "gcp-project-1"
		location  = "europe-west-2"
		proxy     = 1234
	)

	env := &environment.Environment{
		Environment: "production",
		Platform: &environment.AWSVendor{
			AccountId: "aws-production-5678",
		},
	}
	streams := userio.NewIOStreams(
		os.Stdin,
		os.Stdout,
	)
	err := Connect(streams, env, mockCommandFail{}, projectID, location, proxy)
	assert.Error(t, err)
}

type mockCommandSuccess struct {
}

func (m mockCommandSuccess) Execute(c string, args ...string) ([]byte, error) {
	cs := []string{"-test.run=TestOutputSuccess", "--"}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd.CombinedOutput()
}

func TestOutputSuccess(*testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}

	defer os.Exit(0)
	fmt.Printf("binary exists")
}

type mockCommandFail struct {
}

func (m mockCommandFail) Execute(c string, args ...string) ([]byte, error) {
	cs := []string{"-test.run=TestOutputFail", "--"}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd.CombinedOutput()
}

func TestOutputFail(*testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}

	defer os.Exit(1)
	fmt.Printf("gcloud doesn't exist")
}
