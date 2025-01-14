package export

import (
	"bytes"
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
	"os"
	"testing"

	"github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/p2p"
	"github.com/coreeng/corectl/pkg/testutil/gittest"
	"github.com/coreeng/corectl/testdata"
	"github.com/stretchr/testify/assert"
)

var streams = userio.NewIOStreams(os.Stdin, os.Stdout, os.Stderr)

func TestMain(m *testing.M) {
	m.Run()
}

func TestRunExportPrintsEnvVarsToStdOut(t *testing.T) {
	var output, stderr bytes.Buffer

	_ = testLocalRepo(t, testdata.CPlatformEnvsPath()).Path()
	cfg, err := config.DiscoverConfig()
	assert.NoError(t, err)
	err = run(&exportOpts{
		tenant:          testdata.DefaultTenant(),
		environmentName: testdata.DevEnvironment(),
		repoPath:        testLocalRepo(t, testdata.CPlatformEnvsPath()).Path(),
		streams:         userio.NewIOStreams(os.Stdin, &output, &stderr),
	}, cfg)

	assert.NoError(t, err)
	assert.Contains(t, stderr.String(), "export", p2p.BaseDomain, p2p.Registry, p2p.Version, p2p.RepoPath, p2p.TenantName, p2p.Region)
}

func TestRunExportNonExistingAppRepo(t *testing.T) {
	_ = testLocalRepo(t, testdata.CPlatformEnvsPath()).Path()
	cfg, err := config.DiscoverConfig()
	assert.NoError(t, err)

	appRepoPath := t.TempDir()
	err = run(&exportOpts{
		tenant:          testdata.DefaultTenant(),
		environmentName: testdata.DevEnvironment(),
		repoPath:        appRepoPath,
		streams:         streams,
	}, cfg)

	assert.ErrorContains(t, err, fmt.Sprintf("repository on path %s not found: repository does not exist", appRepoPath))
}

func TestRunExportNonExistingTenant(t *testing.T) {
	tenantName := fmt.Sprintf("%s-tenant", t.Name())
	_ = testLocalRepo(t, testdata.CPlatformEnvsPath()).Path()
	cfg, err := config.DiscoverConfig()
	assert.NoError(t, err)

	err = run(&exportOpts{
		tenant:          tenantName,
		environmentName: testdata.DevEnvironment(),
		repoPath:        testLocalRepo(t, testdata.CPlatformEnvsPath()).Path(),
		streams:         streams,
	}, cfg)

	assert.ErrorContains(t, err, fmt.Sprintf("config repo path %s/tenants: tenant %s invalid: cannot find %s tenant, available tenants: [default-tenant parent root]",
		configpath.GetCorectlCPlatformDir(), tenantName, tenantName))
}

func TestFailureWithRootTenant(t *testing.T) {
	tenantName := tenant.RootName
	_ = testLocalRepo(t, testdata.CPlatformEnvsPath()).Path()
	cfg, err := config.DiscoverConfig()
	assert.NoError(t, err)

	err = run(&exportOpts{
		tenant:          tenantName,
		environmentName: testdata.DevEnvironment(),
		repoPath:        testLocalRepo(t, testdata.CPlatformEnvsPath()).Path(),
		streams:         streams,
	}, cfg)

	assert.ErrorContains(t, err, fmt.Sprintf("cannot connect to '%s' as that's the root tenant and cannot be used", tenant.RootName))
}

func TestRunExportNonExistingEnvironment(t *testing.T) {
	envName := fmt.Sprintf("%s-env", t.Name())
	_ = testLocalRepo(t, testdata.CPlatformEnvsPath()).Path()
	cfg, err := config.DiscoverConfig()
	assert.NoError(t, err)

	err = run(&exportOpts{
		tenant:          testdata.DefaultTenant(),
		environmentName: envName,
		repoPath:        testLocalRepo(t, testdata.CPlatformEnvsPath()).Path(),
		streams:         streams,
	}, cfg)

	assert.ErrorContains(t, err, fmt.Sprintf("config repo path %s/environments: environment %s invalid: cannot find %s environment, available envs: [dev prod]",
		configpath.GetCorectlCPlatformDir(), envName, envName))
}

func testLocalRepo(t *testing.T, path string) *git.LocalRepository {
	corectlDir := t.TempDir()
	configpath.SetCorectlHome(corectlDir)
	_, repo, err := gittest.CreateBareAndLocalRepoFromDir(&gittest.CreateBareAndLocalRepoOp{
		SourceDir:          path,
		TargetBareRepoDir:  t.TempDir(),
		TargetLocalRepoDir: configpath.GetCorectlCPlatformDir(),
	})
	assert.NoError(t, err)
	return repo
}
