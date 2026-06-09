package auditlog

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
	report := p.Report()

	err := reportHTML(report).Render(context.Background(), w)
	if err != nil {
		return fmt.Errorf("render HTML report: %w", err)
	}

	return nil
}
