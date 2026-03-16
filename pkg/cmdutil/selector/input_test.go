package selector

import (
	"fmt"
	"os"
	"testing"

	"github.com/coreeng/corectl/pkg/cmdutil/configpath"

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

	assert.ErrorContains(t, err, fmt.Sprintf("config repo path %s/tenants: tenant %s invalid: cannot find %s tenant, available tenants: [default-tenant parent root]", cPlatRepo.Path(), tenantName, tenantName))
	assert.Nil(t, tenant)
}

func TestTenantSelectorInvalidCPlatRepo(t *testing.T) {
	cPlatRepoPath := t.TempDir()
	configpath.SetCorectlHome(cPlatRepoPath)

	tenant, err := Tenant(cPlatRepoPath, testdata.DefaultTenant(), streams)

	assert.ErrorContains(t, err, fmt.Sprintf("couldn't load tenant configuration in path %s/repositories/cplatform/tenants: stat .: no such file or directory", cPlatRepoPath))
	assert.Nil(t, tenant)
}

func TestOrgUnitSelectorReturnsOrgUnit(t *testing.T) {
	cPlatRepo := testLocalRepo(t, testdata.CPlatformEnvsPath())

	orgUnit, err := OrgUnit(cPlatRepo.Path(), "parent", streams)

	assert.NoError(t, err)
	assert.Equal(t, "parent", orgUnit.Name)
	assert.Equal(t, "OrgUnit", orgUnit.Kind)
}

func TestOrgUnitSelectorRejectsDeliveryUnit(t *testing.T) {
	cPlatRepo := testLocalRepo(t, testdata.CPlatformEnvsPath())

	orgUnit, err := OrgUnit(cPlatRepo.Path(), testdata.DefaultTenant(), streams)

	assert.ErrorContains(t, err, fmt.Sprintf("config repo path %s: org unit %s invalid: cannot find %s org unit, available org units: [parent]", cPlatRepo.Path(), testdata.DefaultTenant(), testdata.DefaultTenant()))
	assert.Nil(t, orgUnit)
}

func TestOrgUnitSelectorNonExistingOrgUnit(t *testing.T) {
	cPlatRepo := testLocalRepo(t, testdata.CPlatformEnvsPath())
	orgUnitName := fmt.Sprintf("%s-ou", t.Name())

	orgUnit, err := OrgUnit(cPlatRepo.Path(), orgUnitName, streams)

	assert.ErrorContains(t, err, fmt.Sprintf("config repo path %s: org unit %s invalid: cannot find %s org unit, available org units: [parent]", cPlatRepo.Path(), orgUnitName, orgUnitName))
	assert.Nil(t, orgUnit)
}

func TestOrgUnitSelectorInvalidCPlatRepo(t *testing.T) {
	cPlatRepoPath := t.TempDir() + "some-non-existent-path"
	configpath.SetCorectlHome(cPlatRepoPath)

	orgUnit, err := OrgUnit(cPlatRepoPath, "parent", streams)

	assert.ErrorContains(t, err, "stat .: no such file or directory")
	assert.Nil(t, orgUnit)
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

	assert.ErrorContains(t, err, fmt.Sprintf("config repo path %s: environment %s invalid: cannot find %s environment, available envs: [dev prod]", configpath.GetCorectlCPlatformDir("environments"), env, env))
	assert.Nil(t, tenant)
}

func TestEnvironmentSelectorInvalidCPlatRepo(t *testing.T) {
	cPlatRepoPath := t.TempDir()
	_, err := gittest.CreateTestCorectlConfig(cPlatRepoPath)
	assert.NoError(t, err)

	tenant, err := Environment(cPlatRepoPath, testdata.DevEnvironment(), testdata.TenantEnvs(), streams)
	assert.ErrorContains(t, err, fmt.Sprintf("couldn't load environment configuration: open %s/repositories/cplatform/environments: no such file or directory", cPlatRepoPath))
	assert.Nil(t, tenant)
}

func testLocalRepo(t *testing.T, path string) *git.LocalRepository {
	_, err := gittest.CreateTestCorectlConfig(t.TempDir())
	assert.NoError(t, err)
	_, repo, err := gittest.CreateBareAndLocalRepoFromDir(&gittest.CreateBareAndLocalRepoOp{
		SourceDir:          path,
		TargetBareRepoDir:  t.TempDir(),
		TargetLocalRepoDir: configpath.GetCorectlCPlatformDir(),
	})
	assert.NoError(t, err)
	return repo
}
