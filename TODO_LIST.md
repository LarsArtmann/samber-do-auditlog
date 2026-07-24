# TODO List

Short- and mid-term improvement tasks, verified against actual code state.
Completed items are in [CHANGELOG.md](CHANGELOG.md). Rejected proposals are in [ROADMAP.md](ROADMAP.md).
Last updated: 2026-07-24

---

## Publishing & Release

- [ ] **Make `go-sse` public on GitHub** — The repo at `github.com/LarsArtmann/go-sse` is private. The `replace github.com/larsartmann/go-sse => ../go-sse` directive in `go.mod` must stay until the repo is public and tagged. (Already tagged: v0.1.0, v0.2.0.) Once public, update `go.mod` to `require github.com/larsartmann/go-sse v0.2.0` and remove the replace directive.
- [ ] **Tag and release v0.7.0** once `go-sse` is public and all [Unreleased] items in CHANGELOG.md are verified.
