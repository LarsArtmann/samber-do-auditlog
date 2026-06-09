package auditlog

import (
	"context"
	"fmt"
	"io"
	"os"
)

// ExportToHTML writes a self-contained HTML visualization to a file.
func (p *Plugin) ExportToHTML(path string) error {
	file, err := os.Create(path) //nolint:gosec
	if err != nil {
		return fmt.Errorf("create HTML file %q: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	return p.WriteHTML(file)
}

// WriteHTML writes a self-contained HTML visualization to w.
func (p *Plugin) WriteHTML(w io.Writer) error {
	report := p.Report()

	err := reportHTML(report).Render(context.Background(), w)
	if err != nil {
		return fmt.Errorf("render HTML report: %w", err)
	}

	return nil
}
