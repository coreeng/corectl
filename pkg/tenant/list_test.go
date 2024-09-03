package tenant

import (
	"bytes"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	coretnt "github.com/coreeng/developer-platform/pkg/tenant"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("tenant table", func() {
	DescribeTable("render",
		func(tenants []coretnt.Tenant, expectedOutput string) {
			stdin, stdout := bytes.Buffer{}, bytes.Buffer{}
			streams := userio.NewIOStreams(&stdin, &stdout)
			table := NewTable(streams)
			for _, t := range tenants {
				table.AppendRow(t)
			}
			result := table.Render()
			Expect(result).To(Equal(expectedOutput))
		},
		Entry("no tenants", []coretnt.Tenant{}, ` NAME  PARENT  CONTACT EMAIL `),
		Entry("normal list", []coretnt.Tenant{
			{
				Name:         "tenant1",
				Parent:       "parent1",
				ContactEmail: "tenant1@company.com",
			},
			{
				Name:         "tenant2",
				Parent:       "parent2",
				ContactEmail: "tenant2@company.com",
			},
		}, ` NAME     PARENT   CONTACT EMAIL       
 tenant1  parent1  tenant1@company.com 
 tenant2  parent2  tenant2@company.com `),
	)
})
