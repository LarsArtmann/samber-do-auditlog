package main

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/samber/do/v2"
)

// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

// --- Value types (injected via ProvideValue / ProvideNamedValue) ---

type AppConfig struct {
	AppName string
	Port    int
	Debug   bool
}

type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// --- Core services ---

type Logger struct {
	Prefix string
}

func (l *Logger) Printf(format string, args ...any) {
	fmt.Printf("[%s] "+format+"\n", append([]any{l.Prefix}, args...)...)
}

// Database implements do.ShutdownerWithError and do.Healthchecker.
var (
	_ do.ShutdownerWithError = (*Database)(nil)
	_ do.Healthchecker       = (*Database)(nil)
)

type Database struct {
	DSN string
}

func (d *Database) HealthCheck() error {
	if d.DSN == "" {
		return errDatabaseNoConn
	}

	return nil
}

func (d *Database) Shutdown() error {
	fmt.Println("  Database: closing connection")

	return nil
}

// Cache implements do.ShutdownerWithError and do.HealthcheckerWithContext.
var (
	errDatabaseNoConn = errors.New("database: no connection string")
	errCacheUnhealthy = errors.New("cache: unhealthy")
	errLeakyRelease   = errors.New("leaky: failed to release connection pool")
	errUnreliableDep  = errors.New("unreliable: dependency 'payment-gateway' unavailable")

	_ do.ShutdownerWithError      = (*Cache)(nil)
	_ do.HealthcheckerWithContext = (*Cache)(nil)
)

type Cache struct {
	Healthy bool
}

func (c *Cache) HealthCheck(_ context.Context) error {
	if !c.Healthy {
		return errCacheUnhealthy
	}

	return nil
}

func (c *Cache) Shutdown() error {
	fmt.Println("  Cache: flushing and closing")

	return nil
}

// --- Interface aliasing: accept interfaces, return structs ---

// Notifier is the interface consumers depend on.
type Notifier interface {
	Send(to, body string) error
}

// EmailNotifier is the concrete struct producers return.
type EmailNotifier struct {
	From string
}

func (e *EmailNotifier) Send(to, body string) error {
	fmt.Printf("  Email: %s → %s: %s\n", e.From, to, body)

	return nil
}

func (e *EmailNotifier) Shutdown() error {
	fmt.Println("  EmailNotifier: closing SMTP connection")

	return nil
}

// --- Transient services (new instance per invocation) ---

type RideRequest struct {
	ID        int64
	RiderName string
	Pickup    string
	Dropoff   string
	CreatedAt time.Time
}

var rideCounter atomic.Int64 //nolint:gochecknoglobals

// --- Named services: multiple instances of the same type ---

type Vehicle struct {
	Name     string
	Capacity int
	Active   bool
}

func (v *Vehicle) Shutdown() error {
	fmt.Printf("  Vehicle %q: decommissioning\n", v.Name)
	v.Active = false

	return nil
}

// provideVehicle registers a named *Vehicle provider.
func provideVehicle(injector do.Injector, name string, capacity int) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*Vehicle, error) {
		return &Vehicle{Name: name, Capacity: capacity, Active: true}, nil
	})
}

// --- Scoped services: driver and passenger modules ---

type DriverService struct {
	Name    string
	Vehicle *Vehicle
}

func (d *DriverService) Shutdown() error {
	fmt.Printf("  DriverService(%s): going offline\n", d.Name)

	return nil
}

// provideDriverService registers a *DriverService that pulls a named *Vehicle.
func provideDriverService(injector do.Injector, driverName, vehicleName string) {
	do.ProvideNamed(injector, driverName, func(i do.Injector) (*DriverService, error) {
		vehicle := do.MustInvokeNamed[*Vehicle](i, vehicleName)

		return &DriverService{Name: driverName, Vehicle: vehicle}, nil
	})
}

type PassengerService struct {
	Name string
}

func (p *PassengerService) Shutdown() error {
	fmt.Printf("  PassengerService(%s): logging out\n", p.Name)

	return nil
}

// providePassengerService registers a named *PassengerService provider.
func providePassengerService(injector do.Injector, name string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*PassengerService, error) {
		return &PassengerService{Name: name}, nil
	})
}

type MatchingEngine struct {
	Drivers    []*DriverService
	Passengers []*PassengerService
}

func (m *MatchingEngine) Shutdown() error {
	fmt.Println("  MatchingEngine: stopping match loop")

	return nil
}

// --- HTTP server (entry point) ---

type HTTPServer struct {
	Config *AppConfig
	Server *ServerConfig
	DB     *Database
	Cache  *Cache
	Notify Notifier
	Port   int
}

func (s *HTTPServer) ListenAndServe() error {
	fmt.Printf("  HTTP server listening on :%d (timeout: %v)\n", s.Port, s.Server.WriteTimeout)

	return nil
}

func (s *HTTPServer) Shutdown() error {
	fmt.Printf("  HTTPServer: draining connections on :%d\n", s.Port)

	return nil
}

// --- Error demo: a service whose provider fails at invocation time ---

type UnreliableService struct {
	Reason string
}

// --- Error demo: a service whose shutdown fails ---

type LeakyService struct{}

func (l *LeakyService) Shutdown() error {
	return errLeakyRelease
}
