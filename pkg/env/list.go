package env

import (
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/jedib0t/go-pretty/v6/table"
)

type TableEnv struct {
	table table.Writer
}

func NewTable(streams userio.IOStreams) TableEnv {
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Name", "ID", "Cloud Platform"})
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false
	t.Style().Options.SeparateFooter = false
	t.Style().Options.SeparateHeader = false
	t.Style().Options.SeparateRows = false
	t.SetOutputMirror(streams.GetOutput())

	return TableEnv{table: t}
}

func (t TableEnv) AppendRow(name, id, platform string) {
	t.table.AppendRows([]table.Row{{name, id, platform}})
}

func (t TableEnv) Render() string {
	return t.table.Render()
}

func (t TableEnv) AppendEnv(env environment.Environment) {
	var (
		platform string
		id       string
	)

	switch p := env.Platform.(type) {
	case *environment.GCPVendor:
		id = p.ProjectId
		platform = "GCP"
	case *environment.AWSVendor:
		id = p.AccountId
		platform = "AWS"
	}

	t.AppendRow(env.Environment, id, platform)
}
