# Frame packet: 05-install

## Project inputs

- Project: /home/lars/projects/samber-do-auditlog/website/video/videos/do-auditlog-demo
- Design tokens: /home/lars/projects/samber-do-auditlog/website/video/videos/do-auditlog-demo/frame.md
- RULES_DIR: /home/lars/.agents/skills/hyperframes-animation/rules

## Assigned storyboard block

## Frame 5 — Install

- scene: Caret types `go get github.com/larsartmann/samber-do-auditlog`, collapses, pops the wordmark do-auditlog + "do-auditlog.lars.software"
- duration: 4s
- poster: 2.5s
- transition_in: cut
- status: outline
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

## Selected blueprint: typewriter-reveal

# typewriter-reveal — Typewriter Reveal

**intent**: A live text caret types (and edits) a line as a human would, then either collapses it to a point and pops a brand payoff, or holds it under a persistent brand mark while a sub-line types/swaps into the final CTA — making "someone is typing this" the engine of the shot.

**roles served**

- Hook (from hook-typed-line-to-reveal): Type a relatable question/statement live, then COLLAPSE it and spring-pop the brand — a logo lockup OR a product-UI moment ("here's the everyday pain, now here's us").
- Brand_Outro (from brand-outro-persistent-mark-cta-rail): Hold the hero mark dead-center/top the whole shot while a sub-line beneath it swaps or types its way into the final CTA — landing the ask once the logo is already established.

**duration**: 3.6–7s (Brand_Outro 3.6–6.0s · Hook 5.5–7s)

**shot structure** (one consolidated template; `[slots]` are product-agnostic)

- Scene 1 (0.0–~2.0s): On a solid `[bg color]` field, a blinking text-input caret `|` sits at the line start, then `[primary line]` TYPES on character-by-character with the caret trailing.
  - _Variant — Hook_: nothing else is on screen; the typed `[hook line]` owns the frame. (Sub-variant: the line types inside UI chrome — a rounded `[input/pill]` — and the whole assembly continuously TRANSLATES leftward + scales slightly so the active caret stays pinned near frame-center while earlier words scroll off and clip past the left edge — a ticker push.)
  - _Variant — Brand_Outro_: a `[logo mark]` (+ optional `[wordmark]`) is already centered/upper and STAYS fully visible for the entire shot; an entry flourish plays on the mark itself (e.g. `[checkmark/icon]` strokes into the mark, or thin concentric rings ripple outward from it), and the typed `[tagline / product label]` is the SUB-LINE beneath the mark.

- Scene 2 (~2.0–4.5s): The typed line is MODIFIED in place — the active text is edited rather than re-shot.
  - _Variant — Hook_: final word(s) BACKSPACE out and a new word RETYPES (`[word A]` → `[word B]`), or the fill/caret snaps to `[accent color]` on the final word. Holds briefly.
  - _Variant — Brand_Outro_: the sub-line is REMOVED in place — a direct hard CUT/replace (NO backspace) or a moving mask-WIPE erases it — while the mark performs a small idle move (gentle rotate / sparkle reposition); the mark never leaves frame.

- Scene 3 — resolve:
  - _Variant — Hook (collapse, ~0.3–0.7s)_: caret vanishes; the whole text/assembly COLLAPSES to a point at center (horizontal X-collapse or scale-to-0 zoom-out) and disappears, leaving a clean `[bg]`. Then (remainder) a centered `[brand element]` SPRING-POPS in:
    - _logo-lockup sub-variant_: a `[mark/icon]` pops, then slides aside as a `[wordmark]` UNMASKS / slides out from behind it; both settle into a centered lockup.
    - _product-UI sub-variant_: a `[UI control]` (e.g. button) pops; a `[cursor]` sweeps in from a corner and homes onto it; on contact a ~150ms state-FLIP — base cross-fades to `[accent color]`, icon inverts, and a soft radial GLOW blooms outward and persists.
  - _Variant — Brand_Outro (~4.5s–end)_: the final `[CTA]` resolves in the sub-line slot — TYPED in with a caret and/or shown as a `[CTA in accent-color button]` beside plain text; an optional `[accent color]` GLOW ring / halo settles around the persistent mark. Holds to end. Final frame: `[logo mark]` + (glow ring) + `[CTA]`.

**motion vocabulary**: blinking text caret; character-by-character type-on; backspace-and-retype OR in-place hard-cut/mask-wipe text swap; optional leftward ticker push (assembly translates to keep caret centered); persistent centered hero mark (never vanishes) with entry flourish (icon stroke-draw, concentric ripple rings) and small idle move (rotate / sparkle); X-collapse / scale-to-0 zoom-out of the typed line; spring-pop brand reveal; wordmark unmask-slide into lockup; cursor sweep + UI state-flip + radial glow bloom; accent glow/halo ring settle; pill/button CTA reveal; hold.

