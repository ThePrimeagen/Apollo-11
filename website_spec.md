# Descent Replay Website — Implementation Spec

A browser recreation of the Apollo 11 powered descent: a black-as-night sky full of stars,
the lunar surface, and the LM *Eagle* descending — driven by the **same simulation engine
as `exec-tui`**, with full time control (play, slow-motion, 10 ms stepping, scrubbing),
a working DSKY, the ACA joystick and ROD switch, every keystroke the crew typed, the
program running at each moment, and the five 1201/1202 alarms at their exact times and
altitudes.

This document is written so the feature can be implemented milestone by milestone.
**Tests come first**: no production code for a milestone is written until that milestone's
tests exist and fail.

Sources of truth: [`timeline.markdown`](timeline.markdown) (validated against Cherry's
*Exegesis*, Eyles' *Tales From the Lunar Module Guidance Computer*, and the ALSJ
transcript), [`exec-tui/RESEARCH.md`](exec-tui/RESEARCH.md) (every sim constant, sourced),
and the Luminary 099 assembly in this repository.

---

## 1. Test plan (write these before any code)

### 1.1 Existing tests — reviewed, with verdicts

Every current test was reviewed against this feature. The engine gains **new** capability
(GET clock, script driver, snapshots, joystick inputs); it must not change existing
behavior, so **no existing test is altered** — they become the regression fence:

| Existing test | Location | Verdict |
| :--- | :--- | :--- |
| `TestTimeScaleWallToAGC`, `TestIdleBaselineFreeCompute`, `TestReadaccsPunctuality`, `TestServicerAllocation`, `TestPriorityPreemption`, `TestFreeComputeAccounting`, `TestBucketsClosed` | `exec-tui/sim` | Unchanged — core scheduling/accounting invariants the website relies on |
| `TestNoVacBailout1201`, `TestNoCoreSetBailout1202`, `TestBailoutRestartRecovery`, `TestServicerOverrunLeak`, `TestStubRecovery`, `TestStubSlotMarking`, `TestStubCount`, `TestLeakEvents` | `exec-tui/sim` | Unchanged — alarm/restart semantics shown on the site |
| `TestRadarBugTLOSS`, `TestRadarPing`, `TestKeystrokeCost`, `TestMonitorVerbLoad`, `TestHistoricalScenario`, `TestKnifeEdgeLogThrottling`, `TestPostRestartHint` | `exec-tui/sim` | Unchanged — load-injection paths reused by the bridge |
| All `exec-tui/ui` tests (header, DSKY panel, timelines, keybindings, …) | `exec-tui/ui` | Unchanged — TUI untouched by this feature |
| `timeline-tui` render tests | `timeline-tui` | Unchanged — not part of this feature |
| `npm run lint` (markdownlint) | root | Must pass for this spec and all new docs |

### 1.2 New tests — Milestone M1 (engine additions, Go, `exec-tui/sim`)

- [ ] `TestGETClockMapping` — happy: with the PDI anchor set, `AGCTimeMs` ↔ GET converts
  both ways and `GET 102:33:05.01` ≡ scenario T+0; unhappy: querying GET before the
  scenario window clamps to the window start and reports `ok=false` rather than panicking.
- [ ] `TestFlightScriptDriver` — happy: running the historical script unattended fires
  `StartDescent`, LR lock, `V16N68` keystrokes, `EnterP64`, `AttHold`, P66 at their
  scripted GETs (order and ±1 cycle tolerance asserted); unhappy: a script with
  out-of-order or duplicate entries is rejected at load with a descriptive error.
- [ ] `TestDeterministicReplay` — happy: two engines, same seed and same input trace,
  produce identical event logs and identical final `Accounting()`; unhappy: differing
  seeds may diverge but never violate pool invariants (core sets ≤ 8, VACs ≤ 5).
- [ ] `TestSnapshotRestoreRoundTrip` — happy: `Snapshot()` at an arbitrary time, then
  `Restore()` into a fresh engine, then advancing both originals and restores in lockstep
  yields identical states and events; unhappy: restoring a truncated/corrupted snapshot
  returns an error and leaves the target engine untouched.
- [ ] `TestSeekEqualsContinuousRun` — happy: seek (nearest earlier keyframe + deterministic
  replay) to GET *t* equals the state of an uninterrupted run at *t*; unhappy: seeking
  outside the scenario window clamps to the window edges.
