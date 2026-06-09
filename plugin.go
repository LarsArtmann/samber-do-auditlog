// Package auditlog provides a samber/do v2 plugin that records every
// registration, invocation, and shutdown as timestamped events with
// dependency graph inference, build duration tracking, and exporters
// for JSON, NDJSON, and a self-contained HTML visualization.
package auditlog

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/samber/do/v2"
)

// EnvKeyEnabled is the environment variable that controls audit logging.
// Set to "true", "1", or "yes" to enable. Any other value (or unset) disables it.
const EnvKeyEnabled = "DO_AUDITLOG_ENABLED"

// Config controls the audit log plugin behaviour.
type Config struct {
	// Enabled turns audit logging on or off. When false the plugin is a no-op.
	// If left as zero-value (false), New() checks the DO_AUDITLOG_ENABLED env var.
	Enabled bool
	// ContainerID is an optional human-readable identifier for the injector.
	ContainerID string
}

// defaultContainerID is used when Config.ContainerID is empty.
const defaultContainerID = "default"

// Plugin wraps a samber/do v2 container with audit logging hooks.
type Plugin struct {
	recorder    *Recorder
	config      Config
	containerID string
}

// New creates an audit log plugin.
//
// When Config.Enabled is false (the zero value), New checks the DO_AUDITLOG_ENABLED
// environment variable. Set it to "true", "1", or "yes" to enable audit logging
// without changing code. Explicitly setting Enabled to true overrides the env var.
//
// If ContainerID is empty it defaults to "default".
func New(config Config) *Plugin {
	if config.ContainerID == "" {
		config.ContainerID = defaultContainerID
	}

	if !config.Enabled {
		config.Enabled = envIsEnabled()
	}

	return &Plugin{
		recorder:    NewRecorder(config.ContainerID),
		config:      config,
		containerID: config.ContainerID,
	}
}

// envIsEnabled checks the DO_AUDITLOG_ENABLED environment variable.
func envIsEnabled() bool {
	switch os.Getenv(EnvKeyEnabled) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

// Opts returns a *do.InjectorOpts ready to pass to do.NewWithOpts.
// When Enabled is false the returned opts are harmless no-ops.
func (p *Plugin) Opts() *do.InjectorOpts {
	if !p.config.Enabled {
		return &do.InjectorOpts{}
	}

	return &do.InjectorOpts{
		HookBeforeRegistration: []func(*do.Scope, string){p.recorder.OnBeforeRegistration},
		HookAfterRegistration:  []func(*do.Scope, string){p.recorder.OnAfterRegistration},
		HookBeforeInvocation:   []func(*do.Scope, string){p.recorder.OnBeforeInvocation},
		HookAfterInvocation:    []func(*do.Scope, string, error){p.recorder.OnAfterInvocation},
		HookBeforeShutdown:     []func(*do.Scope, string){p.recorder.OnBeforeShutdown},
		HookAfterShutdown:      []func(*do.Scope, string, error){p.recorder.OnAfterShutdown},
	}
}

// Report returns a consolidated snapshot of everything observed so far.
func (p *Plugin) Report() Report {
	return p.recorder.BuildReport(p.containerID)
}

// WriteReportJSON writes the full Report as indented JSON to writer.
func (p *Plugin) WriteReportJSON(writer io.Writer) error {
	report := p.Report()
	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")

	err := enc.Encode(report)
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}

	return nil
}

// WriteEventsNDJSON writes every captured event as a line-delimited JSON stream to writer.
func (p *Plugin) WriteEventsNDJSON(writer io.Writer) error {
	events := p.recorder.Events()

	enc := json.NewEncoder(writer)
	for _, event := range events {
		err := enc.Encode(event)
		if err != nil {
			return fmt.Errorf("encode event %d: %w", event.Sequence, err)
		}
	}

	return nil
}

// ExportToFile writes the full Report as indented JSON to path.
//

func (p *Plugin) ExportToFile(path string) error {
	file, err := os.Create(path) //nolint:gosec,noinlineerr
	if err != nil {
		return fmt.Errorf("create report file %q: %w", path, err)
	}

	defer func() { _ = file.Close() }()

	return p.WriteReportJSON(file)
}

// ExportEventsToNDJSON writes every captured event as a line-delimited JSON stream to path.
//

func (p *Plugin) ExportEventsToNDJSON(path string) error {
	file, err := os.Create(path) //nolint:gosec,noinlineerr
	if err != nil {
		return fmt.Errorf("create events file %q: %w", path, err)
	}

	defer func() { _ = file.Close() }()

	return p.WriteEventsNDJSON(file)
}

// Events returns a defensive copy of all captured events.
func (p *Plugin) Events() []Event {
	return p.recorder.Events()
}
