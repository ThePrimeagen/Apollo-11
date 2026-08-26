# Rough Outline — What Ran While the Descent Engine Burned

Every program on the LGC during powered descent: its interval, its cost, and whether it
held a core set / VAC area. Verified against the primary sources — Cherry's *Exegesis*
(MIT, 4 Aug 1969, incl. his Table of Jobs and Priorities), the Tillman memo (Grumman,
31 Jul 1969, LGC downlist analysis), Eyles (AAS 04-064), Adler (ALSJ), and the Luminary
099 source in this repo. Mission-time context in
[`rough_descent_timeline.markdown`](rough_descent_timeline.markdown).

## The two pools (what overflowed)

- **8 core sets** (12 words each) — *every* Executive job holds one. Pool empty → **1202**.
- **5 VAC areas** (44 words each) — interpretive (vector-math) jobs hold one *in
  addition*. Pool empty → **1201**.
- Tasks (WAITLIST) and interrupts take **neither**.
- Allocation order (`EXECUTIVE.agc`): FINDVAC scans the 5 VACs first, then the 8 core
  sets. NOVAC skips the VAC scan entirely — **a NOVAC request can only ever raise 1202.**

## The full map — Cherry's 1969 job table, verified against the code

| Program | Kind | Interval / trigger | CPU per 2 s cycle | Core set held | VAC held |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **SERVICER** (average-G nav, LR incorporation, P63/P64 guidance, throttle, FINDCDUW, display prep) | job, prio 20, FINDVAC | every 2.000 s via READACCS, unconditional | ~1.30–1.45 s (65–72%) | full run — **kept forever if a new copy is scheduled before it finishes (the leak)** | same span |
| **MAKEPLAY** (display job spawned by every guidance pass at DISPEXIT) | job, prio 20 (self-raises to 33), **NOVAC when static** (P63 V06N63, P66 V06N60), **SPVAC when flashing** (early-P64 V06N64) | 1 per cycle | ~ms | yes — and a blocked/flashing one **sleeps holding it until the crew responds** (next pass wakes-and-kills the sleeper) | **only the P64 flashing form** — sleeps holding it until PRO |
| **MONDO** — the V16N68 DELTAH monitor | job, prio 30, NOVAC | 1/s — MONREQ waitlist task re-arms itself every 1.00 s and spawns a fresh MONDO; keyed **once**, runs until killed | ~30–60 ms | yes, briefly — **never sleeps** (display busy → lights KEY REL, ends job) | no |
| **CHARIN** (each DSKY keystroke) | job, prio 30, NOVAC | per keystroke | ~5 ms | yes, briefly | no |
| **LRHJOB** (LR range read) | job, prio 32, NOVAC | 1 per cycle, fired 50 ms *before* each READACCS | ~2 ms | yes ~97 ms (1 ms + ~95 ms radar gate + 1 ms), **straddling the cycle boundary** | no |
| **LRVJOB** (LR velocity, 5 samples) | job, prio 32, NOVAC | 1 per cycle when velocity reads enabled | ~2 ms | yes ~500 ms (sleeps through sampling) | no |
| **HIGATJOB** (LR antenna to position 2) | job, prio 32→23, FINDVAC | once, ~6 s before high gate | ~2 ms | yes — sleeps on the position-2 discrete, **up to 22 s** | **yes, same span** |
| **1/GYRO** (gyro compensation, from 1/PIPA inside SERVICER) | job, prio 21, NOVAC | per cycle when accumulated compensation ≥ 2 pulses (~1/s) | ~7 ms | yes, briefly | no |
| **RODCOMP** (P66 rate-of-descent update) | job, prio 22, FINDVAC | every 1 s (RODTASK), P66 only | ~ms | yes, briefly | yes, briefly |
| **READACCS** (read PIPAs, spawn SERVICER, re-arm +2 s) | WAITLIST task | every 2.000 s | ~1 ms | no | no |
| **R10/R11 → LANDISP** (tape meters / cross-pointers) | task | every 0.25 s | ~ms | no | no |
| **DAP** (digital autopilot) | T5RUPT interrupt | every 100 ms | ~0.24 s (12%) | no | no |
| **T4RUPT** (DSKY relays, monitors) | interrupt | every 120 ms | ~16 ms (0.8%) | no | no |
| **DOWNRUPT** (telemetry) | interrupt | every 20 ms (50/s) | ~20 ms (1%) | no | no |
| **RR CDU counter theft** (the bug) | hardware counter steal | 12,800/s continuous | **0.30 s (15%)** | no — time only, zero memory | no |

Only **SERVICER, MAKEPLAY (flashing form), HIGATJOB** — plus P66's RODCOMP — ever take a
VAC. Everything else is NOVAC. In a P63 cycle with the monitor up, roughly **six of the
seven allocation requests are NOVAC** (display job, 2× MONDO, LRH, LRV, gyro) and can
only fail as 1202.

## One 2-second cycle, mapped (P63, radar locked, monitor up)