- [ ] `TestRedesignationInput` — happy: an ACA click in P64 while LPD time remains queues
  redesignation work in the next SERVICER pass and shifts the LPD angle by the configured
  quantum; unhappy: clicks in P63, in P66, or after LPD time expires change nothing and
  emit a "redesignation unavailable" event instead.
- [ ] `TestRODSwitchClicks` — happy: in P66 each ROD click changes commanded descent rate
  by exactly ±1 ft/s and costs the documented compute; unhappy: ROD clicks outside P66 are
  ignored and logged, never scheduled.
- [ ] `TestAttHoldJoystickLoad` — happy: in ATT HOLD, stick-out-of-detent raises DAP
  activity while deflected and rates null after release; unhappy: deflection while paused
  accumulates no load and no time.

### 1.3 New tests — Milestone M2 (record tool + bridge, Go)

- [ ] `TestRecordProducesFlightJSON` — happy: `cmd/record` emits a frame stream whose
  first/last GET match the scenario window and whose five alarms carry the exact GETs and
  altitudes from `events.json`; unhappy: an engine/script mismatch (e.g. missing alarm)
  fails the run with a diff report, not a silent file.
- [ ] `TestStateFrameSchema` — happy: every emitted frame validates against the StateFrame
  schema (§6.2); unhappy: a hand-mutated invalid frame fails validation with the offending
  field named.
- [ ] `TestBridgeControlCommands` — happy: `play`, `pause`, `rate`, `stepMs:10`,
  `seekGet`, `reset`, `key`, `joyClick`, `rod` each mutate engine state observably over the
  WebSocket; unhappy: malformed JSON or an unknown `op` returns an error frame and the
  engine keeps running.
- [ ] `TestBridgeClientLifecycle` — happy: two clients receive identical frame sequences;
  unhappy: one client disconnecting mid-stream neither stalls the engine nor the other
  client.
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
- [ ] `playback.test.ts` — happy: play advances GET at the selected rate; pause freezes it;
  `step(+10 ms)` moves exactly 10 ms; rates 0.1×/0.25×/1×/4×/16× each verified; scrub to an
  event lands on its GET; unhappy: stepping while playing pauses first; scrubbing past
  either end clamps; rate ≤ 0 is rejected.
- [ ] `dsky.test.ts` — happy: at 102:38:04 the keystroke script renders `V16 N68` with
  R3 = −02900, at 102:38:22 the PROG lamp lights and the display reverts to `V06 N63`,
  V05 N09 readback shows `01202`; unhappy: an unknown verb/noun in a frame renders blanks
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

### 1.5 New tests — Milestone M6 (end-to-end, Playwright)

- [ ] `replay.spec.ts` — happy: load site, play at 16×, assert all five alarm flashes
  occur at their GETs (±0.5 s scaled), phase banner walks P63 → P64 → P66, touchdown at
  102:45:40, "The Eagle has landed" caption at 102:45:58; unhappy: with `flight.json`
  blocked the site shows a load-error banner and controls stay disabled (no white screen).
- [ ] `live-sim.spec.ts` — happy: with the bridge running, enabling the RR bug and the
  monitor verb produces an alarm within the historical envelope, and ATT HOLD stops the
  alarm train; unhappy: killing the bridge mid-session shows "connection lost — switch to
  replay?" and replay mode still works.

---

## 2. What the user sees (experience spec)

One full-viewport scene, one instrument strip, one transport bar:

```text
┌────────────────────────────────────────────────────────────────────────────┐
│  ★ ·   ˚      ·        ✦     GET 102:38:22   UTC 20:10:22   PDI +317 s     │
│      ·        ✦    ·                             ┌──────────────┐          │
│         (starfield, 2–3 parallax layers)         │ PROG ALARM   │          │
│   🌍 (Earth, after the 102:36:55 yaw-around)     │  1202        │          │
│                        ╱╲                        └──────────────┘          │
│                   LM  ▕▂▂▏← plume ∝ throttle        ALT 33,500 ft          │
│                       ╱  ╲                          ḢDOT −120 ft/s         │
│ ── lunar terrain ──────────────▄▄▀▀▄▄──────────────────────────── ▪ site   │
│                            West Crater                                     │
├────────────────────────────────────────────────────────────────────────────┤
│ [DSKY: PROG 63 VERB 16 NOUN 68 · R1 R2 R3 · PROG lamp · keyboard]          │
│ [ACA joystick + ROD switch + AUTO/ATT HOLD]  [CORE 8/8 · VAC 5/5 · FREE %] │
├────────────────────────────────────────────────────────────────────────────┤
│ ⏮ ◀ ▶ ⏭   ⏯   rates: .1× .25× 1× 4× 16×   step: −10ms +10ms +2s   🔊 CC   │
│ |—•———•——•———————•———•—•—•——•——•————| scrub bar with event pips            │
└────────────────────────────────────────────────────────────────────────────┘
```

