package main

import (
	"fmt"
	"log"
	"time"

	auditlog "github.com/larsartmann/do-auditlog"
	"github.com/samber/do/v2"
)

type Config struct {
	Port int
}

type Database struct {
	Config *Config
}

func (d *Database) Connect() error {
	time.Sleep(5 * time.Millisecond) // simulate work
	fmt.Println("Database connected")

	return nil
}

func (d *Database) Shutdown() {
	fmt.Println("Database disconnected")
}

type Cache struct {
	Config *Config
}

func (c *Cache) Connect() error {
	time.Sleep(3 * time.Millisecond) // simulate work
	fmt.Println("Cache connected")

	return nil
}

func (c *Cache) Shutdown() {
	fmt.Println("Cache disconnected")
}

type UserService struct {
	DB    *Database
	Cache *Cache
}

type HTTPServer struct {
	Users *UserService
	Port  int
}

func (s *HTTPServer) Start() error {
	fmt.Printf("HTTP server listening on :%d\n", s.Port)

	return nil
}

func main() {
	// 1. Create the audit log plugin
	plugin := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "demo-app",
	})

	// 2. Pass plugin options into the DI container
	injector := do.NewWithOpts(plugin.Opts())

	// 3. Register services
	do.Provide(injector, func(i do.Injector) (*Config, error) {
		return &Config{Port: 8080}, nil
	})

	do.Provide(injector, func(i do.Injector) (*Database, error) {
		cfg := do.MustInvoke[*Config](i)

		db := &Database{Config: cfg}
		err := db.Connect()
		if err != nil {
			return nil, err
		}

		return db, nil
	})

	do.Provide(injector, func(i do.Injector) (*Cache, error) {
		cfg := do.MustInvoke[*Config](i)

		cache := &Cache{Config: cfg}
		err := cache.Connect()
		if err != nil {
			return nil, err
		}

		return cache, nil
	})

	do.Provide(injector, func(i do.Injector) (*UserService, error) {
		return &UserService{
			DB:    do.MustInvoke[*Database](i),
			Cache: do.MustInvoke[*Cache](i),
		}, nil
	})

	do.Provide(injector, func(i do.Injector) (*HTTPServer, error) {
		cfg := do.MustInvoke[*Config](i)

		return &HTTPServer{
			Users: do.MustInvoke[*UserService](i),
			Port:  cfg.Port,
		}, nil
	})

	// 4. Invoke the entry-point service
	server, err := do.Invoke[*HTTPServer](injector)
	if err != nil {
		log.Fatal(err)
	}

	server.Start()

	// 5. Shutdown gracefully
	report := injector.Shutdown()
	if !report.Succeed {
		log.Printf("shutdown errors: %v", report.Errors)
	}

	// 6. Export audit logs
	if err := plugin.ExportToFile("audit-report.json"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Exported audit-report.json")

	if err := plugin.ExportEventsToNDJSON("audit-events.ndjson"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Exported audit-events.ndjson")

	// 7. Print a quick summary
	rep := plugin.Report()

	fmt.Printf("\nAudit Summary:\n")
	fmt.Printf("  Container:   %s\n", rep.ContainerID)
	fmt.Printf("  Services:    %d\n", rep.ServiceCount)
	fmt.Printf("  Events:      %d\n", rep.EventCount)

	for _, s := range rep.Services {
		fmt.Printf("  - %s (invoked %d times", s.ServiceName, s.InvocationCount)

		if s.BuildDurationMs != nil {
			fmt.Printf(", build %.3f ms", *s.BuildDurationMs)
		}

		if len(s.Dependencies) > 0 {
			fmt.Printf(", deps: %v", s.Dependencies)
		}

		fmt.Println(")")
	}
}
