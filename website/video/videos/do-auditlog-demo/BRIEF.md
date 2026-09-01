---
workflow: product-launch-video
flow: automation
storyboard: no
message: "See every dependency, every event — one line wires full DI lifecycle telemetry into samber/do v2."
destination: website-embed
aspect: 1920x1080
language: en
length: 25s
angle: before-after
---

## Intent

A 25-second sales asset for the do-auditlog landing page (embedded above the
fold at do-auditlog.lars.software/#demo). Audience: backend engineers using
samber/do v2, platform teams needing lifecycle audit trails, and SREs wiring
DI metrics. Tone: confident, technical, terminal-centric — warm amber on dark
charcoal, matching the product's own "Container Telemetry" aesthetic. The
video must sell the outcome (no more flying blind in your DI container), not
tour features.

## Assets

- website/public/images/html-services.jpg — real HTML report screenshot (Services tab: waveform, stats, table); the primary evidence beat.
- website/public/images/html-graph.jpg — real dependency-graph screenshot (Sugiyama DAG); secondary evidence.
- Real terminal output of `DO_AUDITLOG_ENABLED=true go run ./example` — Audit Summary: 20 services, 145 events, 4 scopes; captured 2026-09-01 from the actual example app.

## Customizations

- Silent project: no narration, no BGM (muted-test governs — on-screen text carries the argument).
- Final beat shows the install command `go get github.com/larsartmann/samber-do-auditlog` and the site URL.

## Notes

- One narrative, shared with README/landing hero: "See every dependency, every event."
- Evidence beats must use real product output (terminal + report screenshot), never fabricated UI.
- Brand colors from capture tokens (warm amber #e8a838 on #14110d charcoal).
