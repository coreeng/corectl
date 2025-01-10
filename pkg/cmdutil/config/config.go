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

const DEFAULT_CORECTL_CONFIG = "corectl.yaml" // I don't think this should be a variable, always keep it as corectl.yaml
var DEFAULT_CORECTL_DIRS = []string{".config", "corectl"} // this should be variable, the folder used can be changed if so desired
// can't seem to make this constant, since it's a slice

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
	path         string				`yaml:"path"`		
	ConfigPaths  ConfigPaths 		`yaml:"configPaths"`
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

type ConfigPaths struct {
	Directory Parameter[string] `yaml:"directory"`
	Filename  Parameter[string] `yaml:"filename"`
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
				help: "GitHub organization to create the new app into (if different from default, ignored for monorepos)",
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
		ConfigPaths: ConfigPaths{
			Directory: Parameter[string]{
				name: "corectl config directory",
				flag: "config-dir",
				help: "allow specifying the folder for configuration",
			},
			Filename: Parameter[string]{
				name: "corectl config file name",
				flag: "config-file",
				help: "allow specifying the filename of the corectl.yaml configuration file. Must be held in the config-dir.",
			},
		},
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
	configPath, err := Path(nil)
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
	return config, nil
}

func (c *Config) Save() (string, error) {
	fmt.Println("Saving config")
	path, err := c.Path()
	fmt.Println("got the path: ", path)
	if err != nil {
		fmt.Println("Error getting path for Save, so just returning an empty string")
		return "", err
	}
	fmt.Println("marshalling yaml")
	configBytes, err := yaml.Marshal(c)
	fmt.Println("yaml marshalled")
	if err != nil {
		return "", err
	}
	fmt.Println("writing file to path: ", path)
	if err = os.WriteFile(path, configBytes, 0o600); err != nil {
		fmt.Println("Error writing file for Save, so just returning an empty string and error")
		return "", err
	}
	fmt.Println("returning path: ", path)
	c.path = path
	return path, nil
}

func (c *Config) BaseDir() (string, error) {
	return BaseDir(c)
}

// BaseDir returns the path to the base directory, usually <user-home>/.config/corectl unless otherwise specified, for holding logs, repositories, config files, etc.
func BaseDir(c *Config) (string, error) {
	if c != nil && c.ConfigPaths.Directory.Value != "" {
		return c.ConfigPaths.Directory.Value, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting user home directory for BaseDir, so just returning an empty string")
		return "", err
	}
	return filepath.Join(homeDir, filepath.Join(DEFAULT_CORECTL_DIRS...)), nil
}

func (c *Config) RepositoriesDir() (string, error) {
	return RepositoriesDir(c)
}

// RepositoriesDir returns the path to the repositories directory, always <base-dir>/repositories, for holding config repos
func RepositoriesDir(c *Config) (string, error) {
	if c != nil {
		baseDir, err := c.BaseDir()
		if err != nil {
			fmt.Println("Error getting base directory for RepositoriesDir, so just returning an empty string")
			return "", err
		}
		return filepath.Join(baseDir, "repositories"), nil
	}
	baseDir, err := BaseDir(nil)
	if err != nil {
		fmt.Println("Error getting base directory for RepositoriesDir, so just returning an empty string")
		return "", err
	}
	return filepath.Join(baseDir, "repositories"), nil
}

func (c *Config) Path() (string, error) {
	return Path(c)
}

// Path returns the path to the configuration file, usually <config-dir>/corectl.yaml
func Path(c *Config) (string, error) {
	// first half of this should be in BaseDir()
	var corectlDir, corectlFile string
	corectlDir, err := BaseDir(c)
	if err != nil {
		fmt.Println("Error getting base directory for Path, so just returning an empty string")
		return "", err
	}
	if c != nil && c.ConfigPaths.Filename.Value != "" {
		corectlFile = c.ConfigPaths.Filename.Value
	} else {
		corectlFile = DEFAULT_CORECTL_CONFIG
	}
	path := filepath.Join(corectlDir, corectlFile)
	fmt.Println("Returning path: ", path)
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
