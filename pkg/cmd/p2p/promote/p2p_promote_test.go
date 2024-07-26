package promote

import (
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
			ImageWithTag:   "imageName:1.1.1",
			SourceRegistry: "europe-west2-docker.pkg.dev/tenant/grizzly",
			SourceStage:    "extended-test",
			DestRegistry:   "eu.gcr.io/tenant/grizzly",
			DestStage:      "prod",
			Streams: userio.NewIOStreams(
				os.Stdin,
				os.Stdout,
			),
			Exec: mockCommand,
		}
		err := run(opts)

		if err != nil {
			t.Fatalf("run() error = %v", err)
		}

		assert.Equal(t, mockCommand.executedCommands, []string{
			"gcloud help", // validate that gcloud exists
			"gcloud auth configure-docker --quiet europe-west2-docker.pkg.dev", // docker configure source registry
			"gcloud auth configure-docker --quiet eu.gcr.io",                   // docker configure destination registry
			"docker pull europe-west2-docker.pkg.dev/tenant/grizzly/extended-test/imageName:1.1.1",
			"docker tag europe-west2-docker.pkg.dev/tenant/grizzly/extended-test/imageName:1.1.1 eu.gcr.io/tenant/grizzly/prod/imageName:1.1.1",
			"docker push eu.gcr.io/tenant/grizzly/prod/imageName:1.1.1",
		})

	})
}

type mockCommand struct {
	executedCommands []string
}

func (m *mockCommand) Execute(c string, args ...string) ([]byte, error) {
	appendExecutedCommands(c, args, m)
	return nil, nil
}

func (m *mockCommand) ExecuteWithEnv(c string, _ map[string]string, args ...string) ([]byte, error) {
	appendExecutedCommands(c, args, m)
	return nil, nil
}

func appendExecutedCommands(c string, args []string, m *mockCommand) {
	concatenated := strings.Join(append([]string{c}, args...), " ")
	m.executedCommands = append(m.executedCommands, concatenated)
}
