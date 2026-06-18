package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// runConvert loads a report and writes it in the requested format.
func runConvert(args []string) (err error) {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)

	output := fs.String("o", "", "output file (default: stdout)")
	format := fs.String("f", "", "output format: json, ndjson, csv, tsv, html, mermaid, plantuml, dot")

	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return errors.New("usage: auditlog convert <input> [-o output] [-f format]")
	}

	report, err := loadFile(fs.Arg(0))
	if err != nil {
		return err
	}

	fmtName := *format
	if fmtName == "" && *output != "" {
		fmtName = formatFromExt(*output)
	}

	if fmtName == "" {
		return errors.New("output format not specified; use -f or an -o file extension")
	}

	out := os.Stdout

	if *output != "" {
		f, createErr := os.Create(*output)
		if createErr != nil {
			return fmt.Errorf("create %s: %w", *output, createErr)
		}

		defer func() {
			if cerr := f.Close(); err == nil {
				err = cerr
			}
		}()

		out = f
	}

	return writeFormat(out, report, fmtName)
}

// reorderFlags moves flag arguments (and their values) before positional
// arguments so users can write `convert report.json -f csv` as naturally as
// `convert -f csv report.json`. Go's flag package otherwise stops parsing at
// the first non-flag token.
func reorderFlags(args []string) []string {
	var flags, positionals []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}

		if strings.HasPrefix(arg, "-") && arg != "-" {
			flags = append(flags, arg)

			// Consume the value of a separate-value flag (e.g. -f csv).
			if !strings.Contains(arg, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags = append(flags, args[i+1])
				i++
			}
		} else {
			positionals = append(positionals, arg)
		}
	}

	return append(flags, positionals...)
}

// formatFromExt infers an output format from a file extension.
func formatFromExt(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return "json"
	case ".ndjson":
		return "ndjson"
	case ".csv":
		return "csv"
	case ".tsv":
		return "tsv"
	case ".html", ".htm":
		return "html"
	case ".mmd", ".mermaid":
		return "mermaid"
	case ".puml", ".plantuml":
		return "plantuml"
	case ".dot", ".gv":
		return "dot"
	default:
		return ""
	}
}

func writeFormat(w io.Writer, report auditlog.Report, format string) error {
	switch format {
	case "json":
		return report.WriteJSON(w)
	case "ndjson":
		return report.WriteNDJSON(w)
	case "csv":
		return report.WriteCSV(w)
	case "tsv":
		return report.WriteTSV(w)
	case "html":
		return report.WriteHTML(w)
	case "mermaid":
		return report.WriteMermaid(w)
	case "plantuml":
		return report.WritePlantUML(w)
	case "dot":
		return report.WriteDOT(w)
	default:
		return fmt.Errorf("unknown format %q (want: json, ndjson, csv, tsv, html, mermaid, plantuml, dot)", format)
	}
}
