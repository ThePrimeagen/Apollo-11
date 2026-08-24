# Descent Replay Website — Implementation Spec

A browser recreation of the Apollo 11 powered descent, built as a **companion window to
`exec-tui`**: both run side by side on one monitor, driven by the **same simulation
engine**. The website is an extremely narrow portrait column — **600 px wide by design
(800 px hard max), full 4K height (2160 px)** — showing the night sky, the stars, the
Moon, and the LM *Eagle* descending; a working DSKY; the ACA joystick and ROD switch;
every keystroke the crew typed; the program running at each moment; and the five
1201/1202 alarms at their exact times and altitudes.

Beyond replay, the site is a **teaching instrument**: a cog menu (bottom right) selects
between the **actual flight** (RR bug on → five alarms) and the **happy case** (RR switch
in LGC → no alarms), auto-pauses at each alarm, and opens an *allocation forensics* view
that shows — cycle by cycle — exactly how the core sets and VAC areas ran out.

This document is written so the feature can be implemented milestone by milestone.
**Tests come first**: no production code for a milestone is written until that milestone's
tests exist and fail.

Sources of truth: [`timeline.markdown`](timeline.markdown) (validated against Cherry's
*Exegesis*, Eyles' *Tales From the Lunar Module Guidance Computer*, and the ALSJ
transcript), [`exec-tui/RESEARCH.md`](exec-tui/RESEARCH.md) (every sim constant, sourced),
[`operations_and_timing.md`](operations_and_timing.md) (the full job/interrupt inventory
and duty-cycle ledger), and the Luminary 099 assembly in this repository.

The TUI exposes the three flight-critical controls as a **cockpit switch panel**
(h/l or ←/→ focuses a switch, space/Enter flips it; rendered by the reusable
`button-lab/button` toggle component): **DESCENT** (flipping it types `V37E 63E` at human
cadence — the deceleration burn starts because the engine parses the keystrokes),
**DELTAH** (types `V16N68E`; flipping it off types `V34E`, the terminate-monitor verb),
and **RR STEAL** (the SLEW/AUTO mode switch — OFF is LGC, the clean case). Pause in the
TUI is **`.`** (space is a switch flip). The website mirrors these three controls
one-for-one; they are the shared vocabulary of §6.3.

---

## 1. Test plan (write these before any code)

### 1.1 Existing tests — reviewed, with verdicts

Every current test was reviewed against this feature. Most are untouched regression
fences; a specific group in `exec-tui/ui` becomes the **contract for the Director
refactor** (§3): the Model keeps its public accessors (`Paused()`, `PendingKeys()`,
`TypingMode()`) delegating to the shared Director, and these tests must pass **without
modification** to prove the TUI's behavior did not change.

