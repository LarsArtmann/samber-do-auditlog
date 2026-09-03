module github.com/larsartmann/samber-do-auditlog

go 1.18

// v0.9.0 is retracted: missing live/fragments_templ.go (gitignored generated file).
// Broke Nix builds that vendor source without running templ generate.
retract v0.9.0

require (
	github.com/samber/do/v2 v2.1.0
)

require github.com/samber/go-type-to-string v1.8.0 // indirect
