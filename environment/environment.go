package environment

import "path/filepath"

var environmentsRelativePath string = filepath.Join("environments")

type Name string

type Environment struct {
	Environment Name `yaml:"environment"`
	Platform    struct {
		ProjectId     string `yaml:"projectId"`
		ProjectNumber string `yaml:"projectNumber"`
	}
	IngressDomains   []Domain `yaml:"ingressDomains"`
	InternalServices Domain   `yaml:"internalServices"`
}

type Domain struct {
	Name   string `yaml:"name"`
	Domain string `yaml:"domain"`
}

func (e *Environment) GetDefaultIngressDomain() *Domain {
	for i := range e.IngressDomains {
		if e.IngressDomains[i].Name == "default" {
			return &e.IngressDomains[i]
		}
	}
	if len(e.IngressDomains) > 0 {
		return &e.IngressDomains[0]
	}
	return nil
}
