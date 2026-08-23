# RESEARCH — Every number in the simulation, with its source

Primary sources:

- **[Eyles]** Don Eyles, *Tales From the Lunar Module Guidance Computer*, AAS 04-064
  (2004). First-hand author of the landing software.
- **[Cherry]** George W. Cherry, *Exegesis of the 1201 and 1202 Alarms* (MIT, Aug 1969).
- **[L099]** The Luminary 099 assembly source in this repository (`Luminary099/`).
- **[Repo]** This repository's annotated traces: `radar_problem.md`, `memory_leak.md`,
  `alarm_recovery.md`, `definitions.md`, `timeline.markdown`.
- **[Tillman]** Clint Tillman (Grumman), RR-CDU interface simulation report, Aug 9 1969
  (as cited by [Eyles]).

## Machine constants

| Constant | Value | Source |
| :--- | :--- | :--- |
| Memory cycle time (MCT) | 11.72 µs | [Eyles] "The memory-cycle time for the AGC was 11.7 microseconds"; [Repo] definitions.md |
| Counter increment (PINC/MINC) cost | 1 MCT, invisible to software | [Eyles]; [Repo] radar_problem.md ("not a software instruction and not an interrupt") |
| Erasable (RAM) | 2,048 words × 15 bits | [Eyles]; [Repo] |
| Core sets | **8 × 12 words** | [Eyles] "an array of eight such core sets of 12 registers each"; [L099] `COREINC DEC 12`, `ERASE +83D  # EIGHT SETS OF 12 REGISTERS EACH` |
| VAC areas | **5 × 44 words** (43 workspace + 1 use flag) | [Eyles] "There were five such Vector Accumulator (VAC) areas"; [L099] `VACn ERASE +42D`, `VACnUSE` |
| Interpretive vector cross product | ≈ 5 ms | [Eyles] |

## The rendezvous radar bug (the "TLOSS")

| Fact | Value | Source |
| :--- | :--- | :--- |
| Cause | RR mode switch in SLEW/AUTO; ATCA 800 Hz excitation frequency-locked but **not phase-synchronized** with the CDU reference; worst near 90°/270° | [Eyles]; [Repo] radar_problem.md |
| Counter rate | 6,400 pulses/s **per angle**, 2 angles (shaft `CDUS`, trunnion `CDUT`) | [Eyles] "at the maximum rate of 6400 pulses per seconds for each angle"; [L099] `CDUS EQUALS 36`, `CDUT EQUALS 35` |
| Theft | 2 × 6,400 × 11.72 µs = **0.150 s/s ≈ 15.0%** | [Eyles] "consumed approximately 15% of the available computation time"; [Repo] radar_problem.md |
| Grumman's measured worst case | 13.36% (used by MIT as ~13%) | [Tillman] via [Eyles] |
| Software tested against | ~10% unexplained TLOSS | [Repo] radar_problem.md |
| What it does NOT cost | Memory. Zero words. Pure time. | [Repo] radar_problem.md, memory_leak.md |

The simulation uses the theoretical max **15.0%** when the bug is enabled (the flight
value fluctuated 13–15%; a `RadarBugTLOSS` constant holds 6400 × 2 × 11.72 µs exactly).

## Duty cycle by configuration ([Eyles], Figure 8 narrative)

| Configuration | Duty cycle | Margin |
| :--- | :--- | :--- |
| P63 braking, before landing-radar lock | < 85% | > 15% |
| After landing radar acquired ("data good") | ≈ 87% | ≈ 13% |
| + V16 N68 (DELTAH monitor) keyed up | ≥ 90% | ≤ 10% |
| P64 (landing-site redesignation logic) | > 90% | < 10% |

Simulation budget chosen to hit those aggregates (per 2.000 s cycle, in AGC ms):