| Existing test | Location | Verdict |
| :--- | :--- | :--- |
| `TestTimeScaleWallToAGC`, `TestIdleBaselineFreeCompute`, `TestReadaccsPunctuality`, `TestServicerAllocation`, `TestPriorityPreemption`, `TestFreeComputeAccounting`, `TestBucketsClosed` | `exec-tui/sim` | Unchanged — core scheduling/accounting invariants |
| `TestNoVacBailout1201`, `TestNoCoreSetBailout1202`, `TestBailoutRestartRecovery`, `TestServicerOverrunLeak`, `TestStubRecovery`, `TestStubSlotMarking`, `TestStubCount`, `TestLeakEvents` | `exec-tui/sim` | Unchanged — alarm/restart semantics shown on the site |
| `TestRadarBugTLOSS`, `TestRadarPing`, `TestKeystrokeCost`, `TestMonitorVerbLoad`, `TestHistoricalScenario`, `TestKnifeEdgeLogThrottling`, `TestPostRestartHint` | `exec-tui/sim` | Unchanged — load-injection paths reused by both frontends |
| Fidelity suite: `TestSleepingJobHoldsResources`, `TestLRReadCadence`, `TestLRLockDutyCost`, `TestHigatjobVACHold`, `TestP64FirstAlarmIs1201`, `TestP63TypingGives1202`, `TestBoundaryAlignmentNoFalseAlarm` (`fidelity_test.go`); `TestV37E63EStartsDescent` (`dsky_program_test.go`) | `exec-tui/sim` | Unchanged — these pin the sleep-segment mechanics, the P63-1202/P64-1201 alarm-code split, and typed program selection that the website's forensics and DSKY depend on |
| Pause/typing/keybinding tests: `ui_test.go` (pause freeze, state preservation across pause, keybindings) and `typing_test.go` (cadence, speed scaling, paused-holds-keys) | `exec-tui/ui` | **Refactor contract — must pass unchanged.** They assert through `m.Paused()` / `m.PendingKeys()` / `Update(FrameMsg{})`; after the Director refactor these delegate but behave identically |
| Switch-panel tests: `switches_test.go` (`TestSwitchPanelRender`, `TestSwitchFlip` incl. the typing-mode-swallows-space case, `TestPauseMovedToDot`), `active_state_test.go` (key-bar active states) | `exec-tui/ui` | **Refactor contract — must pass unchanged.** Switch flips queue DSKY keys through the same pending-key path the Director absorbs; `.`-pause semantics carry into the Director |
| Flight-recreation tests: `flight_test.go` (`TestFlightPlan` — the 'f' plan fires real engine actions at true mission times; `TestLanderPanel`) | `exec-tui/ui` | **Refactor contract — must pass unchanged.** The `flightPlan()` queue lives in the model exactly like pending keys; M2 promotes it into the Director as the scenario script driver |
| `lander-lab` component tests (`TestGeometry`, `TestAltitudeScale`, `TestAttitudes`, `TestAlarmMarkers`, `TestCountdown`, `TestPlumeFlicker`, `TestCaptions`, plus the demo's descent/playback/truthfulness suite) | `lander-lab/` | Unchanged — pin the √-altitude scale, per-phase attitudes, and persistent alarm markers the web scene mirrors |
| `stars-lab` component tests (`TestGlyphs`, `TestGeometry`, `TestPaintFirst`, `TestStrategies`, `TestPopulation`, `TestStarColors`, demo suite) | `stars-lab/` | Unchanged — pin the four-glyph depth classes, tint palette, no-twinkle rule, and paint-first z-order the web starfield mirrors |
| `lander-lab/sprite` tests (`TestJSONRoundTrip`, `TestTransparentAndColors`) and the `editor`/`cmd` tooling suites | `lander-lab/` | Unchanged — `sprites/lm.json` is the atlas contract the web scene consumes directly; the editor is authoring tooling only |
| `seg-lab` tests (`seg/seg_test.go`, `font_test.go`) | `seg-lab/` | Unchanged — pin the segment glyph mapping and `SegmentedAlpha.ttf` the web DSKY embeds |
| `button-lab` component tests (`button/button_test.go`, `lab_test.go`) | `button-lab/` | Unchanged — the reusable cockpit toggle; the web widgets mirror its look (§7), not its code |
| `dsky-lab` component tests (`TestGeometry`, `TestSevenSegmentDigits`, `TestVerbNounFlash`, `TestLights`) and `dsky_panel_test.go` (`TestDSKYStateMapping`, `TestDSKYPanelEmbedded`, `TestEngineLampAccessors`) | `dsky-lab/`, `exec-tui/ui` | Unchanged — pin the DSKY layout, flash semantics, and the engine's `RestartRecently`/`CompActy` lamp accessors the web replica consumes |
| Remaining `exec-tui/ui` render tests (header, DSKY panel, timelines, badges, knife-edge, stubs, `color_test.go`, the compact-layout suite `layout_test.go`/`layout2_test.go`/`layout3_test.go`/`zoom_test.go`, and `rowcost_test.go`) | `exec-tui/ui` | Unchanged — TUI rendering untouched by this feature |
| `timeline-tui` render tests | `timeline-tui` | Unchanged — not part of this feature |
| `npm run lint` (markdownlint) | root | Must pass for this spec and all new docs |

### 1.2 New tests — Milestone M1 (engine additions, Go, `exec-tui/sim`)

- [ ] `TestGETClockMapping` — happy: with the PDI anchor set, `AGCTimeMs` ↔ GET converts
  both ways and `GET 102:33:05.01` ≡ scenario T+0; unhappy: querying GET before the
  scenario window clamps to the window start and reports `ok=false` rather than panicking.
- [ ] `TestFlightScriptDriver` — happy: the historical script — **promoting the existing
  UI-side 'f' recreation (`flightPlan()` in `exec-tui/ui`) into the Director** — fires
  descent start, LR lock, `V16N68`/`V57E` keystrokes, `EnterP64`, `AttHold`, P66 at their
  scripted GETs (order and ±1 cycle tolerance asserted, timings matching the proven UI
  plan); unhappy: a script with out-of-order or duplicate entries is rejected at load
  with a descriptive error.
- [ ] `TestScenarioHappyCase` — happy: the same full script with the RR switch scripted to
  LGC (no ECDU theft) runs PDI → touchdown with **zero alarms, zero restarts**, and core
  set usage never exceeding a small bound (≤ 5 of 8) even with V16N68 up; unhappy: the
  actual-case scenario run back-to-back on the same engine build still produces exactly
  five alarms (2× P63, 3× P64) — the two scenarios must diverge only through the TLOSS
  input.
- [ ] `TestEventBreakpoints` — happy: with `pauseOn: [alarm]` armed, the engine halts on
  the exact step that raises 1202 (time does not advance past the BAILOUT event; the frame
  shows FAILREG populated and the failing request identified); resume continues cleanly;
  breakpoints also work for `restart`, `program`, and `keystroke` kinds; unhappy: an
  unknown breakpoint kind is rejected; breakpoints never fire in a scenario that lacks the
  event (happy case → the alarm breakpoint never triggers, run completes).
- [ ] `TestAllocationForensics` — happy: the engine keeps a bounded per-cycle allocation
  log (per 2 s cycle: slot owners with state running/**sleeping**/stub, new claims,
  releases, stub count) plus, on BAILOUT, a `FailedRequest` record naming the requesting
  job, whether it needed a VAC, and a snapshot of all 8 core-set / 5 VAC owners at that
  instant — at the first P63 1202 the owner counts sum to 8 with ≥ 1 SERVICER stub, and
  at the P64 1201 the VAC snapshot includes HIGATJOB sleeping on its VAC (consistent with
  `TestP64FirstAlarmIs1201`); unhappy: the log is a ring buffer — after hours of sim time
  memory stays bounded and the oldest cycles evict without corrupting the newest.
- [ ] `TestDeterministicReplay` — happy: two engines, same seed and same input trace,
  produce identical event logs and identical final `Accounting()`; unhappy: differing
  seeds may diverge but never violate pool invariants (core sets ≤ 8, VACs ≤ 5).
- [ ] `TestSnapshotRestoreRoundTrip` — happy: `Snapshot()` at an arbitrary time, then
  `Restore()` into a fresh engine, then advancing both in lockstep yields identical states
  and events; unhappy: restoring a truncated/corrupted snapshot returns an error and
  leaves the target engine untouched.
- [ ] `TestSeekEqualsContinuousRun` — happy: seek (nearest earlier keyframe +
  deterministic replay) to GET *t* equals the state of an uninterrupted run at *t*;
  unhappy: seeking outside the scenario window clamps to the window edges.
- [ ] `TestRedesignationInput` — happy: an ACA click in P64 (after the flashing-V06N64
  PRO) while TREDES remains shifts the landing site by the flight quanta — 0.5°
  elevation per pitch click, 2° azimuth per roll click (`ELEACH`/`AZEACH`) — and queues
  retargeting work in the next SERVICER pass; unhappy: clicks in P63, in P66, in ATT
  HOLD (REDESMON's channel-31 BIT13 skip), before PRO, or after TREDES expires change
  nothing and emit a "redesignation unavailable" event instead.
- [ ] `TestRODSwitchClicks` — happy: in P66 each ROD click changes commanded descent rate
  by exactly ±1 ft/s and costs the documented compute; unhappy: ROD clicks outside P66 are
  ignored and logged, never scheduled.
- [ ] `TestAttHoldJoystickLoad` — happy: in ATT HOLD, stick-out-of-detent raises DAP
  activity while deflected and rates null after release; unhappy: deflection while paused
  accumulates no load and no time.

### 1.3 New tests — Milestone M2 (Director, companion bridge, record tool; Go)

- [ ] `TestDirectorSingleWriter` — happy: all engine mutation goes through the Director's
  command loop; concurrent commands from two goroutines (simulating TUI + WebSocket
  client) interleave without a data race (`go test -race`) and every command is applied
  exactly once in arrival order; unhappy: a command arriving while a breakpoint holds the
  clock is applied without advancing time.
- [ ] `TestBothFrontendsShareState` — happy: a pause issued as a TUI keypress freezes the
  frames streamed to a bridge client on the same tick; a `{"op":"rate"}` from the bridge
  changes the TUI header's wall↔AGC scale on its next frame; DSKY keys typed in the TUI
  appear in the website's frame and vice versa; unhappy: a frontend sending commands after
  disconnect is dropped without affecting the other frontend.
- [ ] `TestServeOffIsInert` — happy: running `exec-tui` without `--serve` starts no
  listener, spawns no bridge goroutines, and the Director-backed TUI passes the entire
  pre-existing `ui` test suite byte-identically; unhappy: `--serve` on an occupied port
  exits with a clear error instead of half-starting.
- [ ] `TestControlStateBroadcast` — happy: pause/rate/scenario/breakpoint changes are
  broadcast in the `control` block of the next frame to **all** clients (late-joining
  clients receive current control state + latest frame immediately on connect); unhappy: a
  slow client's full buffer drops frames for that client only, never blocks the Director.
- [ ] `TestScenarioSwitchPreservesGET` — happy: switching actual → happy at GET
  102:40:00 restores the happy timeline at the same GET (via keyframe + replay) with the
  clock, rate, and pause state preserved; unhappy: switching scenarios mid-P66 in a live
  diverged (sandbox) run warns that sandbox state is discarded and requires confirmation.
- [ ] `TestRecordProducesFlightJSON` — happy: `cmd/record` emits **both**
  `flight-actual.json` and `flight-happy.json` whose first/last GET match the scenario
  window; the actual file carries the five alarms at the exact GETs and altitudes from
  `events.json`, the happy file carries none; both share identical frame cadence and GET
  indexing so the client can switch between them at any GET; unhappy: an engine/script
  mismatch (e.g. missing alarm) fails the run with a diff report, not a silent file.
- [ ] `TestStateFrameSchema` — happy: every emitted frame validates against the StateFrame
  schema (§6.2), including the `control` and `forensics` blocks; unhappy: a hand-mutated
  invalid frame fails validation with the offending field named.
- [ ] `TestBridgeControlCommands` — happy: `play`, `pause`, `rate`, `stepMs:10`,
  `seekGet`, `reset`, `key`, `joyClick`, `rod`, `scenario`, `pauseOn`, `forensics` each
  mutate observable state over the WebSocket; unhappy: malformed JSON or an unknown `op`
  returns an error frame and the engine keeps running.
- [ ] `TestBridgeStepGranularity` — happy: `stepMs:10` while paused advances AGC time by
  exactly 10 ms and emits exactly one frame; unhappy: `stepMs` while playing pauses first
  (no double-advance), `stepMs:0` or negative is rejected.

### 1.4 New tests — Milestone M0/M3+ (web app, Vitest)

- [ ] `events.test.ts` — happy: loader parses `events.json`; the five alarms are exactly
  (1202 @ 102:38:22 ≈ 33,500 ft), (1202 @ 102:39:02 ≈ 29,000 ft), (1201 @ 102:42:18
  ≈ 3,000 ft), (1202 @ 102:42:43 ≈ 2,000 ft), (1202 @ 102:42:58, 770 ft) and Cherry PDI
  offsets +316/+356/+552/+578/+594 s are each within 1 s of the GET values; unhappy:
  out-of-order timestamps, a sixth alarm, or a missing `source` field reject the file.
- [ ] `trajectory.test.ts` — happy: interpolator returns the anchor values exactly
  (49,971 ft @ PDI; 7,400 ft @ 102:41:32; 770 ft @ 102:42:58; 0 ft @ 102:45:40) and is
  monotonically non-increasing in altitude after PDI; unhappy: queries before window start
  / after touchdown clamp and flag `extrapolated`.
- [ ] `layout.test.ts` — happy: at 600×2160 every zone of §2's budget renders with no
  horizontal overflow and no zone collapsed below its minimum; at 800×2160 the column
  caps at 800 px and centers; at reduced heights (1440) the low-priority zones (captions,
  forensics drawer) collapse first per the priority order; unhappy: below 560 px width an
  "unsupported width" notice replaces the app (no broken layout), and hidden zones remain
  reachable through the cog.
- [ ] `playback.test.ts` — happy: play advances GET at the selected rate; pause freezes
  it; `step(+10 ms)` moves exactly 10 ms; rates 0.1×/0.25×/1×/4×/16× each verified; scrub
  to an event lands on its GET; unhappy: stepping while playing pauses first; scrubbing
  past either end clamps; rate ≤ 0 is rejected.
- [ ] `pause-on-alarm.test.ts` — happy: with the cog's "pause on alarms" enabled, replay
  halts on the 102:38:22 frame, the alarm card opens showing the FAILREG code, the failing
  request, and the 8-slot owner breakdown (stubs vs running vs sleeping); resume plays to
  the next alarm; "pause on program change / keystroke / restart" behave the same for their kinds;
  unhappy: with the toggle off nothing pauses; a seek past an alarm does not retrigger its
  card.
- [ ] `scenario.test.ts` — happy: switching actual ↔ happy in the cog keeps the current
  GET and pause state; in compare mode the exec board renders the happy case as ghost
  outlines behind the actual fills and the divergence annotation appears once the actual
  case leaks its first stub; unhappy: a missing `flight-happy.json` disables the scenario
  entry with an explanatory tooltip instead of a dead toggle.
- [ ] `forensics.test.ts` — happy: paused at the first 1202, the forensics strip lists the
  preceding cycles with per-cycle stub growth and the final failing request row matching
  the frame's `forensics` block; unhappy: opening forensics before PDI shows an empty
  state ("no allocations yet"), and a frame without a `forensics` block renders the strip
  from the last known cycle with a stale marker.
- [ ] `cog.test.ts` — happy: the cog button sits bottom-right, opens the options sheet,
  every §8 option round-trips (change → applied → persisted to `localStorage` → restored
  on reload); unhappy: Esc/outside-click closes without applying a pending destructive
  choice (scenario reset asks for confirmation); unknown persisted keys from an older
  version are ignored, not fatal.
- [ ] `dsky.test.ts` — happy: at 102:38:04 the keystroke script renders `V16 N68` with
  R3 = −02900; at 102:38:22 the PROG lamp lights, the RESTART lamp holds through its
  window, and the display reverts to `V06 N63`; V05 N09 readback shows `01202`; the P64
  V06 N64 arrives **flashing** and stops on PRO; the ALT/VEL lights come on at the
  102:44:13 radar-dropout event; unhappy: an unknown verb/noun in a frame renders blanks
  and logs a warning instead of crashing.
- [ ] `lander.test.ts` — happy: sprite altitude/pitch track the trajectory (yaw-around
  begins 102:36:55; pitch-over at high gate 102:41:32; vertical by touchdown); unhappy: a
  missing/late frame holds last-known-good pose and shows a stale-data indicator.
- [ ] `joystick.test.ts` — happy: dragging the ACA widget in P64 emits `joyClick`
  commands with the right axis/sign, and the LPD reticle angle updates from the next
  frame; unhappy: input while paused or outside P64/ATT HOLD/P66 emits nothing and the
  widget shows "inactive" state.
- [ ] `hud.test.ts` — happy: GET/UTC/T+PDI clocks agree (UTC = GET − 82:28:00); the
  light-delay toggle shifts only voice captions by ±1.3 s; unhappy: toggling mid-caption
  does not duplicate or drop captions.

### 1.5 New tests — Milestone M6 (end-to-end, Playwright, viewport 600×2160)

- [ ] `replay.spec.ts` — happy: load site at 600×2160, play the actual case at 16×,
  assert all five alarm flashes occur at their GETs (±0.5 s scaled), phase banner walks
  P63 → P64 → P66, touchdown at 102:45:40, "The Eagle has landed" caption at 102:45:58;
  unhappy: with `flight-actual.json` blocked the site shows a load-error banner and
  controls stay disabled (no white screen).
- [ ] `compare.spec.ts` — happy: enable pause-on-alarm, play the actual case to the first
  halt, open forensics, switch to the happy case at the same GET and verify zero alarms
  through touchdown with the ghost/actual divergence annotation shown; unhappy: deleting
  `flight-happy.json` leaves the actual case fully playable with the happy option
  disabled.
- [ ] `live-sim.spec.ts` — happy: with `exec-tui --serve` running headless, the site
  connects, enabling the RR bug and the monitor verb produces an alarm within the
  historical envelope, and ATT HOLD stops the alarm train; a pause sent from the site is
  visible in a captured TUI frame (integration harness); unhappy: killing the bridge
  mid-session shows "connection lost — switch to replay?" and replay mode still works.

---

## 2. What the user sees — the narrow companion column

**Context: one 4K monitor (3840×2160), two windows.** `exec-tui` fills the left ~3,200 px;
the browser window is snapped to the right edge. The site is therefore designed
**portrait-first and narrow-only**:

| Constraint | Value |
| :--- | :--- |
| Design width | **600 px** |
| Maximum width | **800 px** (content column caps and centers beyond that) |
| Minimum width | 560 px (below: "unsupported width" notice) |
| Design height | **2160 px** (full 4K height) |
| Minimum height | 1440 px (low-priority zones collapse; see priority order below) |
| Type scale | Large: base 18 px, clocks/alarm codes 28–40 px — readable from across a room and legible in a screen recording |
| Contrast | Near-black `#050608` background, high-contrast panel text; no information conveyed by color alone |

There is **no horizontal scrolling, ever**. Vertical stacking only — which suits the
subject: altitude *is* the vertical axis.

### Zone budget at 600×2160 (top to bottom)

| # | Zone | Height | Contents |
| :--- | :--- | :--- | :--- |
| 1 | Mission clocks | 90 px | GET (large), UTC, T+PDI, phase badge P63/P64/P66, scenario badge (ACTUAL/HAPPY/SANDBOX) |
| 2 | Descent scene | 840 px | Star-field backdrop in the `stars-lab` convention: four depth classes (· ˚ * ✦, far/dim → near/bright), white with light-blue and faint-red tints (**no gold** — gold belongs to the LM foil), **no twinkle** (depth reads through per-class parallax speed, the DustRush model), painted first with everything drawing over it, none below the horizon. The LM is **rasterized from `lander-lab/sprites/lm.json`** — the same 4-size × 8-heading atlas the TUI renders (per-cell glyph + xterm-256 color; JSON round-trip pinned by the `sprite` tests) — so both windows show the same craft; the 8 headings give smooth attitude steps across the phases (horizontal in P63, pitched at high gate, vertical in P66). Descent against a **√-scaled altitude ladder** (the `lander-lab` convention — the final thousand feet stay readable), **persistent alarm markers pinned at the altitudes where they fired**, plume ∝ throttle with flicker, touchdown countdown; terrain + West Crater + site marker at the bottom; Earth appears after the 102:36:55 yaw-around; dust < 100 ft; alarm flash overlay (cog toggle) |
| 3 | DSKY | 300 px | PROG/VERB/NOUN + R1–R3 in segment digits rendered with the repo's own **`seg-lab/font/SegmentedAlpha.ttf`** via `@font-face` (digits at the official Unicode 7-seg codepoints U+1FBF0–9, letters from the 14-seg PUA range — the same glyphs as the TUI), COMP ACTY, verb/noun **flash**, and the four story lights (**PROG, RESTART, ALT, VEL**) — the same compact layout as `dsky-lab/dsky`; keyboard with replay key-lighting |
| 4 | Executive board | 420 px | 8 core-set + 5 VAC cells (owner/prio, with running/**sleeping**/stub states distinct), free-compute bar, duty rows with **per-job ms costs over the trailing 2 s window** (engine `UsedMs`, as in the TUI's row-cost display), restart counter — same semantics as the TUI panels; **ghost overlay** of the happy case in compare mode; forensics drawer expands from here |
| 5 | Hand controls | 200 px | ACA joystick, ROD switch, AUTO/ATT HOLD mode switch |
| 6 | Event feed / captions | 200 px | Air-to-ground captions at logged GETs (light-delay toggle), clickable event index — the TUI's compact layout dropped its own event log, so in companion mode this feed is the narrative record |
| 7 | Transport | 110 px | Play/pause, rate presets 0.1–16×, −10 ms/+10 ms/+2 s steps, scrub bar with event pips (alarms red, keystrokes amber, program changes cyan, voice grey) |
| — | **Cog** | 48 px floating | Bottom-right, floating above zones 6–7; opens the options sheet (§8) |

Collapse priority when height < 2160: captions (6) → hand controls (5, replaced by a
one-line status chip) → scene shrinks to 600 px. Zones 1, 3, 4, 7 and the cog never
collapse. Every collapsed zone can be re-pinned from the cog.

```text
┌────600px────┐
│GET 102:38:22│  zone 1 · clocks + P63 + ACTUAL
│  ·  ✦    ·  │
│ ·    ★   ˚ ·│
│   🌍        │  zone 2 · the descent column:
│      ▲      │  LM at 33,500 ft on a √ ladder,
│     ▕▂▏     │  plume, stars, terrain rising
│    ══╧══    │
│─── terrain ─│
│ DSKY  16 68 │  zone 3 · PROG lamp lit, R3 −02900
│ R3 −02900 ⚠ │
│ CORE ████████ 8/8 → 1202 │ zone 4 · exec board
│ VAC  █████ 5/5   + ghost + forensics drawer │
│ [ACA] [ROD] │  zone 5
│ 102:38:42 “Give us a reading…” │ zone 6
│ ⏮ ⏯ ⏭ .1×…16× −10ms +10ms │ zone 7
│ |—•—•———•—•—•—| scrub    ⚙ │ ← cog, bottom right
└─────────────┘
```

### Time controls (hard requirements, unchanged semantics)

| Control | Behavior |
| :--- | :--- |
| Play / Pause | Space **or `.`** (the TUI's pause key — same muscle memory in both windows); frame-accurate freeze |
| Rates | 0.1× (slow play), 0.25×, 1× (real time), 4×, 16× — keys `1`–`5` |
| **Step +10 ms / −10 ms** | `→` / `←` while paused; advances/rewinds AGC time exactly 10 ms (backward = seek) |
| Step ±2 s | Shift+`→` / Shift+`←` — one full guidance cycle |
| Scrub bar | Whole window 102:32:00 → 102:46:10, event pips, snap-to-pip |
| Jump to event | Prev/next event buttons + clickable event index |
| **Auto-pause** | Cog toggles: pause on alarms / restarts / program changes / keystrokes — playback halts on the exact event frame and opens its explainer card (§5.3) |

---

## 3. Linking the website and the TUI (shared-engine architecture)

This is the load-bearing requirement, so it is specified against the code as it exists
today. Currently the TUI **owns the clock and some control state**: a Bubble Tea ticker
fires every 33.34 ms and the model advances the engine and its typing queue itself, and
pause state lives in the model:

```76:91:exec-tui/ui/ui.go
	case FrameMsg:
		if !m.paused {
			m.eng.AdvanceWall(frameWallMs)
			for len(m.pending) > 0 && m.eng.AGCTimeMs() >= m.pending[0].dueAGC {
				m.eng.PressKey(m.pending[0].key)
				m.pending = m.pending[1:]
			}
```

A website bolted onto that would fight the TUI for the engine. The fix is a small,
explicit refactor:

### 3.1 The Director (new: `exec-tui/sim/director.go`)

One object owns everything two frontends must agree on:

- the **engine** (sole writer — all mutation flows through the Director's command loop,
  one goroutine, verified with `-race`),
- the **clock**: paused, rate (wall↔AGC), pending scripted keystrokes,
- the **scenario** (actual / happy / sandbox) and the flight-script driver (today this is
  the model-owned `flightPlan()` queue behind the TUI's 'f' key — it moves here whole),
- **breakpoints** (`pauseOn` event kinds) — evaluated *inside* the advance loop so a halt
  lands on the exact event frame, not the next UI tick,
- the **frame broadcaster**: after each advance it publishes an immutable `StateFrame`
  (§6.2) to all subscribers (TUI model, WebSocket clients, recorder).

The Bubble Tea model shrinks to a view: `FrameMsg` becomes "ask the Director to advance
and hand me the latest frame"; every keybinding and **cockpit switch** maps to the **same
`ControlCommand` values the website sends** (`.` → `{"op":"pause"}`, `[`/`]` →
`{"op":"rate"}`, DESCENT flip → the `key` sequence `V37E63E`, DELTAH flip →
`V16N68E` on / `V34E` off, RR STEAL flip → the RR mode command, …). Which switch is
*focused* (h/l) is TUI-local view state, never shared. Program selection needs no
privileged op: the engine parses typed keystrokes (`TestV37E63EStartsDescent`), so the
website's DSKY starts the deceleration burn exactly the way the TUI's DESCENT switch and
the flight crew did — by typing. One command vocabulary, two frontends. The model keeps
its public accessors (`Paused()`, `PendingKeys()`, `TypingMode()`) as thin delegates so
the existing `ui` test suite passes unchanged (§1.1).

### 3.2 Serving the companion

- **`exec-tui --serve :8443`** (primary): the TUI you are looking at *is* the server. The
  browser column and the terminal window render the same engine tick-for-tick: pause in
  either, both freeze; type `V16N68` in either, both DSKYs light; the alarm flashes in
  both on the same frame. Without the flag, nothing listens and the TUI is byte-for-byte
  the current behavior (`TestServeOffIsInert`).
- **`exec-tui/cmd/bridge`** (secondary): the same Director + WebSocket server without the
  terminal UI, for hosting the live mode when no TUI window is wanted (CI, demos).
- **`exec-tui/cmd/record`**: runs the Director headless through a scenario script and
  writes the Mode A frame files.

### 3.3 What is shared vs local

| State | Owner | Notes |
| :--- | :--- | :--- |
| AGC time, pause, rate, scenario, breakpoints, DSKY/joystick/ROD inputs, scripted keystrokes | **Director** (shared) | Changing it anywhere changes it everywhere; broadcast in every frame's `control` block |
| Cog display preferences (zone visibility, font scale, light-delay toggle, compare-ghost on/off) | Website only (`localStorage`) | Never sent to the Director |
| TUI-only view state (flash counters, switch-panel focus, timeline zoom `z`, layout) | TUI model | Unchanged |

Sync rules: commands are applied in arrival order; late-joining clients get current
control state + the latest frame immediately; a slow client drops its own frames but can
never block the Director or the TUI (`TestControlStateBroadcast`).

### 3.4 Modes

- **Mode A — Replay (default, static hosting).** No TUI required: the site plays the
  pre-recorded `flight-actual.json` / `flight-happy.json` (both produced by the same
  engine via `cmd/record`). All §6 walkthrough features work, because the frames carry
  the same `exec`/`forensics` blocks the live engine emits.
- **Mode B — Companion (live, shared engine).** WebSocket to `exec-tui --serve`. This is
  the two-windows-one-monitor setup this spec is built around.
- Mode B degrades to Mode A when no bridge is reachable (banner + fallback); the cog
  shows connection state and the bridge URL.

---

## 4. Ground truth: the validated replay dataset

All values below are cross-checked against Cherry (MIT, 4 Aug 1969), Eyles (AAS 04-064),
NASA SP-4029, and the ALSJ transcript; they match `timeline.markdown`. `~` =
interpolated/approximate.

### 4.1 Event and keystroke script (`events.json`)

| GET | T+PDI | Program | Event / DSKY activity | Altitude | ḢDOT |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 102:10:16 | −1,369 s | — | Aldrin (onboard): *"Okay, Auto Track."* — the **RR mode switch goes to AUTO TRACK**, the crew-checklist step Cherry's memo cites ("the RR switch be in AUTO TRACK immediately before calling P63"). The ECDU-theft ingredient is armed from here (Fjeld traces the switch's use back to a post-DOI ranging test) | — | — |
| ~102:10:40 | ~−1,345 s | P63 | Crew keys **V37E 63E** (AFJ onboard transcript: *"Okay, we ready to go to P63?"* 102:10:32; Armstrong: *"Yes."* 102:10:36; by 102:11:07 they are checking P63's computed burn time against the PAD's 9:50). DSKY answers with **flashing V06 N61** (TTG in braking / time from ignition / crossrange). Klumpp's "~10 min before ignition" design note understates the flight practice — the crew ran it ~22 min before PDI | — | — |
| ≤102:26:55 | ≤−370 s | P63 | **Checklist code 500** — `P63SPOT3` (`Luminary099/THE_LUNAR_LANDING.agc:245`) finds the LR antenna off position 1 and flashes **V50 N25 / 00500** (source comment: *"ASTRONAUT: PLEASE CRANK THE SILLY THING AROUND"*). Crew sets the LR antenna switch to Descent 1, keys **PRO** (the code re-checks the discrete — *"SEE IF HE'S LYING"*), returns it to Auto; Aldrin reports it at 102:26:55 | — | — |
| 102:32:30 | −35 s | P63 | **V06 N62** blanks (5 s), returns at T−30 s — Average-G running | ~49,971 ft | ~0 |
| 102:32:58 | −7.5 s | P63 | Ullage — RCS settles propellant | — | — |
| 102:33:00 | −5 s | P63 | **Flashing V99 N62** — BURNBABY requests the final engine-enable go; **Aldrin keys PRO** | — | — |
| 102:33:05.01 | +0 s | P63 | **PDI** — ignition at 10%; **V06 N63**: R1 +5559.7 (velocity), R2 −2.2 (ḢDOT), R3 +49971 (alt) | 49,971 ft | 2.2 ft/s |
| 102:33:31 | +26 s | P63 | Throttle up to FTP (~9,870 lb); guidance enabled | ~49,000 ft | — |
| 102:35:38 | +153 s | P63 | Armstrong moves the **RR mode switch AUTO TRACK → SLEW** (onboard: *"You're Slew? Okay."*); it had been in AUTO TRACK since 102:10:16 — the ECDU theft is active in both non-LGC modes throughout the burn (ALSJ/Fjeld) | — | — |
| 102:36:55 | +230 s | P63 | Armstrong yaws face-up (rate switch 5→25 deg/s); Earth in the windows | — | — |
| ~102:37:53 | ~+288 s | P63 | Landing radar **"data good"** | ~35,000 ft | — |
| ~102:38:04 | ~+299 s | P63 | **Aldrin keys V16 N68 E** — DELTAH monitor; R3 −02900 (callout 102:38:06) | ~34,000 ft | — |
| **102:38:22** | **+317 s** | P63 | **ALARM 1 — 1202** (no core sets). PROG lamp; DSKY reverts **V06 N63**; crew reads code with **V05 N09 E** | **~33,500 ft** | ~120 ft/s |
| 102:38:42 | +337 s | P63 | Armstrong: *"Give us a reading on the 1202 Program Alarm."* | ~32,000 ft | — |
| 102:38:53 | +348 s | P63 | Duke: *"We're Go on that alarm."* (crew hears ~:54) | ~31,000 ft | — |
| ~102:38:55 | ~+350 s | P63 | **Aldrin keys V57 E** (accept LR updates), re-keys **V16 N68 E**; DELTAH → ~900 ft | ~30,000 ft | — |
| **102:39:02** | **+357 s** | P63 | **ALARM 2 — 1202**; V05 N09 readback again | **~29,000 ft** | ~125 ft/s |
| 102:39:14 | +369 s | P63 | Aldrin: *"…it appears to come up when we have a 1668 up."* | ~27,000 ft | — |
| 102:39:31 | +386 s | P63 | **Throttle down** — on time (*"better than the simulator"*) | ~24,500 ft | — |
| 102:41:32 | +507 s | **P64** | **High gate.** Pitch-over; **flashing V06 N64** comes up (R1 = TREDES + LPD angle packed, R2 ḢDOT, R3 alt). Crew keys **PRO** (`P64CEED`) — this zeroes the click counters and sets REDFLAG, enabling redesignation. Armstrong's *"P64."* call logged 102:41:35 (Table 5‑I: 102:41:32) | **7,400 ft** | 125 ft/s |
| 102:42:10 | +545 s | P64 | Duke: *"You're Go for landing."* | ~3,500 ft | — |
| **102:42:18** | **+553 s** | P64 | **ALARM 3 — 1201** (no VAC areas) | **~3,000 ft** | ~60 ft/s |
| 102:42:32 | +567 s | P64 | Armstrong: *"Give me an LPD."* — Aldrin reads the reticle series **47° → 35° → 33° → 30°** over the next ~45 s (the site animates the reticle from these) | ~2,000 ft | — |
| **102:42:43** | **+578 s** | P64 | **ALARM 4 — 1202** | **~2,000 ft** | ~50 ft/s |
| late P64 | — | P64 | One **inadvertent LPD redesignation** (Eyles/debriefing; exact GET unknown — M0 item) | — | — |
| **102:42:58** | **+593 s** | P64 | **ALARM 5 — 1202** (last) | **770 ft** | 27 ft/s |
| 102:43:08 | +603 s | P64 | **AUTO → ATT HOLD** — Armstrong takes attitude, sheds load; no further alarms | ~650 ft | ~20 ft/s |
| 102:43:20 | +615 s | **P66** | **ROD switch flick enters P66**; joystick = rate-command attitude, ROD = ±1 ft/s per click | ~430 ft | ~15 ft/s |
| 102:44:13 | +668 s | P66 | LR **altitude & velocity lights** — radar dropouts in the low, dusty final approach | ~230 ft | — |
| 102:44:31 | +686 s | P66 | **Propellant low-level sensor latches** (~5.6% left; slosh tripped it ~30 s early — Fjeld); 94 s "Bingo" countdown starts. Aldrin calls *"Five percent. Quantity light"* at 102:44:45 (100 ft) | ~160 ft | ~6.5 ft/s |
| 102:45:02 | +717 s | P66 | Duke: *"60 seconds"* (to the Bingo land-in-20-s-or-abort call); *"30 seconds"* follows at 102:45:31 | ~65 ft | — |
| 102:45:40 | +755 s | P66 | **Contact light / TOUCHDOWN** in the Sea of Tranquility; Armstrong *"Shutdown"* 102:45:43, Aldrin *"Engine Stop… ACA out of Detent"* 102:45:44–45 | 0 ft | 0 |
| 102:45:58 | +773 s | — | *"Houston, Tranquility Base here. The Eagle has landed."* | — | — |

Rows before 102:32:00 are context events: the replay window (§2) starts at 102:32:00,
and the event index lists earlier rows as "pre-window" (jump-to clamps to window start).

Alarm cross-check embedded in `events.json` and enforced by `events.test.ts`: Cherry PDI
offsets **+316, +356, +552, +578, +594 s** (1202, 1202, 1201, 1202, 1202); two in P63,
three in P64; four 1202s (core sets), one 1201 (VAC areas); three restarts inside 40 s in
P64.

Note on readback keystrokes: Eyles' 2004 paper misprints the alarm readback as "Verb 90
Noun 50"; the correct sequence is **V05 N09 E** (Tillman memo: "V5N9E"), which is what the
site replays.

### 4.2 Trajectory anchors (`trajectory.json`)

Piecewise-monotone (PCHIP) interpolation of altitude vs GET through the anchors in §4.1's
altitude column (exact anchors bold, `~` anchors soft-weighted), with recorded descent
rates used as derivative constraints (125 ft/s at high gate, 27 ft/s at 770 ft, 3.5 ft/s
at 100 ft). Horizontal state: forward velocity from N63 R1 anchors (5,559.7 ft/s at PDI →
~0 at touchdown) integrated to downrange position, with the P66 "9 forward / 4 forward"
callouts as low-altitude constraints. Attitude: windows-down ≈ face-prone through
102:36:55, then face-up; pitch ≈ 77° off-vertical early P63 → ~45° at high gate → near
vertical in P66 (digitize from Bennett AIAA 70-1028 / Klumpp R-695 in M0).

The P64/P66 fine-grain anchors are transcribed from the ALSJ corrected transcript
(crew callouts; blank = not called):

| GET | Alt (ft) | ḢDOT (ft/s) | Fwd (ft/s) | LPD | Note |
| :--- | ---: | ---: | ---: | ---: | :--- |
| 102:42:13 | 3,000 | 70 | | | Armstrong onboard |
| 102:42:24 | 2,000 | 50 | | | Armstrong onboard |
| ~102:42:50 | 1,000 | 30 | | 47° | spoken during the pause after 102:42:37 |
| 102:43:01 | 750 | 23 | | 35° | |
| 102:43:07 | 700 | 21 | | 33° | |
| 102:43:11 | 600 | 19 | | | |
| 102:43:16 | 540 | 15 | | 30° | |
| 102:43:26 | 400 | 9 | 58 | | |
| 102:43:33 | 350 | 4 | | | |
| 102:43:35 | 330 | 3.5 | | | |
| 102:43:46 | 300 | 3.5 | 47 | | *"Slow it up"* |
| 102:43:52 | 270 | 1.5 | | | *"Ease her down"* |
| 102:44:07 | 250 | 2.5 | 19 | | |
| 102:44:16 | 220 | 3.5 | 13 | | |
| 102:44:24 | 200 | 4.5 | | | 5.5 down at :26 |
| 102:44:31 | 160 | 6.5 | | | quantity sensor latches |
| 102:44:33 | | 5.5 | 9 | | |
| 102:44:40 | 120 | | | | |
| 102:44:45 | 100 | 3.5 | 9 | | *"Five percent. Quantity light."* |
| 102:44:54 | 75 | 0.5 | 6 | | |
| 102:45:17 | 40 | 2.5 | | | *"Picking up some dust"* |
| 102:45:21 | 30 | 2.5 | | | |
| 102:45:25 | 20 | 0.5 | 4 | | *"Drifting to the right a little"* |
| 102:45:40 | ~3 | | | | probe contact light |

Remaining trajectory M0 work: P63 mid-phase altitude points and the pitch profile
(digitize from Bennett AIAA 70-1028 / Klumpp R-695).

Note: exec-tui's 'f' recreation carries a second in-repo encoding of these anchors
(`flightPath` in `exec-tui/ui/ui.go`, ALT/VEL vs T+PDI). The M0 validation script
cross-checks `events.json`/`trajectory.json` against `flightPlan()`/`flightPath` and
flags drift beyond tolerance (exact events ±2 s, `~` events ±20 s). Known current
deltas, all inside tolerance: LR data good +274 s vs ~+288 s (both interpolations) and
contact +757 s vs +755 s.

### 4.3 Computer-state truth (from the engine, not hand-authored)

Program phases, verb/noun, R1–R3, PROG lamp, FAILREG contents, core-set/VAC occupancy,
free-compute %, duty breakdown, stub count, and restart events all come from
`exec-tui/sim` frames — the constants are sourced in `exec-tui/RESEARCH.md` and
`operations_and_timing.md` (15.0% RR theft = 2 × 6,400 × 11.72 µs; duty margins >15% →
≈13% → ≤10% → <10%; SERVICER 1,320 ms base, +70 ms/cycle LR conversion, +60 ms/cycle in
P64; monitor 30 ms/s; restart 20 ms + 20 ms REREADAC). The engine also models the
**head/sleep/tail structure** of the short jobs — and sleeping jobs **hold their core
set/VAC while asleep**: LRHJOB 1/80/1 ms (fired 50 ms before each READACCS), LRVJOB
1/500/1 ms, MONDO 15/250/15 ms (display-wait), CHARIN 3/150/2 ms (echo-wait), HIGATJOB
2 ms then ~8 s asleep **holding a VAC** awaiting the antenna discrete. That last fact is
why P64's first alarm is the historical **1201** (VAC wall) while P63's typing-loaded
overloads hit the core-set wall as **1202** — pinned by `TestP64FirstAlarmIs1201` and
`TestP63TypingGives1202`.

**DSKY noun layouts — resolved** from `Luminary099/PINBALL_NOUN_TABLES.agc` (mixed-noun
table) and `ASSEMBLY_AND_OPERATION_INFORMATION.agc`; the replica renders exactly these:

| Noun | R1 | R2 | R3 | Where it appears |
| :--- | :--- | :--- | :--- | :--- |
| **N61** | `TTFDISP` — TTG in braking (min/s) | `TTOGO` — time from ignition (min/s) | `OUTOFPLN` — crossrange (n mi) | Flashing reply to V37E 63E |
| **N62** | `ABVEL` — \|V\| (xxxx.x ft/s) | `TTOGO` — time from ignition (min/s) | `DVTOTAL` — accumulated ΔV (ft/s) | BURNBABY pre-ignition countdown |
| **N63** | `ABVEL` — \|V\| | `HDOTDISP` — altitude rate | `HCALC1` — computed altitude | P63 (matches the PDI downlink: +5559.7 / −2.2 / +49971) |
| **N64** | `FUNNYDSP` — TREDES + LPD angle packed as two 2-digit ints (`xxBxx`) | `HDOTDISP` | `HCALC` | P64 (flashing until PRO) |
| **N60** | `VHORIZ` — horizontal velocity | `HDOTDISP` | `HCALC` | P66 (`VERTDISP` → V06 N60) |
| **N68** | `RANGEDSP` — slant range to site (xxxx.x n mi) | `TTFDISP` — time-to-go in braking (min/s) | `DELTAH` — LR−computed altitude | Aldrin's monitor (Eyles: "third register showed DELTAH" ✓) |
| **N09** | `FAILREG` | `FAILREG +1` | `FAILREG +2` | V05 N09 alarm readback |

---

## 5. Scenarios: the actual case, the happy case, and the walkthrough

The point of this feature is to *talk over* the landing while it plays: run the timeline,
stop on each alarm, and make the cause visible enough that the audience can answer "why
did the core sets run out?" from the screen. Accuracy target: **causally faithful, not
cycle-perfect** — the engine is a calibrated model (§4.3), and the UI says so.

### 5.1 Scenario definitions

| Scenario | Definition | Outcome |
| :--- | :--- | :--- |
| **ACTUAL** | Historical script, RR mode switch in AUTO/SLEW → ECDU theft ≈ 15% from before PDI | Five alarms at the §4.1 times; four software restarts; V16N68 monitor shed twice |
| **HAPPY** | *Identical* script — same keystrokes, same phases, same GETs — but the RR switch scripted to **LGC**, so the CDUs are zeroed and steal nothing (the TUI's RR STEAL switch OFF — the clean case) | Zero alarms, zero restarts; SERVICER finishes every cycle; core sets hover ~2–3 used; DELTAH monitor stays up |
| **SANDBOX** | Live Mode B free-play: every toggle (bug, monitor, typing, radar ping, ATT HOLD timing) under user control | Whatever you fly |

The two canned scenarios differ **only in the TLOSS input** (`TestScenarioHappyCase`),
which is precisely the historical counterfactual: "if the ICD had said *phase
synchronized*, the same descent produces no alarms." Both are recorded with identical GET
indexing so the site can flip between them at any instant (`TestScenarioSwitchPreservesGET`).

### 5.2 Compare mode (fits 600 px: overlay, not split-screen)

A side-by-side split is unreadable at 600 px, so comparison is an **overlay**: with
compare enabled, the executive board draws the HAPPY occupancy as **ghost outlines**
behind the ACTUAL fills. At 102:38:20 the audience sees eight filled core-set cells over
three ghost outlines — the five-cell difference *is* the leak. A one-line annotation
("HAPPY would be using 3 of 8 here") appears whenever the two diverge, and the scrub bar
carries a second, thinner pip row for the happy case (which has no red pips — visibly
empty where the alarms would be).

### 5.3 Auto-pause and alarm cards

With **pause on alarms** enabled (cog, default ON in ACTUAL), the Director halts on the
exact BAILOUT frame (`TestEventBreakpoints`). The alarm card opens over zone 2 and states,
from the frame's `forensics` block — never hand-written prose for the numbers:

1. **What fired:** `1202 — EXECUTIVE OVERFLOW, NO CORE SETS` (FAILREG `01202`). On the
   DSKY replica the codes read out flight-style — **V05 N09 with the FAILREG codes,
   unsigned, in the registers** (the TUI dropped its banner text for exactly this
   presentation); the scene's full-width alarm flash remains a website-only affordance
   with a cog toggle.
2. **The failing request:** which job asked (e.g. `READACCS → FINDVAC: SERVICER`,
   needs core set + VAC) and that it was the request the Executive could not fill.
3. **Who holds everything:** the 8 core sets / 5 VAC areas by owner **and state** at that
   instant — e.g. `4× SERVICER (stub, unfinished)`, `1× SERVICER (running)`,
   `1× MONDO (sleeping in display-wait)`, `1× CHARIN (sleeping in echo-wait)`,
   `1× LRHJOB (sleeping across the radar gate)` — stubs, runners, and sleepers each
   visually distinct. For the P64 1201, the VAC panel shows `HIGATJOB (sleeping ~8 s on
   the antenna discrete)` pinning the pool that ran out.
4. **Why they're stuck:** one sentence + a "show me" button that opens the forensics
   strip (§5.4).
5. Links to [`memory_leak.md`](memory_leak.md) / [`alarm_recovery.md`](alarm_recovery.md)
   / [`radar_problem.md`](radar_problem.md) for the deep dive.

The same card system covers restarts ("what the phase tables rebuilt, what got shed"),
program changes, and keystrokes (narrative mode: pause on *every* scripted event, for a
fully talked-through walkthrough).

### 5.4 Allocation forensics — making the exhaustion obvious

The intuition to validate on screen: *"the 2-second cycle keeps allocating new SERVICERs,
several stale copies pile up holding core sets, and then the handful of short jobs that
pop in at once push it over."* The forensics strip shows exactly that, and puts precise
numbers on it. It expands from the executive board and renders the last ~15 guidance
cycles from the engine's allocation log (`TestAllocationForensics`), one row per 2 s
cycle:

```text
cycle  GET        core sets (8)   VAC (5)   note
-7     102:38:08  ██▁▁▁▁▁▁  2     ██▁▁▁  2  SERVICER finishes late — first overrun
-6     102:38:10  ███▁▁▁▁▁  3     ███▁▁  3  stub A retained; new SERVICER B
-5     102:38:12  ████▁▁▁▁  4     ████▁  4  stub B retained; new SERVICER C
-4     102:38:14  █████▁▁▁  5     █████  5  ← VAC pool full
-3     102:38:16  ██████▁▁  6     █████  5  MONDO refresh takes a core set
-2     102:38:18  ███████▁  7     █████  5  CHARIN (keystroke) takes a core set
-1     102:38:20  ████████  8     █████  5  ← core pool full
 0     102:38:22  REQUEST: READACCS→FINDVAC(SERVICER)  → no core set → 1202 BAILOUT
```

(Illustrative shape — the real rows come from the engine log; the mix of stub growth,
**sleepers** (radar gates, display waits) briefly pinning core sets, and transient jobs
is whatever the simulation actually did, which is the point: the display *validates* the
mental model rather than asserting it. The engine now pins the historical pattern: with
the crew typing, a keystroke job usually finds the eighth core set gone first — the P63
**1202**s; in P64, HIGATJOB parked asleep on a VAC makes the five-slot VAC wall trip
first — the historical **1201**. The strip shows whichever request actually failed.)

Interactions: each row is clickable → seeks to that cycle boundary (Mode A: frame index;
Mode B: keyframe + deterministic replay), so "let's watch that cycle again at 0.1×" is one
click. In HAPPY the same strip shows a flat 2–3-cell line — the contrast slide. The strip
is also available at any pause, not just at alarms.

---

## 6. Data contracts

### 6.1 `events.json` entry

```json
{
  "id": "alarm-1202-1",
  "get": "102:38:22",
  "tPdiSeconds": 317,
  "kind": "alarm",
  "program": "P63",
  "label": "ALARM 1 — 1202 (no core sets)",
  "dsky": { "keys": null, "verb": "06", "noun": "63", "failreg": "01202" },
  "altitudeFt": 33500,
  "hdotFps": 120,
  "approx": { "altitude": true },
  "source": ["SP-4029", "Cherry +316 s", "timeline.markdown"]
}
```

`kind ∈ {alarm, keystroke, program, flight, voice, switch, restart}`. Keystroke events
carry the full key list with per-key cadence (the TUI's 230–330 ms pattern) so the DSKY
keyboard lights replay realistically.

### 6.2 `StateFrame` (bridge + flight files; ~10 ms cadence)

```json
{
  "agcMs": 317000, "get": "102:38:22.00", "phase": "P63",
  "control": { "paused": true, "rate": 1.0, "scenario": "actual",
               "pauseOn": ["alarm"], "haltedBy": "alarm-1202-1" },
  "dsky": { "prog": "63", "verb": "16", "noun": "68",
            "r1": "+00513", "r2": "+0310", "r3": "-02900",
            "progLamp": true, "restartLamp": false, "altLamp": false,
            "velLamp": false, "compActy": true, "flash": false,
            "typing": false },
  "failreg": ["01202"],
  "exec": { "coreSets": [{"owner": "SERVICER", "prio": 20, "stub": true}],
            "vacs": [{"owner": "SERVICER", "stub": false}],
            "freePct": -3.1, "runningJob": "CHARIN", "stubs": 4,
            "restarts": 1, "cycleMs": 1240, "monitor": true },
  "forensics": { "failedRequest": {"job": "SERVICER", "via": "FINDVAC", "needsVac": true},
                 "cycles": [{"get": "102:38:20", "core": 8, "vac": 5, "stubs": 4,
                             "claims": ["CHARIN"], "releases": []}] },
  "traj": { "altFt": 33500, "hdotFps": -120, "vFps": 1200,
            "downrangeFt": 91000, "pitchDeg": 62, "throttlePct": 92.5 },
  "events": ["alarm-1202-1"]
}
```

`control` is present in every frame (both windows stay in sync from it). `forensics` is
included on breakpoint-halt frames and on request (`{"op":"forensics"}`); replay files
carry it on every cycle-boundary frame.

### 6.3 `ControlCommand` (client → Director; also the TUI's internal command vocabulary)

```json
{ "op": "play" } { "op": "pause" } { "op": "rate", "x": 0.25 }
{ "op": "stepMs", "ms": 10 } { "op": "seekGet", "get": "102:41:32" }
{ "op": "key", "k": "V" } { "op": "joyClick", "axis": "pitch", "dir": 1 }
{ "op": "rod", "dir": -1 } { "op": "attHold" } { "op": "reset" }
{ "op": "scenario", "name": "actual" }
{ "op": "pauseOn", "kinds": ["alarm", "restart"] }
{ "op": "forensics" }
{ "op": "sandbox", "radarBug": true, "monitor": true }
```

Every command is acknowledged with the next frame or `{"op":"error","reason":"…"}`.
`sandbox.radarBug` is the RR mode switch (true ≡ SLEW/AUTO stealing cycles, false ≡ LGC)
— the TUI's RR STEAL switch. Starting the descent and keying the monitor need no
dedicated ops: they are `key` sequences (`V37E63E`; `V16N68E` on and `V34E` — the
terminate-monitor verb — off), the same way the TUI's DESCENT and DELTAH switches and
the flight crew did it.

---

## 7. The joystick (ACA) and ROD switch

The physical controls Armstrong used, and what each input costs the computer:

- **P64 (102:41:32 → 102:43:08), LPD redesignation.** N64 R1 shows TREDES (redesignation
  time left) and the LPD angle Armstrong sighted along his window reticle (flight series:
  47° → 35° → 33° → 30°). Redesignation goes live only after the crew keys **PRO** on the
  flashing V06 N64 (`P64CEED` zeroes the counters and sets REDFLAG). The click quanta are
  **confirmed from the flight source**, not estimated: pitch clicks shift the site
  **0.5° in elevation** and roll clicks **2° in azimuth** per click —
  `Luminary099/LUNAR_LANDING_GUIDANCE_EQUATIONS.agc`: `ELEACH DEC .00873 # 1/2 DEGREE`,
  `AZEACH DEC .03491 # 2 DEGREES`. The pipeline the site reproduces: each stick click
  raises the **RUPT10 (`PITFALL`) interrupt**, which reads channel 31 (BIT1/BIT2 =
  ∓elevation, BIT5/BIT6 = ±azimuth) and schedules the **REDESMON** waitlist task to count
  clicks into `ELINCR1`/`AZINCR1`; the next SERVICER guidance pass (`REDESIG`) moves the
  LAND vector by interpretive vector math, with a `DEPRCRIT` guard against redesignating
  too near the horizon. So **each click costs an interrupt + a monitor task + extra
  guidance retargeting** — joystick activity adds computation, and P64's redesignation
  logic is precisely the protected load that made its alarms unshedable. In Mode B,
  clicking during the overload measurably deepens the knife edge. Two source-mandated
  edge cases: REDESMON runs only in P64 (`CHECKMM DEC 64`), and it **skips counting in
  ATT HOLD** (channel 31 BIT13) — the widget mirrors both. The historical script includes
  the one inadvertent redesignation (M0 pins its GET or marks it "late P64").
- **ATT HOLD (102:43:08).** Joystick becomes rate-command/attitude-hold: deflection
  commands a rate, release holds attitude. Autopilot cost drops (the engine's ATT HOLD
  DAP model) — visibly recovering free-compute on the duty bar, which is *why* the alarms
  stopped.
- **P66 (102:43:20).** ROD switch: one click = ±1 ft/s commanded descent rate; SERVICER's
  P66 profile (900 ms) replaces the full guidance load. The widget is a vertical
  spring-loaded toggle next to the ACA; clicks show as ±1 ft/s deltas on the ḢDOT tape.

Widget spec: on-screen ACA (draggable/arrow keys, detent center, click quantization in
P64), ROD toggle, and the AUTO/ATT HOLD mode switch — disabled states grey out with a
tooltip explaining *why* (e.g. "LPD time expired"). At 600 px the three controls share
zone 5 in a single row. Toggles are styled after the repo's `button-lab/button` cockpit
switch (dark slot, lever, lit orange tip when on; fixed footprint so flipping never
shifts layout) so the web panel and the TUI panel read as the same instrument.

---

## 8. The cog menu (bottom right)

A single floating ⚙ button, 48 px, anchored bottom-right (above the scrub bar), opens an
options sheet sliding up. Contents, top to bottom:

| Group | Options |
| :--- | :--- |
| **Scenario** | ACTUAL · HAPPY · SANDBOX (radio); in SANDBOX, the three mission switches (DESCENT · DELTAH · RR STEAL, mirroring the TUI panel); "Compare (ghost happy case)" toggle; Restart scenario (confirmation required) |
| **Auto-pause** | Pause on: alarms (default ON) · restarts · program changes · keystrokes · every event (narrative mode) |
| **Playback** | Rate presets; light-delay toggle (±1.3 s on captions); snap-scrub-to-events |
| **Panels** | Show/hide: captions · hand controls · executive board · forensics drawer; re-pin collapsed zones |
| **Display** | Font scale (100/125/150%); width preset 600/800; high-contrast alarm flash on/off |
| **Connection** | Mode indicator (REPLAY / LIVE); bridge URL; connect/disconnect; "TUI linked" status with last-frame age |
| **About** | Accuracy note ("calibrated model, not cycle-perfect telemetry") + source links |

Behavior: every option round-trips to `localStorage` and restores on reload; Esc or
outside-click closes; destructive actions (scenario restart, discarding a sandbox run)
require confirmation; unknown persisted keys from older versions are ignored
(`cog.test.ts`). Keyboard: `g` opens the cog; the sheet is fully keyboard-navigable.

---

## 9. Milestones (each begins by landing its §1 tests, failing)

| # | Deliverable | Contents |
| :--- | :--- | :--- |
| M0 | Data | `events.json`, `trajectory.json` from §4 (noun layouts, LPD quanta, code-500 context, RR-SLEW time, and the P64/P66 callout series are already resolved in-spec from Luminary099 + ALSJ); remaining research: inadvertent-redesignation GET, P63 mid-phase altitude + pitch profile (Bennett AIAA 70-1028 / Klumpp R-695); validation script wired into CI |
| M1 | Engine | GET clock, flight-script driver, **happy/actual scenarios**, **event breakpoints**, **allocation forensics log**, snapshot/restore, determinism, ACA/ROD/redesignation inputs (tests §1.2) |
| M2 | Director + companion | **Director refactor** (TUI passes existing `ui` tests unchanged), `--serve` WebSocket in `exec-tui`, headless `cmd/bridge`, `cmd/record` → `flight-actual.json` + `flight-happy.json` (tests §1.3) |
| M3 | Web scene | Vite scaffold, loaders (including the consumed repo assets: `lander-lab/sprites/lm.json` sprite atlas and `seg-lab/font/SegmentedAlpha.ttf`), playback clock, **600×2160 portrait layout system**, vertical descent scene from `flight-actual.json` (tests §1.4: events, trajectory, layout, playback, lander) |
| M4 | Panels + transport | DSKY replica, executive board, captions, transport bar, scrub, ±10 ms step, event index (tests §1.4: dsky, hud) |
| M5 | Walkthrough + controls | **Cog menu, auto-pause cards, forensics strip, compare ghost overlay**, ACA/ROD widgets, live Mode B client (tests §1.4: cog, pause-on-alarm, scenario, forensics, joystick) |
| M6 | Polish + e2e | Camera, dust, light-delay toggle, Playwright e2e at 600×2160 (tests §1.5), README |

Tooling: Go 1.26 (stdlib or `nhooyr.io/websocket`), Vite + TypeScript + Vitest +
Playwright, Canvas 2D, zero runtime UI framework dependencies unless M3 review demands
one. New code lives in `descent-web/` and `exec-tui/cmd/`; `exec-tui/sim` and
`exec-tui/ui` grow but their existing APIs and tests stay intact.

## 10. Risks and open questions

- **Director refactor risk.** Moving the clock out of the Bubble Tea model touches the
  TUI's hot path. Mitigation: the existing `ui` pause/typing tests are the frozen
  contract (§1.1), the Model keeps its accessors, and `TestServeOffIsInert` proves the
  no-server path is unchanged.
- **Narrow-scene legibility.** 600 px must carry sky + lander + ladder + flash overlays.
  Mitigation: the vertical column layout makes altitude the long axis; the √ altitude
  scale is already proven legible in `lander-lab`'s 40×30 cell view; alarm flashes take
  the full column width; M3 includes a screenshot review at true 600×2160 before M4
  builds on it.
- **Dual-recording alignment.** Actual/happy files must stay frame-index-compatible or
  scenario switching jumps. Mitigation: one recorder run emits both from the same script
  with a shared GET index; `TestRecordProducesFlightJSON` asserts alignment.
- **Sub-second truth.** GETs are known to the second (alarms also as Cherry PDI offsets);
  the 10 ms step is a *navigation* affordance over engine frames, not a claim of 10 ms
  historical telemetry. The forensics strip is the simulation's own truth, presented as
  such ("calibrated model" note in the cog's About group).
- **First-alarm labeling.** Mission Report Table 5-I labels the first 1202 at 102:39:02;
  SP-4029 and Cherry support 102:38:22. We follow `timeline.markdown` (both listed, main
  scrub pips at SP-4029/Cherry times).
- **Trajectory fidelity between anchors** is interpolation; M0's ALSJ transcription makes
  P66 dense, but P63 mid-phase altitude is ~±1,000 ft. Acceptable for a profile view;
  document in the site's "accuracy" page.
- **Mode B trajectory divergence** (user flies differently than history) is approximate
  by design; the banner + docs must be honest about it.
- **The inadvertent redesignation's GET** still needs M0 confirmation (and Fjeld notes
  propellant slosh had made the LPD "essentially useless" by late P66 — the site's
  accuracy page should say so). The click quanta themselves are confirmed from the
  Luminary 099 source (`ELEACH`/`AZEACH`), so this no longer blocks the joystick.
