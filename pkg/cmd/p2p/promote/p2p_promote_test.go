package promote

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

func Test_run(t *testing.T) {
	t.Run("Run Promote successfully", func(t *testing.T) {

		mockCommand := &mockCommand{executedCommands: []string{}}
		opts := &promoteOpts{
			ImageWithTag:       "imageName:1.1.1",
			SourceRegistry:     "europe-west2-docker.pkg.dev/tenant/grizzly",
			SourceStage:        "extended-test",
			SourceAuthOverride: "/source-auth.json",
			DestRegistry:       "eu.gcr.io/tenant/grizzly",
			DestStage:          "prod",
			DestAuthOverride:   "/dest-auth.json",
			Exec:               mockCommand,
			Streams:            userio.NewIOStreams(os.Stdin, os.Stdout),
		}
		err := run(opts)

		if err != nil {
			t.Fatalf("run() error = %v", err)
		}

		assert.Equal(t,
			[]string{
				"which -s gcloud", // validate that gcloud exists
				"gcloud auth configure-docker --quiet europe-west2-docker.pkg.dev", // docker configure source registry
				"gcloud auth configure-docker --quiet eu.gcr.io",                   // docker configure destination registry
				"CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=/source-auth.json docker pull europe-west2-docker.pkg.dev/tenant/grizzly/extended-test/imageName:1.1.1",
				"docker tag europe-west2-docker.pkg.dev/tenant/grizzly/extended-test/imageName:1.1.1 eu.gcr.io/tenant/grizzly/prod/imageName:1.1.1",
				"CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=/dest-auth.json docker push eu.gcr.io/tenant/grizzly/prod/imageName:1.1.1",
			},
			mockCommand.executedCommands,
		)

	})
}

type mockCommand struct {
	executedCommands []string
}

func (m *mockCommand) Execute(c string, args ...string) ([]byte, error) {
	concatenated := strings.Join(append([]string{c}, args...), " ")
	m.executedCommands = append(m.executedCommands, concatenated)
	return nil, nil
}

func (m *mockCommand) ExecuteWithEnv(c string, envs map[string]string, args ...string) ([]byte, error) {
	var envArray []string
	for k, v := range envs {
		envArray = append(envArray, fmt.Sprintf("%s=%s", k, v))
	}

	commandParts := make([]string, 0, len(envArray)+1+len(args))
	commandParts = append(commandParts, envArray...)
	commandParts = append(commandParts, c)
	commandParts = append(commandParts, args...)

	concatenated := strings.Join(commandParts, " ")
	m.executedCommands = append(m.executedCommands, concatenated)
	return nil, nil
}

type NoopWriter struct {
}

func (NoopWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
