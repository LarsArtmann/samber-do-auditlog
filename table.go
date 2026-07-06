package auditlog

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/larsartmann/go-output"
	// Blank imports register table data renderers for RenderTable dispatch.
	_ "github.com/larsartmann/go-output/delimited"
	_ "github.com/larsartmann/go-output/markdown"
	_ "github.com/larsartmann/go-output/serialization"
	_ "github.com/larsartmann/go-output/table"
)

// buildServiceTableData converts a Report into go-output Table.
// Columns: Service, Scope, Type, Status, Invocations, Build(ms), Error.
func (r Report) buildServiceTableData() *output.Table {
	data := output.NewTable([]string{"Service", "Scope", "Type", "Status", "Invocations", "Build(ms)", "Error"})

	for _, svc := range r.Services {
		errStr := ""
		if svc.InvocationError != nil {
			errStr = *svc.InvocationError
		} else if svc.ShutdownError != nil {
			errStr = *svc.ShutdownError
		}

		buildStr := ""
		if svc.FirstBuildDurationMs != nil {
			buildStr = fmt.Sprintf("%.2f", *svc.FirstBuildDurationMs)
		}

		data.AddRow([]string{
			svc.ServiceName,
			svc.ScopeName,
			string(svc.ServiceType),
			string(svc.Status),
			strconv.Itoa(svc.InvocationCount),
			buildStr,
			errStr,
		})
	}

	return data
}

// WriteTable writes the service summary as a table in the specified format.
// Supported formats (when respective sub-modules are imported): table,
// json, csv, tsv, markdown, xml, d2, yaml, html, tree, mermaid, dot,
// jsonl, asciidoc, toml, plantuml.
func (r Report) WriteTable(writer io.Writer, format output.Format, opts output.RenderOptions) error {
	data := r.buildServiceTableData()

	opts.Writer = writer

	err := output.RenderTable(data, format, opts)
	if err != nil {
		return fmt.Errorf("render table: %w", err)
	}

	return nil
}

// WriteTableString returns the service summary table as a string in the
// specified format. See WriteTable for supported formats.
func (r Report) WriteTableString(format output.Format, opts output.RenderOptions) (string, error) {
	var buf strings.Builder

	err := r.WriteTable(&buf, format, opts)
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