| Component | Cost | Duty | Rationale |
| :--- | :--- | :--- | :--- |
| SERVICER (prio 20, FINDVAC) | 1,320 ms/cycle | 66% | The dominant job: average-G nav → guidance → throttle → attitude → displays [Eyles] |
| DAP (autopilot interrupt, 10 Hz) | 12 ms per 100 ms | 12% | [Eyles] lists the digital autopilot among dedicated interrupts; 10 Hz is the LM DAP RCS period |
| READACCS (waitlist task, 2 s) | 1 ms | ~0.05% | "deliberately short" [Repo] memory_leak.md |
| GYRO COMP (prio 21, 1 Hz) | 7 ms | 0.7% | priority from [Repo] memory_leak.md job table; fires at a +137 ms phase offset from the 2 s boundary (anti-collision phasing, cf. SERVICER.agc's deliberate 70 ms READACCS/DAP offset) |
| T4RUPT (120 ms) | 0.96 ms per fire | ~0.8% | [L099] T4RUPT_PROGRAM.agc: `120MS`; RRAUTCHK "entered every 480 MS" |
| DOWNRUPT telemetry (50/s) | 0.2 ms per fire | 1% | downlink 50 words/s |
| PIPA + misc counters | — | 0.5% | powered-flight accelerometer traffic |
| **Total P63 base** | | **≈ 81%** | < 85% ✓ |
| + LR data conversion (in SERVICER) | +70 ms/cycle | +3.5% | "extra computations involved in converting the body-referenced radar data" [Eyles]; sized so P63+LR+bug ≈ 99.8% (see the 13-vs-15 note below) |
| + LRHJOB / LRVJOB (prio 32, per cycle) | ~4 ms CPU per cycle | +0.2% | ~1 ms run + sleep + ~1 ms run [Cherry job table]; the cost is memory-holding, not CPU |
| + V16N68 MONITOR (prio 30, 1 Hz) | 30 ms CPU per refresh | +3% | monitor verbs are "DISPLAYS THAT ARE UPDATED ONCE PER SECOND" [L099] PINBALL; margin 13% → ≤10% [Eyles] |
| + P64 redesignation (in SERVICER) | +60 ms/cycle | +3% | "Added to the regular guidance equations was new processing" [Eyles] |

**The 13-vs-15 reconciliation.** The duty percentages above sit ~2.5% under Eyles'
narrative because the sim steals the theoretical **15.0%** while the flight lost ~13%
(Grumman's measured 13.36%). What determines behavior is the **aggregate demand**, and
that is what the budget preserves: P63+LR+bug ≈ **99.8%** — the flight's quiet knife
edge (theft active, no monitor, no alarms for ~5 minutes) — and adding the monitor's
~3.3% tips it past 100%, exactly as on July 20, 1969.

## Sleep mechanics (JOBSLEEP / JOBWAKE)

Jobs can run a head segment, **sleep holding their core set (and VAC)**, then wake for a
tail segment. This is the missing memory pressure that shaped the flight's alarm codes:

| Job | Head / sleep / tail | Source |
| :--- | :--- | :--- |
| LRHJOB (prio 32, per cycle) | 1 ms / 80 ms / 1 ms, fired **50 ms before each READACCS** so the gate straddles the boundary | [Cherry] "run for a millisecond or so and then sleep for about 80 milliseconds"; [L099] "50 MS PRIOR TO THE NEXT READACCS TASK" |
| LRVJOB (prio 32, per cycle) | 1 ms / 500 ms / 1 ms, mid-cycle | [L099] "5 VELOCITY SAMPLES AND GOES TO SLEEP WHILE THE SAMPLING IS DONE -- ABOUT 500 MS" |
| MONITOR (MONDO, prio 30, 1 Hz) | 15 ms / 250 ms / 15 ms | display-wait sleep per [L099] PINBALL (display users sleep holding their core set; alarm 1206 exists for a second display sleeper); duration est. |
| CHARIN (prio 30, per keystroke) | 3 ms / 150 ms / 2 ms | DSPTAB echo wait, same PINBALL mechanism; duration est. |
| HIGATJOB (prio 32, VAC, one-shot at P64) | 2 ms / 8 s / 2 ms | [Cherry] "Sleeps until position #2 discrete is received"; antenna slew time est. |

A woken job preempts like a fresh schedule (real JOBWAKE set NEWJOB). Sleeping jobs are
skipped by the EJSCAN.

## Alarm-code fidelity (which pool empties first)

Flight record: P63 gave **1202** twice (no core sets); P64 gave **1201** first (no VAC
areas). A stubs-only model can never produce the P63 1202s: every stub holds a core set
*and* a VAC, there are only 5 VACs against 8 core sets, and FINDVAC scans VACs first —
so the VAC wall always trips first (always 1201). The flight's 1202s therefore require
concurrent **core-set-only** holders at the failing moment, which is exactly what the
sleep mechanics supply: radar gates (LRV ~500 ms + LRH 80 ms per cycle) and display
waits (monitor refreshes, crew keystrokes). Reproduced behavior:

- P63 overload **with DSKY activity** (the flight case — Aldrin was working the DSKY
  throughout): a keystroke lands while 5 SERVICERs + LRV gate + MONDO wait + a prior
  CHARIN hold all 8 core sets → **1202**, ~10 s after the monitor (flight: ~12 s).
- P63 overload with the monitor but an idle keyboard: the VAC wall trips at the sixth
  READACCS → **1201**. The code genuinely depends on crew activity.
- P64: stub VACs plus HIGATJOB's held VAC hit the 5-VAC wall first → **1201** (flight:
  1201 at 102:42:17). Known residual deviation: the flight's *later* P64 alarms were
  1202s; the sim's recurrences are 1201s.

### The tie-break that makes the leak fast

`EXECUTIVE.agc` `SETLOC` lets a new job preempt only a *strictly greater* priority, and
the `EJSCAN` rescan runs at every job completion. The simulation resolves equal-priority
rescans in favor of the **most recently scheduled** copy. Justification: the flight
timeline (V16N68 keyed ≈102:38:10, first 1202 at 102:38:22 — about six 2-second cycles)
requires roughly one leaked core-set/VAC pair per overloaded cycle, which only happens
when new SERVICERs win and old stubs starve — precisely the accumulation of "uncompleted
SERVICER 'stubs'" Eyles describes the restart flushing. A strict FIFO/backlog model would
take minutes to exhaust the pools, contradicting the flight record. An almost-finished
old copy that nothing preempts still runs to completion and frees its pair (RECOVERED) —
which is why the quiet knife edge stays quiet.

## The DSKY keystroke pipeline

| Fact | Source |
| :--- | :--- |
| Each keypress puts a 5-bit code in channel 15 and fires interrupt **KEYRUPT1** | [L099] PINBALL comments: "EACH DEPRESSION OF A KEYBOARD BUTTON ACTIVATES AN INTERRUPT KEYRUPT1" |
| KEYRUPT1 schedules job **CHARIN** and resumes | [L099] KEYRUPT_UPRUPT.agc: `2CADR CHARIN`, "LEAVE 5 BIT KEY CDE IN MPAC FOR CHARIN" |
| CHARIN runs at **priority 30** — above SERVICER's 20 | [L099] PINBALL: `CHRPRIO OCT 30000` |
| Display output: T4RUPT's DSPOUT sends **one DSPTAB relay word per 120 ms pass** | [L099] T4RUPT_PROGRAM.agc DSPOUT |
| Monitor verbs refresh once per second | [L099] PINBALL: "MONITORS (DISPLAYS THAT ARE UPDATED ONCE PER SECOND)" |

Simulated cost per keystroke: KEYRUPT ~0.1 ms + CHARIN job 5 ms (core set, no VAC) +
queued DSPTAB changes adding ~0.5 ms to following T4RUPT passes. A human typing ~3 keys/s
costs ~1.5–2% duty — small, but *real*, and visible on the CHARIN timeline row; V16N68
(7 keystrokes: V-1-6-N-6-8-ENTR) then leaves the +3% monitor running. Aldrin keying it up
is what moved the margin from ~13% to ≤10% [Eyles]: "Buzz Aldrin was perceptive when he
said... 'It appears to come up when we have a 1668 up.'"

The software restart (BAILOUT → ENEMA → phase-table rebuild) is modeled as 20 ms of
CPU-blocking restart work plus a 20 ms delay before REREADAC recreates one SERVICER and
re-arms the READACCS chain — the accelerometer counters keep counting through it, so no
velocity data is lost [Repo] alarm_recovery.md.

## Executive / Waitlist mechanics

| Fact | Source |
| :--- | :--- |
| Highest-priority ready job always runs; preempted jobs keep core set/VAC | [Eyles] (Laning design) |
| SERVICER: priority 20, **lowest** of the landing jobs, longest; "got last crack at the available computation time" | [Eyles]; [L099] `CA PRIO20 / TC FINDVAC / 2CADR SERVICER` |
| Radar reads prio 32, keyboard 30, monitor ~30, gyro comp 21 | [Repo] memory_leak.md |
| READACCS re-arms every 2.000 s via `CA 2SECS / TC VARDELAY`; no "is the old copy done?" check | [L099] SERVICER.agc `GOREADAX`; [Repo] memory_leak.md |
| T3RUPT dispatches punctually regardless of CPU load | [L099] WAITLIST.agc; [Repo] memory_leak.md |
| FINDVAC scan: 5 × `CCS VACnUSE`; all busy → `TC BAILOUT1 / OCT 1201` | [L099] EXECUTIVE.agc `FINDVAC2` |
| Core-set scan: 8 probes (`NO.CORES DEC 7` + 1); all busy → `TC BAILOUT1 / OCT 1202` | [L099] EXECUTIVE.agc `NOVAC2/NOVAC3/NEXTCORE` |
| BAILOUT: code → first free of 3 `FAILREG` slots; PROG lamp via `DSPTAB +11D` bit; `WHIMPER` → `ENEMA` restart | [L099] ALARM_AND_ABORT.agc; [Repo] alarm_recovery.md |
| Restart: frees all 8 PRIORITY words + 5 VACnUSE, wipes waitlist, phase tables rebuild **one** REREADAC task + **one** prio-20 SERVICER; unprotected work (V16N68 monitor) vanishes | [L099] FRESH_START_AND_RESTART.agc, RESTART_TABLES.agc `5.4SPOT`; [Eyles] |
| P64 could not shed load → "The alarms kept coming. There were three 1201 and 1202 alarms within 40 seconds." | [Eyles] |
| Armstrong stopped them at 102:43:08 by AUTO → ATT HOLD, then P66 ("the burden was still lighter") | [Eyles]; [Repo] timeline.markdown |

## Flight timeline anchors ([Repo] timeline.markdown; NASA SP-4029; [Cherry])

Five alarms: 1202 (102:38:22, ~33,500 ft), 1202 (102:39:02), 1201 (102:42:18, ~3,000 ft),
1202 (102:42:43), 1202 (102:42:58, ~770 ft). V16N68 keyed ~102:38:10. P64 at 102:41:32
(7,400 ft). ATT HOLD 102:43:08. Touchdown 102:45:40.

## Time scale

User requirement: **1000 ms wall = 50 ms AGC** (20× slow motion). One 2.000 s guidance
cycle therefore plays out over 40 wall seconds. Engine step: 100 µs of AGC time.
