# PRO/CONTRA: Adopting `go-error-family` in `samber-do-auditlog`

**Scope:** Library (`auditlog` package) + CLI (`cmd/auditlog`). Analysis date: 2026-06-22.

---

## Context Snapshot

| Aspect             | auditlog today                                                                        | go-error-family v0.5.0                                                |
| ------------------ | ------------------------------------------------------------------------------------- | --------------------------------------------------------------------- |
| Role               | Consumer (would import)                                                               | Provider (zero-dep root)                                              |
| Error model        | 14 unexported + 3 exported sentinels, ~45 `fmt.Errorf("%w")` sites, zero custom types | 5 Families, 4 interfaces, `Classify()`, `ExitCode()`, `HandleError()` |
| `errors.As` usage  | **0** (none anywhere)                                                                 | Core to its design                                                    |
| `errors.Is` (prod) | **1** site (`bufio.ErrTooLong`)                                                       | Registry walks `errors.Is`                                            |
| Retry logic        | **None**, none planned                                                                | Core value prop (`IsRetryable`)                                       |
| CLI exit codes     | `0` / `1` / `2` only                                                                  | BSD sysexits.h (`1`, `65`, `69`, `75`)                                |
| Both owned by      | Same author                                                                           | Same author                                                           |

---

## PRO

### 1. CLI boundary is a genuine fit

`cmd/auditlog` is exactly the "error leaves the program" boundary go-error-family targets. Today every error exits `1` with a flat `fmt.Fprintf(os.Stderr, "auditlog %s: %v\n", ...)`. `HandleError(err)` would give structured **What / Why / Fix / WayOut** messages, and `ExitCode(err)` would distinguish "bad input" (exit 1) from "corrupt report file" (exit 65) from "system failure" (exit 69/75). This is the single clearest win — 6 subcommands, immediate user-facing value.

### 2. Classification adds signal to loader/parser errors

The NDJSON reader and migration path have semantically distinct failures that currently all look the same to a caller:

| Current sentinel                          | Natural Family     | Why                               |
| ----------------------------------------- | ------------------ | --------------------------------- |
| `ErrEmpty` / `ErrNoEvents`                | **Rejection**      | Caller passed empty/invalid input |
| `ErrOversizedLine`                        | **Corruption**     | Input data is malformed           |
| `errUnknownEventType` / `errUnknownPhase` | **Corruption**     | Schema violation in data          |
| `errMigrationMissingVersion`              | **Corruption**     | Unparseable/migrated data         |
| `errUnsupportedFormat`                    | **Rejection**      | Caller asked for unknown format   |
| `errContainerIDPathSep`                   | **Rejection**      | Bad config input                  |
| File I/O failures (`writeToFile`)         | **Infrastructure** | System can't serve                |

Today a consumer has **no way** to distinguish "your file is corrupt" from "disk is full" without string-matching. Classification fixes that.

### 3. Machine-readable error codes

The library's public errors (`ErrEmpty`, `ErrNoEvents`, `ErrOversizedLine`) are bare sentinels — no code, no context. `errorfamily.NewRejection("ndjson.empty", ...)` gives consumers a stable `ErrorCode()` string for metrics, logs, and programmatic handling without `errors.Is` chains. This aligns with the auditlog's own philosophy of structured, machine-readable output.

### 4. Context attachment is currently ad hoc

Error context is baked into format strings: `fmt.Errorf("create temp file in %q: %w", dir, err)`. `.WithContext("dir", dir).WithContext("path", path)` is structured and queryable — consumers can extract `ErrorContext()` instead of parsing strings. The `containerID`, file path, and line number are all natural context keys.

### 5. Zero-dependency, same author, same toolchain

The root module has zero third-party deps. Both projects use Go 1.26+, nix flakes, golangci-lint v2, and are owned by the same author — coordination cost is near zero. Adding it doesn't pull in a transitive dependency tree.

### 6. Aligns with go-error-family's own design philosophy

The library explicitly states _"LIBRARIES import go-error-family only"_ — auditlog is a library. This is the intended consumer profile.

---

## CONTRA

### 1. The core value prop is unused: no retry logic exists or is planned

go-error-family's central design decision is _"Transient is the only retryable family; everything else is not."_ auditlog has **zero retry logic** — it's a fire-and-forget audit recorder. `IsRetryable()`, `RetryPolicy()`, and the entire Transient family carry no weight here. Adopting a retry-classification library for a project that never retries is paying for a feature you won't use.

### 2. "Fail-open to Transient" default is semantically dangerous here

`Classify(unknownErr)` returns `Transient` by design (fail-open for retry). For a DI audit plugin, an **unknown error should NOT be retried** — silently classifying a nil-deref or a logic bug as "retryable" could mask real failures in a consumer's retry loop. This default inverts the safety posture a plugin library should have. You'd need to register every sentinel or accept the wrong default.

