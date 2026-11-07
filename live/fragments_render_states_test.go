package live

import (
	"context"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// newRichFixtureReport builds a report exercising the fragment branches the
// happy-path fixture misses: a failing provider (invocation error on the
// service row and the event stream), a successful root service, and services
// in two named child scopes (nested scope tree).
func newRichFixtureReport(t *testing.T) (auditlog.Report, []auditlog.Event) {
	t.Helper()

	collector := &eventCollector{}

	plugin, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "rich-frag-test",
		OnEvent:     collector.onEvent,
	})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	injector := do.NewWithOpts(plugin.Opts())

	do.ProvideNamed(injector, "logger", func(do.Injector) (*strings.Builder, error) {
		return &strings.Builder{}, nil
	})
	do.ProvideNamed(injector, "broken", func(i do.Injector) (*strings.Reader, error) {
		_ = do.MustInvokeNamed[*strings.Builder](i, "logger")

		return nil, errProviderBoom
	})
	// Registered but never invoked: exercises the nil-duration ("—") row.
	do.ProvideNamed(injector, "idle", func(do.Injector) (*strings.Builder, error) {
		return &strings.Builder{}, nil
	})
	// Invoked and shut down with an error: exercises the ShutdownError row.
	do.ProvideNamed(injector, "flaky", func(do.Injector) (*flakyService, error) {
		return &flakyService{}, nil
	})

	driverScope := injector.Scope("drivers")
	do.ProvideNamed(driverScope, "alice", func(do.Injector) (*strings.Builder, error) {
		return &strings.Builder{}, nil
	})

	// The failing invocation is recorded as an after-event with the error.
	if _, err := do.InvokeNamed[*strings.Reader](injector, "broken"); err == nil {
		t.Fatal("invoke broken: expected provider error")
	}

	// Scoped invocation drives the child-scope fragment branches.
	_ = do.MustInvokeNamed[*strings.Builder](driverScope, "alice")
	_ = do.MustInvokeNamed[*flakyService](injector, "flaky")

	// Shutdown records the flaky service's ShutdownError.
	_ = injector.Shutdown()

	return plugin.Report(), collector.snapshot()
}

// flakyService is a do.ShutdownerWithError whose shutdown always fails.
type flakyService struct{}

func (f *flakyService) Shutdown() error {
	return errShutdownBoom
}

func TestRenderAllFragments_FailedAndScopedStates(t *testing.T) {
	t.Parallel()

	report, events := newRichFixtureReport(t)
	meta := auditlog.BuildTypeMetadata()

	patches := renderAllFragments(context.Background(), report, events, meta)

	bySelector := make(map[string]string, len(patches))
	for _, p := range patches {
		bySelector[p.selector] = p.html
	}

	// Failed service row surfaces the truncated invocation error.
	svc := bySelector["#services-tbody"]
	if !strings.Contains(svc, "boom") {
		t.Errorf("#services-tbody missing invocation error text:\n%s", svc)
	}

	// Events table carries the invocation error event.
	evts := bySelector["#events-tbody"]
	if !strings.Contains(evts, "boom") {
		t.Errorf("#events-tbody missing error event text:\n%s", evts)
	}

	// Scope tree nests the child scope under root.
	tree := bySelector["#scope-tree-container"]
	for _, want := range []string{"root", "drivers", "alice"} {
		if !strings.Contains(tree, want) {
			t.Errorf("#scope-tree-container missing %q:\n%s", want, tree)
		}
	}

	// Graph still renders every node across scopes.
	graph := bySelector["#graph-container"]
	for _, want := range []string{"broken", "logger", "alice"} {
		if !strings.Contains(graph, want) {
			t.Errorf("#graph-container missing %q:\n%s", want, graph)
		}
	}
}

func TestRenderAllFragments_CanceledContext(t *testing.T) {
	t.Parallel()

	report, events := newFixtureReport(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	// A canceled context makes every templ component return ctx.Err() at
	// entry; renderAllFragments must still produce all patches (with the
	// error marker per fragment) instead of panicking.
	patches := renderAllFragments(ctx, report, events, auditlog.BuildTypeMetadata())
	if len(patches) != 10 {
		t.Fatalf("expected 10 fragments with canceled context, got %d", len(patches))
	}
}

func TestRenderAllFragments_ShutdownErrorAndIdleStates(t *testing.T) {
	t.Parallel()

	report, events := newRichFixtureReport(t)
	patches := renderAllFragments(context.Background(), report, events, auditlog.BuildTypeMetadata())

	bySelector := make(map[string]string, len(patches))
	for _, p := range patches {
		bySelector[p.selector] = p.html
	}

	svc := bySelector["#services-tbody"]
	if !strings.Contains(svc, "shutdown boom") {
		t.Errorf("#services-tbody missing shutdown error text:\n%s", svc)
	}

	if !strings.Contains(svc, "idle") {
		t.Errorf("#services-tbody missing never-invoked service row:\n%s", svc)
	}
}
