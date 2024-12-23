package tenant

import (
	"github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/jedib0t/go-pretty/v6/table"
)

type Table struct {
	table table.Writer
}

func NewTable(streams userio.IOStreams) Table {
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Name", "Parent", "Contact Email"})
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false
	t.Style().Options.SeparateFooter = false
	t.Style().Options.SeparateHeader = false
	t.Style().Options.SeparateRows = false
	t.SetOutputMirror(streams.GetOutput())

	return Table{table: t}
}

func (t Table) AppendRow(tnnt tenant.Tenant) {
	t.table.AppendRows([]table.Row{{tnnt.Name, tnnt.Parent, tnnt.ContactEmail}})
}

func (t Table) Render() string {
	return t.table.Render()
}
