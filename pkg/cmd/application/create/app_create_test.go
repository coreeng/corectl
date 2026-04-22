package create

import (
	"testing"

	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/coreeng/corectl/pkg/testutil/gittest"
	"github.com/coreeng/corectl/testdata"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAppCreateSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "App Create Suite")
}

var _ = Describe("AppCreateOpt", func() {
	Describe("createTemplateInput", func() {
		var (
			opts              *AppCreateOpt
			existingTemplates []template.Spec
			input             userio.InputSourceSwitch[string, *template.Spec]
		)

		Context("when creating template input with existing templates", func() {
			BeforeEach(func() {
				opts = &AppCreateOpt{}
				existingTemplates = []template.Spec{
					{Name: "template1"},
					{Name: "template2"},
					{Name: "template3"},
				}
				input = opts.createTemplateInput(existingTemplates)
			})

			It("should create a template input with an 'empty' option and existing templates", func() {
				prompt, err := input.InteractivePromptFn()

				Expect(err).NotTo(HaveOccurred())
				singleSelect, ok := prompt.(*userio.SingleSelect)
				Expect(ok).To(BeTrue())
				Expect(singleSelect.Items).To(Equal([]string{"<empty>", "template1", "template2", "template3"}))
			})

			It("should handle empty selection", func() {
				result, err := input.ValidateAndMap("<empty>")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(BeNil())
			})

			It("should handle empty string - default value of a FromTemplate input", func() {
				result, err := input.ValidateAndMap("")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(BeNil())
			})

			It("should handle valid template selection", func() {
				result, err := input.ValidateAndMap("template2")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(&existingTemplates[1]))
			})

			It("should handle invalid template selection", func() {
				result, err := input.ValidateAndMap("nonexistent")
				Expect(err).To(MatchError("unknown template"))
				Expect(result).To(BeNil())
			})
		})

		Context("when creating template input with empty list of existing templates", func() {
			BeforeEach(func() {
				opts = &AppCreateOpt{}
				existingTemplates = []template.Spec{}
				input = opts.createTemplateInput(existingTemplates)
			})

			It("should handle an empty list of existing templates", func() {
				prompt, err := input.InteractivePromptFn()
				Expect(err).NotTo(HaveOccurred())
				singleSelect, ok := prompt.(*userio.SingleSelect)
				Expect(ok).To(BeTrue())
				Expect(singleSelect.Items).To(Equal([]string{"<empty>"}))

				result, err := input.ValidateAndMap("<empty>")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(BeNil())
			})

			It("should handle interactive mode", func() {
				value, err := input.GetValue(userio.NewIOStreamsWithInteractive(nil, nil, nil, false))

				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(BeNil())
			})
		})
	})

	Describe("deliveryUnitTypeFromTemplate", func() {
		It("defaults to application when template is nil", func() {
			duType, err := deliveryUnitTypeFromTemplate(nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(duType).To(Equal("application"))
		})

		It("maps app kind to application", func() {
			s := &template.Spec{Kind: "app"}
			duType, err := deliveryUnitTypeFromTemplate(s)
			Expect(err).NotTo(HaveOccurred())
			Expect(duType).To(Equal("application"))
		})

		It("maps infra kind to infrastructure", func() {
			s := &template.Spec{Kind: "infra"}
			duType, err := deliveryUnitTypeFromTemplate(s)
			Expect(err).NotTo(HaveOccurred())
			Expect(duType).To(Equal("infrastructure"))
		})

		It("rejects unknown kinds", func() {
			s := &template.Spec{Kind: "weird"}
			_, err := deliveryUnitTypeFromTemplate(s)
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("addExistingTenants", func() {
	It("adds pointers to each tenant in the slice", func() {
		tenants := []coretnt.Tenant{
			{Name: "ou-alpha"},
			{Name: "ou-beta"},
		}
		tenantMap := map[string]*coretnt.Tenant{}

		addExistingTenants(tenantMap, tenants)

		Expect(tenantMap).To(HaveKey("ou-alpha"))
		Expect(tenantMap).To(HaveKey("ou-beta"))
		Expect(tenantMap["ou-alpha"]).To(BeIdenticalTo(&tenants[0]))
		Expect(tenantMap["ou-beta"]).To(BeIdenticalTo(&tenants[1]))
	})
})

var _ = Describe("createDeliveryUnitForOrgUnit", func() {
	It("inherits admin groups from the parent org unit", func() {
		t := GinkgoT()

		_, err := gittest.CreateTestCorectlConfig(t.TempDir())
		Expect(err).NotTo(HaveOccurred())
		_, _, err = gittest.CreateBareAndLocalRepoFromDir(&gittest.CreateBareAndLocalRepoOp{
			SourceDir:          testdata.CPlatformEnvsPath(),
			TargetBareRepoDir:  t.TempDir(),
			TargetLocalRepoDir: configpath.GetCorectlCPlatformDir(),
		})
		Expect(err).NotTo(HaveOccurred())

		orgUnit, err := coretnt.FindByName(configpath.GetCorectlCPlatformDir("tenants"), "parent")
		Expect(err).NotTo(HaveOccurred())
		orgUnit.ProdAdminGroup = "prod-admin-group@prod.domain"
		orgUnit.ProdReadOnlyGroup = "prod-readonly-group@prod.domain"

		du, err := createDeliveryUnitForOrgUnit(&AppCreateOpt{Name: "new-app"}, orgUnit, "application")
		Expect(err).NotTo(HaveOccurred())
		Expect(du.AdminGroup).To(Equal(orgUnit.AdminGroup))
		Expect(du.ReadOnlyGroup).To(Equal(orgUnit.ReadOnlyGroup))
		Expect(du.ProdAdminGroup).To(Equal(orgUnit.ProdAdminGroup))
		Expect(du.ProdReadOnlyGroup).To(Equal(orgUnit.ProdReadOnlyGroup))
	})
})
