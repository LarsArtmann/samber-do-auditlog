package auditlog

//go:generate go tool templ generate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/larsartmann/go-output/daghtml"
)

// ExportToHTML writes a self-contained HTML visualization to a file.
func (p *Plugin) ExportToHTML(path string) error {
	return writeToFile(path, p.WriteHTML)
}

// WriteHTML writes a self-contained HTML visualization to w.
func (p *Plugin) WriteHTML(w io.Writer) error {
	return p.Report().WriteHTML(w)
}

// WriteHTML renders a self-contained HTML visualization of the report to w.
// This enables offline report rendering from a loaded Report (e.g. via
// LoadReport) without a live Plugin/container.
func (r Report) WriteHTML(w io.Writer) error {
	var buf bytes.Buffer

	err := reportHTML(r).Render(context.Background(), &buf)
	if err != nil {
		return fmt.Errorf("render HTML report: %w", err)
	}

	// Inject the daghtml SDK's graph JS at the marker point inside the
	// dashboard script block. The templ file contains a placeholder comment
	// because templ v0.3.1020 doesn't evaluate expressions inside <script> tags.
	html := strings.Replace(buf.String(), "// DAGHTML_JS_INJECTION_POINT", daghtml.Script(), 1)

	_, err = w.Write([]byte(html))
	if err != nil {
		return fmt.Errorf("write HTML report: %w", err)
	}

	return nil
}
