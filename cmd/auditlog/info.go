package main

import (
	"errors"
	"fmt"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// runInfo prints a human-readable summary of a report.
func runInfo(args []string) error {
	fs := newFlagSet("info")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return errors.New("usage: auditlog info <file>")
	}

	report, err := loadFile(fs.Arg(0))
	if err != nil {
		return err
	}

	fmt.Printf("version:      %s\n", report.Version)
	fmt.Printf("container:    %s\n", report.ContainerID)
	fmt.Printf("exported at:  %s\n", report.ExportedAt.Format("2006-01-02 15:04:05 MST"))

	fmt.Printf("\nservices:     %d\n", report.ServiceCount)
	fmt.Printf("events:       %d\n", report.EventCount)
	fmt.Printf("scopes:       %d\n", report.ScopeCount)

	if report.DroppedEventCount > 0 {
		fmt.Printf("dropped:      %d (MaxEvents cap reached)\n", report.DroppedEventCount)
	}

	fmt.Printf("build total:  %.2f ms\n", report.TotalBuildDurationMs)
	fmt.Printf("shutdown:     %.2f ms (succeeded=%v)\n",
		report.TotalShutdownDurationMs, report.ShutdownSucceeded)

	if report.HealthCheckedCount > 0 {
		fmt.Printf("health:       %d checked (succeeded=%v)\n",
			report.HealthCheckedCount, report.HealthCheckSucceeded)
	}

	if failed := report.FailedServices(); len(failed) > 0 {
		fmt.Printf("\nfailed services (%d):\n", len(failed))

		for _, svc := range failed {
			fmt.Printf("  • %s [%s]\n", svc.ServiceName, svc.Status)
		}
	}

	if unhealthy := report.UnhealthyServices(); len(unhealthy) > 0 {
		fmt.Printf("\nunhealthy services (%d):\n", len(unhealthy))

		for _, svc := range unhealthy {
			fmt.Printf("  • %s\n", svc.ServiceName)
		}
	}

	if len(report.Services) > 0 {
		fmt.Println("\nservice list:")
		printServices(report)
	}

	return nil
}

func printServices(report auditlog.Report) {
	for _, svc := range report.Services {
		scope := svc.ScopeName

		if scope == "" {
			scope = auditlog.RootScopeName
		}

		fmt.Printf("  • %-6s %-20s [%s] type=%s invocations=%d\n",
			scope, svc.ServiceName, svc.Status, svc.ServiceType, svc.InvocationCount)
	}
}