```text
t=0        READACCS task (1 ms): read PIPAs, FINDVAC a NEW SERVICER, re-arm t+2.000 s
t=0–~1.4s  SERVICER runs at prio 20 (core set + VAC), preempted by:
             every  20 ms   DOWNRUPT      every 100 ms   DAP
             every 120 ms   T4RUPT        every 250 ms   R10/R11 task
             mid-cycle      LRVJOB: 1 ms, sleeps ~500 ms HOLDING A CORE SET
             every 1 s      MONDO refresh (NOVAC, prio 30) — automatic, no typing
             ~1/s           1/GYRO (7 ms, prio 21)
end of pass DISPEXIT spawns the display job (NOVAC in P63) — one more core set
t=1.950s   LRHJOB: 1 ms, sleeps ~95 ms holding a core set ACROSS the boundary
t=2.000s   next READACCS — new SERVICER allocated whether or not the old one finished
all along  RR CDUs steal 11.72 µs 12,800×/s ≈ 0.30 s of the 2.00 s, invisibly
```

## Percent of expectations, by phase

Duty cycle = all known software (jobs + tasks + interrupts). The ~13–15% theft sits on top.

| Phase | Known software | + RR theft | Result |
| :--- | :--- | :--- | :--- |
| P63 before LR lock | < 85% | ~100% | quiet knife edge — no alarms for ~5 min |
| P63 + radar locked | ~87% | ~102% | leaking slowly |
| P63 + V16N68 keyed | ≥ 90% | **~105%** | 1202 ~12 s after each monitor start |
| P64 (redesignation, no monitor) | > 90% | **> 105%** | unsheddable: 1201 + 1202 + 1202 in 40 s |
| P66 (ROD; SERVICER ~0.9 s profile) | lower | < 100% | **zero alarms**, 2 min 20 s to touchdown |

## Why the flight threw 1202 four times and 1201 once

Documented facts (Cherry, Tillman, Adler, and the code — not conjecture):

1. **No typing was involved.** The monitor is keyed once; MONREQ then spawns a NOVAC
   MONDO job every second on its own. Both P63 alarms came "after 12 seconds of a
   monitor verb" (Tillman). The three P64 alarms came "**with no Crew DSKY activity**"
   (Tillman, from the downlist). No 1969 source attributes any alarm to keystrokes.
2. **The alarm code names whichever pool the failing request found empty** (Adler), and
   most requests during descent are NOVAC → 1202-only.
3. **The display pipeline is the swing factor.** P63/P66 display jobs are core-set-only.
   Early P64's *flashing* V06N64 takes a core set + VAC and **sleeps holding both until
   the crew keys PRO** — which Cherry's event log puts at +568, *after* the 1201 (+554)
   and *before* the two P64 1202s (+578, +594). HIGATJOB also parks on a VAC (≤22 s)
   around P64 entry and is not rebuilt after the restart.

Reconstruction consistent with all of the above (the 1969 sources never named the
failing request per alarm — Cherry's deduction stops at "some job like SERVICER was
scheduled two and possibly three or four times"):

```text
P63 (alarms 1–2):   SERVICER pairs leak + heavy NOVAC churn (display job each cycle,
                    MONDO 1/s, LR gates) → the 8-core wall first → 1202, 1202
early P64 (alarm 3): flashing-V06N64 job asleep on a VAC + HIGATJOB asleep on a VAC
                    + SERVICER stub VACs → the 5-VAC wall first → 1201
late P64 (alarms 4–5): after PRO (+568) the display is static/NOVAC, HIGATJOB gone
                    → back to the core-first regime → 1202, 1202
P66:                SERVICER ~0.9 s + RODCOMP 1/s → demand < 100% → clean
```

## What our simulation gets wrong (measured)

Measured behavior of the shipping engine: replaying the `f` flight plan gives **ten
1201s, zero 1202s**; occupancy 1 ms before failure is 6/8 cores (5 stubs + LRH), 5/5
VACs — the boundary FINDVAC always hits the VAC wall. The gaps, in order of impact:

1. **No display job exists at all.** The sim never models MAKEPLAY: no per-cycle NOVAC
   core-holder in P63 (the missing core-set pressure), and no flashing SPVAC sleeper in
   early P64 (the likely actual cause of the flight's single 1201 — the sim substitutes
   HIGATJOB alone).
2. **MONDO is given a ~250 ms core-holding sleep it never had** (real MONDO ends
   immediately if the display is busy), while the job that really did sleep holding
   memory — the display job — is absent.
3. **The RESEARCH.md "typing" mechanism is a workaround, not history**: it manufactures
   CHARIN keystroke pairs mid-cycle to supply the 8th core-set holder that MAKEPLAY
   should have been. Tillman's downlist shows no DSKY activity at the P64 1202s.
4. Minor: HIGATJOB sleep ~8 s (real: ≤22 s, from ~6 s before high gate), LRH gate 80 ms
   (real ~95 ms).
