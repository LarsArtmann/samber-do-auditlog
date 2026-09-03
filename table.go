package auditlog

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Format identifies a table output format. It replaces the go-output Format
// type on the Go 1.18 branch; string literals ("table", "json", "csv", ...)
// remain valid call sites.
type Format string

// Supported table formats.
const (
	// FormatTable is a plain-text ASCII table.
	FormatTable Format = "table"
	// FormatJSON is a JSON object with headers and rows.
	FormatJSON Format = "json"
	// FormatCSV is comma-separated values.
	FormatCSV Format = "csv"
	// FormatTSV is tab-separated values.
	FormatTSV Format = "tsv"
	// FormatMarkdown is a GitHub-flavored Markdown table.
	FormatMarkdown Format = "markdown"
)

// errUnsupportedTableFormat is returned by WriteTable for formats this build
// does not implement (the full 16-format matrix requires go-output, which
// needs Go 1.26).
var errUnsupportedTableFormat = errors.New("unsupported table format")

// ColorMode mirrors go-output's color selection knob. Kept for API
// compatibility; this build never emits ANSI color (deterministic output).
type ColorMode string

// ColorModeAuto lets the renderer decide (never colors in this build).
const ColorModeAuto ColorMode = "auto"

// RenderOptions configures table rendering.
type RenderOptions struct {
	// Title is an optional table title.
	Title string
	// Writer is the destination (set by WriteTable).
	Writer io.Writer
	// ColorMode selects ANSI coloring (no-op in this build).
	ColorMode ColorMode
}

// DefaultTableOpts returns the default RenderOptions for table export.
// Convenience for callers who don't need custom rendering options.
func DefaultTableOpts() RenderOptions {
	return RenderOptions{
		Title:     "",
		Writer:    nil,
		ColorMode: ColorModeAuto,
	}
}

// buildServiceTableRows converts a Report into (headers, rows) using the
// specified columns. If columns is empty, DefaultTableColumns is used.
func (r Report) buildServiceTableRows(columns []TableColumn) ([]string, [][]string) {
	if len(columns) == 0 {
		columns = append([]TableColumn(nil), DefaultTableColumns...)
	}

	headers := make([]string, 0, len(columns))
	for _, col := range columns {
		headers = append(headers, columnDefs[col].header)
	}

	rows := make([][]string, 0, len(r.Services))

	for _, svc := range r.Services {
		row := make([]string, 0, len(columns))
		for _, col := range columns {
			row = append(row, columnDefs[col].extract(svc))
		}

		rows = append(rows, row)
	}

	return headers, rows
}

// WriteTable writes the service summary as a table in the specified format.
// Supported formats: table, json, csv, tsv, markdown.
//
// Use [WithColumns] to customize which columns appear (default: Service,
// Scope, Type, Status, Invocations, Build(ms), Error).
func (r Report) WriteTable(
	writer io.Writer,
	format Format,
	opts RenderOptions,
	tableOpts ...TableOption,
) error {
	cfg := applyTableOpts(tableOpts)
	headers, rows := r.buildServiceTableRows(cfg.columns)

	var err error

	switch format {
	case FormatTable:
		err = writeASCIITable(writer, headers, rows)
	case FormatJSON:
		err = writeJSONTable(writer, headers, rows)
	case FormatCSV:
		err = writeDelimitedTable(writer, headers, rows, ',')
	case FormatTSV:
		err = writeDelimitedTable(writer, headers, rows, '\t')
	case FormatMarkdown:
		err = writeMarkdownTable(writer, headers, rows)
	default:
		err = fmt.Errorf("%w: %s", errUnsupportedTableFormat, format)
	}

	if err != nil {
		return fmt.Errorf("render table: %w", err)
	}

	return nil
}

// WriteTableString returns the service summary table as a string in the
// specified format. See WriteTable for supported formats.
func (r Report) WriteTableString(
	format Format,
	opts RenderOptions,
	tableOpts ...TableOption,
) (string, error) {
	var buf strings.Builder

	if err := r.WriteTable(&buf, format, opts, tableOpts...); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// writeASCIITable renders a plain-text table with padded columns.
func writeASCIITable(writer io.Writer, headers []string, rows [][]string) error {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	divider := buildASCIIDivider(widths)

	var b strings.Builder

	b.WriteString(divider)
	writeASCIIRow(&b, headers, widths)
	b.WriteString(divider)

	for _, row := range rows {
		writeASCIIRow(&b, row, widths)
	}

	b.WriteString(divider)

	if _, err := io.WriteString(writer, b.String()); err != nil {
		return fmt.Errorf("write table output: %w", err)
	}

	return nil
}

// buildASCIIDivider builds a +---+ separator line for the given widths.
func buildASCIIDivider(widths []int) string {
	var b strings.Builder

	b.WriteString("+")

	for _, w := range widths {
		b.WriteString(strings.Repeat("-", w+2))
		b.WriteString("+")
	}

	b.WriteString("\n")

	return b.String()
}

// writeASCIIRow writes one padded | cell | row.
func writeASCIIRow(b *strings.Builder, cells []string, widths []int) {
	b.WriteString("|")

	for i, w := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}

		fmt.Fprintf(b, " %-*s |", w, cell)
	}

	b.WriteString("\n")
}

// writeJSONTable writes the table as {"headers": [...], "rows": [[...]]}.
func writeJSONTable(writer io.Writer, headers []string, rows [][]string) error {
	payload := struct {
		Headers []string   `json:"headers"`
		Rows    [][]string `json:"rows"`
	}{
		Headers: headers,
		Rows:    rows,
	}

	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")

	if err := enc.Encode(payload); err != nil {
		return fmt.Errorf("encode table json: %w", err)
	}

	return nil
}

// writeDelimitedTable writes CSV or TSV via encoding/csv.
func writeDelimitedTable(writer io.Writer, headers []string, rows [][]string, comma rune) error {
	w := csv.NewWriter(writer)
	w.Comma = comma

	if err := w.Write(headers); err != nil {
		return fmt.Errorf("write table header: %w", err)
	}

	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return fmt.Errorf("write table row: %w", err)
		}
	}

	w.Flush()

	return w.Error()
}

// writeMarkdownTable writes a GitHub-flavored Markdown table.
func writeMarkdownTable(writer io.Writer, headers []string, rows [][]string) error {
	var b strings.Builder

	b.WriteString("| ")
	b.WriteString(strings.Join(headers, " | "))
	b.WriteString(" |\n| ")

	for range headers {
		b.WriteString("--- | ")
	}

	b.WriteString("\n")

	for _, row := range rows {
		b.WriteString("| ")
		b.WriteString(strings.Join(row, " | "))
		b.WriteString(" |\n")
	}

	if _, err := io.WriteString(writer, b.String()); err != nil {
		return fmt.Errorf("write table output: %w", err)
	}

	return nil
}
