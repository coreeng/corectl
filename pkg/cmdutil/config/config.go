package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

const (
	CORECTL_DIR    = ".config"
	CORECTL_CONFIG = "corectl.yaml"
)

type Parameter[V interface{}] struct {
	name      string
	flag      string
	shortFlag string
	help      string

	Value V
}

// MarshalYAML It's important to use here value receiver,
// since we define parameters as values (not references) in configuration
func (p Parameter[V]) MarshalYAML() (interface{}, error) {
	return p.Value, nil
}

// UnmarshalYAML It's important to use reference receiver,
// so we modify the actual Parameter struct and not a local copy
func (p *Parameter[V]) UnmarshalYAML(node *yaml.Node) error {
	var value V
	if err := node.Decode(&value); err != nil {
		return err
	}
	p.Value = value
	return nil
}

func (p *Parameter[V]) Name() string {
	return p.name
}

func RegisterStringParameterAsFlag(p *Parameter[string], fs *pflag.FlagSet) {
	if p.flag == "" && p.shortFlag == "" {
		panic("Unexpected flag registration for config parameter")
	}
	if p.shortFlag == "" {
		fs.StringVar(
			&p.Value,
			p.flag,
			p.Value,
			p.help,
		)
	} else {
		fs.StringVarP(
			&p.Value,
			p.flag,
			p.shortFlag,
			p.Value,
			p.help,
		)
	}
	hideDefaultValueFromHelp(p, fs, "")
}

func RegisterBoolParameterAsFlag(p *Parameter[bool], fs *pflag.FlagSet) {
	if p.flag == "" && p.shortFlag == "" {
		panic("Unexpected flag registration for config parameter")
	}
	if p.shortFlag == "" {
		fs.BoolVar(
			&p.Value,
			p.flag,
			p.Value,
			p.help,
		)
	} else {
		fs.BoolVarP(
			&p.Value,
			p.flag,
			p.shortFlag,
			p.Value,
			p.help,
		)
	}
	hideDefaultValueFromHelp(p, fs, "false")
}

func hideDefaultValueFromHelp[V any](p *Parameter[V], fs *pflag.FlagSet, zeroValue string) {
	// do not output value from config in help
	flag := fs.Lookup(p.flag)
	flag.DefValue = zeroValue
}

type Config struct {
	GitHub       GitHubConfig       `yaml:"github"`
	Repositories RepositoriesConfig `yaml:"repositories"`
	P2P          P2PConfig          `yaml:"p2p"`
	path         string
}

type GitHubConfig struct {
	Token        Parameter[string] `yaml:"token"`
	Organization Parameter[string] `yaml:"organization"`
}

type RepositoriesConfig struct {
	CPlatform  Parameter[string] `yaml:"cplatform"`
	Templates  Parameter[string] `yaml:"templates"`
	AllowDirty Parameter[bool]   `yaml:"allow-dirty"`
}

type P2PConfig struct {
	FastFeedback P2PStageConfig `yaml:"fast-feedback"`
	ExtendedTest P2PStageConfig `yaml:"extended-test"`
	Prod         P2PStageConfig `yaml:"prod"`
}

type P2PStageConfig struct {
	DefaultEnvs Parameter[[]string] `yaml:"default-envs"`
}

func NewConfig() *Config {
	return &Config{
		GitHub: GitHubConfig{
			Token: Parameter[string]{
				flag: "github-token",
				help: "Personal GitHub token to use for GitHub authentication",
			},
			Organization: Parameter[string]{
				flag: "github-org",
				help: "GitHub organization to create the new app into (if different from default)",
			},
		},
		Repositories: RepositoriesConfig{
			CPlatform: Parameter[string]{
				name: "cplatform repository",
				flag: "cplatform",
				help: "Path to local repository with core-platform configuration",
			},
			Templates: Parameter[string]{
				name: "template repository",
				flag: "templates",
				help: "Path to local repository with software templates",
			},
			AllowDirty: Parameter[bool]{
				name: "Allow dirty config repositories",
				flag: "allow-dirty-config-repos",
				help: "Allow local changes in configuration repositories",
			},
		},
		path: "",
	}
}

func NewTestPersistedConfig() *Config {
	config := NewConfig()
	config.path = "/this/is/a/mock/path"
	return config
}

func (c *Config) IsPersisted() bool {
	return c.path != ""
}

func DiscoverConfig() (*Config, error) {
	configPath, err := Path()
	if err != nil {
		return nil, err
	}
	return ReadConfig(configPath)
}

func ReadConfig(path string) (*Config, error) {
	fileContent, err := os.ReadFile(path)
	config := NewConfig()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config, nil
		}
		return nil, err
	}

	if err = yaml.Unmarshal(fileContent, &config); err != nil {
		return nil, err
	}
	config.path = path
	return config, nil
}

func (c *Config) Save() error {
	path := c.path
	if path == "" {
		var err error
		path, err = Path()
		if err != nil {
			return err
		}
	}

	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return err
	}

	configBytes, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	if err = os.WriteFile(path, configBytes, 0o600); err != nil {
		return err
	}
	c.path = path
	return nil
}

func (c *Config) Path() string {
	return c.path
}

func (c *Config) BaseDir() (string, error) {
	path, err := Path()
	if err != nil {
		return "", err
	}
	return filepath.Dir(path), nil
}

func Path() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(homeDir, CORECTL_DIR, "corectl", CORECTL_CONFIG)

	return path, nil
}

// SetValue changes config value by yamlPath if possible
// and returns an updated copy of the `c`
func (c *Config) SetValue(yamlPath string, value string) (*Config, error) {
	var cfgYaml yaml.Node
	cfgBytes, err := yaml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}
	if err := yaml.Unmarshal(cfgBytes, &cfgYaml); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	path, err := yamlpath.NewPath(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find property to modify: %w", err)
	}
	nodeToModify, err := path.Find(&cfgYaml)
	if len(nodeToModify) != 1 {
		return nil, fmt.Errorf("path represents multiple nodes: %w", err)
	}
	if nodeToModify[0].Kind != yaml.ScalarNode {
		return nil, fmt.Errorf("path does not represent a scalar node: %w", err)
	}
	nodeToModify[0].Value = value

	cfgBytes, err = yaml.Marshal(&cfgYaml)
	if err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}
	cfgNew := NewConfig()
	if err := yaml.Unmarshal(cfgBytes, &cfgNew); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}
	return cfgNew, nil
}
