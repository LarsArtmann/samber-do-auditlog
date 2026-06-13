package main

import (
	"strconv"
	"time"

	"github.com/samber/do/v2"
)

// registerServices wires the full ride-sharing domain model into the injector
// and returns the named child scopes for cross-scope usage.
func registerServices(injector do.Injector) (driverScope, passengerScope, matchingScope *do.Scope) {
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
		time.Sleep(8 * time.Millisecond)

		return &Database{DSN: dsn}, nil
	})

	do.Provide(injector, func(i do.Injector) (*Cache, error) {
		time.Sleep(3 * time.Millisecond)

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
	do.ProvideNamed(injector, "vehicle.sedan", func(i do.Injector) (*Vehicle, error) {
		return &Vehicle{Name: "Sedan", Capacity: 4, Active: true}, nil
	})

	do.ProvideNamed(injector, "vehicle.suv", func(i do.Injector) (*Vehicle, error) {
		return &Vehicle{Name: "SUV", Capacity: 7, Active: true}, nil
	})

	do.ProvideNamed(injector, "vehicle.van", func(i do.Injector) (*Vehicle, error) {
		return &Vehicle{Name: "Van", Capacity: 12, Active: true}, nil
	})

	// 7. Scopes
	driverScope = injector.Scope("drivers")
	passengerScope = injector.Scope("passengers")
	matchingScope = injector.Scope("matching")

	do.Provide(driverScope, func(i do.Injector) (*DriverService, error) {
		vehicle := do.MustInvokeNamed[*Vehicle](i, "vehicle.sedan")

		return &DriverService{Name: "alice", Vehicle: vehicle}, nil
	})

	do.ProvideNamed(driverScope, "driver.bob", func(i do.Injector) (*DriverService, error) {
		vehicle := do.MustInvokeNamed[*Vehicle](i, "vehicle.suv")

		return &DriverService{Name: "bob", Vehicle: vehicle}, nil
	})

	do.ProvideNamed(passengerScope, "passenger.charlie", func(i do.Injector) (*PassengerService, error) {
		return &PassengerService{Name: "charlie"}, nil
	})

	do.ProvideNamed(passengerScope, "passenger.dana", func(i do.Injector) (*PassengerService, error) {
		return &PassengerService{Name: "dana"}, nil
	})

	do.Provide(matchingScope, func(i do.Injector) (*MatchingEngine, error) {
		alice := do.MustInvoke[*DriverService](driverScope)
		bob := do.MustInvokeNamed[*DriverService](driverScope, "driver.bob")

		charlie := do.MustInvokeNamed[*PassengerService](passengerScope, "passenger.charlie")
		dana := do.MustInvokeNamed[*PassengerService](passengerScope, "passenger.dana")

		return &MatchingEngine{
			Drivers:    []*DriverService{alice, bob},
			Passengers: []*PassengerService{charlie, dana},
		}, nil
	})

	// 8. Override
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

	return driverScope, passengerScope, matchingScope
}
