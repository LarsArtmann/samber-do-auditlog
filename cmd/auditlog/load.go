package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// commonFlags holds the flags every report-consuming subcommand accepts.
// --input-format forces the parse format instead of auto-detection,
// --verbose prints load diagnostics to stderr, and --quiet suppresses
// informational (non-result) output. verbose and quiet are mutually exclusive.
type commonFlags struct {
	inputFormat *string
	verbose     *bool
	quiet       *bool
}

// registerCommonFlags registers the shared flags on a subcommand's flag set.
func registerCommonFlags(fs *flag.FlagSet) commonFlags {
	return commonFlags{
		inputFormat: fs.String("input-format", "auto", "input format: auto, json, ndjson"),
		verbose:     fs.Bool("verbose", false, "print load diagnostics to stderr"),
		quiet:       fs.Bool("quiet", false, "suppress informational output"),
	}
}

// parseInputFormat maps a --input-format value to a library Format.
func parseInputFormat(value string) (auditlog.Format, error) {
	switch value {
	case "auto", "":
		return auditlog.FormatAuto, nil
	case "json":
		return auditlog.FormatJSON, nil
	case "ndjson":
		return auditlog.FormatNDJSON, nil
	default:
		return auditlog.FormatAuto, fmt.Errorf("invalid --input-format %q (want: auto, json, ndjson)", value)
	}
}

// format returns the parsed --input-format value.
func (c commonFlags) format() (auditlog.Format, error) {
	return parseInputFormat(*c.inputFormat)
}

// validate rejects contradictory flag combinations.
func (c commonFlags) validate() error {
	if *c.verbose && *c.quiet {
		return errors.New("--verbose and --quiet are mutually exclusive")
	}

	return nil
}

// logLoaded prints a load diagnostic to w when --verbose is set.
func (c commonFlags) logLoaded(w io.Writer, path string, format auditlog.Format, report auditlog.Report) {
	if !*c.verbose {
		return
	}

	fmt.Fprintf(w, "loaded %s: format=%s services=%d events=%d scopes=%d\n",
		path, format, report.ServiceCount, report.EventCount, report.ScopeCount)
}

// loadFile loads a report from path using the given format (auto-detecting
// when FormatAuto). A path of "-" reads from stdin. It returns the format the
// loader used, for --verbose diagnostics.
func loadFile(path string, format auditlog.Format) (auditlog.Report, auditlog.Format, error) {
	if path == "-" {
		report, used, err := auditlog.LoadReportFromReader(os.Stdin, format)
		if err != nil {
			return auditlog.Report{}, format, fmt.Errorf("load stdin: %w", err)
		}

		return report, used, nil
	}

	report, used, err := auditlog.LoadReport(path, auditlog.WithFormat(format))
	if err != nil {
		return auditlog.Report{}, format, fmt.Errorf("load %s: %w", path, err)
	}

	return report, used, nil
}

// parseCommonFlagSet parses flags for a subcommand that accepts the shared
// flags plus expectedNArg positional arguments. Flags are reordered before
// positional arguments so `info report.json --quiet` parses naturally.
func parseCommonFlagSet(
	name string,
	args []string,
	expectedNArg int,
	usage string,
) (*flag.FlagSet, commonFlags, error) {
	fs := newFlagSet(name)
	flags := registerCommonFlags(fs)

	if err := fs.Parse(reorderFlags(args)); err != nil {
		return nil, commonFlags{}, err
	}

	if fs.NArg() != expectedNArg {
		return nil, commonFlags{}, errors.New(usage)
	}

	if err := flags.validate(); err != nil {
		return nil, commonFlags{}, err
	}

	return fs, flags, nil
}

// loadSingleReportSubcommand is the common preamble for subcommands that take
// exactly one positional report file: parse the shared flags, enforce the arg
// count, load the report, and return it together with the source path. The
// usage string is used in the "usage: ..." error returned when the arg count
// is wrong.
func loadSingleReportSubcommand(
	name string,
	args []string,
	usage string,
) (auditlog.Report, string, commonFlags, error) {
	fs, flags, err := parseCommonFlagSet(name, args, 1, usage)
	if err != nil {
		return auditlog.Report{}, "", commonFlags{}, err
	}

	format, err := flags.format()
	if err != nil {
		return auditlog.Report{}, "", commonFlags{}, err
	}

	report, used, err := loadFile(fs.Arg(0), format)
	if err != nil {
		return auditlog.Report{}, "", commonFlags{}, err
	}

	flags.logLoaded(os.Stderr, fs.Arg(0), used, report)

	return report, fs.Arg(0), flags, nil
}
