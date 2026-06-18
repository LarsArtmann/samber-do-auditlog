package main

import (
	"errors"
	"flag"
	"fmt"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// runDiff loads two reports and prints their structural differences.
func runDiff(args []string) error {
	fs := flag.NewFlagSet("diff", flag.ExitOnError)

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 2 {
		return errors.New("usage: auditlog diff <a> <b>")
	}

	a := loadFile(fs.Arg(0))
	b := loadFile(fs.Arg(1))

	result := a.Diff(b)

	if result.IsEmpty() {
		fmt.Println("reports are identical")

		return nil
	}

	fmt.Printf("event count delta: %+d\n", result.EventCountDelta)

	if len(result.AddedServices) > 0 {
		fmt.Printf("\nadded services (%d):\n", len(result.AddedServices))
		printRefs(result.AddedServices)
	}

	if len(result.RemovedServices) > 0 {
		fmt.Printf("\nremoved services (%d):\n", len(result.RemovedServices))
		printRefs(result.RemovedServices)
	}

	if len(result.ChangedServices) > 0 {
		fmt.Printf("\nchanged services (%d):\n", len(result.ChangedServices))

		for _, c := range result.ChangedServices {
			fmt.Printf("  • %s", c.ServiceName)

			if c.StatusChanged {
				fmt.Print(" [status changed]")
			}

			if c.InvocationCountDelta != 0 {
				fmt.Printf(" invocations %+d", c.InvocationCountDelta)
			}

			if c.HealthCheckCountDelta != 0 {
				fmt.Printf(" health %+d", c.HealthCheckCountDelta)
			}

			if c.HasNewError {
				fmt.Print(" NEW ERROR")
			}

			fmt.Println()
		}
	}

	return nil
}

func printRefs(refs []auditlog.ServiceRef) {
	for _, ref := range refs {
		scope := ref.ScopeName
		if scope == "" {
			scope = auditlog.RootScopeName
		}

		fmt.Printf("  • %-6s %s\n", scope, ref.ServiceName)
	}
}
