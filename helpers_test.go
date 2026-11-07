package auditlog_test

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// --- Shared test types ---.

// epochTime is the canonical fixture base time used by every test that needs
// a deterministic timestamp (replay, merge, new-report). Centralizes the
// `time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)` literal that previously
// appeared in 6 different test files.
var epochTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func newPluginAndInjector() (*auditlog.Plugin, do.Injector) {
	p := mustNew(auditlog.Config{Enabled: true})

	return p, do.NewWithOpts(p.Opts())
}

func newPluginAndInjectorWithID(containerID auditlog.ContainerID) (*auditlog.Plugin, do.Injector) {
	p := mustNew(auditlog.Config{Enabled: true, ContainerID: containerID})

	return p, do.NewWithOpts(p.Opts())
}

// setupWithDB returns a plugin and injector with a single *Database registered
// under "db" and already invoked. The standard 4-line "register + invoke"
// preamble that opens most plugin-level tests.
func setupWithDB(url string) (*auditlog.Plugin, do.Injector) {
	p, injector := newPluginAndInjector()
	provideDB(injector, "db", url)
	_ = do.MustInvokeNamed[*Database](injector, "db")

	return p, injector
}

// setupWithDBReport returns the populated Report from a fresh plugin with a
// single *Database invoked — the 5-line preamble (`mustNew + NewWithOpts +
// provideDB + invoke + Report()`) shared by every "build a populated report
// and assert on it" test.
func setupWithDBReport() auditlog.Report {
	p, _ := setupWithDB("test")

	return p.Report()
}

// writeHTMLToString runs the standard "plugin + provideDB + invoke + WriteHTML"
// pipeline and returns the rendered HTML string. Fails the test on any
// pipeline error. Centralizes the 7-line preamble shared by every HTML
// rendering test.
func writeHTMLToString(t *testing.T) string {
	t.Helper()

	p, _ := setupWithDB("postgres://localhost")

	var buf bytes.Buffer

	if err := p.WriteHTML(&buf); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	return buf.String()
}

// setupRootAndChildScopeDBs returns a plugin + injector + child scope wired
// with a named *Database in each scope (root and child) and both invoked.
// This is the shared preamble for the four "exercise the scope tree" tests
// that otherwise duplicate the same 8 lines of provideDB + MustInvokeNamed
// boilerplate.
//

func setupRootAndChildScopeDBs(
	rootName, rootURL, childName, childURL string,
) (*auditlog.Plugin, do.Injector, *do.Scope) {
	p, injector := newPluginAndInjector()
	child := injector.Scope("child")

	provideDB(injector, rootName, rootURL)
	provideDB(child, childName, childURL)

	_ = do.MustInvokeNamed[*Database](injector, rootName)
	_ = do.MustInvokeNamed[*Database](child, childName)

	return p, injector, child
}

// replayFromPlugin drives the standard round-trip: writes plugin events to
// NDJSON, reads them back, and replays them into a Report. Centralizes the
// 8-line preamble that opens every replay round-trip test.
func replayFromPlugin(t *testing.T, p *auditlog.Plugin) auditlog.Report {
	t.Helper()

	var buf bytes.Buffer

	if err := p.WriteEventsNDJSON(&buf); err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	events, err := auditlog.ReadEvents(&buf)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	return report
}

// assertWriteFails expects a non-nil error from fn writing to a failingWriter.
// Centralizes the "failingWriter error path" assertion used by every plugin
// write-method test.
func assertWriteFails(t *testing.T, label string, fn func(io.Writer) error) {
	t.Helper()

	if err := fn(failingWriter{}); err == nil {
		t.Fatalf("%s: expected error from failing writer", label)
	}
}

// assertErrIs fails the test if err does not match want via errors.Is. Shared
// by every "should return sentinel error" test in the loader/reader paths.
func assertErrIs(t *testing.T, err, want error, label string) {
	t.Helper()

	if !errors.Is(err, want) {
		t.Errorf("%s: want %v, got %v", label, want, err)
	}
}

// assertLen fails the test if the slice length does not equal want.
func assertLen[T any](t *testing.T, label string, slice []T, want int) {
	t.Helper()

	if len(slice) != want {
		t.Errorf("%s length: want %d, got %d", label, want, len(slice))
	}
}

func newPluginWithCapture() (*auditlog.Plugin, *[]auditlog.Event, do.Injector) {
	var captured []auditlog.Event

	p := mustNew(auditlog.Config{
		Enabled: true,
		OnEvent: func(e auditlog.Event) {
			captured = append(captured, e)
		},
	})

	return p, &captured, do.NewWithOpts(p.Opts())
}

// --- Writer error types ---

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

// --- Error sentinels for writer tests ---

var (
	errWriteFailed       = errors.New("write failed")
	errConnectionRefused = errors.New("connection refused")
)
