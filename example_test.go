package auditlog_test

import (
	"bytes"
	"fmt"
	"os"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func ExampleNew() {
	plugin, injector := newPluginAndInjectorWithID("my-app")

	do.ProvideValue(injector, "hello")

	_ = do.MustInvoke[string](injector)

	report := plugin.Report()
	fmt.Println("services:", report.ServiceCount)

	// Output: services: 1
}

// setupNamedRendererInjector builds a minimal injector with one
// registered "renderer" string service that depends on an appConfig
// value. Used by ExampleReport_WriteMermaid and ExampleReport_WriteD2
// to keep the setup code single-sourced.
func setupNamedRendererInjector() *auditlog.Plugin {
	type appConfig struct{ Debug bool }

	plugin := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(plugin.Opts())

	do.ProvideValue(injector, appConfig{Debug: false})

	do.ProvideNamed(injector, "renderer", func(i do.Injector) (string, error) {
		_ = do.MustInvoke[appConfig](i)

		return "rendered", nil
	})

	_ = do.MustInvokeNamed[string](injector, "renderer")

	return plugin
}

func ExamplePlugin_Report() {
	plugin := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(plugin.Opts())

	type reportConfig struct{ Val string }

	do.ProvideNamed(injector, "report-cfg", func(_ do.Injector) (*reportConfig, error) {
		return &reportConfig{Val: "prod"}, nil
	})

	_ = do.MustInvokeNamed[*reportConfig](injector, "report-cfg")

	report := plugin.Report()

	svc := report.ServiceByName("report-cfg")
	if svc != nil {
		fmt.Println("found service")
	}

	// Output: found service
}

func ExamplePlugin_ExportToFile() {
	plugin := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(plugin.Opts())

	do.ProvideValue(injector, 42.0)

	_ = do.MustInvoke[float64](injector)

	path := os.Args[0] + ".audit.json"

	err := plugin.ExportToFile(path)
	if err != nil {
		fmt.Println("export error:", err)

		return
	}

	fmt.Println("exported")

	// Output: exported
}

func ExampleReport_Filtered() {
	plugin := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(plugin.Opts())

	provideString(injector, "alpha", "a")
	provideString(injector, "beta", "b")

	_ = do.MustInvokeNamed[string](injector, "alpha")
	_ = do.MustInvokeNamed[string](injector, "beta")

	filtered := plugin.ReportFiltered(
		auditlog.WithServicesByName("alpha"),
	)

	fmt.Println("filtered services:", filtered.ServiceCount)

	// Output: filtered services: 1
}

func ExamplePlugin_RecordHealthCheck() {
	plugin := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(plugin.Opts())

	type dbConn struct{ Connected bool }

	do.ProvideNamed(injector, "health-db", func(_ do.Injector) (*dbConn, error) {
		return &dbConn{Connected: true}, nil
	})

	_ = do.MustInvokeNamed[*dbConn](injector, "health-db")

	results := plugin.RecordHealthCheck(injector)
	fmt.Println("services checked:", len(results))

	// Output: services checked: 1
}

func ExampleReport_WriteMermaid() {
	plugin := setupNamedRendererInjector()

	var buf bytes.Buffer

	err := plugin.Report().WriteMermaid(&buf)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println("has header:", bytes.Contains(buf.Bytes(), []byte("flowchart TD")))

	// Output: has header: true
}

func ExampleReport_WriteD2() {
	plugin := setupNamedRendererInjector()

	var buf bytes.Buffer

	err := plugin.Report().WriteD2(&buf)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println("has edge:", bytes.Contains(buf.Bytes(), []byte("->")))

	// Output: has edge: true
}

func Example_validate() {
	cfg := auditlog.Config{ContainerID: "my-app"}
	fmt.Println("valid:", cfg.Validate())

	cfg = auditlog.Config{ContainerID: "my/app"}
	fmt.Println("invalid:", cfg.Validate() != nil)

	// Output:
	// valid: <nil>
	// invalid: true
}
