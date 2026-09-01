# Frame packet: 02-see-everything

## Project inputs

- Project: /home/lars/projects/samber-do-auditlog/website/video/videos/do-auditlog-demo
- Design tokens: /home/lars/projects/samber-do-auditlog/website/video/videos/do-auditlog-demo/frame.md
- RULES_DIR: /home/lars/.agents/skills/hyperframes-animation/rules

## Assigned storyboard block

## Frame 2 — See everything

- scene: The claim builds across full-screen beats — "every registration" / "every invocation" / "every shutdown" — then resolves to the thesis line "See every dependency. Every event."
- duration: 5s
- poster: 4s
- transition_in: cut
- status: outline
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

## Selected blueprint: kinetic-type-beats

# kinetic-type-beats — Kinetic-Type Beats

**intent**: A flat, centered, bold-type shot where the motion IS the word/phrase changing — the line either swaps tokens in place by hard cut, or builds a statement across full-screen beats (each with its own move) that lands a spring-pop payoff.

**roles served**

- Hook (from `hook-kinetic-type-flash`): when one stationary line lands a punchy rhetorical question or "you keep doing X" callout and the in-place token swap itself is the joke.
- Hook (from `hook-kinetic-type-escalation`): when ONE statement should escalate across distinct full-screen beats (each a different move) and punctuate on a spring-pop payoff element — a rising-intensity / "transform X into Y" opener.
- Hook (from `kinetic-type-to-logo-reveal`): when rapid centered word beats are the warm-up for a typography-to-brand arc — the swaps resolve into a logo reveal (pop-in whole, or 3D parts assemble and flatten into the flat mark) that hands off to a value card / browser mockup sliding in.
- Hook (from `centered-beat-triptych`): when the open is three center-stage beats on a constant field, each element ALONE on screen — and a beat's payload may be non-text (a logo-lockup rotation-snap, a CTA button spring-pop with a one-shot glow-ring pulse, a benchmark chart that builds and holds); the last beat holds to the end.
- Problem (from `problem-kinetic-type-beats`): when the script is 3–5 short pain statements (or a "what-if?" framing) that should each land alone, on a bare canvas, before the next replaces it — no product visible yet.
- Problem (from `centered-phrase-relay-question-hook`): when the pain is an ordered chain of question/hook phrases that scale-pop through center relay-style (each exits as the next arrives) and land a specially-styled climax word — OR resolve the question on a `[product surface]` entering as an element move, never a camera zoom.
- Product_Intro (from `product-intro-kinetic-type-namedrop`): when the hook IS the words — hard-cut through "Introducing…" / tagline / value beats and resolve on the brand name or logo.
- Product_Intro (from `fixed-line-word-swap`): when a fixed headline holds and ONLY one word-slot changes — cursor-deleted-and-retyped once (optionally by a labeled collaborative cursor) or rapid-cycled through a `[role]` list — then hands off to the product/brand payoff; the purest sub-shape A.
- Product_Intro (from `flat-field-kinetic-word-run`): when a sentence builds word-by-word on a flat brand-color field and each `[hero word]` earns a bespoke one-shot effect payoff (letter-scramble, chromatic glitch, confetti burst, emoji morph) before a punch-word finale.
- Product_Intro (from `anchored-wordmark-transform`): when the anchored type itself mutates — an "Introducing" / predecessor beat builds or swaps into the `[wordmark]` in place, which then TRANSFORMS (a UI collage rushes outward from behind it, a word morphs into a pulsing icon, the title zoom-blurs away) into a short payoff beat.
- Benefits (from `benefits-kinetic-type`): when "what you get" is a rapid-fire staccato montage — 8–12 short value phrases, each flashing and clearing before the next at high tempo.
- Benefits (from `flat-void-statement-relay`): when the value reads as a SLOW statement relay — 2–4 full statements on a flat void, each built by its own engine (typewriter, oversized element-scroll, outline-echo stack, wave-mapped pattern) and held ~1.5s+ before the hard cut; the low-tempo sibling of the staccato montage.
- CTA (from `cta-kinetic-type`): when the sign-off is a punchy closing line (or a short stack of value lines) that snaps/fades in beat-by-beat and lands on the brand lockup or URL — no spatial set, no clicked button.
- CTA (from `kinetic-beat-chain-to-logo`): when the sign-off chains 3–5 message beats and each beat carries a DIFFERENT kinetic gag (marquee scroll-through, flash-swap word list, brief 3D letter extrude, spring-bounce prop, one interleaved mock-UI beat) before the logo/URL forms — optionally out of a preceding glow pulse — and holds.
- Brand_Outro (from `brand-outro-kinetic-type-resolve`): when the close is a rapid center-channel barrage of single-word verbs asserting breadth, resolving on the brand's one defining word (motion-is-the-message, no logo lockup).
- Brand_Outro (from `centered-beat-relay-to-url`): when the close is a short relay of full-frame beats — fade/scale swaps, a spring shrink-to-0, an `[icon]` bounce, a gradient-swept title card — terminating in a centered `[URL / domain]` end card held for the longest stretch of the shot (~40–75% of runtime); an optional `[product UI]` prologue scales down and fades to the canvas first.

