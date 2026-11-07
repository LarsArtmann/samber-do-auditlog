package auditlog_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// mustNew is a test helper that wraps auditlog.New and panics on error.
// Use in tests where the config is known to be valid.
func mustNew(config auditlog.Config) *auditlog.Plugin {
	p, err := auditlog.New(config)
	if err != nil {
		panic(fmt.Sprintf("auditlog.New failed: %v", err))
	}

	return p
}

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

// --- Provider helpers ---.
func provideDB(injector do.Injector, name, url string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*Database, error) {
		return &Database{URL: url}, nil
	})
}

// provideCacheWithSleep is a named *Cache provider kept for test compatibility.
func provideCacheWithSleep(injector do.Injector, name string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*Cache, error) {
		return &Cache{Entries: make(map[string]string)}, nil
	})
}

// provideUserServiceWithDB is a named *UserService provider that depends on a single *Database.
func provideUserServiceWithDB(
	injector do.Injector,
	name, dbName string,
) {
	do.ProvideNamed(injector, name, func(i do.Injector) (*UserService, error) {
		db := do.MustInvokeNamed[*Database](i, dbName)

		return &UserService{DB: db}, nil
	})
}

// provideUserServiceWithDeps is a named *UserService provider that depends on a *Database and *Cache.
//

func provideUserServiceWithDeps(injector do.Injector, name, dbName, cacheName string) {
	do.ProvideNamed(injector, name, func(i do.Injector) (*UserService, error) {
		db := do.MustInvokeNamed[*Database](i, dbName)
		cache := do.MustInvokeNamed[*Cache](i, cacheName)

		return &UserService{DB: db, Cache: cache}, nil
	})
}

// provideHTTPServerWithUsers is a named *HTTPServer provider that depends on a *UserService.
func provideHTTPServerWithUsers(injector do.Injector, name, usersName string) {
	do.ProvideNamed(injector, name, func(i do.Injector) (*HTTPServer, error) {
		users := do.MustInvokeNamed[*UserService](i, usersName)

		return &HTTPServer{Users: users}, nil
	})
}

func provideHealthyDB(injector do.Injector, name, dsn string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*HealthyDB, error) {
		return &HealthyDB{DSN: dsn}, nil
	})
}

// provideString is a named *string* provider that returns the given value.
// Used in example/fuzz tests where the exact string value is the test data.
func provideString(injector do.Injector, name, value string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (string, error) {
		return value, nil
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

func findServiceByName(t *testing.T, report auditlog.Report, name auditlog.ServiceName) *auditlog.ServiceInfo {
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
		if strings.HasSuffix(string(report.Services[i].ServiceName), suffix) {
			return &report.Services[i]
		}
	}

	return nil
}
