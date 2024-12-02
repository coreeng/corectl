package env

import (
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/jedib0t/go-pretty/v6/table"
)

type TableEnv struct {
	table     table.Writer
	showProxy bool
}

func NewTable(streams userio.IOStreams, showProxy bool) TableEnv {
	t := table.NewWriter()
	if showProxy {
		t.AppendHeader(table.Row{"Name", "ID", "CloudPlatform", "Proxy", "Pid"})
	} else {
		t.AppendHeader(table.Row{"Name", "ID", "CloudPlatform"})
	}
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false
	t.Style().Options.SeparateFooter = false
	t.Style().Options.SeparateHeader = false
	t.Style().Options.SeparateRows = false
	t.SetOutputMirror(streams.GetOutput())

	return TableEnv{table: t, showProxy: showProxy}
}

func (t TableEnv) AppendRow(name, id, platform, proxy, pid string) {

	if t.showProxy {
		t.table.AppendRows([]table.Row{{name, id, platform, proxy, pid}})
	} else {
		t.table.AppendRows([]table.Row{{name, id, platform}})
	}

}

func (t TableEnv) Render() string {
	return t.table.Render()
}

func (t TableEnv) AppendEnv(env environment.Environment, proxy string, pid string) {
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

	t.AppendRow(env.Environment, id, platform, proxy, pid)
}
