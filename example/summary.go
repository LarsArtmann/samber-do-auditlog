package main

import (
	"fmt"
	"strings"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// printSummary writes a human-readable summary of the audit report to stdout.
// eventLog is a pointer so the caller and the OnEvent closure share the same
// slice header across append reallocations.
func printSummary(report auditlog.Report, eventLog *[]string) {
	fmt.Println()
	fmt.Println("=== Audit Summary ===")

	fmt.Printf("  Container:            %s\n", report.ContainerID)
	fmt.Printf("  Schema version:       %s\n", report.Version)
	fmt.Printf("  Services:             %d\n", report.ServiceCount)
	fmt.Printf("  Events:               %d\n", report.EventCount)
	fmt.Printf("  Scopes:               %d\n", report.ScopeCount)
	fmt.Printf("  Total build time:     %.3f ms\n", report.TotalBuildDurationMs)
	fmt.Printf("  Total shutdown time:  %.3f ms\n", report.TotalShutdownDurationMs)
	fmt.Printf("  Shutdown succeeded:   %t\n", report.ShutdownSucceeded)
	fmt.Println()
	fmt.Println("  Services:")

	for _, s := range report.Services {
		build := float64(0)
		if s.FirstBuildDurationMs != nil {
			build = *s.FirstBuildDurationMs
		}

		shutdown := float64(0)
		if s.ShutdownDurationMs != nil {
			shutdown = *s.ShutdownDurationMs
		}

		depNames := strings.Join(depRefs(s.Dependencies), " ")
		if depNames != "" {
			depNames = " deps=[" + depNames + "]"
		}

		fmt.Printf("    %-32s status=%-18s invoked=%-2d build=%.3fms shutdown=%.3fms%s\n",
			s.ServiceName, s.Status, s.InvocationCount, build, shutdown, depNames)
	}

	fmt.Println()
	fmt.Println("  Scope tree: " + report.ScopeTree.Name)

	for _, child := range report.ScopeTree.Children {
		fmt.Printf("    └── %s (services: %s)\n", child.Name, strings.Join(serviceNamesToStrings(child.Services), " "))
	}

	// Convenience method demos
	fmt.Println("  Convenience method demos:")

	if db := report.ServiceByName("*main.Database"); db != nil && db.FirstBuildDurationMs != nil {
		fmt.Printf("    ServiceByName(\"*main.Database\"): status=%s, build=%.3fms\n",
			db.Status, *db.FirstBuildDurationMs)
	}

	fmt.Printf("    EventsByType(shutdown): %d events\n", len(report.EventsByType(auditlog.EventTypeShutdown)))

	failed := report.FailedServices()
	fmt.Printf("    FailedServices(): %d failures\n", len(failed))

	for _, f := range failed {
		fmt.Printf("      %s: %s\n", f.ServiceName, f.Status)
	}

	if len(*eventLog) > 0 {
		fmt.Println()
		fmt.Println("  OnEvent callback (invocations only):")

		for _, name := range *eventLog {
			fmt.Println("    " + name)
		}
	}
}

func serviceNamesToStrings(names []auditlog.ServiceName) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = string(n)
	}

	return out
}

// depRefs formats dependency service names for the summary line.
func depRefs(refs []auditlog.ServiceRef) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, string(r.ServiceName))
	}

	return out
}
