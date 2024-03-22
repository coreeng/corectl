package environment

import "errors"

const environmentsRelativePath = "environments"

type Name string

type Environment struct {
	Environment Name `yaml:"environment"`
	Platform    struct {
		ProjectId     string `yaml:"projectId"`
		ProjectNumber string `yaml:"projectNumber"`
	} `yaml:"platform"`
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

func Validate(env *Environment) error {
	if string(env.Environment) == "" {
		return errors.New("environment is empty")
	}
	if env.Platform.ProjectId == "" {
		return errors.New("projectId is missing")
	}
	if env.Platform.ProjectNumber == "" {
		return errors.New("projectNumber is missing")
	}

	defaultIngressDomain := env.GetDefaultIngressDomain()
	if defaultIngressDomain == nil {
		return errors.New("default ingress domain is not found")
	}
	if defaultIngressDomain.Name == "" {
		return errors.New("default ingress domain name is missing")
	}
	if defaultIngressDomain.Domain == "" {
		return errors.New("default ingress domain is missing")
	}

	if env.InternalServices.Name == "" {
		return errors.New("internalServices name is missing")
	}
	if env.InternalServices.Domain == "" {
		return errors.New("internalServices domain is missing")
	}
	return nil
}
