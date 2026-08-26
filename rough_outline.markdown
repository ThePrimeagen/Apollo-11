# Rough Outline — What Ran While the Descent Engine Burned

Every program on the LGC during powered descent: its interval, its cost, and whether it
held a core set / VAC area. Sources: [Cherry] job table, [Eyles], [L099] source in this
repo. Details in [`operations_and_timing.md`](operations_and_timing.md).

## The two pools (what overflowed)

- **8 core sets** (12 words each) — *every* Executive job holds one. Pool empty → **1202**.
- **5 VAC areas** (44 words each) — only interpretive (vector-math) jobs hold one *in
  addition*. Pool empty → **1201**.
- Tasks (WAITLIST) and interrupts take **neither** — they can't cause either alarm.

## The full map — every program, its interval, what it held

| Program | Kind | Interval | CPU per 2 s cycle | Core set held | VAC held |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **SERVICER** (all of guidance: average-G nav, LR incorporation, P63/P64 equations, throttle, FINDCDUW, display prep) | job, prio 20, FINDVAC | every 2.000 s | ~1.30–1.45 s (65–72%) | yes — full run (~1.4 s healthy; **forever if unfinished** — the leak) | yes — same span |
| **READACCS** (read accelerometers, spawn SERVICER, re-arm +2 s) | WAITLIST task | every 2.000 s | ~1 ms | no | no |
| **MAKEPLAY** (display job SERVICER sets up) | job, prio 20, FINDVAC | 1 per cycle | ~ms | yes, briefly | yes, briefly |
| **MONDO** — the V16N68 DELTAH monitor | job, prio 30, NOVAC | refresh 1/s (only while keyed) | ~30–60 ms | yes, incl. ~250 ms display-wait sleep | **no** |
| **CHARIN** (each DSKY keystroke) | job, prio 30, NOVAC | per keystroke | ~5 ms | yes, incl. ~150 ms echo wait | **no** |
| **LRHJOB** (LR range read, below 25 kft) | job, prio 32, NOVAC | 1 per cycle, fired 50 ms *before* each READACCS | ~2 ms | yes ~82 ms (1 ms run + 80 ms sleep + 1 ms), **straddling the cycle boundary** | **no** |
| **LRVJOB** (LR velocity, 1 beam/cycle) | job, prio 32, NOVAC | 1 per cycle | ~2 ms | yes ~500 ms (sleeps through 5 samples) | **no** |
| **1/GYRO** (IMU compensation) | job, prio 21, NOVAC | ~1/s (phase-offset from the 2 s mark) | ~7 ms | yes, briefly | **no** |
| **HIGATJOB** (LR antenna to position 2) | job, prio 32→23, FINDVAC | once, at P64 entry | ~2 ms | yes ~8 s | **yes ~8 s** (parks on a VAC awaiting the discrete) |
| **RODCOMP** (P66 rate-of-descent update, from ROD switch clicks) | job, prio 22, FINDVAC | every 1 s (`RODTASK`), P66 only | ~ms | yes, briefly | yes, briefly |
| **DAP** (digital autopilot — points the thrust vector, fires RCS) | T5RUPT interrupt | every 100 ms | ~0.24 s (12%) | no | no |
| **T4RUPT** (DSKY relays, housekeeping) | interrupt | every 120 ms | ~16 ms (0.8%) | no | no |
| **DOWNRUPT** (telemetry downlink) | interrupt | every 20 ms (50/s) | ~20 ms (1%) | no | no |
| **R10/R11** (cockpit cross-pointer/analog displays) | task | every 0.25 s | ~ms | no | no |
| **RR CDU counter theft** (the bug) | hardware counter steal | 12,800/s, continuous | **0.30 s (15%)** | no — steals time, zero memory | no |

## One 2-second cycle, mapped

