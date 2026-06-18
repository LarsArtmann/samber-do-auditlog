package auditlog

//go:generate go tool templ generate

import (
	"context"
	"fmt"
	"io"
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
	err := reportHTML(r).Render(context.Background(), w)
	if err != nil {
		return fmt.Errorf("render HTML report: %w", err)
	}

	return nil
}
