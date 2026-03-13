package tenant

import (
	"bytes"

	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("tenant table", func() {
	DescribeTable("render",
		func(tenants []coretnt.Tenant, expectedSubstrings []string) {
			var stdin, stdout, stderr bytes.Buffer
			streams := userio.NewIOStreams(&stdin, &stdout, &stderr)
			table := NewTable(streams)
			for _, t := range tenants {
				table.AppendRow(t)
			}
			result := table.Render()
			for _, s := range expectedSubstrings {
				Expect(result).To(ContainSubstring(s))
			}
		},
		Entry("no tenants", []coretnt.Tenant{}, []string{"NAME", "KIND", "OWNER", "TYPE", "PREFIX", "REPO", "CONTACT EMAIL"}),
		Entry("normal list", []coretnt.Tenant{
			{
				Name:         "tenant1",
				Kind:         "DeliveryUnit",
				Owner:        "parent1",
				Type:         "application",
				Prefix:       "",
				Repo:         "https://github.com/org/repo1",
				ContactEmail: "tenant1@company.com",
			},
			{
				Name:         "tenant2",
				Kind:         "DeliveryUnit",
				Owner:        "parent2",
				Type:         "infrastructure",
				Prefix:       "area/subarea",
				Repo:         "",
				ContactEmail: "tenant2@company.com",
			},
		}, []string{"tenant1", "DeliveryUnit", "parent1", "application", "https://github.com/org/repo1", "tenant1@company.com", "tenant2", "parent2", "infrastructure", "area/subarea", "tenant2@company.com"}),
	)
})
