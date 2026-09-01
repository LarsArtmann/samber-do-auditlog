---
format: 1920x1080
duration: 25s
message: "See every dependency, every event — one line wires full DI lifecycle telemetry into samber/do v2."
arc: BAB (before → after tease → bridge → evidence → CTA)
audience: backend engineers using samber/do v2, platform teams, SREs
mode: autonomous
music: none

## Video direction

- **palette** (from frame.md, broadside × brand): ground charcoal `#1C1915`, panel `#1C1915` elevated via 1px hairlines `#4A4030`, primary text warm beige `#C4B89E`, single accent amber `#E8A838` (rationed: one amber element per frame), muted beige `#9C9078` for chrome/labels.
- **type**: display = brand sans (Space Grotesk roles), lowercase, weight 700, negative tracking; chrome/labels/code = IBM Plex Mono roles, uppercase 0.14em tracking; numbers tabular.
- **motion grammar**: long-tail `power3` eases everywhere, smooth over bouncy; this is a SILENT piece — reveals are paced to the implied reading cadence of the on-screen text (each phrase lands when a narrator would say it), never front-loaded; holds prefer stillness; only sanctioned aliveness is subtle jitter or bounded ambient on ONE held hero.
- **rhythm**: frames 1–2 are quick kinetic cuts; frame 3 is the busy centerpiece (typed command → cascading output); frame 4 ends on a long held read; frame 5 is the breather end card.
- **negative list**: no slideshow (front-load then freeze), no screensaver (independent floating elements), no purple/blue "AI" gradients, no browser chrome or fake cursors beyond the terminal caret, no decorative bokeh, no shadows (broadside flat plane), 0px rounded corners on panels.
- **captions**: skipped (silent project); bottom ~17% keep-out band still respected.
---

## Frame 1 — The black box

- scene: Massive lowercase type on charcoal — "your DI container is a black box." — the last two words swap in with a hard cut
- duration: 4s
- poster: 3s
- transition_in: cut
- status: animated
- blueprint: kinetic-type-beats (Reproduce)
- asset_candidates: none
- focal: type only
- src: compositions/frames/01-black-box.html

Adapt: keep the in-place token-swap signature; one swap, landing on the accent word.
Scene 1 (0.0–1.4s): centered hero, display type ~60% of canvas — the phrase "flying blind." slams in on a spring-pop entrance and settles → `spring-pop-entrance`.
Scene 2 (1.4–2.6s): hard-cut token swap — the fixed line "your DI container is a" seats above and the swap slot fires "black box." in amber, instant cut, no fade → `discrete-text-sequence`.
Scene 3 (2.6–4.0s): hold the read; a subtle low-amplitude positional jitter on "black box." only (sanctioned aliveness, no rule recipe needed).

Hook in outcome language; the swap landing on "black box." is the tension.

## Frame 2 — See everything

- scene: The claim builds across full-screen beats — "every registration" / "every invocation" / "every shutdown" — then resolves to the thesis line "See every dependency. Every event."
- duration: 5s
- poster: 4s
- transition_in: cut
- status: animated
- blueprint: kinetic-type-beats (Reproduce — build-across-beats variant)
- asset_candidates: none
- focal: type only
- src: compositions/frames/02-see-everything.html

Adapt: keep the escalation-beat signature; three escalation swaps resolve into one stacked thesis line instead of a spring-pop single.
Scene 1 (0.0–1.3s): full-screen beat 1 — "every registration" slams in centered, weight-700 display (percussive slam entrance, settles fast).
Scene 2 (1.3–2.4s): hard-cut swap → "every invocation" → `discrete-text-sequence`.
Scene 3 (2.4–3.4s): hard-cut swap → "every shutdown" → `discrete-text-sequence`.
Scene 4 (3.4–5.0s): the three beats compress upward into a stacked list at upper-third and the thesis line "See every dependency. Every event." spring-pops beneath, amber on both "every" words → `spring-pop-entrance`; hold the read.

Value claim lands by beat 2 per the reverse-iceberg rule. The three escalation
beats are the lifecycle events the viewer gains visibility into; the thesis
line is the brief's message verbatim.

## Frame 3 — One line, instant answers

