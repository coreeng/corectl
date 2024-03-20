package tenant

import (
	"errors"
	"github.com/coreeng/corectl/pkg/environment"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"unicode/utf8"
)

var tenantsRelativePath = filepath.Join("tenants", "tenants")
var RootName = Name("root")

type Name string

type Tenant struct {
	Name          Name                   `yaml:"name"`
	Parent        Name                   `yaml:"parent"`
	Description   string                 `yaml:"description"`
	ContactEmail  string                 `yaml:"contactEmail"`
	CostCentre    string                 `yaml:"costCentre"`
	Environments  []environment.Name     `yaml:"environments"`
	Repositories  []string               `yaml:"repos"`
	AdminGroup    string                 `yaml:"adminGroup"`
	ReadonlyGroup string                 `yaml:"readonlyGroup"`
	CloudAccess   []interface{}          `yaml:"cloudAccess"`
	RestFields    map[string]interface{} `yaml:",inline"`
	path          *string                `yaml:"-"`
}

func (t *Tenant) AddRepository(repoUrl string) error {
	if slices.Contains(t.Repositories, repoUrl) {
		return errors.New("repository already present")
	}
	t.Repositories = append(t.Repositories, repoUrl)
	return nil
}

var k8sNamespacePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
var k8sLabelValuePattern = regexp.MustCompile(`^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$`)

func ValidateName(name Name) error {
	nameStr := string(name)
	chCount := utf8.RuneCountInString(nameStr)
	if chCount < 1 || chCount > 63 {
		return errors.New("must be a valid K8S namespace name")
	}
	if !k8sNamespacePattern.MatchString(nameStr) {
		return errors.New("must be a valid K8S namespace name")
	}
	return nil
}

func ValidateDescription(description string) error {
	chCount := utf8.RuneCountInString(description)
	if chCount > 253 {
		return errors.New("must be shorter then 253 characters")
	}
	return nil
}

func ValidateCostCentre(costCentre string) error {
	chCount := utf8.RuneCountInString(costCentre)
	if chCount < 1 || chCount > 63 {
		return errors.New("must be a valid K8S label value")
	}
	if !k8sLabelValuePattern.MatchString(costCentre) {
		return errors.New("must be a valida K8S label value")
	}
	return nil
}

func ValidateRepositoryLink(link string) error {
	_, err := url.ParseRequestURI(link)
	if err != nil {
		return err
	}
	return nil
}

func Validate(t *Tenant) error {
	if err := ValidateName(t.Name); err != nil {
		return err
	}
	if err := ValidateDescription(t.Description); err != nil {
		return err
	}
	if err := ValidateCostCentre(t.CostCentre); err != nil {
		return err
	}
	for _, repoLink := range t.Repositories {
		if err := ValidateRepositoryLink(repoLink); err != nil {
			return err
		}
	}
	return nil
}
