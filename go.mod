module github.com/larsartmann/samber-do-auditlog

go 1.26.3

require (
	github.com/a-h/templ v0.3.1020
	github.com/invopop/jsonschema v0.14.0
	// go-output uses mono-versioning: all sub-modules (d2/escape/graph/plantuml)
	// are tagged together at the same version. Pin them in lockstep.
	github.com/larsartmann/go-output v0.17.0
	github.com/larsartmann/go-output/d2 v0.17.0
	github.com/larsartmann/go-output/escape v0.17.0
	github.com/larsartmann/go-output/graph v0.17.0
	github.com/larsartmann/go-output/plantuml v0.17.0
	github.com/samber/do/v2 v2.0.0
)

require (
	github.com/a-h/parse v0.0.0-20250122154542-74294addb73e // indirect
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.2.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cli/browser v1.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.1 // indirect
	github.com/larsartmann/go-output/enum v0.17.0 // indirect
	github.com/larsartmann/go-output/envdetect v0.17.0 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/natefinch/atomic v1.0.1 // indirect
	github.com/pb33f/ordered-map/v2 v2.3.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/samber/go-type-to-string v1.8.0 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.6 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/term v0.44.0 // indirect
	golang.org/x/tools v0.46.0 // indirect
)

tool github.com/a-h/templ/cmd/templ
