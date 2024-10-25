package env

import (
	"os"
	"os/exec"
	"testing"

	"github.com/coreeng/corectl/pkg/command"

	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/stretchr/testify/assert"
)

func TestConnectSuccess(t *testing.T) {
	var proxy = 1234

	env := &environment.Environment{
		Environment: "predev",
		Platform: &environment.GCPVendor{
			ProjectId: "gcp-predev-1234",
		},
	}
	streams := userio.NewIOStreams(
		os.Stdin,
		os.Stdout,
		os.Stderr,
	)
	err := Connect(streams, env, mockCommanderSuccess{}, proxy)
	assert.NoError(t, err)
}

func TestConnectFail(t *testing.T) {
	var proxy = 1234

	env := &environment.Environment{
		Environment: "production",
		Platform: &environment.AWSVendor{
			AccountId: "aws-production-5678",
		},
	}
	streams := userio.NewIOStreams(
		os.Stdin,
		os.Stdout,
		os.Stderr,
	)
	err := Connect(streams, env, mockCommanderFail{}, proxy)
	assert.Error(t, err)
}

type mockCommanderSuccess struct {
}

// helperExecProcess allows us to execute a mock exec.Command
// this function ensures that it will only be run if
// GO_TEST_PROCESS environment variable is present
func helperExecProcess(fn string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=" + fn, "--"}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}

	return cmd
}

func (m mockCommanderSuccess) Execute(c string, opts ...command.Option) ([]byte, error) {
	options := command.ApplyOptions(opts)
	return helperExecProcess("TestOutputSuccess", options.Args...).CombinedOutput()
}

// TestOutputSuccess mocks a command that returns successful command
func TestOutputSuccess(*testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	os.Exit(0)
}

type mockCommanderFail struct {
}

func (m mockCommanderFail) Execute(c string, opts ...command.Option) ([]byte, error) {
	options := command.ApplyOptions(opts)
	return helperExecProcess("TestOutputFail", options.Args...).CombinedOutput()
}

// TestOutputFail mocks a command that returns a non zero exit code
func TestOutputFail(*testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	os.Exit(1)
}
