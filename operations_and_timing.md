# Operations and Timing During the Apollo 11 Powered Descent

Everything the LM Guidance Computer was doing during the landing, how long each thing took,
how often it ran, and — the question this document exists to settle — **what the "1.8 seconds"
actually was and what the 10% margin was actually for.**

Primary sources:

- **[Eyles]** Don Eyles, *Tales From the Lunar Module Guidance Computer*, AAS 04-064 (2004).
- **[Cherry]** George W. Cherry, *Exegesis of the 1201 and 1202 Alarms* (MIT AG# 370-69, Aug 4 1969).
  Includes the Table of Jobs and Priorities and a PDI-relative event log.
- **[Wills]** Robert Wills, *Light Years Ahead* (TNMOC talk, 2019) — a synthesis of the two above.
- **[L099]** The Luminary 099 assembly source in this repository (`Luminary099/`).

Every rate below marked [L099] was verified against the flight source in this repo. Numbers
marked **est.** are allocations chosen to be consistent with the sourced aggregates; the
aggregates themselves are hard numbers.

---

## 1. The currency: one memory cycle = 11.72 µs

Everything the LGC did was paid for in memory cycles of 11.72 µs ([Cherry]; Eyles rounds to
11.7). That is ~85,300 cycles per second, total, ever. A basic instruction costs 1–3 cycles.
Interpretive code (the vector/matrix language SERVICER is mostly written in) is 50–100×
slower: one double-precision vector cross product ≈ 5 ms [Eyles].

Three kinds of work competed for those cycles:

1. **Counter increments** (`PINC`/`MINC`, "unprogrammed sequences") — hardware steals exactly
   one cycle each, invisibly. No interrupt, no job, no trace.
2. **Interrupts** — autopilot, telemetry, keyboard, timers. Serviced regardless of what the
   Executive is doing. These cannot be starved by job-queue congestion.
3. **Executive jobs** — priority-scheduled; each holds one of **8 core sets**, VAC jobs also
   one of **5 VAC areas**. Out of core sets → **1202**; out of VAC areas → **1201**.

---

## 2. The complete inventory

| Operation | Kind | How often | CPU per occurrence | ~Share of CPU |
| --- | --- | --- | --- | --- |
| **RR CDU counter spam** (the bug) | counter steal | 6,400/s × 2 counters | 11.72 µs | **15.0%** (0.30 s per cycle) |
| **SERVICER** | job, prio 20→24, VAC | every 2.000 s | ~1.3–1.45 s **est.** | **~65–72%** |
| **DAP** (digital autopilot) | T5RUPT interrupt | every 100 ms [L099] | ~12 ms **est.** | ~10–12% **est.** |
| **READACCS** | Waitlist task | every 2.000 s [L099] | ~1 ms | ~0.05% |
| **MONDO** (V16N68 monitor) | job, prio 30, NOVAC | refresh 1/s [L099] | ~30–60 ms **est.** | **~3–5%** |
| **MAKEPLAY** (display job) | job, prio 20, VAC | set up by SERVICER each cycle | ms-scale | ~1% |
| **LRHJOB** (LR range read) | job, prio 32, NOVAC | 1/cycle below 25 kft [L099] | ~1 ms + 1 ms (80 ms asleep) | <1% |
| **LRVJOB** (LR velocity read) | job, prio 32, NOVAC | 1 beam/cycle, 5 samples [L099] | ~ms (≈500 ms asleep) | <1% |
| LR data → nav-frame conversion | inside SERVICER | per cycle after "data good" | ~40 ms **est.** | **~2%** [Eyles] |
| **1/GYRO** (IMU compensation) | job, prio 21, NOVAC | ~1/s | ~7 ms **est.** | ~0.35% |
| **CHARIN** (keystroke) | KEYRUPT + job, prio 30, NOVAC | per keystroke [L099] | ~5 ms **est.** | negligible directly |
| **T4RUPT** (DSKY/housekeeping) | interrupt | every 120 ms [L099] | ~1 ms **est.** | ~0.8% |
| **DOWNRUPT** (telemetry) | interrupt | every 20 ms (50/s) [L099] | ~0.2 ms **est.** | ~1% |
| **R10/R11** (cockpit analog displays) | task | every 0.25 s [L099] | ms-scale | ~1–2% **est.** |
| **HIGATJOB** (LR antenna to Pos 2) | job, prio 32→23, VAC | once, at P64 entry | 1–2 ms (sleeps for discrete) | negligible — but holds a VAC |
| PIPA + misc counter traffic | counter steal | continuous in powered flight | 11.72 µs each | ~0.5% **est.** |
| Executive/Waitlist overhead | OS | continuous | µs-scale | ~1% **est.** |
| P64 redesignation + LPD logic | inside SERVICER | per cycle in P64 | ~60 ms **est.** | **~3%** [Eyles] |

Details on the ones that need explaining:

### SERVICER — the big one

One job containing the whole guidance chain, in sequence, because each step feeds the next
[Eyles]: average-G navigation (integrate the accelerometer ΔVs), mass update, landing-radar
data incorporation, guidance equations (P63 braking targets / P64 + redesignation), FINDCDUW
(attitude commands handed to the DAP), throttle command (~2.8 lb per pulse), display data
prep. Mostly Interpretive, hence enormous in time. Lowest priority (20) of every job in
Cherry's table — so it absorbs 100% of any deficit.

### DAP — the digital autopilot (what it is)

Guidance (SERVICER, every 2 s) decides *which way the thrust vector should point*. The DAP
(every 100 ms) *actually points it*: it estimates attitude and rates, compares against the
commanded attitude, picks which of the 16 RCS thrusters to fire and for how long (the TJET
law), and trims the descent-engine gimbal. It runs at **interrupt level** (T5RUPT), not as an
Executive job:

> `# THE NOMINAL TIME BETWEEN THE P-AXIS RUPTS IS 100 MS IN ALL NON-IDLING MODES OF THE DAP.`
> — `Luminary099/P-AXIS_RCS_AUTOPILOT.agc`

Because it is an interrupt, it kept flying the vehicle straight through every alarm and every
restart — job-queue exhaustion could not touch it. It is also why Armstrong's AUTO → ATT HOLD
switch shed real load: rate-command attitude hold is cheaper than auto-guidance steering
[Eyles: "easing the computational burden"].

### The landing radar — how often it actually ran

Two separate costs, both small in CPU and long in wall time:

- **Range (LRHJOB):** below 25,000 ft, `READACCS` sets `LRHTASK` to fire **50 ms before the
  next READACCS**, once per 2-second cycle. The job runs ~1 ms, sleeps ~80 ms while the
  radar's sync pulses gate the data in, wakes, runs ~1 ms more [Cherry's job table; L099
  `SERVICER.agc`].
