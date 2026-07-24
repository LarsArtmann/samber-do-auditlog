package auditlog

import (
	"fmt"
	"io"
	"strings"

	"github.com/larsartmann/go-output"
	// Blank imports register table data renderers for RenderTable dispatch.
	_ "github.com/larsartmann/go-output/delimited"
	_ "github.com/larsartmann/go-output/markdown"
	_ "github.com/larsartmann/go-output/serialization"
	_ "github.com/larsartmann/go-output/table"
)

// buildServiceTableData converts a Report into go-output Table using the
// specified columns. If columns is empty, DefaultTableColumns is used.
func (r Report) buildServiceTableData(columns []TableColumn) *output.Table {
	if len(columns) == 0 {
		columns = append([]TableColumn(nil), DefaultTableColumns...)
	}

	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = columnDefs[col].header
	}

	data := output.NewTable(headers)

	for _, svc := range r.Services {
		row := make([]string, len(columns))
		for i, col := range columns {
			row[i] = columnDefs[col].extract(svc)
		}

		data.AddRow(row)
	}

	return data
}

// WriteTable writes the service summary as a table in the specified format.
// Supported formats (when respective sub-modules are imported): table,
// json, csv, tsv, markdown, xml, d2, yaml, html, tree, mermaid, dot,
// jsonl, asciidoc, toml, plantuml.
//
// Use [WithColumns] to customize which columns appear (default: Service,
// Scope, Type, Status, Invocations, Build(ms), Error).
func (r Report) WriteTable(writer io.Writer, format output.Format, opts output.RenderOptions, tableOpts ...TableOption) error {
	cfg := applyTableOpts(tableOpts)
	data := r.buildServiceTableData(cfg.columns)

	opts.Writer = writer

	err := output.RenderTable(data, format, opts)
	if err != nil {
		return fmt.Errorf("render table: %w", err)
	}

	return nil
}

// WriteTableString returns the service summary table as a string in the
// specified format. See WriteTable for supported formats.
func (r Report) WriteTableString(format output.Format, opts output.RenderOptions, tableOpts ...TableOption) (string, error) {
	var buf strings.Builder

	err := r.WriteTable(&buf, format, opts, tableOpts...)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// DefaultTableOpts returns the default RenderOptions for table export.
// Convenience for callers who don't need custom rendering options.
func DefaultTableOpts() output.RenderOptions {
	return output.RenderOptions{
		Title:     "",
		Writer:    nil,
		ColorMode: output.ColorModeAuto,
	}
}
