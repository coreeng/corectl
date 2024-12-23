package p2p

import (
	"bytes"
	"fmt"
	"github.com/coreeng/core-platform/pkg/environment"
	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/git"
)

const (
	BaseDomain = "BASE_DOMAIN"
	Registry   = "REGISTRY"
	Version    = "VERSION"
	RepoPath   = "REPO_PATH"
	Region     = "REGION"
	TenantName = "TENANT_NAME"
)

type EnvVarContext struct {
	Tenant      *coretnt.Tenant
	Environment *environment.Environment
	AppRepo     *git.LocalRepository
}
type EnvVars map[string]string

func (p2pEnv *EnvVars) AsExportCmd() (string, error) {
	s, err := p2pEnv.asKeyValString("export")
	if err != nil {
		return "", err
	}
	return s, nil
}

func (p2pEnv *EnvVars) asKeyValString(keyPrefix string) (string, error) {
	b := new(bytes.Buffer)
	for key, value := range *p2pEnv {
		_, err := fmt.Fprintf(b, "%s %s=\"%s\"\n", keyPrefix, key, value)
		if err != nil {
			return "", fmt.Errorf("faild to convert vars map %v to string", p2pEnv)
		}
	}
	return b.String(), nil
}

func NewP2pEnvVariables(context *EnvVarContext) (*EnvVars, error) {
	var envVars = make(EnvVars)
	envVars[TenantName] = context.Tenant.Name
	envVars[BaseDomain] = context.Environment.GetDefaultIngressDomain().Domain
	envVars[RepoPath] = context.AppRepo.Path()
	switch p := context.Environment.Platform.(type) {
	case *environment.GCPVendor:
		envVars[Region] = p.Region
		envVars[Registry] = fmt.Sprintf("%s-docker.pkg.dev/%s/tenant/%s", p.Region, p.ProjectId, context.Tenant.Name)
	default:
		return nil, fmt.Errorf("platform vendor not supported: %s", context.Environment.Platform.Type())
	}
	version, err := context.AppRepo.HeadShortCommitHash()
	if err != nil {
		return nil, err
	}
	envVars[Version] = version
	return &envVars, nil
}
