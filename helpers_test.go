package auditlog_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// --- Shared test types ---
type Database struct {
	URL string
}

type Cache struct {
	Entries map[string]string
}

type UserService struct {
	DB    *Database
	Cache *Cache
}
type CrashingService struct{}

var errConnectionReset = errors.New("connection reset")

func (c *CrashingService) Shutdown() error {
	return errConnectionReset
}

type HealthyDB struct {
	DSN string
}

var _ do.Healthchecker = (*HealthyDB)(nil)

func (d *HealthyDB) HealthCheck() error {
	return nil
}

type UnhealthyCache struct {
	Reason string
}

var _ do.HealthcheckerWithContext = (*UnhealthyCache)(nil)

var errCacheUnhealthy = errors.New("cache: unhealthy")

func (c *UnhealthyCache) HealthCheck(_ context.Context) error {
	return errCacheUnhealthy
}

// --- Health check tests ---

type HTTPServer struct {
	Users *UserService
}

type Config struct {
	Port int
}

type CrashingService struct{}

// --- Provider helpers ---
func provideDB(injector do.Injector, name, url string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*Database, error) {
		return &Database{URL: url}, nil
	})
}

func provideHealthyDB(injector do.Injector, name, dsn string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*HealthyDB, error) {
		return &HealthyDB{DSN: dsn}, nil
	})
}

func provideUnhealthyCache(injector do.Injector, name, reason string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*UnhealthyCache, error) {
		return &UnhealthyCache{Reason: reason}, nil
	})
}

func provideFailing(injector do.Injector, name string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*Database, error) {
		return nil, os.ErrNotExist
	})
}

func provideCache(injector do.Injector, name string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*Cache, error) {
		return &Cache{}, nil
	})
}

func provideCrashing(injector do.Injector, name string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*CrashingService, error) {
		return &CrashingService{}, nil
	})
}

func findServiceByName(t *testing.T, report auditlog.Report, name string) *auditlog.ServiceInfo {
	t.Helper()

	for i := range report.Services {
		if report.Services[i].ServiceName == name {
			return &report.Services[i]
		}
	}

	return nil
}

func findServiceBySuffix(t *testing.T, report auditlog.Report, suffix string) *auditlog.ServiceInfo {
	t.Helper()

	for i := range report.Services {
		if strings.HasSuffix(report.Services[i].ServiceName, suffix) {
			return &report.Services[i]
		}
	}

	return nil
}

func assertVersion(t *testing.T, report auditlog.Report) {
	t.Helper()

	if report.Version != auditlog.SchemaVersion {
		t.Errorf("version: want %s, got %s", auditlog.SchemaVersion, report.Version)
	}
}

func newPluginWithCapture() (*auditlog.Plugin, *[]auditlog.Event, do.Injector) { //nolint:ireturn
	var captured []auditlog.Event

	p := auditlog.New(auditlog.Config{
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
