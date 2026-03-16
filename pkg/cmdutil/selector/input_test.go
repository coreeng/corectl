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

func TestDeliveryUnitSelectorReturnsDeliveryUnit(t *testing.T) {
	cPlatRepo := testLocalRepo(t, testdata.CPlatformEnvsPath())

	du, err := DeliveryUnit(cPlatRepo.Path(), testdata.DefaultTenant(), streams)

	assert.NoError(t, err)
	assert.Equal(t, testdata.DefaultTenant(), du.Name)
	assert.Equal(t, "DeliveryUnit", du.Kind)
}

func TestDeliveryUnitSelectorRejectsOrgUnit(t *testing.T) {
	cPlatRepo := testLocalRepo(t, testdata.CPlatformEnvsPath())

	du, err := DeliveryUnit(cPlatRepo.Path(), "parent", streams)

	assert.ErrorContains(t, err, fmt.Sprintf("config repo path %s: delivery unit parent invalid: cannot find parent delivery unit, available delivery units: [default-tenant]", cPlatRepo.Path()))
	assert.Nil(t, du)
}

func TestDeliveryUnitSelectorNonExistingDeliveryUnit(t *testing.T) {
	cPlatRepo := testLocalRepo(t, testdata.CPlatformEnvsPath())
	duName := fmt.Sprintf("%s-du", t.Name())

	du, err := DeliveryUnit(cPlatRepo.Path(), duName, streams)

	assert.ErrorContains(t, err, fmt.Sprintf("config repo path %s: delivery unit %s invalid: cannot find %s delivery unit, available delivery units: [default-tenant]", cPlatRepo.Path(), duName, duName))
	assert.Nil(t, du)
}

func TestDeliveryUnitSelectorInvalidCPlatRepo(t *testing.T) {
	cPlatRepoPath := t.TempDir() + "some-non-existent-path"
	configpath.SetCorectlHome(cPlatRepoPath)

	du, err := DeliveryUnit(cPlatRepoPath, testdata.DefaultTenant(), streams)

	assert.ErrorContains(t, err, "stat .: no such file or directory")
	assert.Nil(t, du)
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
