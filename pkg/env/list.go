package env

import (
	"github.com/coreeng/developer-platform/pkg/environment"
)

func AppendEnv(t TableEnv, env environment.Environment) {
	var (
		platform string
		id       string
	)

	switch p := env.Platform.(type) {
	case *environment.GCPVendor:
		id = p.ProjectId
		platform = "GCP"
	case *environment.AWSVendor:
		id = p.AccountId
		platform = "AWS"
	}

	t.AppendRow(env.Environment, id, platform)
}