**duration**: 3.0–12.9s (Benefits staccato fastest ~3.5–4s at 8–12 sub-0.5s beats, statement-relay Benefits up to ~8.3s; Product_Intro fixed-line as short as ~3.0s; Problem 4.1–12s; CTA spans 3.6–12.9s with beat count; Brand_Outro ~3.6s as a verb barrage, up to ~12.6s when the terminal URL hold carries 40–75% of the runtime)

**shot structure** (flat, fixed center anchor; bold sans-serif text on a solid `[bg color]`; type/tokens are the default subject, though a beat's payload may be ONE non-text center-stage element — a logo lockup, a CTA button, a chart — obeying the same arrive-hold-clear law; camera locked unless a modifier is noted; two folded sub-shapes — **(A) fixed-line token swap** and **(B) multi-beat statement build**)

- **Scene 1 (0.0–~1.0s) — first beat lands.** Solid `[bg color]` field. Bold `[type color]` text arrives dead-center via ONE entrance: type-on character-by-character with a trailing blinking caret, OR a hard-cut FLASH-in (no fade/slide), OR a per-word staggered fade/blur, OR an oversized word that smoothly SCALES DOWN to a small centered word. An optional `[accent color]` move plays on the key word(s): a left→right drawn underline / strike-through, a small particle/dot burst from behind the text, or a `[accent color]` selection-box framing the word.
  - _Variant — Hook (flash)_: just the fixed `[hook line]` (or its first word) parks at center; no escalation move.
  - _Variant — Hook (escalation)_: `[beat 1 text]` arrives big and scale-downs to centered, OR sits over a glowing `[motif]` with a slow camera push-in (see camera modifier); ends on a hard cut.
  - _Variant — Hook (logo reveal)_: centered bold words swap in with quick spring-scale pops on a flat/gradient field while flat `[accent]` circles/dots drift idly; a beat may hard-cut to a contrast bg and enter with an RGB-split glitch stretch that snaps sharp.
  - _Variant — Hook (triptych)_: beat 1 may be non-text — a `[logo mark]` rotates in 3D and snaps flat beside a `[version tag]`, or a statement resolves via a horizontal stretch/slice glitch on a subtle `[grid card]` — holds, then clears (scale-down + fade, or hard cut).
  - _Variant — Problem_: centered `[pain line 1]` reveals in chunks across one or two lines with its `[accent]` underline / particle burst.
  - _Variant — Problem (relay)_: `[hook phrase 1]` scale-pops into center with a quick spring on a flat solid OR drifting-gradient field — optionally the background itself morphs open first (a rounded `[accent shape]` expands into the full-bleed gradient); as the phrase holds, its word-spacing spreads slightly.
  - _Variant — Product_Intro_: bold `[hook word, e.g. "Introducing"]` enters with a typographic accent (split-and-slide apart, drawn underline, or `[accent]` selection-box).
  - _Variant — Product_Intro (fixed-line)_: the full fixed headline `[fixed phrase] [swap-slot]` parks centered (a faint `[plexus / ambient pattern]` may drift behind); no escalation move — the slot is the show.
  - _Variant — Product_Intro (word-run)_: a field-claiming open — horizontal `[brand color]` stripe wipes reveal the `[logo lockup]` then clear, or a giant blob expands from center repainting the frame in the brand color — before the sentence starts building.
  - _Variant — Product_Intro (wordmark transform)_: "Introducing" fades in over ambient sine-wave lines that undulate then snap taut, OR the `[old version wordmark]` holds and swaps away, OR oversized scattered `[gradient]` letters bounce-assemble into the `[name]` while the whole word scales down to center.
  - _Variant — Brand_Outro_: optional single-frame flash of `[product UI / hero asset]` precedes the verb channel, then `[verb 1]` hard-cuts in centered.
  - _Variant — Brand_Outro (relay-to-URL)_: optional prologue — the `[product UI window]` scrolls its content, then scales down and fades out to the flat canvas; the first text beat fades/scales in centered.

- **Scene 2..N — beats replace each other in place (the engine).** The center anchor advances one beat at a time; nothing from the prior beat lingers. Choose the swap mechanism by sub-shape:
  - **Sub-shape A (fixed-line token swap)**: the line stays fixed and only the variable slot changes by an instant hard CUT (no roll/scroll/blur) — `[token A]` → `[token B]` → `[token C]` — OR the final word(s) backspace out and a new word retypes (`[word A]` → `[word B]`). The rest of the line holds. The cycle may run a rapid `[role word]` list at the fixed slot, and the delete-retype may be performed by a labeled collaborative cursor; a faint `[plexus / ambient pattern]` may keep drifting behind the fixed line.
  - **Sub-shape B (multi-beat statement build)**: each full-screen beat hard-cuts to a NEW background/line, and each gets its own distinct entrance/exit MOVE — springy scale-in/scale-out overshoot, 3D letter-tumble (glyphs scatter into a rotating depth cloud, then reassemble into the next phrase), motion-blur fly-in that resolves sharp at center, prior text accelerates/zooms past the camera while fading, letter-spacing collapse, or a bottom-up masked slide. Background may hard-flip `[bg A]`↔`[bg B]` on selected beats with `[type color]` inverting to stay legible.
  - _Variant — Hook (escalation)_: beat 2 `[beat 2 text]` (more emphatic) snaps in; beat 3 `[beat 3 text]` (climax) holds, then a transition-out move on the type itself — a Z-dolly forward THROUGH an oversized glyph, OR a per-word karaoke highlight sweep lighting words left→right.
  - _Variant — Hook (triptych)_: a mid beat may be non-text — a `[CTA button]` spring-pops with overshoot, fires a one-shot blurry glow-ring pulse outward, and settles smaller — the element alone on screen, then cleared like any other beat.
  - _Variant — Problem_: each `[pain line k]` enters by chunk-reveal or motion-blur fly-in as the prior blurs/zooms off; an optional `[accent color]` interstitial word ("[what-if hook]") scales up from center, holds, then zooms past the camera and fades.
  - _Variant — Problem (relay)_: each `[phrase]` scale-pops into center while the prior shrinks and split-slides off toward BOTH left/right edges (clipping off-screen) with fade; an optional emphasis beat lands on a hard-cut contrast bg — a single `[word]` letter-tracking-tightens from wide spacing while scaling up as four thin `[accent]` arrows shoot in diagonally from the corners, converging on it; a left-aligned line-by-line value build may interleave.
  - _Variant — Product_Intro_: each `[tagline phrase]` is a hard-cut/push-through inverted-text beat with its own one-shot accent (strike-through, slider/toggle shapes sliding in, or a bg-invert cycle white→`[accent]`→black flipping fg/bg).
  - _Variant — Product_Intro (word-run)_: the `[sentence]` builds word-by-word with snappy pops (lines re-center as they add; an underline may draw beneath key words), then one beat per `[hero word]` — each lands large and performs its own one-shot effect: a letter-scramble resolve (with thin divider ticks), a chromatic-glitch jitter (offset color copies snapping back clean), a spring bounce + confetti burst that erupts up and drifts down, or a letter-slot swapped for a springing `[emoji / mark]` that morphs; the finale may run an alternating huge/small word scale chain.
  - _Variant — Product_Intro (wordmark transform)_: the `[wordmark]` completes in place — staggered part-by-part pop (`[part 1]` then `[part 2]`), an in-place swap replacing "Introducing", or a `[second phrase]` appending — with a gradient hue-sweep across the type that settles to a solid color snap.
  - _Variant — Benefits_: high tempo (~0.4s/beat) — each `[benefit phrase]` pops via springy scale-in/out or 3D letter-tumble; multiple bg light↔dark flips across the run with text-color invert.
  - _Variant — Benefits (statement relay)_: low tempo — each `[statement]` builds by its own engine and HOLDS ~1.5s+ before the hard cut: line 2 types char-by-char under a static line 1; an oversized `[phrase]` element-scrolls right→left through the frame (a moving window onto a wider line, a gradient sweeping the letters); a solid `[word]` holds while stacked outline-only echo copies cycle vertically behind it; a multi-line block builds fast as small `[accent shapes]` fly in from the edges then drift outward and thin (text may form as a masked grey fill, then snap solid).
  - _Variant — CTA_: each `[value line]` → `[value line]` → `[CTA verb line]` clears by hard cut / zoom-blur cut through near-black / fade-out, then the next pops/fades/slides in. Optional `[accent motif]` draws on behind (rising line-graph trim-path, thin wireframe guides, gutter geometry tiles).
  - _Variant — CTA (beat-chain)_: individual beats carry their own gag — a line enters right and marquee-scrolls continuously left across the frame (exiting); a `[use-case word]` list flash-swaps in place; a beat's letters briefly extrude into simple 3D and flatten back; a `[glyph + prop]` group spring-bounces in then slides off; ONE mock `[compose-window / product UI]` beat may interleave without breaking the chain.
  - _Variant — Brand_Outro_: a centered single `[verb / keyword]` HARD-CUTS to the next at a steady ~0.2s cadence (no fade/scale) over a continuous moving field (see camera modifier).
  - _Variant — Brand_Outro (relay-to-URL)_: 2–3 full-frame beats swap wholesale at a relaxed cadence — each fades/scales in and out, or scales up slightly then spring-shrinks to 0%, or an `[icon]` bounce-pops in from 0% and shrinks back out, or a `[title card]` holds with a continuous in-text horizontal gradient sweep before a HARD CUT to the bare canvas.

- **Scene N (final beat → end) — resolve and HOLD.** The last beat lands and holds to the end (settle only, no further scale-out). Resolution diverges by role:
  - _Variant — Hook (flash)_: last token swap lands and holds; optional tiny punctuation/emphasis snap (`?` → `?!`, or fill snaps to `[accent color]`).
  - _Variant — Hook (escalation)_: resolve on `[payoff bg]` — a `[payoff element]` (colored square / heart-eyes reaction emoji) SPRING-POPS in center; small `[accent motes]` drift outward; subtle settle.
  - _Variant — Hook (logo reveal)_: the word beats resolve on the brand — the `[logo]` pops in whole, or floating 3D `[shapes]` assemble and FLATTEN into the flat 2D mark as the `[wordmark]` slides in beside it; then a `[browser mockup / value card]` slides/scales in on a fresh bg and holds (a bottom caption may build).
  - _Variant — Hook (triptych)_: the final beat may be non-text — a `[benchmark chart]` fades in its framework and grows bars from zero width in a top-down stagger (the `[hero row]` bold/highlighted), then holds static for the back half of the shot; or a closing statement glitch-reveals and holds.
  - _Variant — Problem_: final `[pain line]` reveals (left→right swipe with leading-edge blur, OR letters explode radially then the resolving line fades up); holds the pain on screen.
  - _Variant — Problem (relay)_: the climax `[word]` scales in with special treatment (gradient fill, slight ~-8° rotation) and holds — OR the question resolves on a `[product surface]` as an ELEMENT move: a `[pill / search bar]` slides in from the right and keeps traveling leftward while its text progressively reveals (may end mid-slide, phrase cropped at the frame edge), or the `[page canvas]` scales down while `[app chrome + side panels]` slide in and frame it.
  - _Variant — Product_Intro_: resolve on the brand — `[logo mark]` / `[wordmark]` pops in centered (optional sting: liquid/ink splash, blob backing), OR the final value word holds inside an expanding-iris `[accent]` circle that scales to fill frame and hard-cuts the closing word through it.
  - _Variant — Product_Intro (wordmark transform)_: with the completed `[wordmark]` anchored dead-center, a dense `[UI-screenshot collage]` rushes in and expands outward from behind the text toward the frame edges with parallax (fast pull-back feel), then clears quickly to a clean `[wordmark]` end card; OR one `[word]` morphs into a pulsing `[icon]` completing an icon+text lockup before the field dissolves to its inverse; OR the title rapidly scales up and zoom-blurs away as the next context fades in.
  - _Variant — Benefits_: the last `[benefit phrase]` arrives (optionally on the inverted bg) and SETTLES — does not scale/tumble back out.
  - _Variant — CTA_: land on the lockup — `[logo mark]` SCALES UP small→full and holds, OR a `[logo]`/`[url]` builds segment-by-segment beside its icon. End-card holds dead static.
  - _Variant — CTA (glow-preceded formation)_: the prior letters scatter/clear, a soft `[accent]` glow pulses on the empty field, and the `[logo mark]` FORMS out of the glow with the `[url]` wordmark below; holds to the final frame.
  - _Variant — Brand_Outro_: hard cut to the `[resolve word / brand keyword]` (longest, still centered); HOLDS ~0.5s while the background field keeps moving.
  - _Variant — Brand_Outro (relay-to-URL)_: the centered `[URL / domain]` (+ optional CTA line above) fades/scales in and holds — the LONGEST beat of the shot, ~40–75% of the runtime — optionally fading at the very tail.

**motion vocabulary**: hard-cut / flash word swaps; in-place token cycle (instant cut, no roll/scroll/blur); type-on with trailing blinking caret; backspace-and-retype; per-word staggered fade/blur reveal; big→small scale-down; springy scale-in/scale-out overshoot; 3D letter-tumble scatter-and-reassemble; motion-blur fly-in / blur-off; prior text zoom-through-camera; letter-spacing collapse; bottom-up masked slide; drawn-on `[accent]` underline / strike-through; particle/dot burst from text; `[accent]` selection-box frame; bg-invert hard-flip with text-color invert; karaoke per-word highlight sweep; radial letter-explode; expanding-iris circle wipe-to-next; final spring-pop payoff element (square / emoji / logo mark); drifting `[accent]` motes / ambient shapes; segment-by-segment URL/wordmark build; final-token punctuation snap; settle-and-hold; scale-pop phrase relay (prior shrinks + split-slides off both edges with clip-fade); letter-tracking tighten-from-wide while scaling; corner arrows converging on a word; gradient-fill / hue-sweep across type with settle-to-solid snap; in-text traveling gradient sweep; background shape morph-open into a full-bleed field; RGB-split / chromatic-glitch jitter; horizontal stretch/slice glitch reveal; letter-scramble resolve with divider ticks; confetti burst up-and-drift; letter-slot emoji/mark swap + morph; alternating huge/small word scale chain; color-stripe wipes / blob expand frame-repaint; oversized phrase element-scroll (moving window); right→left marquee scroll-through; stacked outline-echo copies cycling behind a solid word; full-frame repeating-word pattern on a rolling 3D wave; accent shapes fly-in then drift-out-and-thin; masked grey fill snapping solid; brief 3D letter extrude-then-flatten; spring-bounce glyph+prop drop-in; glow-pulse-preceded logo formation; 3D shapes assemble-and-flatten into the mark; logo-lockup 3D rotation-snap; one-shot glow-ring pulse; chart bars growing in a top-down stagger; labeled collaborative cursor delete-and-retype; in-place role-word cycle; ambient plexus/pattern drift; spring shrink-to-0 exit / bounce-in from 0%; scattered-letter bounce-assembly with baseline settle; staggered wordmark part pop; phrase append; word→icon morph with continuous pulse; UI-collage rush-out with parallax from behind anchored type; zoom-blur title exit; long-held URL end card.

**rule mapping**

- hard-cut / flash word swaps, in-place token cycle, whole-line state swaps at time thresholds → `discrete-text-sequence`
- type-on character-by-character + blinking trailing caret → `discrete-text-sequence` (text/typing state progression) + `context-sensitive-cursor` (caret blink/color-switch)
- backspace-and-retype final word(s) → `discrete-text-sequence` (typos/holds/backspace is explicitly in-scope)
- one short distinct phrase per beat / script-driven phrase windows / word-by-word tagline assembly → `dynamic-content-sequencing`
- percussive per-beat phrase entrances on a shared beat array (distinct entrance per phrase, steady cadence) → `kinetic-beat-slam` (best fit for the multi-beat statement-build engine and the ~0.2s Brand_Outro verb march)
- per-word staggered fade/blur reveal → `kinetic-beat-slam` (per-phrase/per-word distinct entrances); the soft-focus blur component → `depth-of-field-blur` (selective-focus blur on the off-focus words)
- big→small scale-down on a word; springy scale-in/scale-out overshoot → `spring-pop-entrance` (spring pop/settle) backed by `gsap-effects` for the plain scale tween
- 3D letter-tumble scatter-into-depth-cloud then reassemble → `depth-scatter-assemble` (glyphs scatter into a 3D depth cloud and reassemble into the next phrase; combine w/ `3d-text-depth-layers` for the extruded read, or `hacker-flip-3d` for an in-place per-char flip flavor)
- karaoke per-word highlight sweep synced across words → `asr-keyword-glow` (keyword glow+scale on a synced rail) OR `css-marker-patterns` (highlight sweep) — choose ASR-driven vs. static-timeline sweep
- drawn-on `[accent]` underline / strike-through / loop / scribble under key word → `css-marker-patterns` (highlight sweep / circle / burst / scribble / sketchout)
- particle/dot burst from behind text → `css-marker-patterns` (burst) backed by `gsap-effects`
- `[accent]` selection-box frame around a word → `css-marker-patterns` (circle/box marker) + `gsap-effects`
- bg-invert hard-flip (light↔dark / white→accent→black) with text-color invert → `discrete-text-sequence` (whole-text/state swap covers the synchronized fg/bg state change)
- letter-spacing collapse; bottom-up masked slide → `gsap-effects` (tween letter-spacing / masked translate) + techniques: per-word kinetic typography / clip-path reveal
- expanding-iris circle wipe that morphs the current word into the next at the same center → `scale-swap-transition` (morph two elements at same center)
- final spring-pop payoff element (colored square / reaction emoji / logo mark) → `spring-pop-entrance` (or `physics-press-reaction` for a weightier pop)
- drifting `[accent]` motes / ambient shapes / soft drifting gradient field beneath the type → `sine-wave-loop` (idle drift loop)
- segment-by-segment URL / wordmark build beside its icon → `discrete-text-sequence` (segment-by-segment state reveal) or `dynamic-content-sequencing`
- final-token punctuation / emphasis snap (`?`→`?!`, fill→accent) → `discrete-text-sequence`
- settle-and-hold final frame → `spring-pop-entrance` (settle phase) / static hold (no rule needed)
- motion-blur fly-in / blur-off / zoom-through-camera streak on type → `motion-blur-streak` (directional velocity blur on a fast fly-in / zoom-through; the heavy motion-blur smear resolves sharp at center)
- radial letter-explode (glyphs explode outward radially then resolve) → `depth-scatter-assemble` (radial per-letter explode-and-resolve is in scope alongside the depth-cloud scatter)
- 3D letter-tumble depth-cloud scatter-and-reassemble → `depth-scatter-assemble` (free tumbling depth-cloud that flies out and snaps back into the next phrase)
- scale-pop phrase relay → `spring-pop-entrance` (the arriving phrase) + `gsap-effects` (the prior phrase's shrink + split-slide clear toward both edges)
- letter-tracking tighten-from-wide while scaling → `gsap-effects` (letter-spacing tween — the inverse of the letter-spacing collapse mapped above)
- corner arrows converging on a word → `css-marker-patterns` (burst geometry with inverted travel — lines converge instead of radiate) + `gsap-effects`
- gradient-fill climax word / hue-sweep across type / in-text traveling gradient sweep → `gradient-text-sweep` (gradient tweened THROUGH letterforms — position/hue sweep with settle-to-solid snap, seek-safe)
- background shape morph-open into a full-bleed field → `card-morph-anchor` (uniform scale + borderRadius paint tween, then the field takes over)
- RGB-split / chromatic-glitch jitter; horizontal stretch/slice glitch reveal → `chromatic-glitch` (deterministic offset color-copy layers, jitter + snap-clean; covers the stretch/slice glitch reveal)
- letter-scramble resolve with divider ticks → `hacker-flip-3d` (the deterministic glyph-substitution decode, minus the 3D rotation)
- 3D shapes assemble-and-flatten into the mark; scattered-letter bounce-assembly → `depth-scatter-assemble` (scatter-to-clean-layout settle) + `spring-pop-entrance` (the bounce settle)
- logo-lockup 3D rotation-snap → `orbit-3d-entry` (the 3D flip-in entry, skipping the orbit phase)
- one-shot glow-ring pulse; glow-pulse-preceded logo formation → `ambient-glow-bloom` (single-pass bloom-and-fade) + `spring-pop-entrance` (the mark forming out of it)
- chart framework fade-in + bars growing from zero width top-down; radial gauge arc-draw + count-up → `stat-bars-and-fills` (+ `counting-dynamic-scale` for the ticking value)
- labeled collaborative cursor delete-and-retype; in-place role-word cycle → `discrete-text-sequence` + `context-sensitive-cursor` (the labeled-pointer look itself is oversized-cursor doctrine, not a rule)
- ambient plexus/pattern drift; accent shapes drift-out-and-thin → `sine-wave-loop` (finite drift) after a `spring-pop-entrance` arrival
- letter-slot emoji/mark swap + morph; word→icon morph with continuous pulse → `scale-swap-transition` (same-center morph) + `svg-icon-enrichment` (the icon's internal pulse)
- oversized phrase element-scroll; right→left marquee scroll-through → `gsap-effects` (linear translate of an oversized element through a static frame)
- stacked outline-echo copies cycling behind a solid word → `3d-text-depth-layers` (the offset echo stack) + `vertical-spring-ticker` (the vertical cycle)
- brief 3D letter extrude-then-flatten → `3d-text-depth-layers` (build the extrusion offsets, then collapse them)
- full-frame repeating-word pattern on a rolling 3D wave → flagged special — a 3D wave-mapped text field is out of rule scope; `sine-wave-loop` only drives the undulation oscillator
- alternating huge/small word scale chain → `kinetic-beat-slam` (distinct per-beat entrances on the shared beat array)
- color-stripe wipes → `gsap-effects` (masked translate tweens); blob expand frame-repaint → `card-morph-anchor`
- UI-collage rush-out with parallax from behind anchored type → `center-outward-expansion` (clustered-at-center → outward to final positions; vary per-tile rates/scales for the parallax read)
- spring shrink-to-0 exit / bounce-in from 0% → `spring-pop-entrance` (in) / `gsap-effects` `back.in` shrink (out)
- product-surface resolve (pill slide with progressive text reveal; canvas scale-down as chrome frames in) → `nudge-curve` (the slide that reveals during travel) + `gsap-effects` (coordinated scale + panel slides)
- zoom-blur title exit as an in-shot beat handoff → `motion-blur-streak`; as a scene-out into the next scene it belongs to the transition layer
- staggered wordmark part pop / phrase append → `spring-pop-entrance` + `dynamic-content-sequencing`

**camera modifier** (optional, layered over the flat shot; most variants are camera-locked)

- Slow continuous global zoom-in / uniform push-in running underneath the whole sequence (Problem, Brand_Outro) → `multi-phase-camera` (push phase) — gives parallax between the fixed type and a moving background field.
- Camera dolly/zoom forward THROUGH an oversized glyph along Z as a beat transition-out (Hook escalation, Product_Intro push-through) → `coordinate-target-zoom` (target the glyph center) or `multi-phase-camera` (push).
- Slow push-in on Scene 1 over a glowing `[motif]` (Hook escalation) → `multi-phase-camera` (push) or `coordinate-target-zoom`.
- Slow continuous card/scene scale-up running UNDER hard-cut beats (Hook triptych) — a push-in feel rendered as element scale on the scene group, never a real dolly → `multi-phase-camera` (push phase) or a plain `gsap-effects` scale tween.
- Note: the in-place token swap (sub-shape A) and most Benefits/Hook-flash/CTA variants are fully camera-static — the swap is the only motion.

## Selected motion rule: discrete-text-sequence

---
name: discrete-text-sequence
description: Replace entire text states at frame thresholds for non-linear typing effects — typos, bulk additions, pauses, backspaces, simulated thinking.
metadata:
  tags: text, typing, discrete, threshold, non-linear, sequence
---

# Discrete Text Sequence

Instead of character-by-character typewriter, replace entire string states at time thresholds — enabling non-linear effects (typos, backspaces, bulk paste, "thinking" gaps) that smooth per-char typing can't achieve. If your effect is "type each character, no edits", this rule is overkill — use the smooth-slice variation below.

## How It Works

The typing is authored as a sparse array of `{ t, text }` states; on every `onUpdate` a **reverse search** finds the latest entry whose `t` has passed and renders its text. Display jumps between states with no animation between them — the realism comes from the schedule shape: fast keystroke clusters (0.06–0.20s apart), pauses at word breaks (0.3–0.6s), a typo, backspaces peeling back to the fork, then a bulk paste replacing many chars in one entry. A block cursor blinks via a deterministic sin square wave on the same timeline.

## Recipe

```html
<!-- inside a standard scene clip (hyperframes-core) -->
<div class="terminal">
  <div class="prompt">$</div>
  <div class="text-wrap">
    <span class="text" id="text"></span><span class="cursor" id="cursor">_</span>
  </div>
</div>
```

```css
.terminal {
  font-family: {monoFont}; /* monospace required — proportional jitters even in a fixed box */
  display: flex;
  align-items: baseline;
  font-size: TERMINAL_FONT_SIZE;
}
.text-wrap {
  display: inline-flex;
  align-items: baseline;
  min-width: TEXT_WRAP_MIN_WIDTH; /* ≥ widest state — stops right-edge jitter */
  white-space: nowrap;
}
.cursor {
  display: inline-block; /* inline ignores width */
  width: CURSOR_WIDTH;
}
```

```js
// Each entry shows from its t until the NEXT entry's t.
// Shape: keystrokes → typo → backspace to the fork → bulk paste → completion mark.
const SEQUENCE = [
  { t: 0.0, text: "" },
  { t: T_K1, text: "{p1}" }, // first keystrokes (~3-5 chars, 0.1-0.2s apart)
  { t: T_K2, text: "{p1 + ' ' + p2_typo}" }, // continuation containing a typo
  { t: T_BS, text: "{p1 + ' ' + p2_partial}" }, // backspace(s) — peel back to the fork
  { t: T_BULK, text: "{fullCorrectedText}" }, // bulk paste — many chars in one jump
  { t: T_DONE, text: "{fullCorrectedText + ' ✓'}" }, // completion marker
];

// Reverse-search for the latest entry whose t has passed
function textAt(time) {
  for (let i = SEQUENCE.length - 1; i >= 0; i--) {
    if (time >= SEQUENCE[i].t) return SEQUENCE[i].text;
  }
  return "";
}

const textEl = document.getElementById("text");
const cursorEl = document.getElementById("cursor");

const driver = { t: 0 };
tl.to(
  driver,
  {
    t: TOTAL_DURATION,
    duration: TOTAL_DURATION,
    ease: "none",
    onUpdate: () => {
      textEl.textContent = textAt(driver.t);
    },
  },
  0,
);

// Cursor blink — deterministic sin square wave, never a CSS animation
const blink = { p: 0 };
tl.to(
  blink,
  {
    p: Math.PI * 2 * BLINK_CYCLES,
    duration: TOTAL_DURATION,
    ease: "none",
    onUpdate: () => {
      cursorEl.style.opacity = Math.sin(blink.p) > 0 ? "1" : "0";
    },
  },
  0,
);
```

## Variations

- **Smooth character slice** (continuous typewriter — no pauses, no edits): faster to author but uniformly "machine-typed", missing the human realism:

```js
const fullText = "{fullPhrase}";
const len = { v: 0 };
tl.to(
  len,
  {
    v: fullText.length,
    duration: TYPE_DUR,
    ease: "power1.inOut",
    onUpdate: () => {
      textEl.textContent = fullText.substring(0, Math.floor(len.v));
    },
  },
  0,
);
```

- **Thinking pause** — hold one state for `THINK_HOLD_DUR` (0.8–2.0s; under 0.5s reads as a stutter, not thought) simply by leaving a gap before the next entry's `t`.
- **State pulse on completion** — when the final state lands, `tl.to(".text", { scale: 1.03–1.08, duration: 0.15–0.3, yoyo: true, repeat: 1 }, T_DONE)`.
- **Per-state color shift** — in `onUpdate`, branch on `driver.t` vs the milestones: success color after `T_DONE`, dim mid-edit, normal while typing.

## Values

| token               | range                                        | notes                                                                  |
| ------------------- | -------------------------------------------- | ---------------------------------------------------------------------- |
| TERMINAL_FONT_SIZE  | 48–96px                                      | full-bleed comps; smaller for terminal-style detail                    |
| TEXT_WRAP_MIN_WIDTH | ≥ widest state                               | measure with a hidden probe after `document.fonts.ready` if unsure     |
| milestone `t`s      | keystrokes 0.06–0.20s apart; pauses 0.3–0.6s | monotonically increasing; `T_DONE ≤ TOTAL_DURATION − ~1s` climax dwell |
| TYPE_DUR (smooth)   | `chars × 0.06–0.12s`                         | fast → relaxed                                                         |
| BLINK_CYCLES        | one cycle per 0.5–0.8s                       | `TOTAL_DURATION / 0.8 ≤ BLINK_CYCLES ≤ TOTAL_DURATION / 0.5`           |
| CURSOR_WIDTH        | ~0.3× font size                              | gap to text single-digit px so the cursor feels attached               |

## Critical Constraints

- **Reverse-search the array each frame** — O(n) with small n (≤30 typical); don't index by frame, the sequence is sparse.
- **`min-width` on the text wrap is mandatory** — without it the right edge jitters as state length changes.
- **Discrete jumps must be INSTANT** — any transition on the text turns the jump into a smear and kills the "typing" feel.
- **Cursor blink is sin/sequence-driven on the timeline**, `display: inline-block`, monospace font, `white-space: nowrap` (wrapping mid-state breaks the illusion; trailing spaces must survive).
- **Discrete vs smooth** — use discrete only for non-linear states (typos, pauses, bulk paste); plain typing takes the smooth-slice variation.

## See also

`context-sensitive-cursor` (same SEQUENCE pattern + segment-colored cursor) · `3d-text-depth-layers` (discrete text with layered depth) · `counting-dynamic-scale` (discrete label beside a smooth counter) · `press-release-spring` (post-completion press beat).

## Selected motion rule: spring-pop-entrance

---
name: spring-pop-entrance
description: The canonical entrance pop — an element (or staggered group) arrives by scaling 0 → 1 on a smooth long-tail settle (power3 default); bouncy overshoot is a rare, explicitly-playful exception. fromTo so it's correct at t=0 under seek.
metadata:
  tags: spring, entrance, pop, scale, power3, settle, stagger, reveal, arrival
---

# Spring-Pop Entrance

> **Smooth beats bouncy.** This entrance defaults to a smooth long-tail settle — `power3.out` (or `expo.out` for a faster front) — that decelerates cleanly into the resting size with **no overshoot**. Bouncy `back.out` is the **#1 instant turn-off** in agent-made videos and is almost never executed well; it is a rare, explicitly-playful exception (consumer / fun brand), never the default. When unsure, settle smoothly.

THE entrance primitive: an element (or staggered group) arrives by springing from nothing — `scale: 0 → 1`, optional small `y` rise — and settles without bouncing. This is **arrival**, not reaction: distinct from [press-release-spring.md](press-release-spring.md) (a click/press → release feedback chain on an element that already rests on screen). Many blueprints used to borrow that rule to fake an entrance; reach for this instead.

## How It Works

One `fromTo` carries the whole arrival: from `{ scale: 0, opacity: 0 }` (explicit, so t=0 is correct under seek) to `{ scale: 1, opacity: 1, ease: "power3.out" }`. For a **group**, the same `fromTo` runs per element at `i * STAGGER`, capped so the group reads as one arriving beat. The `scale` grow is load-bearing; the `y` rise is garnish — drop everything else and it must still read as a clean entrance. Let the ease produce the settle: never hand-key a `scale: 1.1` mid-state (it double-bounces against the curve).

## Recipe

```html
<!-- inside a standard scene clip (hyperframes-core) -->
<div class="pop-hero" id="hero">{heroLabel}</div>

<div class="pop-grid">
  <div class="pop-item">{itemA}</div>
  <div class="pop-item">{itemB}</div>
  <div class="pop-item">{itemC}</div>
</div>
```

```css
.pop-hero,
.pop-item {
  transform-origin: 50% 50%; /* in-place pop; move to the source point for the anchored variation */
  will-change: transform;
}
.pop-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: GRID_GAP;
  place-items: center;
}
```

```js
// Single hero pop — smooth long-tail settle, no overshoot.
tl.fromTo(
  "#hero",
  { scale: 0, opacity: 0 },
  { scale: 1, opacity: 1, duration: POP_DUR, ease: "power3.out" },
  ENTRY_AT,
);

// Staggered group pop — one arriving beat.
gsap.utils.toArray(".pop-item").forEach((el, i) => {
  tl.fromTo(
    el,
    { scale: 0, opacity: 0, y: Y_RISE },
    { scale: 1, opacity: 1, y: 0, duration: POP_DUR, ease: "power3.out" },
    GROUP_ENTRY_AT + i * STAGGER,
  );
});
```

## Variations

- **Calm settle** (premium / enterprise): `power3.out`, no rotation, `Y_RISE` 0–12px — a weighted, confident landing for a hero wordmark or product shot.
- **Firm settle** (everyday default): `power3.out` or `expo.out` for a punchier front, `Y_RISE` ~24px — cards, icons, callouts.
- **Exact-physics settle**: when the settle IS the shot, swap the ease for `springEase({ response: 0.4 })` (critically damped) from `../adapters/gsap-easing-and-stagger.md` → Spring Eases; take `duration` from the helper.
- **Origin-anchored pop**: a callout growing out of a specific point (marker, pointer tip) sets `transform-origin` to that point (e.g. `0% 100%`) so `scale: 0 → 1` reads as "emerging from the source", not "inflating in place".
- **Pop into a held slot**: land the pop and hold still — no idle loop baked into the entrance. If the held frame genuinely needs life, hand off to [sine-wave-loop.md](sine-wave-loop.md) for subtle jitter on a separate later tween; prefer revealing the next element on its VO cue.
- **Bouncy pop (RARE — explicitly-playful only)**: swap the ease for `back.out(OVERSHOOT)` and optionally settle a small `rotation: ROT_FROM → 0` so elements look hand-placed. Only for a deliberately playful register — never product / enterprise / serious tone:

```js
tl.fromTo(
  el,
  { scale: 0, opacity: 0, rotation: ROT_FROM },
  { scale: 1, opacity: 1, rotation: 0, duration: POP_DUR, ease: `back.out(${OVERSHOOT})` },
  GROUP_ENTRY_AT + i * STAGGER,
);
```

Even here keep `OVERSHOOT ≤ ~2` — past that it reads as cartoon wobble. Better still: the baked spring at `dampingFraction: 0.6–0.7` (same adapters doc) gives ~5–10% overshoot that reads physical where `back.out` reads cartoon.

## Values

| token      | range                                     | notes                                                            |
| ---------- | ----------------------------------------- | ---------------------------------------------------------------- |
| EASE       | `power3.out` default; `expo.out` punchier | `back.out(OVERSHOOT)` only in the playful variant                |
| POP_DUR    | 0.4–0.7s                                  | shorter = tight snap; hero must be visible by **t ≤ 0.5s**       |
| STAGGER    | 0.04–0.08s                                | `min(0.06, 0.5 / ITEM_COUNT)` — self-caps the window             |
| ITEM_COUNT | 3–9                                       | >9 makes the stagger vanish — switch to a wipe/sweep reveal      |
| Y_RISE     | 0–32px                                    | small; never large enough to read as a slide-up                  |
| ROT_FROM   | −10°–+10°                                 | playful variant only; alternate sign by index (`i % 2 ? 6 : -6`) |
| ENTRY_AT   | 0–0.4s                                    | a beat of quiet, but keep the subject landing by t ≤ 0.5s        |

## Critical Constraints

- Default ease `power3.out` (no overshoot); `back.out` only in the explicitly-playful variant, and there `OVERSHOOT ≤ ~2`.
- `ITEM_COUNT × STAGGER ≤ ~0.5s` — the group must land inside one beat.
- Entrances state the collapsed from-state in `fromTo` — never rely on a CSS-hidden start (it renders visible before the tween claims it under seek).
- `transform-origin: 50% 50%` for an in-place pop; the source point only for the anchored variation.
- This is a finite arrival — idle motion on a held element is a separate, later `sine-wave-loop` tween.

## See also

`center-outward-expansion` (pop while radiating to slots) · `press-release-spring` (the click-feedback counterpart) · `sine-wave-loop` (post-arrival jitter, sparingly).
