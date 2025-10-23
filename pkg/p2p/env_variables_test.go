package p2p

import (
	"fmt"
	"testing"

	"github.com/coreeng/core-platform/pkg/environment"
	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/testutil/gittest"
	"github.com/coreeng/corectl/testdata"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	m.Run()
}

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
	exportCmd := asExportCmd(t, vars)
	// Check that all variables are present in the export command
	assert.Contains(t, exportCmd, fmt.Sprintf("export %s=", BaseDomain))
	assert.Contains(t, exportCmd, fmt.Sprintf("export %s=", Registry))
	assert.Contains(t, exportCmd, fmt.Sprintf("export %s=", Version))
	assert.Contains(t, exportCmd, fmt.Sprintf("export %s=", RepoPath))
	assert.Contains(t, exportCmd, fmt.Sprintf("export %s=", Region))
	assert.Contains(t, exportCmd, fmt.Sprintf("export %s=", TenantName))
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
	})
	assert.NoError(t, err)
	return repo
}

var commitHash = func(t *testing.T, repo *git.LocalRepository) string {
	hash, err := repo.HeadShortCommitHash()
	assert.NoError(t, err)
	return hash
}

var asExportCmd = func(t *testing.T, vars *EnvVars) string {
	cmd, err := vars.AsExportCmd(ShellBash)
	assert.NoError(t, err)
	return cmd
}

var toRegistry = func(_ *testing.T, vendor *environment.GCPVendor, tenantName string) string {
	return fmt.Sprintf("%s-docker.pkg.dev/%s/tenant/%s", vendor.Region, vendor.ProjectId, tenantName)
}

