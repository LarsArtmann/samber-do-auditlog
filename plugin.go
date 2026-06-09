package auditlog

import (
	"encoding/json"
	"io"
	"os"

	"github.com/samber/do/v2"
)

// Config controls the audit log plugin behaviour.
type Config struct {
	// Enabled turns audit logging on or off. When false the plugin is a no-op.
	Enabled bool
	// ContainerID is an optional human-readable identifier for the injector.
	ContainerID string
}

// Plugin wraps a samber/do v2 container with audit logging hooks.
type Plugin struct {
	recorder    *Recorder
	config      Config
	containerID string
}

// New creates an audit log plugin. If ContainerID is empty it defaults to "default".
func New(config Config) *Plugin {
	if config.ContainerID == "" {
		config.ContainerID = "default"
	}

	return &Plugin{
		recorder:    NewRecorder(),
		config:      config,
		containerID: config.ContainerID,
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

// WriteReportJSON writes the full Report as indented JSON to w.
func (p *Plugin) WriteReportJSON(w io.Writer) error {
	report := p.Report()
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	return enc.Encode(report)
}

// WriteEventsNDJSON writes every captured event as a line-delimited JSON stream to w.
func (p *Plugin) WriteEventsNDJSON(w io.Writer) error {
	events := p.recorder.Events()

	enc := json.NewEncoder(w)
	for _, e := range events {
		err := enc.Encode(e)
		if err != nil {
			return err
		}
	}

	return nil
}

// ExportToFile writes the full Report as indented JSON.
func (p *Plugin) ExportToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	return p.WriteReportJSON(f)
}

// ExportEventsToNDJSON writes every captured event as a line-delimited JSON stream.
func (p *Plugin) ExportEventsToNDJSON(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	return p.WriteEventsNDJSON(f)
}

// Events returns a defensive copy of all captured events.
func (p *Plugin) Events() []Event {
	return p.recorder.Events()
}
