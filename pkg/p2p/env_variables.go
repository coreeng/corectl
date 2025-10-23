package p2p

import (
	"bytes"
	"fmt"
	"strings"

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

// ShellType represents the type of shell for environment variable export
type ShellType string

const (
	ShellBash       ShellType = "bash"
	ShellZsh        ShellType = "zsh"
	ShellFish       ShellType = "fish"
	ShellPowershell ShellType = "powershell"
	ShellCmd        ShellType = "cmd"
)

// SupportedShells returns a list of all supported shell types
func SupportedShells() []ShellType {
	return []ShellType{ShellBash, ShellZsh, ShellFish, ShellPowershell, ShellCmd}
}

// IsValidShell checks if the given shell type is supported
func IsValidShell(shell string) bool {
	shellType := ShellType(strings.ToLower(shell))
	for _, s := range SupportedShells() {
		if s == shellType {
			return true
		}
	}
	return false
}

type EnvVarContext struct {
	Tenant      *coretnt.Tenant
	Environment *environment.Environment
	AppRepo     *git.LocalRepository
}
type EnvVars map[string]string

// AsExportCmd converts environment variables to shell-specific export statements
func (p2pEnv *EnvVars) AsExportCmd(shell ShellType) (string, error) {
	switch shell {
	case ShellBash, ShellZsh:
		return p2pEnv.formatBashZsh()
	case ShellFish:
		return p2pEnv.formatFish()
	case ShellPowershell:
		return p2pEnv.formatPowershell()
	case ShellCmd:
		return p2pEnv.formatCmd()
	default:
		return "", fmt.Errorf("unsupported shell type: %s", shell)
	}
}

// formatBashZsh formats variables for bash/zsh using double quotes
func (p2pEnv *EnvVars) formatBashZsh() (string, error) {
	b := new(bytes.Buffer)
	for key, value := range *p2pEnv {
		// Always use double quotes and escape special characters within them
		// In double quotes, we need to escape: $ ` " \ and newline
		escapedValue := value
		escapedValue = strings.ReplaceAll(escapedValue, "\\", "\\\\") // Must escape backslash first
		escapedValue = strings.ReplaceAll(escapedValue, "\"", "\\\"") // Escape double quotes
		escapedValue = strings.ReplaceAll(escapedValue, "$", "\\$")   // Escape dollar signs
		escapedValue = strings.ReplaceAll(escapedValue, "`", "\\`")   // Escape backticks
		escapedValue = strings.ReplaceAll(escapedValue, "\n", "\\n")  // Escape newlines

		_, err := fmt.Fprintf(b, "export %s=\"%s\"\n", key, escapedValue)
		if err != nil {
			return "", fmt.Errorf("failed to format bash/zsh export for %s: %w", key, err)
		}
	}
	return b.String(), nil
}

// formatFish formats variables for fish shell
func (p2pEnv *EnvVars) formatFish() (string, error) {
	b := new(bytes.Buffer)
	for key, value := range *p2pEnv {
		// Fish uses single quotes for literal strings, escape single quotes by ending quote, adding escaped quote, and starting new quote
		escapedValue := strings.ReplaceAll(value, "'", "'\\''")
		_, err := fmt.Fprintf(b, "set -gx %s '%s'\n", key, escapedValue)
		if err != nil {
			return "", fmt.Errorf("failed to format fish export for %s: %w", key, err)
		}
	}
	return b.String(), nil
}

// formatPowershell formats variables for PowerShell
func (p2pEnv *EnvVars) formatPowershell() (string, error) {
	b := new(bytes.Buffer)
	for key, value := range *p2pEnv {
		// PowerShell uses single quotes for literal strings, escape single quotes with double single quotes
		escapedValue := strings.ReplaceAll(value, "'", "''")
		_, err := fmt.Fprintf(b, "$Env:%s = '%s'\n", key, escapedValue)
		if err != nil {
			return "", fmt.Errorf("failed to format powershell export for %s: %w", key, err)
		}
	}
	return b.String(), nil
}

// formatCmd formats variables for Windows cmd.exe
func (p2pEnv *EnvVars) formatCmd() (string, error) {
	b := new(bytes.Buffer)
	for key, value := range *p2pEnv {
		// cmd.exe doesn't handle quotes well in set commands, and special chars need escaping with ^
		// We'll keep it simple and not quote unless necessary
		escapedValue := value
		// Escape special characters for cmd (note: must escape ^ first to avoid double-escaping)
		specialChars := []string{"^", "&", "|", "<", ">", "%"}
		for _, char := range specialChars {
			escapedValue = strings.ReplaceAll(escapedValue, char, "^"+char)
		}
		_, err := fmt.Fprintf(b, "set %s=%s\n", key, escapedValue)
		if err != nil {
			return "", fmt.Errorf("failed to format cmd export for %s: %w", key, err)
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
