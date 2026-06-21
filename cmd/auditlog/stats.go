package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// runStats prints aggregate statistics for a report.
func runStats(args []string) error {
	fs := flag.NewFlagSet("stats", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return errors.New("usage: auditlog stats <file>")
	}

	report, err := loadFile(fs.Arg(0))
	if err != nil {
		return err
	}

	totalInvocations := 0

	for _, svc := range report.Services {
		totalInvocations += svc.InvocationCount
	}

	failedCount := len(report.FailedServices())
	unhealthyCount := len(report.UnhealthyServices())

	typeBreakdown := make(map[auditlog.ProviderType]int)

	for _, svc := range report.Services {
		typeBreakdown[svc.ServiceType]++
	}

	statusBreakdown := make(map[auditlog.ServiceStatus]int)

	for _, svc := range report.Services {
		statusBreakdown[svc.Status]++
	}

	fmt.Printf("container:        %s\n", report.ContainerID)
	fmt.Printf("services:         %d\n", report.ServiceCount)
	fmt.Printf("events:           %d\n", report.EventCount)
	fmt.Printf("scopes:           %d\n", report.ScopeCount)
	fmt.Printf("invocations:      %d\n", totalInvocations)
	fmt.Printf("failed:           %d\n", failedCount)
	fmt.Printf("unhealthy:        %d\n", unhealthyCount)
	fmt.Printf("build total:      %.2f ms\n", report.TotalBuildDurationMs)
	fmt.Printf("shutdown total:   %.2f ms\n", report.TotalShutdownDurationMs)
	fmt.Printf("avg build:        %.2f ms\n", avgMs(report.TotalBuildDurationMs, report.ServiceCount))

	if len(typeBreakdown) > 0 {
		fmt.Println("\nprovider types:")
		printBreakdown(typeBreakdown)
	}

	if len(statusBreakdown) > 0 {
		fmt.Println("\nstatus breakdown:")
		printBreakdown(statusBreakdown)
	}

	return nil
}

func avgMs(total float64, count int) float64 {
	if count == 0 {
		return 0
	}

	return total / float64(count)
}

func printBreakdown[T ~string](m map[T]int) {
	for val, count := range m {
		fmt.Printf("  %-20s %d\n", val, count)
	}
}
