package auditlog

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	// MaxEvents caps the number of events stored in memory. When 0 (default),
	// events grow without bound. When > 0, the recorder stops appending new
	// events after reaching the cap and increments DroppedEventCount.
	// Use this to prevent OOM in long-running processes.
	MaxEvents int
	// InitialEventCapacity pre-allocates the events slice to avoid runtime
	// growslice reallocations. When 0, defaults to 1024. Set this to the
	// expected number of events for your workload to eliminate slice growth cost.
	InitialEventCapacity int
}

// Validate returns an error if the config is invalid.
//
// Checks that ContainerID does not contain path separators ("/" or "\")
// since it is used in file export paths and event metadata.
var errContainerIDPathSep = errors.New("config.ContainerID must not contain path separators")

func (c Config) Validate() error {
	if strings.ContainsAny(c.ContainerID, "/\\") {
		return fmt.Errorf("%w: %q", errContainerIDPathSep, c.ContainerID)
	}

	return nil
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
//
// Returns an error if Config.Validate() fails (e.g., ContainerID contains
// path separators).
func New(config Config) (*Plugin, error) {
	if config.ContainerID == "" {
		config.ContainerID = defaultContainerID
	}

	err := config.Validate()
	if err != nil {
		return nil, err
	}

	if !config.Enabled {
		config.Enabled = envIsEnabled()
	}

	recorder := NewRecorder(config.ContainerID, config.OnEvent)
	if config.MaxEvents > 0 {
		recorder.maxEvents = config.MaxEvents
	}

	if config.InitialEventCapacity > 0 && len(recorder.events) == 0 {
		recorder.events = make([]Event, 0, config.InitialEventCapacity)
	}

	return &Plugin{
		recorder: recorder,
		config:   config,
	}, nil
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

// ReportFiltered returns a filtered Report with the given options applied.
func (p *Plugin) ReportFiltered(opts ...ReportOption) Report {
	return p.Report().Filtered(opts...)
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

	return writeToFile(path, func(w io.Writer) error {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		err := enc.Encode(filtered)
		if err != nil {
			return fmt.Errorf("encode filtered report: %w", err)
		}

		return nil
	})
}

// Events returns a defensive copy of all captured events.
func (p *Plugin) Events() []Event {
	return p.recorder.Events()
}

// EventsCount returns the number of captured events without copying the slice.
func (p *Plugin) EventsCount() int {
	return p.recorder.EventsCount()
}

// DroppedEventCount returns the number of events dropped due to Config.MaxEvents.
func (p *Plugin) DroppedEventCount() int64 {
	return p.recorder.DroppedEventCount()
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

	results := injector.HealthCheckWithContext(ctx)

	for svcName, svcErr := range results {
		scopeID, scopeName, found := p.recorder.ResolveServiceScope(injector, svcName)
		if !found {
			continue
		}

		p.recorder.RecordHealthCheck(scopeID, scopeName, svcName, svcErr)
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

// fileWriteBufferSize is the bufio buffer size used for atomic file exports.
const fileWriteBufferSize = 65536

// writeToFile creates a file at path and calls fn with a buffered writer.
// The bufio.Writer batches small writes into 64KB blocks, reducing syscall count
// by 10-100x compared to writing directly to os.File.
//
// Writes are atomic: data is written to a temporary file in the same directory,
// then atomically renamed to the final path. A crash during write leaves the
// previous file (if any) intact rather than a partial file.
func writeToFile(path string, fn func(io.Writer) error) error {
	dir := filepath.Dir(path)

	tmpFile, err := os.CreateTemp(dir, ".tmp-auditlog-*")
	if err != nil {
		return fmt.Errorf("create temp file in %q: %w", dir, err)
	}

	tmpPath := tmpFile.Name()
	cleanup := true

	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	bw := bufio.NewWriterSize(tmpFile, fileWriteBufferSize)

	writeErr := fn(bw)

	flushErr := bw.Flush()

	closeErr := tmpFile.Close()

	if writeErr != nil {
		return writeErr
	}

	if flushErr != nil {
		return fmt.Errorf("flush temp file %q: %w", tmpPath, flushErr)
	}

	if closeErr != nil {
		return fmt.Errorf("close temp file %q: %w", tmpPath, closeErr)
	}

	renameErr := os.Rename(tmpPath, path)
	if renameErr != nil {
		return fmt.Errorf("rename %q → %q: %w", tmpPath, path, renameErr)
	}

	cleanup = false

	return nil
}