- **Velocity (LRVJOB):** one beam per cycle (the beam index rotates X→Y→Z), **5 samples
  averaged over ~500 ms of sleep** [L099 `SERVICER.agc`: "INITIALIZES THE LANDING RADAR READ
  ROUTINE FOR 5 VELOCITY SAMPLES AND GOES TO SLEEP WHILE THE SAMPLING IS DONE -- ABOUT
  500 MS"].

The reads themselves are ~1% of the CPU. The real radar cost is *inside SERVICER*: converting
body-referenced radar data to the navigation frame, which cut the margin from >15% to ~13%
[Eyles] — call it ~2%, ~40 ms per cycle.

### The monitor — V16N68

Monitor verbs refresh **once per second** [L099 PINBALL: "MONITORS (DISPLAYS THAT ARE UPDATED
ONCE PER SECOND)"], each refresh running the MONDO job at priority 30 (above SERVICER's 20).
Keying it shrank the margin from ~13% to "10% or less" [Eyles] → the monitor plus its display
machinery cost **~3–5%**. It was *not* restart-protected, so every BAILOUT killed it — the
built-in load shedding that made P63 self-healing.

### Typing

Each keystroke: KEYRUPT interrupt (µs) → CHARIN job, priority 30, core set only, ~5 ms. Even
frantic typing is well under 1% of the machine. Typing's real contributions: it is what turned
the monitor **on** (twice), each CHARIN transiently takes a core set from the pool of 8, and
at priority 30 it preempts SERVICER. Both P63 alarms trail a fresh `V16N68` entry by ~12 s
(Cherry's event log: keyed +304 → alarm +316; re-keyed +346 → alarm +358).

---

## 3. THE LEDGER — what was budgeted, and what the 10% was for

This section answers the question directly: **were the landing radar, the DAP, the monitor,
and the typing supposed to come out of the 10%?**

**No.** All of them were *known* loads, simulated during development, and budgeted inside the
~85–90%. The 10% was **pure contingency for unknown time loss**, on top of everything they
knew about. Both sources are explicit:

Eyles defines duty cycle as the aggregate:

> "During the lunar descent, duty-cycle simply describes how much time was used **in
> aggregate by jobs, tasks, and interrupts**, during each 2-second period."

Cherry states the design requirement — note "with a monitor verb running":

> "We simulate an **unknown source or sources** of memory cycle loss (we call it TLOSS) and
> insist that a certain TLOSS be tolerable (no 1201's, no 1202's). ... We insisted on a
> tolerable TLOSS of about 10% during landing. The coding and the guidance period were
> therefore massaged until 10% TLOSS was tolerable (**with a monitor verb running**).
> Unfortunately, the RR ECDU's stole about 15% (most of the time)."

So the accounting, per 2.000-second cycle, in the worst pre-P64 configuration (P63, radar
locked, V16N68 up):

```text
DEMAND (all known software: jobs + tasks + interrupts)   ~1.80 s   (~90%)
  of which SERVICER itself                               ~1.35–1.45 s
  of which everything else (DAP, monitor, LR, displays,
    telemetry, keystrokes, overhead)                     ~0.35–0.45 s
CONTINGENCY for unknown TLOSS (the requirement)           0.20 s   (10%)
                                                         -------
TOTAL                                                     2.00 s   (100%)

ACTUAL unknown TLOSS (RR CDU counter theft)               0.30 s   (15%)
DEFICIT, absorbed entirely by SERVICER                   ~0.10 s per cycle
```

The margin by configuration [Eyles]:

| Configuration | Duty cycle (all software) | Margin for unknowns |
| --- | --- | --- |
| P63 before landing-radar lock | <85% | >15% |
| + radar acquired (nav-frame conversion) | ~87% | ~13% |
| + V16N68 monitor keyed | ≥90% | ≤10% |
| P64 (redesignation logic), essential software alone | >90% | <10% |

**So there was no oopsie in the calculation.** MIT did the accounting correctly: every known
consumer was in the budget, the requirement was met (the software genuinely tolerated 10%
unknown loss with a monitor running), and the requirement was even re-verified in simulation.
The failure was that the *unknown* loss turned out to be 13–15% — larger than the 10%
contingency — and it was unknown only because the all-digital simulator did not model the RR
CDU interface. Cherry concedes: "if the simulator had this feature for our software
verification tests we would have detected the computer overload during our testing."

One nuance worth saying out loud: the 10% requirement was set *for the worst planned
configuration*. Early in P63, before radar lock and before any monitor, the real margin was
>15% — which is exactly why the first alarm did not fire at ignition but only after the crew
keyed the monitor up (+304 s), stacking the last ~3–5% of known load on top of the 15% theft.

---

## 4. VALIDATION — the "1.8 seconds" in this repo's artifacts

The repo currently contains **two different meanings** of "1.8 s", and that conflation is
exactly the confusion this document resolves:

| Artifact | What it says | Verdict |
| --- | --- | --- |
| `memory_leak.md` ("The same idea in seconds") | "work the **software** needed ~1.800 s (90%)" | **Correct.** 1.8 s = aggregate of all jobs+tasks+interrupts. |
| `exec-tui/RESEARCH.md` + `ROADMAP.md` | SERVICER ≈ 1,320 ms (66%), DAP 12%, monitor 3%, LR +3%, T4RUPT/DOWNRUPT/misc ~3% → aggregates match Eyles' table | **Correct decomposition.** SERVICER-the-job is ~1.3 s, not 1.8 s. |
| `timeline.c` (`SERVICER_CPU_NEED 1.80` — "work this copy needs") | 1.8 s attributed to **SERVICER alone** | **Mislabeled.** That figure is the whole software demand. SERVICER's own need is ~1.35 s. |
| `timeline-tui` (`servicerNeedS = 1.80`; healthy scenario = SERVICER 1.80 + other jobs 0.15 → 97.5% busy, no DAP row) | Same mislabel, plus a "healthy" state with only 2.5% margin and no autopilot cost | **Mislabeled + miscalibrated.** Healthy late-P63 margin should be ~10–15%, and the DAP (~12%) is missing entirely. |

Recommended corrections (not applied in this change — the tools still demonstrate the leak
mechanism correctly, but their labels contradict the sources):

1. In `timeline.c` and `timeline-tui`, either rename the 1.80 s constant to "software demand
   per cycle (all jobs + interrupts)" and show SERVICER's own portion as ~1.35 s, **or** keep
   a SERVICER-only bar at ~1.35 s and add a DAP/interrupt row (~0.25 s) so the healthy total
   comes out at ~1.80 s busy / 0.20 s idle.
2. In `timeline-tui`'s healthy scenario, the idle slice should be ~0.20 s (10%, monitor up)
   to ~0.30 s (15%, pre-monitor), not 0.05 s.
3. `exec-tui`'s budget needs no correction; its decomposition sums ~3% under Eyles'
   aggregates (conservative), which `RESEARCH.md` already notes.

Sanity check of the corrected model, overload case:

```text
available          2.000 s
- RR theft         0.300 s   (15%)
- interrupts etc.  0.250 s   (DAP ~0.24 + T4RUPT/DOWNRUPT ~0.04, also slowed by theft)
- higher jobs      0.100 s   (monitor, LR reads, gyro, keystrokes)
= left for SERVICER ~1.35 s  vs. need ~1.40–1.45 s (P63+LR+monitor)
→ deficit ~0.05–0.10 s per cycle → one leaked core set + VAC pair per cycle-ish
→ pools of 8 / 5 exhaust in tens of seconds → 1202 (or 1201)
```

That matches the flight record: V16N68 keyed at PDI+304 s, first 1202 at +316 s — about six
2-second cycles from "monitor up" to "no core sets."

---

## 5. Timeline of the five alarms (PDI = MET 102:33:05)

From Cherry's event log (PDI-relative) and Eyles (MET):

| PDI+ | Event |
| --- | --- |
| 0 | Ignition, 10% throttle; SERVICER cycle already running (started PDI−30 s) |
| +26 s | Throttle to fixed max; descent guidance enabled |
| +232 s | Yaw windows-up maneuver |
| +262–286 s | Landing radar "data good" (margin 15% → ~13%) |
| +304 s | **V16N68 keyed**; DELTAH ≈ −2,900 ft (margin → ≤10%; demand now ~105% with theft) |
| +316 s | **1202** — restart, monitor shed, display back to N63 |
| +322 s | V05N09E — crew reads code; Houston "go" ~30 s later |
| +338 s | V57E — accept LR updates into navigation |
| +346 s | **V16N68 re-keyed** |
| +358 s | **1202** — monitor shed again |
| +384 s | Throttle down, on time ("better than the simulator") |
| +506 s | High gate → **P64**; LPD; HIGATJOB; redesignation logic now essential (unsheddable) |
| +552 s (102:42:17) | **1201** (no VAC areas) |
| +578 s (102:42:43) | **1202** |
| +594 s (102:42:58) | **1202** (~770 ft) |
| 102:43:08 | AUTO → ATT HOLD (load shed #1) |
| 102:43:20 | **P66** rate-of-descent mode (load shed #2) — alarms stop |
| 102:45:40 | Touchdown; 2 min 20 s in P66 alarm-free |

Why 1202s in P63 but a 1201 in P64: only three job types used VACs (SERVICER, MAKEPLAY,
HIGATJOB — Cherry's table). In P63 the queue also filled with NOVAC jobs (CHARIN, MONDO, LR
reads), so the 8 core sets ran out first. In P64, stacked VAC-holding jobs hit the 5-VAC wall
first.

Source discrepancies worth knowing: **7 vs. 8 core sets** — the LM's LUMINARY had **8**
(Eyles: "an array of eight such 'core sets' of 12 registers each"; Cherry: "8 coresets and 5
VAC areas"; [L099] `ERASE +83D # EIGHT SETS OF 12 REGISTERS EACH`), while the CM's COLOSSUS
had **7** (Cherry's parenthetical). Secondary sources (including the Wills talk) sometimes say
7 for the LM; the likely ancestry is the flight code itself: the scan loop constant is
literally `NO.CORES DEC 7` (meaning "one probe + seven repeats" = 8), and the original 1969
comment beside it — "SEVEN SETS OF ELEVEN REGISTERS EACH" — is stale on both counts
(`COREINC DEC 12` sits five lines below). The klabs copy of Eyles says Aldrin keyed "V90N50" to
read the alarm — Cherry's event log and Eyles' own corrected website say **V05N09E**. Cherry's
log also shows an ATT HOLD entry at +528 s (~4,770 ft) while Eyles puts the definitive
takeover at 102:43:08 (~650 ft) — most plausibly an earlier manual-control check versus the
final one. And MIT conservatively used **13%** for the theft in 1969 analysis (Grumman's
measured worst case was 13.36%) while the theoretical max is **15.0%**; both appear in the
sources.

---

## 6. The one-breath version

The machine had 2.000 s per cycle. Everything MIT knew about — SERVICER (~1.35 s), the DAP
(~0.24 s), the monitor, the radar reads, the displays, the telemetry, the typing — was
budgeted, and it summed to ~1.80 s (90%). The remaining 0.20 s (10%) was, by explicit
requirement, contingency for *unknown* loss. The rendezvous-radar counters then stole 0.30 s
(15%) — an unknown bigger than the contingency — and the shortfall of ~0.10 s per cycle was
paid not in lateness but in *allocations*, because a punctual clock kept scheduling new
SERVICERs while unfinished ones held their memory. Eight core sets and five VACs later:
1202.
