package create

import (
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/template"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestAppCreateSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "App Create Suite")
}

var _ = Describe("AppCreateOpt", func() {
	Describe("createTemplateInput", func() {
		var opts *AppCreateOpt

		BeforeEach(func() {
			opts = &AppCreateOpt{}
		})

		It("should create a template input with an 'empty' option and existing templates", func() {
			existingTemplates := []template.Spec{
				{Name: "template1"},
				{Name: "template2"},
				{Name: "template3"},
			}

			input := opts.createTemplateInput(existingTemplates)

			// Check the interactive prompt
			prompt, err := input.InteractivePromptFn()

			Expect(err).NotTo(HaveOccurred())
			singleSelect, ok := prompt.(*userio.SingleSelect)
			Expect(ok).To(BeTrue())
			Expect(singleSelect.Items).To(Equal([]string{"<empty>", "template1", "template2", "template3"}))

			// Test empty selection
			result, err := input.ValidateAndMap("<empty>")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())

			// Test valid template selection
			result, err = input.ValidateAndMap("template2")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(&existingTemplates[1]))

			// Test invalid template selection
			result, err = input.ValidateAndMap("nonexistent")
			Expect(err).To(MatchError("unknown template"))
			Expect(result).To(BeNil())

		})

		It("should handle an empty list of existing templates", func() {
			existingTemplates := []template.Spec{}

			input := opts.createTemplateInput(existingTemplates)

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
			existingTemplates := []template.Spec{}

			input := opts.createTemplateInput(existingTemplates)

			value, err := input.GetValue(userio.NewIOStreamsWithInteractive(nil, nil, false))

			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(BeNil())
		})
	})
})
