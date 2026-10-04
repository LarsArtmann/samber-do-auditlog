package auditlog

import "time"

// RecordCommand records a command execution event on the recorder.
//
// Commands are not DI services: no service record is created and command
// events never appear in Report.Services — they surface in Report.Events (and
// the NDJSON event stream) only. ServiceRef.ServiceName carries the command
// name; distinguish command events from service events via Event.IsCommand().
//
// Phase PhaseBefore marks the start of a command (durationMs must be nil);
// PhaseAfter marks completion, carrying the wall-clock duration and the
// command's error (nil on success). scopeID/scopeName attribute the command
// to the reporting scope when known (empty values are fine).
func (r *Recorder) RecordCommand(
	scopeID ScopeID,
	scopeName string,
	commandName string,
	phase Phase,
	durationMs *float64,
	err error,
) {
	now := time.Now()
	errStr := errorToStringPtr(err)
	seq := r.nextSequence()

	ref := ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: ServiceName(commandName)}

	r.mu.Lock()

	evt := newEventFromRef(
		seq, now, EventTypeCommand, phase,
		ref, r.containerID, r.runID, ProviderType(""), durationMs, errStr,
	)
	r.appendEventLocked(evt)

	r.mu.Unlock()

	r.fireEvent(evt)
}

// RecordCommand records a command execution event on the plugin's recorder,
// attributed to the root scope. See [Recorder.RecordCommand] for phase and
// duration semantics.
func (p *Plugin) RecordCommand(commandName string, phase Phase, durationMs *float64, err error) {
	p.recorder.RecordCommand("", RootScopeName, commandName, phase, durationMs, err)
}
