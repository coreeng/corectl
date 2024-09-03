package config

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("config", func() {
	When("set value", func() {
		var config *Config
		BeforeEach(func() {
			config = NewConfig()
			fillCfgWithMockValues(config)
		})

		It("string value updated", func() {
			newCfg, err := config.SetValue("github.token", "new-token-value")
			Expect(err).NotTo(HaveOccurred())
			Expect(newCfg.GitHub.Token.Value).To(Equal("new-token-value"))
			Expect(config.GitHub.Token.Value).To(Equal("gh_token-qwerty"))

			newCfg.GitHub.Token.Value = "gh_token-qwerty"
			Expect(newCfg).To(Equal(config))
		})

		It("boolean value updated", func() {
			newCfg, err := config.SetValue("repositories.allow-dirty", "false")
			Expect(err).NotTo(HaveOccurred())
			Expect(newCfg.Repositories.AllowDirty.Value).To(BeFalse())
			Expect(config.Repositories.AllowDirty.Value).To(BeTrue())

			newCfg.Repositories.AllowDirty.Value = true
			Expect(newCfg).To(Equal(config))
		})

		It("boolean value - invalid", func() {
			newCfg, err := config.SetValue("repositories.allow-dirty", "random value")
			Expect(err).To(HaveOccurred())
			Expect(newCfg).To(BeNil())
		})

		It("invalid path", func() {
			newCfg, err := config.SetValue("non.existing.path", "random value")
			Expect(err).To(HaveOccurred())
			Expect(newCfg).To(BeNil())
		})

		It("partial path", func() {
			newCfg, err := config.SetValue("repositories", "random value")
			Expect(err).To(HaveOccurred())
			Expect(newCfg).To(BeNil())
		})
	})
})

func fillCfgWithMockValues(cfg *Config) {
	cfg.GitHub.Token.Value = "gh_token-qwerty"
	cfg.GitHub.Organization.Value = "organization"

	cfg.Repositories.CPlatform.Value = "https://github.com/org/cplatform"
	cfg.Repositories.Templates.Value = "https://github.com/org/templates"
	cfg.Repositories.AllowDirty.Value = true

	cfg.P2P.FastFeedback.DefaultEnvs.Value = []string{"dev"}
	cfg.P2P.ExtendedTest.DefaultEnvs.Value = []string{"dev"}
	cfg.P2P.Prod.DefaultEnvs.Value = []string{"prod"}
}
