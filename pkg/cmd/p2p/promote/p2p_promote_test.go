package promote

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/command"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

type MockFileSystem struct {
}

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	return nil, nil
}

func Test_run(t *testing.T) {
	t.Run("Run Promote successfully", func(t *testing.T) {
		mockFS := new(MockFileSystem)

		mockCommander := &mockCommander{executedCommands: []string{}}
		opts := &promoteOpts{
			ImageWithTag:       "imageName:1.1.1",
			SourceRegistry:     "europe-west2-docker.pkg.dev/tenant/grizzly",
			SourceStage:        "extended-test",
			SourceAuthOverride: "/source-auth.json",
			DestRegistry:       "eu.gcr.io/tenant/grizzly",
			DestStage:          "prod",
			DestAuthOverride:   "/dest-auth.json",
			Exec:               mockCommander,
			Streams:            userio.NewIOStreams(os.Stdin, os.Stdout),
			FileSystem:         mockFS,
		}
		err := run(opts)

		assert.NoError(t, err)

		assert.Equal(t,
			[]string{
				"which -s gcloud", // validate that gcloud exists
				"CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=/source-auth.json gcloud artifacts docker images list europe-west2-docker.pkg.dev/tenant/grizzly --limit=1",
				"CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=/dest-auth.json gcloud artifacts docker images list eu.gcr.io/tenant/grizzly --limit=1",
				"gcloud auth configure-docker --quiet europe-west2-docker.pkg.dev", // docker configure source registry
				"gcloud auth configure-docker --quiet eu.gcr.io",                   // docker configure destination registry
				"CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=/source-auth.json docker pull europe-west2-docker.pkg.dev/tenant/grizzly/extended-test/imageName:1.1.1",
				"docker tag europe-west2-docker.pkg.dev/tenant/grizzly/extended-test/imageName:1.1.1 eu.gcr.io/tenant/grizzly/prod/imageName:1.1.1",
				"CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=/dest-auth.json docker push eu.gcr.io/tenant/grizzly/prod/imageName:1.1.1",
			},
			mockCommander.executedCommands,
		)

	})
}

type mockCommander struct {
	executedCommands []string
}

func (m *mockCommander) Execute(c string, opts ...command.Option) ([]byte, error) {
	options := command.ApplyOptions(opts)
	var envArray []string
	for k, v := range options.Env {
		envArray = append(envArray, fmt.Sprintf("%s=%s", k, v))
	}

	commandParts := make([]string, 0, len(envArray)+1+len(options.Args))
	commandParts = append(commandParts, envArray...)
	commandParts = append(commandParts, c)
	commandParts = append(commandParts, options.Args...)

	concatenated := strings.Join(commandParts, " ")
	m.executedCommands = append(m.executedCommands, concatenated)
	return nil, nil
}
