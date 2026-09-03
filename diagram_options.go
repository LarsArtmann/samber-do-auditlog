package auditlog

import (
	"strings"
)

// Direction represents a canonical layout direction for diagrams.
// It bridges D2 vocabulary ("down"/"right") and DOT vocabulary ("TB"/"LR")
// through a single canonical type.
type Direction string

const (
	// DirectionDown lays out top-to-bottom (the default for most formats).
	DirectionDown Direction = "down"
	// DirectionUp lays out bottom-to-top.
	DirectionUp Direction = "up"
	// DirectionLeft lays out right-to-left.
	DirectionLeft Direction = "left"
	// DirectionRight lays out left-to-right.
	DirectionRight Direction = "right"
)

// DiagramOption configures diagram output (Mermaid, DOT, PlantUML, D2).
// Pass options to WriteMermaid, WriteDOT, WritePlantUML, WriteD2, and
// their Plugin wrapper methods.
type DiagramOption func(*diagramConfig)

type diagramConfig struct {
	direction Direction
}

// WithDirection sets the layout direction for diagram output.
// Applies to all diagram formats (Mermaid, DOT, D2, PlantUML).
//
// The default is top-down ([DirectionDown]). For wide DAGs,
// [DirectionRight] (left-to-right) often produces more readable diagrams.
//
// Example:
//
//	report.WriteMermaid(w, auditlog.WithDirection(auditlog.DirectionRight))
func WithDirection(d Direction) DiagramOption {
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
	return c.direction != "" && c.direction != DirectionDown
}

// toRankDir converts Direction to DOT's rankdir string.
func (d Direction) toRankDir() string {
	switch d {
	case DirectionDown:
		return "TB"
	case DirectionUp:
		return "BT"
	case DirectionLeft:
		return "RL"
	case DirectionRight:
		return "LR"
	default:
		return "TB"
	}
}

// mermaidDirection maps the canonical Direction to the Mermaid flowchart
// direction keyword (TD, LR, BT, RL).
func mermaidDirection(d Direction) string {
	switch d {
	case DirectionDown:
		return "TD"
	case DirectionUp:
		return "BT"
	case DirectionRight:
		return "LR"
	case DirectionLeft:
		return "RL"
	default:
		return "TD"
	}
}

// plantumlDirectionCommand returns the PlantUML layout command for the
// given direction. PlantUML supports two layouts: top-to-bottom (default)
// and left-to-right. Directions that map to left-to-right return the
// command; others return empty (use the default).
func plantumlDirectionCommand(d Direction) string {
	if d == DirectionRight || d == DirectionLeft {
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
