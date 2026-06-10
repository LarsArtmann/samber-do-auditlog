package auditlog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

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
	// OnEvent is called after each event is captured. Must not block.
	// Called sequentially in hook order. Nil disables the callback.
	OnEvent func(Event)
}

// defaultContainerID is used when Config.ContainerID is empty.
const defaultContainerID = "default"

// Plugin wraps a samber/do v2 container with audit logging hooks.
type Plugin struct {
	recorder *Recorder
	config   Config
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
		recorder: NewRecorder(config.ContainerID, config.OnEvent),
		config:   config,
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
		return &do.InjectorOpts{} //nolint:exhaustruct
	}

	return &do.InjectorOpts{ //nolint:exhaustruct
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
	return p.recorder.BuildReport()
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
func (p *Plugin) ExportToFile(path string) error {
	return writeToFile(path, p.WriteReportJSON)
}

// ExportEventsToNDJSON writes every captured event as a line-delimited JSON stream to path.
func (p *Plugin) ExportEventsToNDJSON(path string) error {
	return writeToFile(path, p.WriteEventsNDJSON)
}

// Events returns a defensive copy of all captured events.
func (p *Plugin) Events() []Event {
	return p.recorder.Events()
}

// RecordHealthCheckWithContext performs health checks on all services in the injector
// and records the results as audit events. It wraps injector.HealthCheckWithContext(ctx)
// with audit logging for each service result.
//
// When the plugin is disabled, it delegates directly to the injector without recording.
//
// Returns the same map[string]error as the underlying call (nil error = healthy).
func (p *Plugin) RecordHealthCheckWithContext(ctx context.Context, injector do.Injector) map[string]error {
	if !p.config.Enabled {
		return injector.HealthCheckWithContext(ctx)
	}

	start := time.Now()
	results := injector.HealthCheckWithContext(ctx)

	for svcName, svcErr := range results {
		elapsed := time.Since(start)
		durationMs := float64(elapsed.Microseconds()) / microsPerMs

		scopeID, scopeName, found := p.recorder.ResolveServiceScope(injector, svcName)
		if !found {
			continue
		}

		p.recorder.RecordHealthCheck(scopeID, scopeName, svcName, svcErr, durationMs)
	}

	return results
}

// RecordHealthCheck performs health checks on all services in the injector
// and records the results as audit events. It wraps injector.HealthCheck()
// with audit logging for each service result.
//
// When the plugin is disabled, it delegates directly to the injector without recording.
//
// Returns the same map[string]error as the underlying call (nil error = healthy).
func (p *Plugin) RecordHealthCheck(injector do.Injector) map[string]error {
	return p.RecordHealthCheckWithContext(context.Background(), injector)
}

// writeToFile creates a file at path and calls fn with the writer.
func writeToFile(path string, fn func(io.Writer) error) error {
	file, err := os.Create(path) //nolint:gosec
	if err != nil {
		return fmt.Errorf("create file %q: %w", path, err)
	}

	defer func() { _ = file.Close() }()

	return fn(file)
}
