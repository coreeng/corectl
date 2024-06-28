package tenant

import (
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/developer-platform/pkg/tenant"
	"github.com/jedib0t/go-pretty/v6/table"
)

var (
	tableHeaders = []string{
		"Name",
		"Parent",
		"Contact Email",
		"Description",
	}
)

type TableTenant struct {
	table table.Writer
}

func NewTable(streams userio.IOStreams) TableTenant {
	t := table.NewWriter()
	rows := make(table.Row, len(tableHeaders))
	for i, header := range tableHeaders {
		rows[i] = header
	}
	t.AppendHeader(rows)
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false
	t.Style().Options.SeparateFooter = false
	t.Style().Options.SeparateHeader = false
	t.Style().Options.SeparateRows = false
	t.SetOutputMirror(streams.GetOutput())
	return TableTenant{table: t}
}

func (t TableTenant) Append(tnt tenant.Tenant) {
	t.table.AppendRows([]table.Row{{
		tnt.Name,
		tnt.Parent,
		tnt.ContactEmail,
		tnt.Description,
	}})
}

func (t TableTenant) Render() string {
	return t.table.Render()
}
