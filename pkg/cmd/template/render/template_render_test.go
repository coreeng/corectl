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

	It("should render the template correctly when params file is provided", func() {
		paramsFile = createArgsFile(tempDir, map[string]string{
			"name":   expectedName,
			"tenant": expectedTenantName,
		})

		opts := TemplateRenderOpts{
			IgnoreChecks:  false,
			ArgsFile:      paramsFile,
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

	It("should render the template correctly when params passed with args", func() {
		argsFilePath := createArgsFile(tempDir, map[string]string{
			"name":   "app-name",
			"tenant": "tenant-name",
		})
		opts := TemplateRenderOpts{
			// params3 is an optional parameter
			ArgsFile:      argsFilePath,
			Args:          []string{"param1=param1 value", "param2=321"},
			TemplateName:  testdata.TemplateWithArgs(),
			TargetPath:    targetDir,
			TemplatesPath: templatesLocalRepo.Path(),
		}

		err := run(opts)
		Expect(err).NotTo(HaveOccurred())

		renderedContent, err := os.ReadFile(filepath.Join(targetDir, "args.txt"))
		Expect(err).NotTo(HaveOccurred())
		expectedArgsFileContent :=
			`app-name
tenant-name
param1 value
321
param3 default value

param2 is integer!`
		Expect(string(renderedContent)).To(Equal(expectedArgsFileContent))
	})
})

func createArgsFile(dir string, args map[string]string) string {
	argsContent, err := yaml.Marshal(args)
	Expect(err).NotTo(HaveOccurred())
	argsFile := filepath.Join(dir, "args.yaml")
	err = os.WriteFile(argsFile, argsContent, 0644)
	Expect(err).NotTo(HaveOccurred())
	return argsFile
}
