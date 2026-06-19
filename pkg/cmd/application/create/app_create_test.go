package create

import (
	"testing"

	"github.com/coreeng/core-platform/pkg/environment"
	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
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

var _ = Describe("p2pBaseNamespace", func() {
	It("uses the app name when owner and app are the same", func() {
		Expect(p2pBaseNamespace("orders", "orders")).To(Equal("orders"))
	})

	It("uses owner-app when owner and app differ", func() {
		Expect(p2pBaseNamespace("payments", "orders")).To(Equal("payments-orders"))
	})
})

var _ = Describe("cloudAccessKubernetesServiceAccounts", func() {
	It("returns fast-feedback and extended-test service accounts for dev environments", func() {
		result := cloudAccessKubernetesServiceAccounts("payments-orders", "orders", environment.DevEnvironmentTier)

		Expect(result).To(Equal([]string{
			"payments-orders-functional/orders",
			"payments-orders-nft/orders",
			"payments-orders-integration/orders",
			"payments-orders-extended/orders",
		}))
	})

	It("returns fast-feedback and extended-test service accounts for pre-dev environments", func() {
		result := cloudAccessKubernetesServiceAccounts("payments-orders", "orders", environment.PreDevEnvironmentTier)

		Expect(result).To(Equal([]string{
			"payments-orders-functional/orders",
			"payments-orders-nft/orders",
			"payments-orders-integration/orders",
			"payments-orders-extended/orders",
		}))
	})

	It("returns prod service account for prod environments", func() {
		result := cloudAccessKubernetesServiceAccounts("payments-orders", "orders", environment.ProdEnvironmentTier)

		Expect(result).To(Equal([]string{
			"payments-orders-prod/orders",
		}))
	})
})

var _ = Describe("cloudAccessForApp", func() {
	It("creates GCP cloud access entries for each selected environment", func() {
		opts := &AppCreateOpt{Name: "orders", CloudAccess: true}
		orgUnit := &coretnt.Tenant{Name: "payments"}
		envs := []environment.Environment{
			{Environment: "gcp-dev", Tier: environment.DevEnvironmentTier},
			{Environment: "gcp-prod", Tier: environment.ProdEnvironmentTier},
		}

		result := cloudAccessForApp(opts, orgUnit, envs)

		Expect(result).To(Equal([]coretnt.CloudAccess{
			{
				Name:        "ca",
				Provider:    "gcp",
				Environment: "gcp-dev",
				KubernetesServiceAccounts: []string{
					"payments-orders-functional/orders",
					"payments-orders-nft/orders",
					"payments-orders-integration/orders",
					"payments-orders-extended/orders",
				},
			},
			{
				Name:        "ca",
				Provider:    "gcp",
				Environment: "gcp-prod",
				KubernetesServiceAccounts: []string{
					"payments-orders-prod/orders",
				},
			},
		}))
	})

	It("creates no entries when cloud access is disabled", func() {
		opts := &AppCreateOpt{Name: "orders", CloudAccess: false}
		orgUnit := &coretnt.Tenant{Name: "payments"}
		envs := []environment.Environment{
			{Environment: "gcp-dev", Tier: environment.DevEnvironmentTier},
		}

		result := cloudAccessForApp(opts, orgUnit, envs)

		Expect(result).To(BeEmpty())
	})
})

var _ = Describe("cloudAccessEnvironments", func() {
	It("excludes existing environments that are not selected in P2P defaults", func() {
		cfg := config.NewConfig()
		cfg.P2P.FastFeedback.DefaultEnvs.Value = []string{"gcp-dev"}
		cfg.P2P.ExtendedTest.DefaultEnvs.Value = []string{"gcp-dev"}
		cfg.P2P.Prod.DefaultEnvs.Value = []string{"gcp-prod"}
		envs := []environment.Environment{
			{Environment: "gcp-pre-dev", Tier: environment.PreDevEnvironmentTier},
			{Environment: "gcp-dev", Tier: environment.DevEnvironmentTier},
			{Environment: "gcp-prod", Tier: environment.ProdEnvironmentTier},
		}

		result := cloudAccessEnvironments(cfg, envs)

		Expect(result).To(Equal([]environment.Environment{
			{Environment: "gcp-dev", Tier: environment.DevEnvironmentTier},
			{Environment: "gcp-prod", Tier: environment.ProdEnvironmentTier},
		}))
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

		du, err := createDeliveryUnitForOrgUnit(&AppCreateOpt{Name: "new-app"}, orgUnit, "application", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(du.AdminGroup).To(Equal(orgUnit.AdminGroup))
		Expect(du.ReadOnlyGroup).To(Equal(orgUnit.ReadOnlyGroup))
		Expect(du.ProdAdminGroup).To(Equal(orgUnit.ProdAdminGroup))
		Expect(du.ProdReadOnlyGroup).To(Equal(orgUnit.ProdReadOnlyGroup))
	})

	It("adds cloud access when requested", func() {
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

		envs := []environment.Environment{
			{Environment: "dev", Tier: environment.DevEnvironmentTier},
			{Environment: "prod", Tier: environment.ProdEnvironmentTier},
		}

		du, err := createDeliveryUnitForOrgUnit(
			&AppCreateOpt{Name: "new-app", CloudAccess: true},
			orgUnit,
			"application",
			envs,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(du.CloudAccess).To(Equal([]coretnt.CloudAccess{
			{
				Name:        "ca",
				Provider:    "gcp",
				Environment: "dev",
				KubernetesServiceAccounts: []string{
					"parent-new-app-functional/new-app",
					"parent-new-app-nft/new-app",
					"parent-new-app-integration/new-app",
					"parent-new-app-extended/new-app",
				},
			},
			{
				Name:        "ca",
				Provider:    "gcp",
				Environment: "prod",
				KubernetesServiceAccounts: []string{
					"parent-new-app-prod/new-app",
				},
			},
		}))
	})

	It("rejects cloud access for infrastructure delivery units", func() {
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

		du, err := createDeliveryUnitForOrgUnit(
			&AppCreateOpt{Name: "new-infra", CloudAccess: true},
			orgUnit,
			"infrastructure",
			nil,
		)

		Expect(err).To(MatchError("--cloud-access can only be used with application templates"))
		Expect(du).To(BeNil())
	})
})
