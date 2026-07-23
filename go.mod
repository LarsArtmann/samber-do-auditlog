module github.com/larsartmann/samber-do-auditlog

go 1.26.4

require github.com/larsartmann/auditlog-core v0.0.0

require (
	github.com/a-h/templ v0.3.1020
	github.com/invopop/jsonschema v0.14.0
	// go-output uses mono-versioning: all sub-modules (d2/escape/graph/plantuml)
	// are tagged together at the same version. Pin them in lockstep.
	github.com/larsartmann/go-output v0.30.1
	github.com/larsartmann/go-output/d2 v0.30.1
	github.com/larsartmann/go-output/daghtml v0.30.1
	github.com/larsartmann/go-output/escape v0.30.1
	github.com/larsartmann/go-output/graph v0.30.1
	github.com/larsartmann/go-output/plantuml v0.30.1
	github.com/samber/do/v2 v2.0.0
)

require (
	github.com/larsartmann/go-output/delimited v0.30.1
	github.com/larsartmann/go-output/markdown v0.30.1
	github.com/larsartmann/go-output/markup v0.30.1
	github.com/larsartmann/go-output/serialization v0.30.1
	github.com/larsartmann/go-output/table v0.30.1
	github.com/larsartmann/go-output/tree v0.30.1
)

require (
	charm.land/lipgloss/v2 v2.0.5 // indirect
	github.com/a-h/parse v0.0.0-20250122154542-74294addb73e // indirect
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.2.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20260703014108-f5a850f9c2b7 // indirect
	github.com/charmbracelet/x/ansi v0.11.7 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/cli/browser v1.3.0 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-faster/errors v0.7.1 // indirect
	github.com/go-faster/jx v1.2.0 // indirect
	github.com/go-faster/yaml v0.4.6 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.1 // indirect
	github.com/larsartmann/go-output/testhelpers v0.30.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/mattn/go-runewidth v0.0.24 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/natefinch/atomic v1.0.1 // indirect
	github.com/pb33f/ordered-map/v2 v2.3.1 // indirect
	github.com/pelletier/go-toml/v2 v2.4.3 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/samber/go-type-to-string v1.8.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.6 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/term v0.45.0 // indirect
	golang.org/x/tools v0.46.0 // indirect
)

tool github.com/a-h/templ/cmd/templ

replace github.com/larsartmann/auditlog-core => ../auditlog-core