- **Sky:** near-black (`#050608`), deterministic starfield (seeded, 2–3 parallax layers,
  subtle twinkle, none below the horizon). After the 102:36:55 yaw-around, Earth rises
  into the window view — exactly why Armstrong rolled the LM.
- **Moon:** side-view terrain strip (grey ramp with craters), the landing area, and
  **West Crater** — the boulder field Armstrong flew over in P66. Terrain scrolls with
  downrange position; a marker shows the current computed landing site (moves on LPD
  redesignation).
- **Lander:** 2D vector LM (descent + ascent stage, gear, bell). Pitch follows the
  attitude profile; plume length ∝ throttle (10% → FTP → throttle-down at 102:39:31 →
  P66 modulation); dust sheets below ~100 ft; contact probes; contact light at ~5 ft.
- **Camera:** logarithmic auto-zoom — braking phase framed with an altitude ladder,
  tight close-up through P66 and touchdown.
- **Captions:** the air-to-ground transcript lines from `timeline.markdown`, at their
  logged GETs, with a **light-delay toggle** (±1.3 s) to show what the crew actually
  heard when.
- **Explainers:** clicking any alarm pip opens a card summarizing the cause and linking
  to `memory_leak.md` / `alarm_recovery.md` / `radar_problem.md`.

### Time controls (hard requirements)

| Control | Behavior |
| :--- | :--- |
| Play / Pause | Space; frame-accurate freeze |
| Rates | 0.1× (slow play), 0.25×, 1× (real time), 4×, 16× |
| **Step +10 ms / −10 ms** | `.` / `,` while paused; advances/rewinds AGC time exactly 10 ms (backward = seek) |
| Step +2 s | One full guidance cycle (`]` on the cycle boundary) |
| Scrub bar | Whole window 102:32:00 → 102:46:10, event pips (alarms red, keystrokes amber, program changes cyan, voice grey), snap-to-pip |
| Jump to event | Prev/next event buttons + clickable event index (the §4 table) |

---

## 3. Modes: replay is the default, the TUI engine is the source

- **Mode A — Replay (default, static hosting).** The site plays `flight.json`, a frame
  stream **pre-recorded from the `exec-tui/sim` engine** running the historical script
  (§5, `cmd/record`). No backend needed. Scrubbing/stepping reads frames; 10 ms frame
  cadence makes the 10 ms step exact.
- **Mode B — Live sim ("fly it yourself").** The site connects over WebSocket to a small
  Go bridge embedding the same engine. Everything interactive becomes real: type on the
  DSKY, key V16N68, toggle the RR bug, click the ACA, flip ATT HOLD, enter P66 — and
  either reproduce the alarms or prevent them. This is the "hooked up to the TUI"
  requirement in the strongest sense: it *is* the TUI's engine.
- Mode B degrades to Mode A when no bridge is reachable (banner + fallback).

---

## 4. Ground truth: the validated replay dataset

All values below are cross-checked against Cherry (MIT, 4 Aug 1969), Eyles (AAS 04-064),
NASA SP-4029, and the ALSJ transcript; they match `timeline.markdown` after this branch's
corrections. `~` = interpolated/approximate.

### 4.1 Event and keystroke script (`events.json`)

