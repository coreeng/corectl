package env

import (
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
)

type TableEnv struct {
	table table.Writer
}

func NewTable(headers ...string) TableEnv {
	t := table.NewWriter()
	row := make(table.Row, len(headers))
	for i, header := range headers {
		row[i] = header
	}
	t.AppendHeader(row)
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false
	t.Style().Options.SeparateFooter = false
	t.Style().Options.SeparateHeader = false
	t.Style().Options.SeparateRows = false
	t.SetOutputMirror(os.Stdout)

	return TableEnv{table: t}
}

func (t TableEnv) AppendRow(platform, id, cluster string) {
	t.table.AppendRows([]table.Row{{platform, id, cluster}})
}

func (t TableEnv) Render() string {
	return t.table.Render()
}
