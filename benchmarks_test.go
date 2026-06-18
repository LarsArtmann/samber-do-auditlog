package auditlog_test

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func BenchmarkHookOverhead_Invocation(b *testing.B) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")

	b.ResetTimer()

	for b.Loop() {
		_, _ = do.InvokeNamed[*Database](injector, "db")
	}
}

func BenchmarkHookOverhead_Disabled(b *testing.B) {
	p := mustNew(auditlog.Config{Enabled: false})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")

	b.ResetTimer()

	for b.Loop() {
		_, _ = do.InvokeNamed[*Database](injector, "db")
	}
}

func BenchmarkHookOverhead_Registration(b *testing.B) {
	b.ResetTimer()

	for b.Loop() {
		p := mustNew(auditlog.Config{Enabled: true})
		injector := do.NewWithOpts(p.Opts())
		provideDB(injector, "svc", "test")
	}
}

func BenchmarkHookOnAfterInvocation(b *testing.B) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	b.ResetTimer()

	for b.Loop() {
		_, _ = do.InvokeNamed[*Database](injector, "db")
	}
}

func BenchmarkHookRegistrationOnly(b *testing.B) {
	b.ResetTimer()

	for b.Loop() {
		p := mustNew(auditlog.Config{Enabled: true})
		injector := do.NewWithOpts(p.Opts())
		do.ProvideValue(injector, &Database{URL: "test"})
	}
}

func BenchmarkConcurrentInvocation(b *testing.B) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = do.InvokeNamed[*Database](injector, "db")
		}
	})
}

func BenchmarkBuildReport(b *testing.B) {
	for _, count := range []int{50, 100, 500} {
		b.Run(fmt.Sprintf("services=%d", count), func(b *testing.B) {
			p := mustNew(auditlog.Config{Enabled: true})
			injector := do.NewWithOpts(p.Opts())

			populateDBServices(injector, count)

			b.ResetTimer()

			for b.Loop() {
				_ = p.Report()
			}
		})
	}
}

func BenchmarkEnrichCapabilities(b *testing.B) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	for i := range 50 {
		name := "svc-" + strconv.Itoa(i)
		do.ProvideNamed(injector, name, func(i do.Injector) (*HealthyDB, error) {
			return &HealthyDB{DSN: "test"}, nil
		})
		_ = do.MustInvokeNamed[*HealthyDB](injector, name)
	}

	b.ResetTimer()

	for b.Loop() {
		_ = p.Report()
	}
}

func BenchmarkEventsCopy(b *testing.B) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	populateDBServices(injector, 50)

	b.ResetTimer()

	for b.Loop() {
		_ = p.Events()
	}
}

func BenchmarkOnEventCallback(b *testing.B) {
	var called atomic.Int64

	p := mustNew(auditlog.Config{
		Enabled: true,
		OnEvent: func(_ auditlog.Event) { called.Add(1) },
	})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")

	b.ResetTimer()

	for b.Loop() {
		_, _ = do.InvokeNamed[*Database](injector, "db")
	}
}

func BenchmarkHealthCheck(b *testing.B) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	for i := range 10 {
		name := "healthy-" + strconv.Itoa(i)
		provideHealthyDB(injector, name, "test")
		_ = do.MustInvokeNamed[*HealthyDB](injector, name)
	}

	b.ResetTimer()

	for b.Loop() {
		_ = p.RecordHealthCheck(injector)
	}
}

func populateDBServices(injector do.Injector, count int) {
	for i := range count {
		name := "svc-" + strconv.Itoa(i)
		provideDB(injector, name, "test")
		_ = do.MustInvokeNamed[*Database](injector, name)
	}
}
