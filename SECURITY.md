# Security Policy

## Supported versions

samber-do-auditlog is pre-v1 (BETA): only the latest release line receives
security fixes. Please keep your `go get` pin current.

| Version                     | Supported |
| --------------------------- | --------- |
| latest release on this repo | yes       |
| older releases              | no        |

## Reporting a vulnerability

**Do not open a public issue for a security report.**

Use GitHub's private vulnerability reporting:
[Security → Report a vulnerability](https://github.com/LarsArtmann/samber-do-auditlog/security/advisories/new).

Include what you can of:

- Affected version (`go list -m github.com/larsartmann/samber-do-auditlog`)
  and Go version (note `GOEXPERIMENT=jsonv2` is required).
- A minimal reproduction: plugin `Config`, service graph, and the exact
  export/stream path (HTML, NDJSON, live dashboard, CLI).
- Observed vs expected behavior.

## What counts as a vulnerability here

This library records dependency-injection lifecycle events and renders them.
Reports are most valuable when they affect that contract, for example:

- **XSS in any export**: attacker-controlled strings (service names, scope
  names, error messages, run IDs) escaping into HTML — the static report
  (`html.templ`, guarded by `FuzzPluginHTML` + syntax validation) or the live
  dashboard fragments.
- **CSP weakening**: a regression that lets the static report or live
  dashboard execute untrusted script (e.g. dropping `default-src 'none'`,
  adding wildcards beyond the documented Google-Fonts/Datastar exceptions).
- **Panic or unbounded resource use from untrusted input**: `LoadReport`,
  `MigrateReport`, `ReadEvents`, or `ReplayEvents` crashing or amplifying on
  crafted JSON/NDJSON.
- **Live-server injection**: response splitting or header injection through
  any `Config` field (prefix, CORS) or request path.
- **Sensitive-data leaks in recorded events** beyond what the container
  operator explicitly registered.

## What is explicitly out of scope

- Exposing the live debug dashboard (`live/`) to untrusted networks. The
  dashboard is a development tool that intentionally renders your full
  service graph — bind it to localhost or keep it behind your own auth.
- Vulnerabilities in [samber/do](https://github.com/samber/do) service
  containers themselves; report those upstream.
- Secrets that host applications pass into service names, DSNs, or provider
  closures — the logger records names it is given; do not put secrets there.

## Response targets

- Acknowledgement: within 7 days.
- Triage and severity assessment: within 14 days.
- Fix or mitigation for accepted reports: within 30 days, released as a patch
  version with credit to the reporter unless anonymity is requested.

## Hardening guidance for users

- The live dashboard is for development: serve it on `127.0.0.1` or inside a
  trusted network. It ships a strict CSP but no authentication.
- The static HTML report is self-contained (`default-src 'none'`) and safe to
  share as a file; treat its content as a disclosure of your architecture.
- Audit logs may embed error strings from services — scrub or filter before
  shipping reports to third parties.
