package auditlog_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// registerNUniqueDatabases launches goroutines that each call provideDB
// with a unique service name derived from nameCounter. Used by stress
// tests to flood the hook system with concurrent registrations.
func registerNUniqueDatabases(goroutines, regsPerGoroutine int, nameCounter *atomic.Int64, injector do.Injector) {
	var wg sync.WaitGroup

	for range goroutines {
		wg.Go(func() {
			for range regsPerGoroutine {
				num := nameCounter.Add(1)

				provideDB(injector, "db-"+strconv.FormatInt(num, 10), "test")
			}
		})
	}

	wg.Wait()
}

// TestPlugin_MaxEventsConcurrentStress fires many goroutines racing against
// hooks with a tight MaxEvents cap and verifies the invariant:
//
//	stored events + dropped events == total events fired
//
// Run with -race to catch data races in appendEventLocked / droppedEvents.
func TestPlugin_MaxEventsConcurrentStress(t *testing.T) {
	t.Parallel()

	// Each registration produces 2 events (before+after). With MaxEvents=10
	// we can store 5 registrations, the rest are dropped.
	const (
		maxEvents        = 10
		goroutines       = 50
		regsPerGoroutine = 3
	)

	totalRegs := goroutines * regsPerGoroutine
	totalEventsExpected := totalRegs * 2

	p := mustNew(auditlog.Config{Enabled: true, MaxEvents: maxEvents, InitialEventCapacity: maxEvents})
	injector := do.NewWithOpts(p.Opts())

	var nameCounter atomic.Int64
	registerNUniqueDatabases(goroutines, regsPerGoroutine, &nameCounter, injector)

	report := p.Report()
	stored := report.EventCount
	dropped := int(report.DroppedEventCount)

	// Invariant: no events are lost or double-counted.
	if stored+dropped != totalEventsExpected {
		t.Fatalf("invariant broken: stored(%d)+dropped(%d)=%d != total fired(%d)",
			stored, dropped, stored+dropped, totalEventsExpected)
	}

	// Cap must be respected: never store more than MaxEvents.
	if stored > maxEvents {
		t.Fatalf("stored %d events exceeds MaxEvents cap %d", stored, maxEvents)
	}

	// Dropped count from Plugin must match the report.
	pluginDropped := int(p.DroppedEventCount())

	if pluginDropped != dropped {
		t.Fatalf("Plugin.DroppedEventCount(%d) != Report.DroppedEventCount(%d)", pluginDropped, dropped)
	}

	t.Logf("stored=%d dropped=%d (total fired=%d)", stored, dropped, totalEventsExpected)
}

// TestPlugin_MaxEventsConcurrentRepeat runs the stress test many times to
// catch flaky races that only surface under repeated scheduling.
func TestPlugin_MaxEventsConcurrentRepeat(t *testing.T) {
	t.Parallel()

	const iterations = 20

	for range iterations {
		p := mustNew(auditlog.Config{Enabled: true, MaxEvents: 4, InitialEventCapacity: 4})
		injector := do.NewWithOpts(p.Opts())

		var nameCounter atomic.Int64
		registerNUniqueDatabases(8, 1, &nameCounter, injector)

		report := p.Report()
		stored := report.EventCount
		dropped := int(report.DroppedEventCount)

		// 8 goroutines * 2 events = 16 total.
		if stored+dropped != 16 {
			t.Fatalf("iteration: stored(%d)+dropped(%d) != 16", stored, dropped)
		}

		if stored > 4 {
			t.Fatalf("stored %d exceeds cap 4", stored)
		}
	}
}

// TestPlugin_AtomicWriteRenameFailure verifies that when os.Rename fails (here:
// the target path is a directory), writeToFile returns an error AND cleans up
// its temp file, leaving no stray .tmp-auditlog-* files behind.
func TestPlugin_AtomicWriteRenameFailure(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()
	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	tmpDir := t.TempDir()

	// Create a directory at the target path so os.Rename(tmpFile, dir) fails
	// with "is a directory" on Linux.
	targetIsDir := filepath.Join(tmpDir, "output.html")

	if err := os.Mkdir(targetIsDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// ExportToHTML should fail because rename targets a directory.
	err := p.ExportToHTML(targetIsDir)
	if err == nil {
		t.Fatal("expected ExportToHTML to fail when target is a directory, got nil")
	}

	// Verify no temp files are left behind in the parent directory.
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".tmp-") {
			t.Errorf("stray temp file left behind after rename failure: %s", name)
		}
	}

	// The directory itself should still exist (not clobbered).
	if _, err := os.Stat(targetIsDir); err != nil {
		t.Errorf("target directory should still exist after failed rename: %v", err)
	}
}

// TestPlugin_AtomicWriteWriteErrorCleanup verifies that when the temp-file
// creation fails (read-only directory), the error propagates and no partial
// output lands at the target path.
func TestPlugin_AtomicWriteWriteErrorCleanup(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()
	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	tmpDir := t.TempDir()

	// Writing to a read-only directory will cause the temp-file creation to
	// fail, proving the error propagates and no partial file appears.
	readOnlyDir := filepath.Join(tmpDir, "readonly")

	if err := os.Mkdir(readOnlyDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.Chmod(readOnlyDir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	err := p.ExportToHTML(filepath.Join(readOnlyDir, "report.html"))
	if err == nil {
		// On some systems (root) chmod is ignored; skip if it succeeded.
		// The rename-failure test above is the authoritative one.
		t.Skip("write succeeded (likely running as root); skipping error-path assertion")
	}

	// No report.html should exist in the read-only dir.
	if _, err := os.Stat(filepath.Join(readOnlyDir, "report.html")); err == nil {
		t.Error("partial output file exists after write error")
	}
}