- scene: Terminal window on charcoal — the command `DO_AUDITLOG_ENABLED=true go run ./example` types in, submit fires, the real Audit Summary cascades in line by line: 20 services, 145 events, 4 scopes
- duration: 7s
- poster: 5.5s
- transition_in: cut
- status: animated
- blueprint: prompt-type-submit-generate (Reproduce)
- asset_candidates: none
- focal: the terminal window
- src: compositions/frames/03-terminal.html

Reproduce: the full typed-ask → machine-answers loop; output lines are the real Audit Summary.
Scene 1 (0.0–2.0s): full-width terminal strip upper-76% on charcoal, 1px hairline border, mono chrome title row "ride-share-app — demo"; the command `DO_AUDITLOG_ENABLED=true go run ./example` types character-by-character behind a blinking amber caret → `discrete-text-sequence` (caret blink handled inline by the cursor rule it ships with).
Scene 2 (2.0–2.5s): submit beat — caret holds, one blink, a beat of silence (the machine thinking).
Scene 3 (2.5–5.8s): the real Audit Summary cascades line-by-line down the terminal, each line landing on its own beat — header, Container, then the stat rows — and each number counts up as its row lands (20 / 145 / 4) → `dynamic-content-sequencing` + `counting-dynamic-scale`; amber accent reserved for the three stat values.
Scene 4 (5.8–7.0s): hold; numbers settle at final size, static read → stillness.

Evidence A: real product behavior — the actual demo command and its actual
output (captured 2026-09-01). Keyboard drives, machine answers. The typed
command doubles as the "one line" bridge from the value claim.

## Frame 4 — The report

- scene: Open tight on the waveform strip of the real HTML report screenshot, then one continuous decelerating zoom-out reveals the full Services tab — stats cards, type badges, dependency columns
- duration: 5s
- poster: 3.5s
- transition_in: cut
- status: animated
- blueprint: zoom-out-workspace-reveal (Adapt)
- asset_candidates: what-does-it-look-like.jpg
- focal: what-does-it-look-like.jpg = cutout (hero, full-bleed)
- src: compositions/frames/04-report.html

Adapt: keep the ONE continuous decelerating zoom-out as the signature; the close-up micro-action becomes a bounded luminance pulse on the screenshot's waveform strip instead of live UI.
Scene 1 (0.0–1.4s): open TIGHT (~1.5x) on the waveform strip at the top of the report screenshot, full-bleed; the amber bars pulse softly in place → `sine-wave-loop` (low register).
Scene 2 (1.4–4.0s): one continuous decelerating zoom-out reveals the containing report — stats cards, services table, type badges resolve into frame → `viewport-change`; layered depth via the panel hairlines (3 depth layers: page glow, panel, table rows).
Scene 3 (4.0–5.0s): zoom locks at 1.0x; hold the full-report read, still → stillness.

Evidence B: the exported artifact itself — a real screenshot of the
self-contained HTML report (what-does-it-look-like.jpg). The zoom-out lands
the scale payoff: "that waveform was one corner of the whole report."

## Frame 5 — Install

- scene: Caret types `go get github.com/larsartmann/samber-do-auditlog`, collapses, pops the wordmark do-auditlog + "do-auditlog.lars.software"
- duration: 4s
- poster: 2.5s
- transition_in: cut
- status: animated
- blueprint: typewriter-reveal (Reproduce)
- asset_candidates: none
- focal: type only
- src: compositions/frames/05-install.html

Reproduce: typed CTA rail collapsing into the brand payoff.
Scene 1 (0.0–1.7s): centered; mono chrome label "add it to your container" seats upper-third; below it a `$` prompt and the command `go get github.com/larsartmann/samber-do-auditlog` types behind a blinking amber caret → `discrete-text-sequence` + `context-sensitive-cursor`.
Scene 2 (1.7–2.6s): the command collapses (fast backspace) and the wordmark "do-auditlog" spring-pops centered at display scale → `discrete-text-sequence` + `spring-pop-entrance`.
Scene 3 (2.6–4.0s): the sub-line "do-auditlog.lars.software" types in beneath the lockup in mono chrome → `discrete-text-sequence`; hold — this is the end card.

CTA: the one command to start, then brand lockup + site URL. Typed-command end
card shape.
