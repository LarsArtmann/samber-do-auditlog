package auditlog_test

import (
	"bytes"
	"fmt"
	"os"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func ExampleNew() {
	plugin := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "my-app",
	})

	injector := do.NewWithOpts(plugin.Opts())

	do.ProvideValue(injector, "hello")

	_ = do.MustInvoke[string](injector)

	report := plugin.Report()
	fmt.Println("services:", report.ServiceCount)

	// Output: services: 1
}

func ExamplePlugin_Report() {
	plugin := auditlog.New(auditlog.Config{Enabled: true})
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
	plugin := auditlog.New(auditlog.Config{Enabled: true})
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
	plugin := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(plugin.Opts())

	do.ProvideNamed(injector, "alpha", func(_ do.Injector) (string, error) {
		return "a", nil
	})

	do.ProvideNamed(injector, "beta", func(_ do.Injector) (string, error) {
		return "b", nil
	})

	_ = do.MustInvokeNamed[string](injector, "alpha")
	_ = do.MustInvokeNamed[string](injector, "beta")

	filtered := plugin.ReportFiltered(
		auditlog.WithServicesByName("alpha"),
	)

	fmt.Println("filtered services:", filtered.ServiceCount)

	// Output: filtered services: 1
}

func ExamplePlugin_RecordHealthCheck() {
	plugin := auditlog.New(auditlog.Config{Enabled: true})
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
	plugin := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(plugin.Opts())

	type appConfig struct{ Debug bool }

	do.ProvideValue(injector, appConfig{Debug: false})

	do.ProvideNamed(injector, "renderer", func(i do.Injector) (string, error) {
		_ = do.MustInvoke[appConfig](i)

		return "rendered", nil
	})

	_ = do.MustInvokeNamed[string](injector, "renderer")

	var buf bytes.Buffer

	err := plugin.Report().WriteMermaid(&buf)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println("has header:", bytes.HasPrefix(buf.Bytes(), []byte("flowchart TD")))

	// Output: has header: true
}
