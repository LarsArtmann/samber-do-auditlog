package auditlog_test

import (
	"fmt"
	"sync"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// eventCollector is a concurrency-safe OnEvent sink for tests. SetOnEvent
// may swap callbacks while recording goroutines fire events, so collectors
// shared across goroutines must synchronize their append.
type eventCollector struct {
	mu     sync.Mutex
	events []auditlog.Event
}

// OnEvent matches the Config.OnEvent / Plugin.SetOnEvent callback signature.
func (c *eventCollector) OnEvent(evt auditlog.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.events = append(c.events, evt)
}

// len returns the number of collected events.
func (c *eventCollector) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.events)
}

// snapshot returns a copy of the collected events.
func (c *eventCollector) snapshot() []auditlog.Event {
	c.mu.Lock()
	defer c.mu.Unlock()

	return append([]auditlog.Event(nil), c.events...)
}

// provideTriggerNamed registers one named value service on a fresh injector
// wired with the plugin's hooks. Each call records exactly two events
// (before + after registration).
func provideTriggerNamed(p *auditlog.Plugin, name string) {
	injector := do.NewWithOpts(p.Opts())
	do.ProvideNamedValue(injector, name, name)
}

func TestPlugin_SetOnEvent_ReceivesOnlySubsequentEvents(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})

	// Events recorded before SetOnEvent must NOT be replayed to the callback.
	provideTriggerNamed(p, "before-svc")

	collector := &eventCollector{}
	p.SetOnEvent(collector.OnEvent)

	provideTriggerNamed(p, "after-svc")

	if got := collector.len(); got != 2 {
		t.Fatalf("collector events: want 2 (before+after registration), got %d", got)
	}

	for _, evt := range collector.snapshot() {
		if string(evt.ServiceName) != "after-svc" {
			t.Errorf("service name: want after-svc, got %q", evt.ServiceName)
		}
	}
}

func TestPlugin_SetOnEvent_NilDisablesCallback(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})

	collector := &eventCollector{}
	p.SetOnEvent(collector.OnEvent)
	p.SetOnEvent(nil)

	provideTriggerNamed(p, "nil-svc")

	if got := collector.len(); got != 0 {
		t.Fatalf("collector events after SetOnEvent(nil): want 0, got %d", got)
	}

	if got := p.EventsCount(); got < 2 {
		t.Fatalf("recorder events: want >= 2 (recording must continue), got %d", got)
	}
}

func TestPlugin_SetOnEvent_ReplacesConfiguredCallback(t *testing.T) {
	t.Parallel()

	first := &eventCollector{}
	p := mustNew(auditlog.Config{Enabled: true, OnEvent: first.OnEvent})

	second := &eventCollector{}
	p.SetOnEvent(second.OnEvent)

	provideTriggerNamed(p, "swap-svc")

	if got := first.len(); got != 0 {
		t.Fatalf("first callback events after replace: want 0, got %d", got)
	}

	if got := second.len(); got != 2 {
		t.Fatalf("second callback events: want 2, got %d", got)
	}
}

func TestPlugin_Enable_InstallsHooksWhenCalledBeforeOpts(t *testing.T) {
	// Not parallel: mutates the DO_AUDITLOG_ENABLED environment (t.Setenv
	// is incompatible with t.Parallel).
	t.Setenv("DO_AUDITLOG_ENABLED", "")

	p := mustNew(auditlog.Config{Enabled: false})

	if got := len(p.Opts().HookBeforeRegistration); got != 0 {
		t.Fatalf("disabled plugin Opts() hooks: want 0, got %d", got)
	}

	p.Enable()

	opts := p.Opts()
	if got := len(opts.HookBeforeRegistration); got != 1 {
		t.Fatalf("enabled plugin Opts() HookBeforeRegistration: want 1, got %d", got)
	}

	// Opts() consumed after Enable: a fresh injector records events.
	injector := do.NewWithOpts(opts)
	do.ProvideNamedValue(injector, "enabled-svc", "enabled-svc")

	if got := p.EventsCount(); got != 2 {
		t.Fatalf("events after Enable + Provide: want 2, got %d", got)
	}

	// Idempotent: a second Enable must not duplicate hooks.
	p.Enable()

	if got := len(p.Opts().HookBeforeRegistration); got != 1 {
		t.Fatalf("HookBeforeRegistration after double Enable: want 1, got %d", got)
	}
}

func TestPlugin_SetOnEvent_ConcurrentSwapWhileRecording(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})

	a, b := &eventCollector{}, &eventCollector{}
	p.SetOnEvent(a.OnEvent)

	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		for i := range iterations {
			provideTriggerNamed(p, fmt.Sprintf("svc-%d", i))
		}
	}()

	go func() {
		defer wg.Done()

		for i := range iterations {
			if i%2 == 0 {
				p.SetOnEvent(b.OnEvent)
			} else {
				p.SetOnEvent(a.OnEvent)
			}
		}
	}()

	wg.Wait()

	if total := a.len() + b.len(); total == 0 {
		t.Fatal("no events delivered to any callback")
	}
}
