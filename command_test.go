package auditlog_test

import (
	"errors"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestRecorder_RecordCommand_BeforeAfterPair(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})

	p.RecordCommand("deploy", auditlog.PhaseBefore, nil, nil)

	duration := 12.5
	p.RecordCommand("deploy", auditlog.PhaseAfter, &duration, nil)

	events := p.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	before, after := events[0], events[1]

	if !before.IsCommand() || !after.IsCommand() {
		t.Error("expected both events to be command events")
	}

	if before.EventType != auditlog.EventTypeCommand {
		t.Errorf("before event type: want %q, got %q", auditlog.EventTypeCommand, before.EventType)
	}

	if !before.IsBefore() || !after.IsAfter() {
		t.Error("expected before/after phases on the pair")
	}

	if string(before.ServiceName) != "deploy" {
		t.Errorf("before service name: want %q, got %q", "deploy", before.ServiceName)
	}

	if after.DurationMs == nil || *after.DurationMs != 12.5 {
		t.Errorf("after duration: want 12.5, got %v", after.DurationMs)
	}

	if after.HasError() {
		t.Error("expected no error on successful command")
	}

	if before.Sequence >= after.Sequence {
		t.Errorf("expected strictly increasing sequence, got %d then %d", before.Sequence, after.Sequence)
	}
}

func TestRecorder_RecordCommand_Error(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})

	duration := 3.0
	p.RecordCommand("migrate", auditlog.PhaseAfter, &duration, errors.New("connection refused"))

	events := p.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt := events[0]

	if !evt.HasError() {
		t.Fatal("expected error on command event")
	}

	if *evt.Error != "connection refused" {
		t.Errorf("error: want %q, got %q", "connection refused", *evt.Error)
	}
}

func TestRecorder_RecordCommand_DoesNotCreateServiceRecords(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()

	provideHealthyDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")

	p.RecordCommand("deploy", auditlog.PhaseBefore, nil, nil)

	report := p.Report()

	if len(report.Services) != 1 {
		t.Fatalf("command event must not create service records: expected 1 service, got %d", len(report.Services))
	}

	commandEvents := 0
	for _, evt := range report.Events {
		if evt.IsCommand() {
			commandEvents++
		}
	}

	if commandEvents != 1 {
		t.Fatalf("expected 1 command event in report, got %d (of %d total)", commandEvents, len(report.Events))
	}
}

func TestPlugin_RecordCommand_FiresOnEventCallback(t *testing.T) {
	t.Parallel()

	var received []auditlog.Event

	p := mustNew(auditlog.Config{
		Enabled: true,
		OnEvent: func(evt auditlog.Event) {
			received = append(received, evt)
		},
	})

	p.RecordCommand("build", auditlog.PhaseBefore, nil, nil)

	if len(received) != 1 {
		t.Fatalf("expected onEvent callback to fire once, got %d", len(received))
	}

	if !received[0].IsCommand() {
		t.Error("expected callback event to be a command event")
	}
}

func TestEventTypeCommand_Meta(t *testing.T) {
	t.Parallel()

	if !auditlog.EventTypeCommand.IsKnown() {
		t.Error("EventTypeCommand should be a known event type")
	}

	if auditlog.EventTypeCommand.Label() == "" {
		t.Error("EventTypeCommand should have a display label")
	}
}