### 3. HTTP surface is irrelevant

`Family.HTTPStatus()` (Rejection→400, Transient→503, etc.) is useless — auditlog has no HTTP layer. It's a DI container plugin and a file-format CLI. This removes ~20% of the library's advertised value.

### 4. Breaking change to the public sentinel contract

`ErrEmpty`, `ErrNoEvents`, `ErrOversizedLine` are exported sentinels that consumers match via `errors.Is(err, auditlog.ErrEmpty)`. Converting them to `*errorfamily.Error` changes the identity contract:

- `errors.Is` matching on `*Error` uses **code + family**, ignoring message — a semantic shift
- Consumers doing `errors.Is(err, auditlog.ErrEmpty)` would need the sentinel to still satisfy the comparison, requiring careful `Is()` method wiring
- Any consumer currently using `errors.Is` would break or silently behave differently

This is a v0.x library with no known external consumers yet, so the blast radius is small — but it sets a precedent for the public API.

### 5. Marginal benefit vs. churn: ~45 sites for little behavioral change

The bulk of `fmt.Errorf("...: %w", err)` sites are **I/O delegation** (render → write → close). Wrapping each in `errorfamily.WrapInfrastructure(err, "export.write", "...")` adds a constructor call and a code string to every site, but the consumer behavior doesn't change — they still just log or return. The classification is metadata that nobody currently consumes. This is **speculative API design** — adding structure before a consumer asks for it.

### 6. Coupling two v0.x / ALPHA APIs

Both projects are pre-1.0. go-error-family has had breaking changes (`{key}` template syntax, `errors.AsType` requirement, `HandleConfig.Registry` addition). Adopting it means auditlog's error contract inherits go-error-family's instability. A breaking change in go-error-family's `Error` struct or `Classify` semantics becomes a breaking change in auditlog.

### 7. depguard + exhaustruct friction

The `.golangci.yml` depguard allow-list is extremely restrictive (`$gostd`, `templ`, `go-output`, `samber`). Adding `go-error-family` requires an allow-list entry. Additionally, `exhaustruct` is enabled — if any `errorfamily.Error` or `HandleConfig` literals appear, all fields must be initialized. These are solvable but add config surface.

### 8. The double-`%w` pattern in `replay.go` doesn't map cleanly

`replay.go:91` uses `fmt.Errorf("%w: %w", errReplayValidationFailed, err)` — Go 1.20+ multi-error join. go-error-family's `*Error` has a single `Cause()`, not a multi-cause chain. This pattern would need rethinking (use `errors.Join` + let `Classify` pick worst-family, or lose one layer of context).

### 9. Diagnostic rules / agent are irrelevant

The `diagnose/` and `agent/` submodules (PostgreSQL, git, filesystem, network root-cause analysis) have no application in a DI audit log. That's ~60% of go-error-family's codebase — dead weight if imported conceptually, though not literally (separate modules).

---

## Verdict Matrix

| Dimension                            | Fit          | Weight                            |
| ------------------------------------ | ------------ | --------------------------------- |
| CLI exit codes + structured messages | **Strong**   | Medium (6 subcommands)            |
| Loader/parser error classification   | **Moderate** | Medium                            |
| Retry decisions                      | **None**     | High (core value prop)            |
| HTTP status mapping                  | **None**     | Low (no HTTP layer)               |
| Context attachment                   | **Moderate** | Low (currently works via strings) |
| Machine-readable codes               | **Moderate** | Low (no consumer yet)             |
| Diagnostic rules / agent             | **None**     | Low                               |

---

## Recommendation

**Do not adopt now. Revisit on a specific trigger.**

The CLI boundary is the one genuine fit, but it alone doesn't justify coupling two v0.x APIs across ~45 error sites for a library with no retry logic, no HTTP surface, and no consumer requesting classification. The fail-open-to-Transient default actively conflicts with a plugin library's safety posture.

**Adopt when ANY of these become true:**

1. The CLI grows rich user-facing error guidance (What/Why/Fix) — then adopt **only in `cmd/auditlog`**, leaving the library's sentinel contract untouched.
2. A consumer requests machine-readable error codes or programmatic classification.
3. Retry/recovery logic is added (currently implausible for an audit recorder).
4. Either project hits v1.0 and the API stabilizes.

**If adopted, scope it to the CLI only** (`cmd/auditlog` calls `errorfamily.HandleError(err)` at the exit boundary). The library keeps its current sentinels — zero breaking change, zero depguard churn in the hot path, and the CLI gets structured exit codes. This is the 80/20 win.
