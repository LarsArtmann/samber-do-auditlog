package auditlog_test

import (
	"sync"
	"sync/atomic"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestMultiWriter_FansOutToAllCallbacks(t *testing.T) {
	t.Parallel()

	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		got []auditlog.Event
	)

	callback := func(e auditlog.Event) {
		mu.Lock()
		got = append(got, e)
		mu.Unlock()
		wg.Done()
	}

	mw := auditlog.NewMultiWriter(callback, callback, callback)

	evt := auditlog.Event{
		Sequence:  1,
		EventType: auditlog.EventTypeRegistration,
		Phase:     auditlog.PhaseBefore,
		ServiceRef: auditlog.ServiceRef{
			ServiceName: "s1",
		},
	}

	wg.Add(3)
	mw.OnEvent(evt)
	wg.Wait()

	if len(got) != 3 {
		t.Errorf("expected 3 callback invocations, got %d", len(got))
	}

	for i, e := range got {
		if e != evt {
			t.Errorf("invocation %d: got %+v, want %+v", i, e, evt)
		}
	}
}

func TestMultiWriter_PreservesRegistrationOrder(t *testing.T) {
	t.Parallel()

	var (
		order []int
		mu    sync.Mutex
	)

	cb := func(n int) auditlog.MultiWriterCallback {
		return func(auditlog.Event) {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, n)
		}
	}

	mw := auditlog.NewMultiWriter(cb(1), cb(2), cb(3))
	mw.OnEvent(auditlog.Event{})

	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Errorf("callback order violated: %v", order)
	}
}

func TestMultiWriter_ComposableWithConfigOnEvent(t *testing.T) {
	t.Parallel()

	var called atomic.Int64

	mw := auditlog.NewMultiWriter(
		func(auditlog.Event) { called.Add(1) },
		func(auditlog.Event) { called.Add(1) },
	)

	cfg := auditlog.Config{Enabled: true, OnEvent: mw.OnEvent}
	cfg.OnEvent(auditlog.Event{Sequence: 7})

	if got := called.Load(); got != 2 {
		t.Errorf("expected 2 invocations via Config.OnEvent, got %d", got)
	}
}

func TestMultiWriter_ConcurrentSafety(t *testing.T) {
	t.Parallel()

	var counter atomic.Int64

	cb := func() auditlog.MultiWriterCallback {
		return func(auditlog.Event) { counter.Add(1) }
	}

	mw := auditlog.NewMultiWriter(cb(), cb(), cb())

	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func(seq int) {
			defer wg.Done()
			mw.OnEvent(auditlog.Event{Sequence: seq})
		}(i)
	}

	wg.Wait()

	if got := counter.Load(); got != 300 {
		t.Errorf("expected 300 total invocations, got %d", got)
	}
}

func TestMultiWriter_NoCallbacks(t *testing.T) {
	t.Parallel()

	if mw := auditlog.NewMultiWriter(); mw != nil {
		t.Errorf("NewMultiWriter() should return nil for empty input, got %+v", mw)
	}
}

func TestMultiWriter_NilReceiver(t *testing.T) {
	t.Parallel()

	var mw *auditlog.MultiWriter
	mw.OnEvent(auditlog.Event{})

	if got := mw.CallbackCount(); got != 0 {
		t.Errorf("nil MultiWriter.CallbackCount should return 0, got %d", got)
	}
}

func TestMultiWriter_SkipsNilCallback(t *testing.T) {
	t.Parallel()

	called := false

	mw := auditlog.NewMultiWriter(
		nil,
		func(auditlog.Event) { called = true },
		nil,
	)

	mw.OnEvent(auditlog.Event{})

	if !called {
		t.Error("non-nil callback should run between nil callbacks")
	}
}

func TestMultiWriter_CallbackCount(t *testing.T) {
	t.Parallel()

	mw := auditlog.NewMultiWriter(
		func(auditlog.Event) {},
		func(auditlog.Event) {},
	)

	if got := mw.CallbackCount(); got != 2 {
		t.Errorf("CallbackCount = %d, want 2", got)
	}
}
