package env

import (
	"errors"

	"github.com/coreeng/developer-platform/pkg/environment"
)

var (
	ErrorUnknownResourceType = errors.New("unknown resource type")
)

type ResourceType string

const (
	GrafanaResourceType               = "grafana"
	GrafanaContinuousLoadResourceType = "grafana/continuous-load"
)

func ResourceStringList() []string {
	return []string{
		GrafanaResourceType,
		GrafanaContinuousLoadResourceType,
	}
}

func OpenResource(resourceType ResourceType, env *environment.Environment) (string, error) {
	switch resourceType {
	case GrafanaResourceType:
		return openGrafana(env), nil
	case GrafanaContinuousLoadResourceType:
		return openGrafanaContinuousLoad(env), nil
	default:
		return "", ErrorUnknownResourceType
	}
}

func openGrafana(env *environment.Environment) string {
	return "https://grafana." + env.InternalServices.Domain
}

func openGrafanaContinuousLoad(env *environment.Environment) string {
	return "https://grafana." + env.InternalServices.Domain + "/d/zDpLnqaMz/continuous-load?orgId=1&refresh=5s"
}
