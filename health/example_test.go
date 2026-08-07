package health_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/larsartmann/samber-do-auditlog/health"
	"github.com/samber/do/v2"
)

// exampleDB is a minimal service that satisfies do.HealthcheckerWithContext.
type exampleDB struct{}

var _ do.HealthcheckerWithContext = (*exampleDB)(nil)

func (*exampleDB) HealthCheck(_ context.Context) error { return nil }

// ExampleNew shows how to create a health Probe wired to a samber/do
// injector, register a critical service, and evaluate its health.
func ExampleNew() {
	injector := do.New()

	do.ProvideNamed(injector, "database", func(_ do.Injector) (*exampleDB, error) {
		return &exampleDB{}, nil
	})
	_ = do.MustInvokeNamed[*exampleDB](injector, "database")

	probe := health.New(injector,
		health.WithVersion("1.0.0"),
		health.WithCriticalServices("database"),
	)

	resp := probe.Evaluate(context.Background())
	fmt.Println("status:", resp.Status)
	fmt.Println("checks:", len(resp.Checks))

	// Output:
	// status: pass
	// checks: 1
}

// ExampleProbe_LivenessHandler shows the liveness handler in action. Liveness
// never checks dependencies and always returns 200 with status "pass".
func ExampleProbe_LivenessHandler() {
	injector := do.New()
	probe := health.New(injector, health.WithVersion("1.0.0"))

	handler := probe.LivenessHandler()

	w := httptest.NewRecorder()

	r, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/healthz", nil)
	if err != nil {
		panic(err)
	}

	handler(w, r)

	fmt.Println("HTTP status:", w.Code)

	var resp health.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		panic(err)
	}

	fmt.Println("health:", resp.Status)

	// Output:
	// HTTP status: 200
	// health: pass
}

// ExampleProbe_ReadinessHandler shows the readiness handler checking critical
// services. All healthy services return 200; a failing critical service would
// return 503.
func ExampleProbe_ReadinessHandler() {
	injector := do.New()

	do.ProvideNamed(injector, "database", func(_ do.Injector) (*exampleDB, error) {
		return &exampleDB{}, nil
	})
	_ = do.MustInvokeNamed[*exampleDB](injector, "database")

	probe := health.New(injector,
		health.WithCriticalServices("database"),
		health.WithRefreshInterval(0), // live evaluation for deterministic output
	)

	handler := probe.ReadinessHandler()

	w := httptest.NewRecorder()

	r, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/readyz", nil)
	if err != nil {
		panic(err)
	}

	handler(w, r)

	fmt.Println("HTTP status:", w.Code)

	// Output: HTTP status: 200
}

// ExampleProbe_RegisterRoutes shows the one-liner for mounting all three
// Kubernetes probe endpoints on a standard http.ServeMux.
func ExampleProbe_RegisterRoutes() {
	injector := do.New()
	probe := health.New(injector, health.WithRefreshInterval(0))

	mux := http.NewServeMux()
	probe.RegisterRoutes(mux, health.DefaultRoutes())

	for _, path := range []string{"/healthz", "/readyz", "/startupz"} {
		w := httptest.NewRecorder()

		r, err := http.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
		if err != nil {
			panic(err)
		}

		mux.ServeHTTP(w, r)
		fmt.Printf("%s: %d\n", path, w.Code)
	}

	// Output:
	// /healthz: 200
	// /readyz: 200
	// /startupz: 200
}
