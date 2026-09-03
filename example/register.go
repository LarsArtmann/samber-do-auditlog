package main

import (
	"strconv"
	"time"

	"github.com/samber/do/v2"
)

// registerServices wires the full ride-sharing domain model into the injector
// and returns the matching scope for cross-scope usage.
func registerServices(injector do.Injector) *do.Scope {
	// 2. Eager value injection
	do.ProvideValue(injector, &AppConfig{
		AppName: "RideShare",
		Port:    8080,
		Debug:   true,
	})

	do.ProvideNamedValue(injector, "config.db.dsn", "postgres://localhost:5432/rideshare?sslmode=disable")

	// 3. Lazy singletons
	do.Provide(injector, func(i do.Injector) (*Logger, error) {
		cfg := do.MustInvoke[*AppConfig](i)

		return &Logger{Prefix: cfg.AppName}, nil
	})

	do.Provide(injector, func(i do.Injector) (*Database, error) {
		dsn := do.MustInvokeNamed[string](i, "config.db.dsn")
		logger := do.MustInvoke[*Logger](i)

		logger.Printf("connecting to database: %s", dsn)

		return &Database{DSN: dsn}, nil
	})

	do.Provide(injector, func(i do.Injector) (*Cache, error) {
		return &Cache{Healthy: true}, nil
	})

	// 4. Interface aliasing
	do.Provide(injector, func(i do.Injector) (*EmailNotifier, error) {
		return &EmailNotifier{From: "no-reply@rideshare.app"}, nil
	})

	_ = do.As[*EmailNotifier, Notifier](injector)

	// 5. Transient provider
	do.ProvideTransient(injector, func(i do.Injector) (*RideRequest, error) {
		id := rideCounter.Add(1)

		return &RideRequest{
			ID:        id,
			RiderName: "rider-" + strconv.FormatInt(id, 10),
			Pickup:    "123 Main St",
			Dropoff:   "456 Elm Ave",
			CreatedAt: time.Now(),
		}, nil
	})

	// 6. Named services
	provideVehicle(injector, "vehicle.sedan", 4)
	provideVehicle(injector, "vehicle.suv", 7)
	provideVehicle(injector, "vehicle.van", 12)

	// 7. Scopes
	driverScope := injector.Scope("drivers")
	passengerScope := injector.Scope("passengers")
	matchingScope := injector.Scope("matching")

	provideDriverService(driverScope, "alice", "vehicle.sedan")
	provideDriverService(driverScope, "driver.bob", "vehicle.suv")
	providePassengerService(passengerScope, "passenger.charlie")
	providePassengerService(passengerScope, "passenger.dana")

	do.Provide(matchingScope, func(i do.Injector) (*MatchingEngine, error) {
		alice := do.MustInvokeNamed[*DriverService](driverScope, "alice")
		bob := do.MustInvokeNamed[*DriverService](driverScope, "driver.bob")

		charlie := do.MustInvokeNamed[*PassengerService](passengerScope, "passenger.charlie")
		dana := do.MustInvokeNamed[*PassengerService](passengerScope, "passenger.dana")

		return &MatchingEngine{
			Drivers:    []*DriverService{alice, bob},
			Passengers: []*PassengerService{charlie, dana},
		}, nil
	})

	// 8. Override (hot-swap) — ProvideValue + OverrideValue use the same struct
	// type intentionally to demonstrate the override feature (see AGENTS.md
	// duplication policy, category 4: demo pairs).
	do.ProvideValue(injector, &ServerConfig{
		Port:         80,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	})

	do.OverrideValue(injector, &ServerConfig{
		Port:         8080,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	// 9. HTTP server
	do.Provide(injector, func(i do.Injector) (*HTTPServer, error) {
		cfg := do.MustInvoke[*AppConfig](i)
		srvCfg := do.MustInvoke[*ServerConfig](i)
		db := do.MustInvoke[*Database](i)
		cache := do.MustInvoke[*Cache](i)
		notifier := do.MustInvoke[Notifier](i)

		return &HTTPServer{
			Config: cfg,
			Server: srvCfg,
			DB:     db,
			Cache:  cache,
			Notify: notifier,
			Port:   cfg.Port,
		}, nil
	})

	// 10. Error-case services
	do.Provide(injector, func(i do.Injector) (*UnreliableService, error) {
		return nil, errUnreliableDep
	})

	do.Provide(injector, func(i do.Injector) (*LeakyService, error) {
		return &LeakyService{}, nil
	})

	return matchingScope
}