**rule mapping** (per motion verb → `rules/<id>.md`)

- blinking text caret → `context-sensitive-cursor` (caret color-switch + blink)
- character-by-character type-on → `discrete-text-sequence` (typing/typos/holds/backspace); recipe `gsap-effects` (typewriter)
- backspace-and-retype → `discrete-text-sequence`
- in-place hard-cut / replace text swap → `discrete-text-sequence` (whole-text state swaps)
- mask-wipe erase of sub-line → `techniques.md` clip-path reveal (run in reverse)
- leftward ticker push (assembly translates to keep caret centered) → `camera-cursor-tracking` (viewport follows a moving caret)
- persistent hero mark hold → no motion rule needed (static anchor; intentional — it's the absence of motion)
- entry flourish: icon stroke-draw into mark → `svg-path-draw`
- entry flourish: concentric ripple rings from mark → `cursor-click-ripple` (ripple bloom)
- small idle mark move (rotate / sparkle reposition) → `sine-wave-loop` (idle)
- X-collapse / scale-to-0 zoom-out of typed line → `scale-swap-transition` (closest fit — it morphs/collapses elements at a shared center; approximation, since a standalone collapse-and-vanish without the paired same-center brand pop isn't its exact case)
- spring-pop brand reveal → `spring-pop-entrance` (alt `physics-press-reaction`)
- collapse-text → pop-brand as a same-center morph pair → `scale-swap-transition` (morph two elements at same center)
- wordmark unmask-slide into lockup → `techniques.md` clip-path reveal (unmask); slide via `spring-pop-entrance`
- cursor sweep onto UI control + press → `cursor-click-ripple` (cursor→target press + ripple)
- UI state-flip (base/icon invert on contact) → `hacker-flip-3d`
- radial glow bloom / accent glow-halo ring settle → `asr-keyword-glow` (accent glow); ring expansion via `center-outward-expansion`
- pill/button CTA reveal → `spring-pop-entrance` (alt `scale-swap-transition`)

**camera modifier**: none required — camera is static for both roles. The Hook ticker push is an ELEMENT translate (the typed assembly slides leftward to keep the caret centered), not a camera move → modeled by `camera-cursor-tracking` rather than a true camera rule.

## Selected motion rule: context-sensitive-cursor

---
name: context-sensitive-cursor
description: Cursor color and styling that adapt to the current text segment being typed — accent color on highlights, dim on placeholders, etc.
metadata:
  tags: cursor, color, context, typewriter, styling, segment
---

# Context-Sensitive Cursor

In a typewriter sequence, the cursor's color (and optionally height / blink behavior) matches the **active text segment** — brand accent while typing the brand name, dim on placeholders, success color on the completion mark. The eye lands on the keyword being typed because the cursor shifts with it; a fixed single-color cursor is visual noise by comparison. Layers on top of [discrete-text-sequence](discrete-text-sequence.md)'s SEQUENCE pattern.

## How It Works

The text is authored as a SEQUENCE of `{ t, text, segment, color }` entries; a linear driver's `onUpdate` reverse-searches for the current entry and writes both the visible text and the cursor's `background` (the cursor is a colored block, so `background`, NOT `color`). A second linear tween sweeps a phase `p` through `2π × BLINK_CYCLES_PER_SCENE` and gates cursor opacity on `sin(p) > 0` — a deterministic square-wave blink on the timeline.

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
  font-family: {monoFont}; /* proportional fonts drift the cursor mid-segment */
  display: flex;
  align-items: baseline;
  white-space: pre; /* preserve trailing spaces — cursor sits at segment end */
}
.text {
  white-space: pre;
}
.cursor {
  display: inline-block; /* inline ignores width/height */
  width: {cursorWidth}px;
  height: {cursorHeight}px;
  background: {textColor}; /* default — overridden per segment in onUpdate */
  vertical-align: {cursorBaselineFix}px; /* small negative — anchor to baseline, not line-height */
}
```

```js
// Adjacent entries usually share a text prefix but may differ in `segment` —
// that's what shifts the cursor color mid-line.
const SEQUENCE = [
  { t: 0, text: "", segment: "main", color: "{mainColor}" },
  { t: T_LEADIN_END, text: "{leadInChunk}", segment: "main", color: "{mainColor}" },
  { t: T_BRAND_IN, text: "{leadInBrandPrefix}", segment: "brand", color: "{brandColor}" },
  { t: T_BRAND_OUT, text: "{leadInBrandFull}", segment: "main", color: "{mainColor}" },
  { t: T_CMD_IN, text: "{leadInCmdPrefix}", segment: "cmd", color: "{cmdColor}" },
  { t: T_SUCCESS, text: "{leadInDone}", segment: "success", color: "{successColor}" },
];

function entryAt(time) {
  for (let i = SEQUENCE.length - 1; i >= 0; i--) {
    if (time >= SEQUENCE[i].t) return SEQUENCE[i];
  }
  return SEQUENCE[0];
}

const textEl = document.getElementById("text");
const cursorEl = document.getElementById("cursor");

const driver = { t: 0 };
tl.to(
  driver,
  {
    t: DURATION,
    duration: DURATION,
    ease: "none",
    onUpdate: () => {
      const entry = entryAt(driver.t);
      textEl.textContent = entry.text;
      cursorEl.style.background = entry.color;
    },
  },
  0,
);

// Deterministic square-wave blink
const blink = { p: 0 };
tl.to(
  blink,
  {
    p: Math.PI * 2 * BLINK_CYCLES_PER_SCENE,
    duration: DURATION,
    ease: "none",
    onUpdate: () => {
      cursorEl.style.opacity = Math.sin(blink.p) > 0 ? "1" : "0";
    },
  },
  0,
);
```

## Variations

- **Non-blinking during active typing** — suppress blink while letters are appearing (solid cursor), resume on idle. This MUST be a pure function of the driver's time: tracking a mutable `lastChangeTime` in `onUpdate` is not reverse-seek-safe (scrubbing backwards leaves the stale forward-pass value behind and the cursor blinks — or holds solid — at the wrong frames). Bake the change times from the SEQUENCE instead — every entry whose `text` differs from its predecessor is a typing event:

```js
// Baked once at build time — no runtime state.
const CHANGE_TIMES = SEQUENCE.filter((e, i) => i > 0 && e.text !== SEQUENCE[i - 1].text).map(
  (e) => e.t,
);
// In onUpdate — identical result at any seek, either direction:
const isTyping = CHANGE_TIMES.some((t) => t <= driver.t && driver.t - t < TYPING_GRACE);
cursorEl.style.opacity = isTyping ? "1" : Math.sin(blink.p) > 0 ? "1" : "0";
```

- **Cursor HEIGHT shifts on segment** — larger cursor on the brand segment: `cursorEl.style.height = entry.segment === "brand" ? cursorHeightEmphasis : cursorHeight` (1.1–1.25×; more reads as glitch).
- **Contrast reversal** — a dark-text-on-light segment needs a dark cursor too; keep `entry.color` as the single source of truth and read from it.

## Values

| token                  | range                       | notes                                                                                           |
| ---------------------- | --------------------------- | ----------------------------------------------------------------------------------------------- |
| DURATION               | 4–8s per typed line         | `≥ SEQUENCE[last].t + closing dwell`                                                            |
| entry `t` spacing      | 0.2–0.5s micro-additions    | ascending, non-uniform — slow down on highlights                                                |
| segment palette        | 3–4 colors max              | more reads as random; brand vs success should differ in saturation/luminance                    |
| cursorWidth / Height   | 8–24px / 0.85–1.0× fontSize | too thin vanishes in render compression; too tall outranks the text                             |
| cursorBaselineFix      | small negative px           | drop the block to the text baseline                                                             |
| BLINK_CYCLES_PER_SCENE | period ≈ 0.6–1.2s           | **whole number** — otherwise the sin sweep ends mid-cycle and the cursor pops on the last frame |
| TYPING_GRACE           | 0.15–0.3s                   | **< shortest dwell between adjacent entries** — otherwise the cursor never blinks               |

## Critical Constraints

- **Cursor color goes on `background`** — it's a colored block, not a glyph.
- **Blink is timeline-driven sin, pure of any mutable tracker** — the typing-grace variation shows the seek-safe form.
- **`white-space: pre` on text and container** — collapsed trailing spaces park the cursor in the wrong column.
- **Monospace font + `display: inline-block` cursor** — proportional faces drift the cursor mid-segment; inline ignores the block geometry.
- **BLINK_CYCLES_PER_SCENE is a whole number** for the fixed DURATION.

## See also

`discrete-text-sequence` (the underlying SEQUENCE pattern) · `camera-cursor-tracking` (camera follows the cursor) · `press-release-spring` (post-typing confirm press).

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
