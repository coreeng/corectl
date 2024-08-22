package render

import (
	"github.com/coreeng/corectl/pkg/git"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"

	"github.com/coreeng/corectl/pkg/testutil/gittest"
	"github.com/coreeng/corectl/testdata"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Template Render", Ordered, func() {
	t := GinkgoTB()

	var (
		tempDir            string
		targetDir          string
		paramsFile         string
		templatesLocalRepo *git.LocalRepository
		err                error

		expectedTenantName = "some-tenant"
		expectedName       = "some-name"
	)

	BeforeAll(func() {
		_, templatesLocalRepo, err = gittest.CreateBareAndLocalRepoFromDir(&gittest.CreateBareAndLocalRepoOp{
			SourceDir:          testdata.TemplatesPath(),
			TargetBareRepoDir:  t.TempDir(),
			TargetLocalRepoDir: t.TempDir(),
		})
		Expect(err).NotTo(HaveOccurred())

		tempDir = t.TempDir()

		targetDir = filepath.Join(tempDir, "target")
		err = os.MkdirAll(targetDir, 0755)
		Expect(err).NotTo(HaveOccurred())

	})

	AfterAll(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	It("should render the template correctly when params file is provided", func() {
		paramsFile = createParamsFile(tempDir, map[string]string{
			"name":   expectedName,
			"tenant": expectedTenantName,
		})

		opts := TemplateRenderOpts{
			IgnoreChecks:  false,
			ParamsFile:    paramsFile,
			TemplateName:  testdata.BlankTemplate(),
			TargetPath:    targetDir,
			TemplatesPath: templatesLocalRepo.Path(),
		}

		err := run(opts)
		Expect(err).NotTo(HaveOccurred())

		renderedContent, err := os.ReadFile(filepath.Join(targetDir, ".github", "workflows", "extended-test.yaml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(renderedContent)).To(ContainSubstring(expectedName))
		Expect(string(renderedContent)).To(ContainSubstring(expectedTenantName))
	})

	It("should render the template correctly when params file is not provided", func() {

		opts := TemplateRenderOpts{
			IgnoreChecks:  false,
			ParamsFile:    "",
			TemplateName:  testdata.BlankTemplate(),
			TargetPath:    targetDir,
			TemplatesPath: templatesLocalRepo.Path(),
		}

		err := run(opts)
		Expect(err).NotTo(HaveOccurred())

		renderedContent, err := os.ReadFile(filepath.Join(targetDir, ".github", "workflows", "extended-test.yaml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(renderedContent)).NotTo(ContainSubstring(expectedName))
		Expect(string(renderedContent)).NotTo(ContainSubstring(expectedTenantName))
	})
})

func createParamsFile(dir string, params map[string]string) string {
	paramsContent, err := yaml.Marshal(params)
	Expect(err).NotTo(HaveOccurred())
	paramsFile := filepath.Join(dir, "params.yaml")
	err = os.WriteFile(paramsFile, paramsContent, 0644)
	Expect(err).NotTo(HaveOccurred())
	return paramsFile
}
