// Package main is a self-contained demo of the live audit-log dashboard.
//
// It registers services with deliberate delays so you can watch them appear
// on the real-time SSE dashboard, then invokes them, runs health checks,
// and shuts down gracefully.
//
// Run:
//
//	go run ./live/demo
//
// Then open http://localhost:7777/debug/di/ in your browser.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
	"github.com/samber/do/v2"
)

func main() {
	fmt.Println("=== samber-do-auditlog LIVE demo ===")
	fmt.Println("Open http://localhost:7777/debug/di/ in your browser")
	fmt.Println()

	server, plugin, err := live.New(
		auditlog.Config{
			Enabled:     true,
			ContainerID: "live-demo",
		},
		live.Config{
			Addr:              ":7777",
			HeartbeatInterval: 5 * time.Second,
		},
	)
	if err != nil {
		log.Fatalf("create live server: %v", err)
	}

	injector := do.NewWithOpts(plugin.Opts())

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	fmt.Printf("Listening on %s\n", server.Addr())

	// Wait a moment for the server to start.
	time.Sleep(200 * time.Millisecond)

	// Phase 1: Register services with delays.
	registerDemoServices(injector)

	// Phase 2: Invoke services.
	invokeDemoServices(injector)

	// Phase 3: Health checks.
	runHealthChecks(plugin, injector)

	// Phase 4: Signal complete (sends final report to dashboard).
	server.SignalComplete()

	fmt.Println("\nLifecycle complete. Dashboard shows final state.")
	fmt.Println("Press Ctrl+C to exit.")

	// Wait for interrupt signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func registerDemoServices(injector do.Injector) {
	fmt.Print("Registering services")

	do.ProvideNamed(injector, "database", func(i do.Injector) (*Database, error) {
		time.Sleep(300 * time.Millisecond)
		return &Database{DSN: "postgres://localhost:5432/demo"}, nil
	})
	step()

	do.ProvideNamed(injector, "cache", func(i do.Injector) (*Cache, error) {
		time.Sleep(200 * time.Millisecond)
		return &Cache{Addr: "redis://localhost:6379"}, nil
	})
	step()

	do.ProvideNamed(injector, "user-repo", func(i do.Injector) (*UserRepo, error) {
		db := do.MustInvokeNamed[*Database](i, "database")
		return &UserRepo{db: db}, nil
	})
	step()

	do.ProvideNamed(injector, "user-service", func(i do.Injector) (*UserService, error) {
		repo := do.MustInvokeNamed[*UserRepo](i, "user-repo")
		cache := do.MustInvokeNamed[*Cache](i, "cache")

		return &UserService{repo: repo, cache: cache}, nil
	})
	step()

	do.ProvideTransient(injector, func(i do.Injector) (*EmailNotifier, error) {
		return &EmailNotifier{SMTP: "smtp://localhost:587"}, nil
	})

	fmt.Println(" done.")
}

func invokeDemoServices(injector do.Injector) {
	fmt.Print("Invoking services")

	_, _ = do.InvokeNamed[*Database](injector, "database")

	step()

	_, _ = do.InvokeNamed[*Cache](injector, "cache")

	step()

	_, _ = do.InvokeNamed[*UserService](injector, "user-service")

	step()

	_, _ = do.InvokeNamed[*EmailNotifier](injector, "email-notifier")

	step()

	fmt.Println(" done.")
}

func runHealthChecks(plugin *auditlog.Plugin, injector do.Injector) {
	fmt.Print("Running health checks")

	results := plugin.RecordHealthCheck(injector)
	healthy := 0

	for _, err := range results {
		if err == nil {
			healthy++
		}

		step()
	}

	fmt.Printf(" done (%d/%d healthy).\n", healthy, len(results))
}

func step() {
	fmt.Print(".")

	_ = os.Stdout.Sync()

	time.Sleep(400 * time.Millisecond)
}

// --- Demo services ---

type Database struct {
	DSN string
}

type Cache struct {
	Addr string
}

type UserRepo struct {
	db *Database
}

type UserService struct {
	repo  *UserRepo
	cache *Cache
}

type EmailNotifier struct {
	SMTP string
}
