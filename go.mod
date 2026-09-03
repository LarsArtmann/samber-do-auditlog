module github.com/larsartmann/samber-do-auditlog

go 1.23

// v0.9.0 is retracted: missing live/fragments_templ.go (gitignored generated file).
// Broke Nix builds that vendor source without running templ generate.
retract v0.9.0

require (
	github.com/invopop/jsonschema v0.13.0
	github.com/samber/do/v2 v2.1.0
)

require (
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/samber/go-type-to-string v1.8.0 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