```text
t=0        READACCS task (1 ms): read PIPAs, FINDVAC a NEW SERVICER, re-arm t+2.000 s
t=0–~1.4s  SERVICER runs at prio 20 (holds core set + VAC), constantly preempted by:
             every  20 ms   DOWNRUPT      (0.2 ms)
             every 100 ms   DAP           (12 ms)
             every 120 ms   T4RUPT        (1 ms)
             every 250 ms   R10/R11 task
             ~mid-cycle     LRVJOB: 1 ms, sleeps ~500 ms HOLDING A CORE SET
             1/s            MONDO refresh (30–60 ms, prio 30) — only if V16N68 keyed
             1/s            1/GYRO (7 ms, prio 21), phase-offset
             any time       CHARIN per keystroke (5 ms, prio 30, holds a core set)
t=1.950s   LRHJOB: 1 ms, sleeps 80 ms holding a core set ACROSS the boundary
t=2.000s   next READACCS — new SERVICER allocated whether or not the old one finished
all along  RR CDUs steal 11.72 µs 12,800×/s ≈ 0.30 s of the 2.00 s, invisibly
```

## Percent of expectations, by phase

Duty cycle = all known software (jobs + tasks + interrupts). The 15% theft sits on top.

| Phase | Known software | + RR theft | Result |
| :--- | :--- | :--- | :--- |
| P63 before LR lock | < 85% | ~100% | quiet knife edge — no alarms for ~5 min |
| P63 + radar locked | ~87% | ~102% | leaking slowly |
| P63 + V16N68 keyed | ≥ 90% | **~105%** | 1202 in ~6 cycles (12 s) — twice |
| P64 (redesignation, no monitor) | > 90% | **> 105%** | unsheddable: 1201 + 1202 + 1202 in 40 s |
| P66 (ROD; SERVICER drops to ~0.9 s profile) | lower | < 100% | **zero alarms**, 2 min 20 s to touchdown |

Design requirement was 10% margin for *unknown* loss; the unknown loss turned out to be
13–15%. Deficit ≈ 0.05–0.10 s per cycle → about one leaked core-set+VAC pair per cycle.

## Reconciling: flight = core sets 4/5 times; our sim = almost always 1201

Flight record: **1202, 1202, 1201, 1202, 1202** — core sets ran out 4 of 5 times, VAC
areas once. A naive sim does the opposite (1201s), for structural reasons:

1. **In a stubs-only model, every leaked allocation is a core-set+VAC pair.** SERVICER
   stubs pile up in lockstep, 5 VACs < 8 core sets, and FINDVAC scans VACs *first* — so
   the VAC wall always trips first. Always 1201.
2. **What the model misses is the NOVAC traffic.** Only three job types ever took a VAC
   (SERVICER, MAKEPLAY, HIGATJOB — Cherry's table). Everything else — monitor refreshes,
   keystrokes, both radar reads, gyro comp — was core-set-only, and several of them
   **sleep holding the core set** (LRV ~500 ms and LRH ~82 ms every cycle, MONDO ~250 ms
   per refresh, CHARIN ~150 ms per keystroke — and Aldrin was typing constantly).
   That nibbles the 8-set pool down to effective parity with the 5-VAC pool.
3. **A NOVAC request can only ever raise 1202.** It skips the VAC scan entirely. During
   the overload most *allocation attempts* were these small NOVAC jobs (monitor 1/s,
   radar 2/cycle, keystrokes), so the failing request was usually one that reports
   "no core sets."
4. **The one 1201 is the exception that proves it:** at P64 entry HIGATJOB parked on a
   VAC for ~8 s while stub VACs accumulated — a FINDVAC caller hit the 5-VAC wall first,
   once, at 102:42:18.

Status in `exec-tui`: the sleep mechanics were added after the audit
(`exec-tui/RESEARCH.md` §"Alarm-code fidelity") and the P63-with-typing case now yields
**1202** (~10 s after the monitor; flight ~12 s), P64's first alarm **1201** — matching
flight. Known residual gap: the flight's *later* P64 alarms were 1202s, the sim's
recurrences are still 1201s, because the sim's P64 has no DSKY/NOVAC churn. To close it,
model P64 crew activity (keystrokes; the one inadvertent LPD redesignation) so core-set
holders exist at the recurring failure points.

For mission-time context see [`rough_descent_timeline.markdown`](rough_descent_timeline.markdown).
