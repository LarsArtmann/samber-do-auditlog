package auditlog

import (
	"io"

	"github.com/larsartmann/go-output"
)

// WriteReportJSON writes the full Report as indented JSON to writer.
func (p *Plugin) WriteReportJSON(writer io.Writer) error {
	return p.Report().WriteJSON(writer)
}

// WriteEventsNDJSON writes every captured event as a line-delimited JSON stream to writer.
func (p *Plugin) WriteEventsNDJSON(writer io.Writer) error {
	return writeEventsNDJSON(writer, p.recorder.Events())
}

// ExportToFile writes the full Report as indented JSON to path.
func (p *Plugin) ExportToFile(path string) error {
	return writeToFile(path, p.WriteReportJSON)
}

// ExportEventsToNDJSON writes every captured event as a line-delimited JSON stream to path.
func (p *Plugin) ExportEventsToNDJSON(path string) error {
	return writeToFile(path, p.WriteEventsNDJSON)
}

// ExportFilteredToFile writes a filtered Report as indented JSON to path.
func (p *Plugin) ExportFilteredToFile(path string, opts ...ReportOption) error {
	filtered := p.ReportFiltered(opts...)

	return writeToFile(path, filtered.WriteJSON)
}

// WriteReportCSV writes the Report services as comma-separated values to writer.
func (p *Plugin) WriteReportCSV(writer io.Writer) error {
	return p.Report().WriteCSV(writer)
}

// WriteReportTSV writes the Report services as tab-separated values to writer.
func (p *Plugin) WriteReportTSV(writer io.Writer) error {
	return p.Report().WriteTSV(writer)
}

// ExportToCSV writes the Report services as comma-separated values to path.
func (p *Plugin) ExportToCSV(path string) error {
	return writeToFile(path, p.WriteReportCSV)
}

// ExportToTSV writes the Report services as tab-separated values to path.
func (p *Plugin) ExportToTSV(path string) error {
	return writeToFile(path, p.WriteReportTSV)
}

// WriteMermaid writes the dependency graph as a Mermaid flowchart to writer.
func (p *Plugin) WriteMermaid(writer io.Writer, opts ...DiagramOption) error {
	return p.Report().WriteMermaid(writer, opts...)
}

// WritePlantUML writes the dependency graph as a PlantUML component diagram to writer.
func (p *Plugin) WritePlantUML(writer io.Writer, opts ...DiagramOption) error {
	return p.Report().WritePlantUML(writer, opts...)
}

// WriteDOT writes the dependency graph as a Graphviz DOT digraph to writer.
func (p *Plugin) WriteDOT(writer io.Writer, opts ...DiagramOption) error {
	return p.Report().WriteDOT(writer, opts...)
}

// WriteD2 writes the dependency graph as a D2 diagram to writer.
func (p *Plugin) WriteD2(writer io.Writer, opts ...DiagramOption) error {
	return p.Report().WriteD2(writer, opts...)
}

// ExportToMermaid writes the dependency graph as a Mermaid flowchart to path.
func (p *Plugin) ExportToMermaid(path string, opts ...DiagramOption) error {
	return writeToFile(path, func(w io.Writer) error {
		return p.WriteMermaid(w, opts...)
	})
}

// ExportToPlantUML writes the dependency graph as a PlantUML component diagram to path.
func (p *Plugin) ExportToPlantUML(path string, opts ...DiagramOption) error {
	return writeToFile(path, func(w io.Writer) error {
		return p.WritePlantUML(w, opts...)
	})
}

// ExportToDOT writes the dependency graph as a Graphviz DOT digraph to path.
func (p *Plugin) ExportToDOT(path string, opts ...DiagramOption) error {
	return writeToFile(path, func(w io.Writer) error {
		return p.WriteDOT(w, opts...)
	})
}

// ExportToD2 writes the dependency graph as a D2 diagram to path.
func (p *Plugin) ExportToD2(path string, opts ...DiagramOption) error {
	return writeToFile(path, func(w io.Writer) error {
		return p.WriteD2(w, opts...)
	})
}

// WriteTree writes the service dependency DAG as an ASCII tree to writer.
func (p *Plugin) WriteTree(writer io.Writer) error {
	return p.Report().WriteTree(writer)
}

// WriteHTMLTree writes the service dependency DAG as an HTML nested list tree to writer.
func (p *Plugin) WriteHTMLTree(writer io.Writer) error {
	return p.Report().WriteHTMLTree(writer)
}

// WriteTable writes the service summary as a table in the specified format to writer.
func (p *Plugin) WriteTable(
	writer io.Writer,
	format output.Format,
	opts output.RenderOptions,
	tableOpts ...TableOption,
) error {
	return p.Report().WriteTable(writer, format, opts, tableOpts...)
}

// ExportToTree writes the service dependency DAG as an ASCII tree to path.
func (p *Plugin) ExportToTree(path string) error {
	return writeToFile(path, p.WriteTree)
}

// ExportToHTMLTree writes the service dependency DAG as an HTML tree to path.
func (p *Plugin) ExportToHTMLTree(path string) error {
	return writeToFile(path, p.WriteHTMLTree)
}

// ExportToTable writes the service summary table to path in the specified format.
func (p *Plugin) ExportToTable(
	path string,
	format output.Format,
	opts output.RenderOptions,
	tableOpts ...TableOption,
) error {
	return writeToFile(path, func(w io.Writer) error {
		return p.WriteTable(w, format, opts, tableOpts...)
	})
}

// Events returns a defensive copy of all captured events.
