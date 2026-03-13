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
		func(tenants []coretnt.Tenant, expectedOutput string) {
			var stdin, stdout, stderr bytes.Buffer
			streams := userio.NewIOStreams(&stdin, &stdout, &stderr)
			table := NewTable(streams)
			for _, t := range tenants {
				table.AppendRow(t)
			}
			result := table.Render()
			Expect(result).To(Equal(expectedOutput))
		},
		Entry("no tenants", []coretnt.Tenant{}, ` NAME  KIND  OWNER  CONTACT EMAIL `),
		Entry("normal list", []coretnt.Tenant{
			{
				Name:         "team1",
				Kind:         "OrgUnit",
				ContactEmail: "team1@company.com",
			},
			{
				Name:         "app1",
				Kind:         "DeliveryUnit",
				Owner:        "team1",
				ContactEmail: "app1@company.com",
			},
		}, " NAME   KIND          OWNER  CONTACT EMAIL     \n team1  OrgUnit              team1@company.com \n app1   DeliveryUnit  team1  app1@company.com  "),
	)
})
