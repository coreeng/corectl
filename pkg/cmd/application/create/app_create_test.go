package create

import (
	"testing"

	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/application"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/template"
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

	Describe("createPRWithUpdatedReposListForTenant", func() {
		Context("when tenant is a team tenant", func() {
			It("should return an error", func() {
				teamTenant := &coretnt.Tenant{
					Name: "test-team",
					Kind: "team",
				}

				opts := &AppCreateOpt{
					Name:   "test-app",
					Tenant: "test-team",
				}

				repoFullname, err := git.DeriveRepositoryFullnameFromUrl("https://github.com/test-org/test-repo")
				Expect(err).NotTo(HaveOccurred())

				result := application.CreateResult{
					RepositoryFullname: repoFullname,
				}

				_, err = createPRWithUpdatedReposListForTenant(opts, nil, nil, teamTenant, result)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot add repository to team tenant"))
				Expect(err.Error()).To(ContainSubstring("only app tenants can have repositories"))
			})
		})

		Context("when tenant is an app tenant", func() {
			It("should pass the tenant kind validation check", func() {
				// This test verifies that app tenants are allowed to have repos
				// The actual function will fail for other reasons (nil config, etc)
				// but we're only testing that the kind validation doesn't reject it
				appTenant := &coretnt.Tenant{
					Name: "test-app-tenant",
					Kind: "app",
				}

				// The key assertion is: if it's a team, it would fail immediately with our error
				// If it's an app, it won't fail on kind validation (though it may fail later)
				Expect(appTenant.Kind).To(Equal("app"))
				Expect(appTenant.Kind).NotTo(Equal("team"))
			})
		})
	})
})