| GET | T+PDI | Program | Event / DSKY activity | Altitude | ḢDOT |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ~102:27:00 | −365 s | P63 | Crew keys **V37E 63E**; P63 ignition algorithm runs (exact GET: M0 item) | ~50,000 ft | ~0 |
| pre-PDI | — | P63 | **Alarm code 500** (LR antenna position discrete); crew cycles switches, clears (exact GET: M0 item) | — | — |
| 102:32:30 | −35 s | P63 | **V06 N62** blanks (5 s), returns at T−30 s — Average-G running | ~49,971 ft | ~0 |
| 102:32:58 | −7.5 s | P63 | Ullage — RCS settles propellant | — | — |
| 102:33:00 | −5 s | P63 | Display flashes for crew go; **Aldrin keys PRO** | — | — |
| 102:33:05.01 | +0 s | P63 | **PDI** — ignition at 10%; **V06 N63**: R1 +5559.7 (velocity), R2 −2.2 (ḢDOT), R3 +49971 (alt) | 49,971 ft | 2.2 ft/s |
| 102:33:31 | +26 s | P63 | Throttle up to FTP (~9,870 lb); guidance enabled | ~49,000 ft | — |
| 102:36:55 | +230 s | P63 | Armstrong yaws face-up (rate switch 5→25 deg/s); Earth in the windows | — | — |
| ~102:37:53 | ~+288 s | P63 | Landing radar **"data good"** | ~35,000  ft | — |
| ~102:38:04 | ~+299 s | P63 | **Aldrin keys V16 N68 E** — DELTAH monitor; R3 −02900 (callout 102:38:06) | ~34,000 ft | — |
| **102:38:22** | **+317 s** | P63 | **ALARM 1 — 1202** (no core sets). PROG lamp; DSKY reverts **V06 N63**; crew reads code with **V05 N09 E** | **~33,500 ft** | ~120 ft/s |
| 102:38:42 | +337 s | P63 | Armstrong: *"Give us a reading on the 1202 Program Alarm."* | ~32,000 ft | — |
| 102:38:53 | +348 s | P63 | Duke: *"We're Go on that alarm."* (crew hears ~:54) | ~31,000 ft | — |
| ~102:38:55 | ~+350 s | P63 | **Aldrin keys V57 E** (accept LR updates), re-keys **V16 N68 E**; DELTAH → ~900 ft | ~30,000 ft | — |
| **102:39:02** | **+357 s** | P63 | **ALARM 2 — 1202**; V05 N09 readback again | **~29,000 ft** | ~125 ft/s |
| 102:39:14 | +369 s | P63 | Aldrin: *"…it appears to come up when we have a 1668 up."* | ~27,000 ft | — |
| 102:39:31 | +386 s | P63 | **Throttle down** — on time (*"better than the simulator"*) | ~24,500 ft | — |
| 102:41:32 | +507 s | **P64** | **High gate.** Pitch-over; **V06 N64**: R1 = LPD time-left + LPD angle, R3 alt. LPD/redesignation logic active | **7,400 ft** | 125 ft/s |
| 102:42:10 | +545 s | P64 | Duke: *"You're Go for landing."* | ~3,500 ft | — |
| **102:42:18** | **+553 s** | P64 | **ALARM 3 — 1201** (no VAC areas) | **~3,000 ft** | ~60 ft/s |
| **102:42:43** | **+578 s** | P64 | **ALARM 4 — 1202** | **~2,000 ft** | ~50 ft/s |
| late P64 | — | P64 | One **inadvertent LPD redesignation** (Eyles/debriefing; exact GET unknown — M0 item) | — | — |
| **102:42:58** | **+593 s** | P64 | **ALARM 5 — 1202** (last) | **770 ft** | 27 ft/s |
| 102:43:08 | +603 s | P64 | **AUTO → ATT HOLD** — Armstrong takes attitude, sheds load; no further alarms | ~650 ft | ~20 ft/s |
| 102:43:20 | +615 s | **P66** | **ROD switch flick enters P66**; joystick = rate-command attitude, ROD = ±1 ft/s per click | ~430 ft | ~15 ft/s |
| ~102:44:40 | ~+695 s | P66 | Quantity light; *"100 feet, 3½ down, 9 forward. Five percent."* | ~100 ft | 3.5 ft/s |
| 102:45:40 | +755 s | P66 | **Contact light / TOUCHDOWN** in the Sea of Tranquility | 0 ft | 0 |
| 102:45:58 | +773 s | — | *"Houston, Tranquility Base here. The Eagle has landed."* | — | — |

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

**M0 completes this table** by transcribing every Aldrin altitude/rate callout from the
ALSJ transcript between 102:43:01 (*"750, 23"*) and touchdown (~20 additional anchors:
700/21, 600/19, 540/15, 400/9, 350/4, 300/3.5, 270, 250/2.5, 220, 200/4.5, 160/6.5,
100/3.5, 75, 60/2.5, 40/2.5, 30/2.5, 20/0.5, contact).

### 4.3 Computer-state truth (from the engine, not hand-authored)

