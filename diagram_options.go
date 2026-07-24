package auditlog

import (
	"strings"

	"github.com/larsartmann/go-output"
)

// DiagramOption configures diagram output (Mermaid, DOT, PlantUML, D2).
// Pass options to WriteMermaid, WriteDOT, WritePlantUML, WriteD2, and
// their Plugin wrapper methods.
type DiagramOption func(*diagramConfig)

type diagramConfig struct {
	direction output.Direction
}

// WithDirection sets the layout direction for diagram output.
// Applies to all diagram formats (Mermaid, DOT, D2, PlantUML).
//
// The default is top-down ([output.DirectionDown]). For wide DAGs,
// [output.DirectionRight] (left-to-right) often produces more readable diagrams.
//
// Example:
//
//	report.WriteMermaid(w, auditlog.WithDirection(output.DirectionRight))
func WithDirection(d output.Direction) DiagramOption {
	return func(c *diagramConfig) { c.direction = d }
}

func applyDiagramOpts(opts []DiagramOption) diagramConfig {
	var cfg diagramConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

// hasDirection returns true when a non-default direction was configured.
func (c diagramConfig) hasDirection() bool {
	return c.direction != "" && c.direction != output.DirectionDown
}

// mermaidDirection maps the canonical output.Direction to the Mermaid
// flowchart direction keyword (TD, LR, BT, RL).
func mermaidDirection(d output.Direction) string {
	switch d {
	case output.DirectionDown:
		return "TD"
	case output.DirectionUp:
		return "BT"
	case output.DirectionRight:
		return "LR"
	case output.DirectionLeft:
		return "RL"
	default:
		return "TD"
	}
}

// plantumlDirectionCommand returns the PlantUML layout command for the
// given direction. PlantUML supports two layouts: top-to-bottom (default)
// and left-to-right. Directions that map to left-to-right return the
// command; others return empty (use the default).
func plantumlDirectionCommand(d output.Direction) string {
	if d == output.DirectionRight || d == output.DirectionLeft {
		return "left to right direction"
	}

	return ""
}

// applyPlantumlDirection inserts a direction command after the @startuml line.
func applyPlantumlDirection(rendered, command string) string {
	if command == "" {
		return rendered
	}

	return strings.Replace(rendered, "@startuml\n", "@startuml\n"+command+"\n", 1)
}
