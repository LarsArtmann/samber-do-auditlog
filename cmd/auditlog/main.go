// Package main implements the auditlog CLI: report conversion, inspection,
// diffing and validation built on the samber-do-auditlog library.
//
// Usage:
//
//	auditlog info <file>                     print a report summary
//	auditlog convert <input> [-o out] [-f FMT]  convert between formats
//	auditlog diff <a> <b>                    diff two reports
//	auditlog validate <file>                 validate report consistency
//	auditlog schema                          print the JSON Schema for the format
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error

	switch cmd {
	case "info":
		err = runInfo(args)
	case "convert":
		err = runConvert(args)
	case "diff":
		err = runDiff(args)
	case "validate":
		err = runValidate(args)
	case "schema":
		err = runSchema(args)
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "auditlog %s: %v\n", cmd, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `auditlog — inspect and convert do-auditlog reports

Usage:
  auditlog info <file>
      Print a human-readable summary of the report.

  auditlog convert <input> [-o output] [-f format]
      Convert a report between formats. Format: json, ndjson, csv, tsv,
      html, mermaid, plantuml, dot. When -f is omitted it is inferred from
      the -o file extension; when -o is omitted output goes to stdout.

  auditlog diff <a> <b>
      Print the structural differences between two reports.

  auditlog validate <file>
      Load and validate a report (consistency + denormalized counts).

  auditlog schema
      Print the canonical JSON Schema for the report format.
`)
}