Program phases, verb/noun, R1–R3, PROG lamp, FAILREG contents, core-set/VAC occupancy,
free-compute %, duty breakdown, stub count, and restart events all come from
`exec-tui/sim` frames — the constants are already sourced in `exec-tui/RESEARCH.md`
(15.0% RR theft = 2 × 6,400 × 11.72 µs; duty margins >15% → ≈13% → ≤10% → <10%;
SERVICER 1,320 ms base; monitor 30 ms/s; P64 +60 ms/cycle; restart 20 ms + 20 ms
REREADAC). The DSKY noun register layouts for N62/N63/N64/N68 (and P66's noun) are
extracted in M0 from `Luminary099/PINBALL_NOUN_TABLES.agc` so the replica shows exactly
what Luminary 099 defined.

---

## 5. Architecture

```text
exec-tui/sim (Go)            ──┐  existing engine + M1 additions:
  GET anchor · flight script   │  Snapshot/Restore · determinism ·
  ACA/ROD/redesignation input  │  LPD state · trajectory hooks
                               │
exec-tui/cmd/record (Go) ──────┼──► flight.json (10 ms StateFrames) + events.json
exec-tui/cmd/bridge (Go) ──────┘──► WebSocket: StateFrames out, ControlCommands in
                                          │
descent-web/ (TypeScript + Vite)          ▼
  src/data      loaders + schema validation (events, trajectory, flight)
  src/playback  clock, rates, 10 ms step, seek, event index
  src/scene     canvas renderer: starfield, terrain, LM, plume, dust, camera
  src/panels    DSKY replica, ACA/ROD widgets, core/VAC board, duty bar, captions
  src/net       Mode B WebSocket client with Mode A fallback
```

Decisions (made now so implementation doesn't relitigate them):

1. **The engine stays in Go and stays canonical.** No TypeScript re-implementation of
   the Executive. Mode A consumes recorded engine output; Mode B talks to the live
   engine. (WASM compilation of `sim` is a possible later optimization for offline
   Mode B, explicitly out of scope.)
2. **Scrubbing = keyframe + deterministic replay.** Engine snapshots at every 2 s cycle
   boundary; seeks restore the nearest earlier keyframe and replay deterministically.
   Backward 10 ms steps are seeks. In Mode A this is trivial (indexed frames).
3. **Canvas 2D, side view.** A 2D profile view (downrange × altitude) tells this story
   better than 3D, matches the data we actually have, and keeps the site dependency-free
   (no WebGL framework). Panels are DOM.
4. **Trajectory is data, not physics.** The visual flight path replays the historical
   record (§4.2); the computer simulation replays the computer. They are joined by GET.
   In Mode B, user actions change *computer* history faithfully; trajectory deviates only
   in P66-style rate/attitude response (documented approximation, banner shown).

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

`kind ∈ {alarm, keystroke, program, flight, voice, switch}`. Keystroke events carry the
full key list with per-key cadence (the TUI's 230–330 ms pattern) so the DSKY keyboard
lights replay realistically.

### 6.2 `StateFrame` (bridge + flight.json; ~10 ms cadence)

```json
{
  "agcMs": 317000, "get": "102:38:22.00", "phase": "P63",
  "dsky": { "prog": "63", "verb": "16", "noun": "68",
            "r1": "+05559", "r2": "-00120", "r3": "-02900",
            "progLamp": true, "compActy": true, "typing": false },
  "failreg": ["01202"],
  "exec": { "coreSets": [{"owner": "SERVICER", "prio": 20, "stub": true}],
            "vacs": [{"owner": "SERVICER", "stub": false}],
            "freePct": -3.1, "runningJob": "CHARIN", "stubs": 4,
            "restarts": 1, "cycleMs": 1240, "monitor": true },
  "traj": { "altFt": 33500, "hdotFps": -120, "vFps": 1200,
            "downrangeFt": 91000, "pitchDeg": 62, "throttlePct": 92.5 },
  "events": ["alarm-1202-1"]
}
```

### 6.3 `ControlCommand` (Mode B, client → bridge)

```json
{ "op": "play" } { "op": "pause" } { "op": "rate", "x": 0.25 }
{ "op": "stepMs", "ms": 10 } { "op": "seekGet", "get": "102:41:32" }
{ "op": "key", "k": "V" } { "op": "joyClick", "axis": "pitch", "dir": 1 }
{ "op": "rod", "dir": -1 } { "op": "attHold" } { "op": "reset" }
{ "op": "scenario", "radarBug": true, "monitor": true }
```

Every command is acknowledged with the next frame or `{"op":"error","reason":"…"}`.

## 7. The joystick (ACA) and ROD switch

The physical controls Armstrong used, and what each input costs the computer:

- **P64 (102:41:32 → 102:43:08), LPD redesignation.** N64 R1 shows LPD time-left and the
  LPD angle Armstrong sighted along his window reticle. While time remains, each ACA
  click redesignates the site — pitch clicks shift it along-track, roll clicks
  cross-track (published quanta ≈ 0.5° elevation / 2° azimuth per click; confirm from
  Klumpp R-695 in M0). **Each click queues extra guidance retargeting in the next
  SERVICER pass** — the user's intuition is correct: joystick activity adds computation,
  and P64's redesignation logic is precisely the protected load that made its alarms
  unshedable. In Mode B, clicking during the overload measurably deepens the knife edge.
  The site renders the reticle + site marker moving per click; the historical script
  includes the one inadvertent redesignation (M0 pins its time or marks it "late P64").
- **ATT HOLD (102:43:08).** Joystick becomes rate-command/attitude-hold: deflection
  commands a rate, release holds attitude. Autopilot cost drops (the engine's ATT HOLD
  DAP model) — visibly recovering free-compute on the duty bar, which is *why* the alarms
  stopped.
- **P66 (102:43:20).** ROD switch: one click = ±1 ft/s commanded descent rate; SERVICER's
  P66 profile (900 ms) replaces the full guidance load. The widget is a vertical
  spring-loaded toggle next to the ACA; clicks show as ±1 ft/s deltas on the ḢDOT tape.

Widget spec: on-screen ACA (draggable/arrow keys, detent center, click quantization in
P64), ROD toggle, and the AUTO/ATT HOLD mode switch — disabled states grey out with a
tooltip explaining *why* (e.g. "LPD time expired").

## 8. Milestones (each begins by landing its §1 tests, failing)

| # | Deliverable | Contents |
| :--- | :--- | :--- |
| M0 | Data | `events.json`, `trajectory.json` from §4; noun layouts from `PINBALL_NOUN_TABLES.agc`; ALSJ P66 callout transcription; resolve open facts (P63 selection GET, code-500 GET, LPD quanta, inadvertent-redesignation time); validation script wired into CI |
| M1 | Engine | GET clock, flight-script driver, snapshot/restore, determinism, ACA/ROD/redesignation inputs, LPD state (tests §1.2) |
| M2 | Record + bridge | `cmd/record` → `flight.json`; `cmd/bridge` WebSocket server (tests §1.3) |
| M3 | Web scene | Vite scaffold, loaders, playback clock, starfield/terrain/LM scene from `flight.json` (tests §1.4: events, trajectory, playback, lander) |
| M4 | Panels + transport | DSKY replica, captions, core/VAC + duty panels, transport bar, scrub, ±10 ms step, event index (tests §1.4: dsky, hud) |
| M5 | Controls + Mode B | ACA/ROD/mode-switch widgets, live bridge client, scenario toggles (tests §1.4: joystick) |
| M6 | Polish + e2e | Camera, dust, light-delay toggle, explainer cards, Playwright e2e (tests §1.5), README |

Tooling: Go 1.26 (`nhooyr.io/websocket` or stdlib), Vite + TypeScript + Vitest +
Playwright, Canvas 2D, zero runtime UI framework dependencies unless M3 review demands
one. New code lives in `descent-web/` and `exec-tui/cmd/`; `exec-tui/sim` grows but its
existing API and tests stay intact.

## 9. Risks and open questions

- **Sub-second truth.** GETs are known to the second (alarms also as Cherry PDI offsets);
  the 10 ms step is a *navigation* affordance over engine frames, not a claim of 10 ms
  historical telemetry. The UI says so in the event cards.
- **First-alarm labeling.** Mission Report Table 5-I labels the first 1202 at 102:39:02;
  SP-4029 and Cherry support 102:38:22. We follow `timeline.markdown` (both listed, main
  scrub pips at SP-4029/Cherry times).
- **Trajectory fidelity between anchors** is interpolation; M0's ALSJ transcription makes
  P66 dense, but P63 mid-phase altitude is ~±1,000 ft. Acceptable for a profile view;
  document in the site's "accuracy" page.
- **Mode B trajectory divergence** (user flies differently than history) is approximate
  by design; the banner + docs must be honest about it.
- **LPD click quanta and the inadvertent redesignation's GET** need M0 source
  confirmation before the joystick ships.
