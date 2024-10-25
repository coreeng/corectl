package create

import (
	"testing"

	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/template"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/phuslu/log"
)

func TestAppCreateSuite(t *testing.T) {
	log.DefaultLogger.SetLevel(log.PanicLevel)
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
})
