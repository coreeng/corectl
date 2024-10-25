package promote

import (
	"os"
	"testing"

	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/command"
	"github.com/phuslu/log"
	"github.com/stretchr/testify/assert"
)

type MockFileSystem struct {
}

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	return nil, nil
}

func Test_run(t *testing.T) {
	log.DefaultLogger.SetLevel(log.PanicLevel)
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
			Streams:            userio.NewIOStreams(os.Stdin, os.Stdout, os.Stderr),
			FileSystem:         mockFS,
		}
		err := run(opts)

		assert.NoError(t, err)

		assert.Equal(t,
			[]string{
				"gcloud help", // validate that gcloud exists
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
	m.executedCommands = append(m.executedCommands, command.FormatCommand(c, options))
	return nil, nil
}
