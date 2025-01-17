package testdata

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"text/template"
)

var basepath string

func init() {
	_, currentFile, _, _ := runtime.Caller(0)
	basepath = filepath.Dir(currentFile)
}

func Path(rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(basepath, rel)
}

func CPlatformEnvsPath() string {
	return Path("./repositories/cplatform")
}

func TemplatesPath() string {
	return Path("./repositories/templates")
}

func RenderInitFile(
	dst string,
	cplatformRepoUrl string,
	templateRepoUrl string,
) error {
	initFileTemplatePath := Path("./corectl-init.yaml.tmpl")
	templateBytes, err := os.ReadFile(initFileTemplatePath)
	if err != nil {
		return err
	}
	t := template.New("init-file")
	t, err = t.Parse(string(templateBytes))
	if err != nil {
		return err
	}
	buffer := bytes.Buffer{}
	if err = t.Execute(&buffer, map[string]string{
		"cplatformUrl": cplatformRepoUrl,
		"templatesUrl": templateRepoUrl,
	}); err != nil {
		return err
	}
	if err = os.WriteFile(dst, buffer.Bytes(), 0o644); err != nil {
		return err
	}
	return nil
}

func DefaultTenant() string {
	return "default-tenant"
}

func DevEnvironment() string {
	return "dev"
}

func TenantEnvs() []string {
	return []string{DevEnvironment(), ProdEnvironment()}
}

func ProdEnvironment() string {
	return "prod"
}

func BlankTemplate() string {
	return "blank"
}

func TemplateWithArgs() string {
	return "with-args"
}

func Monorepo() string {
	return "monorepo"
}
