package p2p

import (
	"fmt"
	"testing"

	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/testutil/gittest"
	"github.com/coreeng/corectl/testdata"
	"github.com/coreeng/developer-platform/pkg/environment"
	coretnt "github.com/coreeng/developer-platform/pkg/tenant"
	"github.com/stretchr/testify/assert"
)

func TestCreateEnvVarsAsMap(t *testing.T) {
	testRepo := testLocalRepo(t)
	tenant := newTenant(t)
	env := newEnv(t)

	vars, err := NewP2pEnvVariables(&EnvVarContext{Tenant: tenant, Environment: env, AppRepo: testRepo})

	assert.NoError(t, err)
	assert.Equal(t, vars, &EnvVars{
		BaseDomain: env.GetDefaultIngressDomain().Domain,
		Registry:   toRegistry(t, env.Platform.(*environment.GCPVendor), tenant.Name),
		Version:    commitHash(t, testRepo),
		RepoPath:   testRepo.Path(),
		Region:     env.Platform.(*environment.GCPVendor).Region,
		TenantName: tenant.Name})
}

func TestCreateEnvVarsReturnsVarsAsExportCmd(t *testing.T) {
	testRepo := testLocalRepo(t)
	tenant := newTenant(t)
	env := newEnv(t)

	vars, err := NewP2pEnvVariables(&EnvVarContext{Tenant: tenant, Environment: env, AppRepo: testRepo})

	assert.NoError(t, err)
	assert.Contains(t, asExportCmd(t, vars),
		envVarFormat(t, BaseDomain, env.GetDefaultIngressDomain().Domain),
		envVarFormat(t, Registry, toRegistry(t, env.Platform.(*environment.GCPVendor), tenant.Name)),
		envVarFormat(t, Version, commitHash(t, testRepo)),
		envVarFormat(t, RepoPath, testRepo.Path()),
		envVarFormat(t, Region, env.Platform.(*environment.GCPVendor).Region),
		envVarFormat(t, TenantName, tenant.Name))
}

func TestCreateEnvVarsUnsupportedPlatform(t *testing.T) {
	env := &environment.Environment{
		IngressDomains: []environment.Domain{{Name: "default", Domain: fmt.Sprintf("%s-domain", t.Name())}},
		Platform: &environment.AWSVendor{
			Vendor: environment.AWSVendorType,
		},
	}

	_, err := NewP2pEnvVariables(&EnvVarContext{Tenant: newTenant(t), Environment: env, AppRepo: testLocalRepo(t)})

	assert.ErrorContains(t, err, "platform vendor not supported: aws")
}

func newTenant(t *testing.T) *coretnt.Tenant {
	return &coretnt.Tenant{
		Name: fmt.Sprintf("%s-tenant", t.Name()),
	}
}

func newEnv(t *testing.T) *environment.Environment {
	return &environment.Environment{
		IngressDomains: []environment.Domain{{Name: "default", Domain: fmt.Sprintf("%s-domain", t.Name())}},
		Platform: &environment.GCPVendor{
			Region:    fmt.Sprintf("%s-region", t.Name()),
			ProjectId: fmt.Sprintf("%s-projId", t.Name()),
			Vendor:    environment.GCPVendorType,
		},
	}
}

func testLocalRepo(t *testing.T) *git.LocalRepository {
	_, repo, err := gittest.CreateBareAndLocalRepoFromDir(&gittest.CreateBareAndLocalRepoOp{
		SourceDir:          testdata.CPlatformEnvsPath(),
		TargetBareRepoDir:  t.TempDir(),
		TargetLocalRepoDir: t.TempDir(),
		DryRun:             false,
	})
	assert.NoError(t, err)
	return repo
}

var envVarFormat = func(t *testing.T, k, v string) string {
	return fmt.Sprintf("export %s=\"%s\"", k, v)
}

var commitHash = func(t *testing.T, repo *git.LocalRepository) string {
	hash, err := repo.HeadShortCommitHash()
	assert.NoError(t, err)
	return hash
}

var asExportCmd = func(t *testing.T, vars *EnvVars) string {
	cmd, err := vars.AsExportCmd()
	assert.NoError(t, err)
	return cmd
}

var toRegistry = func(t *testing.T, vendor *environment.GCPVendor, tenantName string) string {
	return fmt.Sprintf("%s-docker.pkg.dev/%s/tenant/%s", vendor.Region, vendor.ProjectId, tenantName)
}
