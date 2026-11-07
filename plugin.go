package auditlog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	atomicwrite "github.com/larsartmann/go-atomic-write"
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
	ContainerID ContainerID
	// RunID is an optional identifier for a single execution ("run") of the
	// container. When empty, New() auto-generates a 128-bit hex ID. Stamped
	// on every captured event and the final Report for cross-system correlation
	// (logs, traces, audit files).
	RunID RunID
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

// ErrContainerIDPathSep is returned by [Config.Validate] when ContainerID
// contains a path separator ("/" or "\"), which is forbidden because
// ContainerID is used in file export paths and event metadata.
var ErrContainerIDPathSep = errors.New("config.ContainerID must not contain path separators")

// Validate returns an error if the config is invalid.
//
// Checks that ContainerID does not contain path separators ("/" or "\")
// since it is used in file export paths and event metadata.
func (c Config) Validate() error {
	if strings.ContainsAny(string(c.ContainerID), "/\\") {
		return fmt.Errorf("%w: %q", ErrContainerIDPathSep, c.ContainerID)
	}

	return nil
}

// defaultContainerID is used when Config.ContainerID is empty.
const defaultContainerID ContainerID = "default"

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

	if config.RunID == "" {
		config.RunID = newRunID()
	}

	err := config.Validate()
	if err != nil {
		return nil, err
	}

	if !config.Enabled {
		config.Enabled = envIsEnabled()
	}

	recorder := NewRecorder(config.ContainerID, config.RunID, config.OnEvent)
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
		return &do.InjectorOpts{} //nolint:exhaustruct_v5
	}

	return &do.InjectorOpts{ //nolint:exhaustruct_v5
		HookBeforeRegistration: []func(*do.Scope, string){p.recorder.OnBeforeRegistration},
		HookAfterRegistration:  []func(*do.Scope, string){p.recorder.OnAfterRegistration},
		HookBeforeInvocation:   []func(*do.Scope, string){p.recorder.OnBeforeInvocation},
		HookAfterInvocation:    []func(*do.Scope, string, error){p.recorder.OnAfterInvocation},
		HookBeforeShutdown:     []func(*do.Scope, string){p.recorder.OnBeforeShutdown},
		HookAfterShutdown:      []func(*do.Scope, string, error){p.recorder.OnAfterShutdown},
	}
}

// Enable turns audit logging on after creation, regardless of the
// Config.Enabled value or DO_AUDITLOG_ENABLED env var seen at New().
// Idempotent; there is no Disable counterpart because disabling after
// hooks are installed would silently drop events mid-run.
//
// Only effective while the injector options are still consumable: Opts()
// consults the enabled flag at call time, so Enable must be called before
// the first Opts() result is used to create a samber/do injector. Calling
// it later has no effect on already-installed (or already-empty) hooks.
// Intended for integrations that create the plugin before their feature
// flags are parsed, e.g. a CLI that force-enables DI audit logging when
// the user requests a live dashboard, mirroring an "always on when the
// dashboard is shown" contract.
func (p *Plugin) Enable() {
	p.config.Enabled = true
}

// SetOnEvent sets or replaces the event callback after creation.
//
// Config.OnEvent is captured at New() time; integrations that must create
// the plugin before they know their sink (e.g. a CLI that constructs the
// plugin for its DI container before flags are parsed, then wires a live
// SSE dashboard hub afterward) can attach the callback with SetOnEvent.
//
// Replaces any callback passed via Config.OnEvent; compose with
// NewMultiWriter beforehand if several sinks must all keep receiving
// events. Passing nil disables the callback.
//
// Safe for concurrent use with recording goroutines. Events recorded
// before the call are not replayed to fn; read them via Events or Report
// (the live dashboard sends a full snapshot on connect, so late wiring
// loses nothing).
func (p *Plugin) SetOnEvent(fn func(Event)) {
	p.recorder.setOnEvent(fn)
}

// Report returns a consolidated snapshot of everything observed so far.
func (p *Plugin) Report() Report {
	return p.recorder.BuildReport()
}

// ReportFiltered returns a filtered Report with the given options applied.
func (p *Plugin) ReportFiltered(opts ...ReportOption) Report {
	return p.Report().Filtered(opts...)
}

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
		name := ServiceName(svcName)
		scopeID, scopeName, found := p.recorder.ResolveServiceScope(injector, name)

		if !found {
			continue
		}

		p.recorder.RecordHealthCheck(scopeID, scopeName, name, svcErr)
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

// writeToFile creates a file at path and writes to it atomically via a
// streaming callback. Delegates to go-atomic-write for TOCTOU-safe writes
// with fsync durability and cross-platform atomic rename. A crash during
// write leaves the previous file (if any) intact rather than a partial file.
func writeToFile(path string, fn func(io.Writer) error) error {
	err := atomicwrite.WriteFunc(path, fn)
	if err != nil {
		return fmt.Errorf("atomic write %q: %w", path, err)
	}

	return nil
}