// TestShellFormats tests export formatting for all supported shell types
func TestShellFormats(t *testing.T) {
	testCases := []struct {
		name     string
		shell    ShellType
		vars     EnvVars
		expected map[string]string // key -> expected line in output
	}{
		{
			name:  "bash basic",
			shell: ShellBash,
			vars: EnvVars{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			expected: map[string]string{
				"KEY1": "export KEY1=\"value1\"\n",
				"KEY2": "export KEY2=\"value2\"\n",
			},
		},
		{
			name:  "bash with spaces",
			shell: ShellBash,
			vars: EnvVars{
				"KEY": "value with spaces",
			},
			expected: map[string]string{
				"KEY": "export KEY=\"value with spaces\"\n",
			},
		},
		{
			name:  "bash with special chars",
			shell: ShellBash,
			vars: EnvVars{
				"KEY": "value$with&special|chars",
			},
			expected: map[string]string{
				// Dollar signs must be escaped in double quotes
				"KEY": "export KEY=\"value\\$with&special|chars\"\n",
			},
		},
		{
			name:  "bash with quotes",
			shell: ShellBash,
			vars: EnvVars{
				"KEY": "value'with\"quotes",
			},
			expected: map[string]string{
				// Double quotes must be escaped in double quotes, single quotes don't need escaping
				"KEY": "export KEY=\"value'with\\\"quotes\"\n",
			},
		},
		{
			name:  "bash with backslash",
			shell: ShellBash,
			vars: EnvVars{
				"KEY": "value\\with\\backslashes",
			},
			expected: map[string]string{
				// Backslashes must be escaped
				"KEY": "export KEY=\"value\\\\with\\\\backslashes\"\n",
			},
		},
		{
			name:  "bash with backtick",
			shell: ShellBash,
			vars: EnvVars{
				"KEY": "value`with`backticks",
			},
			expected: map[string]string{
				// Backticks must be escaped
				"KEY": "export KEY=\"value\\`with\\`backticks\"\n",
			},
		},
		{
			name:  "zsh basic",
			shell: ShellZsh,
			vars: EnvVars{
				"KEY": "value",
			},
			expected: map[string]string{
				"KEY": "export KEY=\"value\"\n",
			},
		},
		{
			name:  "fish basic",
			shell: ShellFish,
			vars: EnvVars{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			expected: map[string]string{
				"KEY1": "set -gx KEY1 'value1'\n",
				"KEY2": "set -gx KEY2 'value2'\n",
			},
		},
		{
			name:  "fish with spaces",
			shell: ShellFish,
			vars: EnvVars{
				"KEY": "value with spaces",
			},
			expected: map[string]string{
				"KEY": "set -gx KEY 'value with spaces'\n",
			},
		},
		{
			name:  "fish with single quotes",
			shell: ShellFish,
			vars: EnvVars{
				"KEY": "value'with'quotes",
			},
			expected: map[string]string{
				"KEY": "set -gx KEY 'value'\\''with'\\''quotes'\n",
			},
		},
		{
			name:  "powershell basic",
			shell: ShellPowershell,
			vars: EnvVars{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			expected: map[string]string{
				"KEY1": "$Env:KEY1 = 'value1'\n",
				"KEY2": "$Env:KEY2 = 'value2'\n",
			},
		},
		{
			name:  "powershell with spaces",
			shell: ShellPowershell,
			vars: EnvVars{
				"KEY": "value with spaces",
			},
			expected: map[string]string{
				"KEY": "$Env:KEY = 'value with spaces'\n",
			},
		},
		{
			name:  "powershell with single quotes",
			shell: ShellPowershell,
			vars: EnvVars{
				"KEY": "value'with'quotes",
			},
			expected: map[string]string{
				"KEY": "$Env:KEY = 'value''with''quotes'\n",
			},
		},
		{
			name:  "cmd basic",
			shell: ShellCmd,
			vars: EnvVars{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			expected: map[string]string{
				"KEY1": "set KEY1=value1\n",
				"KEY2": "set KEY2=value2\n",
			},
		},
		{
			name:  "cmd with spaces",
			shell: ShellCmd,
			vars: EnvVars{
				"KEY": "value with spaces",
			},
			expected: map[string]string{
				"KEY": "set KEY=value with spaces\n",
			},
		},
		{
			name:  "cmd with special chars",
			shell: ShellCmd,
			vars: EnvVars{
				"KEY": "value&with|special>chars",
			},
			expected: map[string]string{
				"KEY": "set KEY=value^&with^|special^>chars\n",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := tc.vars.AsExportCmd(tc.shell)
			assert.NoError(t, err)

			for key, expectedLine := range tc.expected {
				assert.Contains(t, output, expectedLine,
					"Expected to find '%s' in output for %s", expectedLine, key)
			}
		})
	}
}

// TestShellTypeValidation tests the shell type validation functions
func TestShellTypeValidation(t *testing.T) {
	testCases := []struct {
		name     string
		shell    string
		isValid  bool
	}{
		{"bash lowercase", "bash", true},
		{"bash uppercase", "BASH", true},
		{"bash mixed case", "BaSh", true},
		{"zsh", "zsh", true},
		{"fish", "fish", true},
		{"powershell", "powershell", true},
		{"pwsh alias", "pwsh", false}, // not supported as alias
		{"cmd", "cmd", true},
		{"invalid shell", "invalid", false},
		{"empty string", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsValidShell(tc.shell)
			assert.Equal(t, tc.isValid, result,
				"IsValidShell(%s) should return %v", tc.shell, tc.isValid)
		})
	}
}

// TestSupportedShells tests that all expected shells are in the supported list
func TestSupportedShells(t *testing.T) {
	shells := SupportedShells()
	assert.Len(t, shells, 5, "Should have exactly 5 supported shells")
	assert.Contains(t, shells, ShellBash)
	assert.Contains(t, shells, ShellZsh)
	assert.Contains(t, shells, ShellFish)
	assert.Contains(t, shells, ShellPowershell)
	assert.Contains(t, shells, ShellCmd)
}

// TestUnsupportedShellType tests error handling for unsupported shell types
func TestUnsupportedShellType(t *testing.T) {
	vars := EnvVars{"KEY": "value"}
	_, err := vars.AsExportCmd(ShellType("unsupported"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported shell type")
}
