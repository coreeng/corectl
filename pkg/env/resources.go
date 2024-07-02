package env

import (
	"errors"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/pkg/browser"
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

func OpenResource(resourceType ResourceType, env *environment.Environment) error {
	switch resourceType {
	case GrafanaResourceType:
		return openGrafana(env)
	case GrafanaContinuousLoadResourceType:
		return openGrafanaContinuousLoad(env)
	default:
		return ErrorUnknownResourceType
	}
}

func openGrafana(env *environment.Environment) error {
	url := "https://grafana." + env.InternalServices.Domain
	return browser.OpenURL(url)
}

func openGrafanaContinuousLoad(env *environment.Environment) error {
	url := "https://grafana." + env.InternalServices.Domain + "/d/zDpLnqaMz/continuous-load?orgId=1&refresh=5s"
	return browser.OpenURL(url)
}
