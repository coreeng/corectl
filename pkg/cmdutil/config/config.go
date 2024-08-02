package config

import (
	"errors"
	"os"
	"path/filepath"

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

	// do not output value from config in help
	flag := fs.Lookup(p.flag)
	flag.DefValue = ""
}

type Config struct {
	GitHub struct {
		Token        Parameter[string] `yaml:"token"`
		Organization Parameter[string] `yaml:"organization"`
	} `yaml:"github"`
	Repositories struct {
		CPlatform Parameter[string] `yaml:"cplatform"`
		Templates Parameter[string] `yaml:"templates"`
	} `yaml:"repositories"`
	P2P struct {
		FastFeedback P2PStageConfig `yaml:"fast-feedback"`
		ExtendedTest P2PStageConfig `yaml:"extended-test"`
		Prod         P2PStageConfig `yaml:"prod"`
	} `yaml:"p2p"`
	path string
}

type P2PStageConfig struct {
	DefaultEnvs Parameter[[]string] `yaml:"default-envs"`
}

func newConfig() *Config {
	config := Config{}
	config.GitHub.Token.flag = "github-token"
	config.GitHub.Token.help = "Personal GitHub token to use for GitHub authentication"

	config.GitHub.Organization.flag = "github-org"
	config.GitHub.Organization.help = "Github organization your company is using"

	config.Repositories.CPlatform.name = "cplatform repository"
	config.Repositories.CPlatform.flag = "cplatform"
	config.Repositories.CPlatform.help = "Path to local repository with core-platform configuration"

	config.Repositories.Templates.name = "template repository"
	config.Repositories.Templates.flag = "templates"
	config.Repositories.Templates.help = "Path to local repository with software templates"
	return &config
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
	config := newConfig()
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
