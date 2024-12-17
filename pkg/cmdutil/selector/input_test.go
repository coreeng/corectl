package selector

import (
	"fmt"
	"os"
	"testing"

	"github.com/coreeng/core-platform/pkg/environment"
	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/testutil/gittest"
	"github.com/coreeng/corectl/testdata"
	"github.com/stretchr/testify/assert"
)

var streams = userio.NewIOStreams(os.Stdin, os.Stdout, os.Stderr)

func TestMain(m *testing.M) {
	m.Run()
}

func TestTenantSelectorReturnsTenant(t *testing.T) {
	cPlatRepo := testLocalRepo(t, testdata.CPlatformEnvsPath())

	tenant, err := Tenant(cPlatRepo.Path(), testdata.DefaultTenant(), streams)

	assert.NoError(t, err)
	assert.Equal(t, tenant.Name, testdata.DefaultTenant())
}

func TestTenantSelectorNonExistingTenant(t *testing.T) {
	cPlatRepo := testLocalRepo(t, testdata.CPlatformEnvsPath())
	tenantName := fmt.Sprintf("%s-tenant", t.Name())

	tenant, err := Tenant(cPlatRepo.Path(), fmt.Sprintf("%s-tenant", t.Name()), streams)

	assert.ErrorContains(t, err, fmt.Sprintf("config repo path %s/tenants/tenants: tenant %s invalid: cannot find %s tenant, available tenants: [default-tenant parent root]", cPlatRepo.Path(), tenantName, tenantName))
	assert.Nil(t, tenant)
}

func TestTenantSelectorInvalidCPlatRepo(t *testing.T) {
	cPlatRepoPath := t.TempDir()

	tenant, err := Tenant(cPlatRepoPath, testdata.DefaultTenant(), streams)

	assert.ErrorContains(t, err, fmt.Sprintf("couldn't load tenant configuration in path %s/tenants/tenants: stat .: no such file or directory", cPlatRepoPath))
	assert.Nil(t, tenant)
}

func TestEnvironmentSelectorReturnsEnvironment(t *testing.T) {
	cPlatRepo := testLocalRepo(t, testdata.CPlatformEnvsPath())

	env, err := Environment(cPlatRepo.Path(), testdata.DevEnvironment(), testdata.TenantEnvs(), streams)

	assert.NoError(t, err)
	assert.Equal(t, env.Environment, testdata.DevEnvironment())
}

func TestEnvironmentSelectorFilterOnTenantEnvs(t *testing.T) {
	cPlatRepo := testLocalRepo(t, testdata.CPlatformEnvsPath())

	env, err := Environment(cPlatRepo.Path(), testdata.DevEnvironment(), []string{"not-tenant-envs"}, streams)

	assert.ErrorContains(t, err, fmt.Sprintf("tenant env %s doesn't exist in tenant configuration %s", testdata.DevEnvironment(), coretnt.DirFromCPlatformPath(cPlatRepo.Path())))
	assert.Nil(t, env)
}

func TestEnvironmentSelectorNonExistingEnvironment(t *testing.T) {
	cPlatRepo := testLocalRepo(t, testdata.CPlatformEnvsPath())
	env := fmt.Sprintf("%s-env", t.Name())

	tenant, err := Environment(cPlatRepo.Path(), env, testdata.TenantEnvs(), streams)

	assert.ErrorContains(t, err, fmt.Sprintf("config repo path %s: environment %s invalid: cannot find %s environment, available envs: [dev prod]", environment.DirFromCPlatformRepoPath(cPlatRepo.Path()), env, env))
	assert.Nil(t, tenant)
}

func TestEnvironmentSelectorInvalidCPlatRepo(t *testing.T) {
	cPlatRepoPath := t.TempDir()

	tenant, err := Environment(cPlatRepoPath, testdata.DevEnvironment(), testdata.TenantEnvs(), streams)

	assert.ErrorContains(t, err, fmt.Sprintf("couldn't load environment configuration: open %s/environments: no such file or directory", cPlatRepoPath))
	assert.Nil(t, tenant)
}

func testLocalRepo(t *testing.T, path string) *git.LocalRepository {
	_, repo, err := gittest.CreateBareAndLocalRepoFromDir(&gittest.CreateBareAndLocalRepoOp{
		SourceDir:          path,
		TargetBareRepoDir:  t.TempDir(),
		TargetLocalRepoDir: t.TempDir(),
	})
	assert.NoError(t, err)
	return repo
}
