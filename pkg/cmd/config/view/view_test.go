package view

import (
	"bytes"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

var _ = Describe("config view", func() {
	It("no options", func() {
		stdin, stdout := bytes.Buffer{}, bytes.Buffer{}
		originalCfg := config.NewTestPersistedConfig()
		fillCfgWithMockValues(originalCfg)

		cfg := *originalCfg
		err := run(&ConfigViewOpts{
			Streams: userio.NewIOStreams(&stdin, &stdout),
			Raw:     false,
		}, &cfg)
		Expect(err).NotTo(HaveOccurred())

		var outputedCfg config.Config
		Expect(yaml.Unmarshal(stdout.Bytes(), &outputedCfg)).To(Succeed())

		assertNonSensitiveConfigValues(GinkgoT(), originalCfg, &outputedCfg)
		Expect(outputedCfg.GitHub.Token.Value).To(Equal("REDACTED"))

		Expect(cfg).To(Equal(*originalCfg), "config value is modified")
	})
	It("raw output", func() {
		stdin, stdout := bytes.Buffer{}, bytes.Buffer{}
		originalCfg := config.NewTestPersistedConfig()
		fillCfgWithMockValues(originalCfg)

		cfg := *originalCfg
		err := run(&ConfigViewOpts{
			Streams: userio.NewIOStreams(&stdin, &stdout),
			Raw:     true,
		}, &cfg)
		Expect(err).NotTo(HaveOccurred())

		var outputedCfg config.Config
		Expect(yaml.Unmarshal(stdout.Bytes(), &outputedCfg)).To(Succeed())

		assertNonSensitiveConfigValues(GinkgoT(), originalCfg, &outputedCfg)
		Expect(outputedCfg.GitHub.Token.Value).To(Equal(originalCfg.GitHub.Token.Value))

		Expect(cfg).To(Equal(*originalCfg), "config value is modified")
	})
	It("config is not persisted", func() {
		stdin, stdout := bytes.Buffer{}, bytes.Buffer{}
		originalCfg := config.NewConfig()
		fillCfgWithMockValues(originalCfg)

		cfg := *originalCfg
		err := run(&ConfigViewOpts{
			Streams: userio.NewIOStreams(&stdin, &stdout),
			Raw:     false,
		}, &cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout.String()).To(ContainSubstring("No config found"))
		Expect(cfg).To(Equal(*originalCfg), "config value is modified")
	})
})

func fillCfgWithMockValues(cfg *config.Config) {
	cfg.GitHub.Token.Value = "gh_token-qwerty"
	cfg.GitHub.Organization.Value = "organization"

	cfg.Repositories.CPlatform.Value = "https://github.com/org/cplatform"
	cfg.Repositories.Templates.Value = "https://github.com/org/templates"
	cfg.Repositories.AllowDirty.Value = true

	cfg.P2P.FastFeedback.DefaultEnvs.Value = []string{"dev"}
	cfg.P2P.ExtendedTest.DefaultEnvs.Value = []string{"dev"}
	cfg.P2P.Prod.DefaultEnvs.Value = []string{"prod"}
}

func assertNonSensitiveConfigValues(t GinkgoTInterface, expected *config.Config, actual *config.Config) {
	assert.EqualExportedValues(t, expected.GitHub.Organization, actual.GitHub.Organization)
	assert.EqualExportedValues(t, expected.Repositories, actual.Repositories)
	assert.EqualExportedValues(t, expected.P2P, actual.P2P)
}
