# Screenplay Outline — What Ran While the Descent Engine Burned

Every program on the LGC during powered descent: its interval, its cost, and whether it
held a core set / VAC area. Every claim carries a bracketed source link — the 1969
primary documents (deep-linked to page or passage) and the flight code (deep-linked to
the line). Mission-time context in
[`screenplay_descent_timeline.markdown`](screenplay_descent_timeline.markdown); the hardware
mechanics of the counter theft in
[`screenplay_memory_cycle_stealing.markdown`](screenplay_memory_cycle_stealing.markdown).

## The two pools (what overflowed)

- **8 core sets** (12 words each) — *every* Executive job holds one. Pool empty → **1202**.[^tillman-p1][^eyles-coresets][^code-1202]
- **5 VAC areas** (44 words each) — interpretive (vector-math) jobs hold one *in
  addition*. Pool empty → **1201**.[^tillman-p1][^eyles-vacs][^code-1201]
- Tasks (WAITLIST) and interrupts take **neither**.[^eyles-interrupts]
- Allocation order: FINDVAC scans the 5 VACs first, then the 8 core sets. NOVAC skips
  the VAC scan entirely — **a NOVAC request can only ever raise 1202**.[^code-findvac2][^code-novac][^adler-skip]

## The full map — Cherry's 1969 job table, verified against the code

| Program | Kind | Interval / trigger | CPU per 2 s cycle | Core set held | VAC held |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **SERVICER** (average-G nav, LR incorporation, P63/P64 guidance, throttle, FINDCDUW, display prep)[^cherry-table][^eyles-servicer] | job, prio 20, FINDVAC[^code-servicer] | every 2.000 s via READACCS, unconditional[^code-goreadax] | ~1.30–1.45 s (65–72%) | full run — **kept forever if a new copy is scheduled before it finishes (the leak)**[^eyles-punctual] | same span |
| **MAKEPLAY** (display job spawned by every guidance pass at DISPEXIT)[^cherry-table][^code-dispexit] | job, prio 20 (self-raises to 33), **NOVAC when static** (P63 V06N63, P66 V06N60), **SPVAC when flashing** (early-P64 V06N64)[^code-makeplay] | 1 per cycle | ~ms | yes — and a blocked/flashing one **sleeps holding it until the crew responds** (next pass wakes-and-kills the sleeper)[^code-1206] | **only the P64 flashing form** — sleeps holding it until PRO[^code-makeplay] |
| **MONDO** — the V16N68 DELTAH monitor[^cherry-table] | job, prio 30, NOVAC | 1/s — MONREQ waitlist task re-arms itself every 1.00 s and spawns a fresh MONDO; keyed **once**, runs until killed[^code-monreq] | ~30–60 ms | yes, briefly — **never sleeps** (display busy → lights KEY REL, ends job)[^code-monbusy] | no |
| **CHARIN** (each DSKY keystroke)[^cherry-table] | job, prio 30, NOVAC[^code-charin] | per keystroke | ~5 ms | yes, briefly | no |
| **LRHJOB** (LR range read, below 25,000 ft)[^cherry-lr] | job, prio 32, NOVAC[^code-lrh] | 1 per cycle, fired 50 ms *before* each READACCS[^code-lrhtask] | ~2 ms | yes ~97 ms (1 ms + ~95 ms radar gate + 1 ms), **straddling the cycle boundary**[^code-lrh95] | no |
| **LRVJOB** (LR velocity, 5 samples)[^cherry-lr] | job, prio 32, NOVAC[^code-lrv] | 1 per cycle when velocity reads enabled | ~2 ms | yes ~500 ms (sleeps through sampling)[^code-lrv500] | no |
| **HIGATJOB** (LR antenna to position 2)[^cherry-table] | job, prio 32→23, FINDVAC[^code-higat] | once, ~6 s before high gate[^code-higatask] | ~2 ms | yes — sleeps on the position-2 discrete, **up to 22 s** | **yes, same span** |
| **1/GYRO** (gyro compensation, from 1/PIPA inside SERVICER)[^cherry-table] | job, prio 21, NOVAC[^code-gyro] | per cycle when accumulated compensation ≥ 2 pulses (~1/s) | ~7 ms | yes, briefly | no |
| **RODCOMP** (P66 rate-of-descent update) | job, prio 22, FINDVAC[^code-rod] | every 1 s (RODTASK), P66 only | ~ms | yes, briefly | yes, briefly |
| **READACCS** (read PIPAs, spawn SERVICER, re-arm +2 s) | WAITLIST task | every 2.000 s[^code-goreadax][^cherry-ineluctable] | ~1 ms | no | no |
| **R10/R11 → LANDISP** (tape meters / cross-pointers) | task | every 0.25 s[^code-r10] | ~ms | no | no |
| **DAP** (digital autopilot) | T5RUPT interrupt | every 100 ms[^code-dap] | ~0.24 s (12%) | no | no |
| **T4RUPT** (DSKY relays, monitors) | interrupt | every 120 ms[^code-t4] | ~16 ms (0.8%) | no | no |
| **DOWNRUPT** (telemetry) | interrupt | every 20 ms (50/s)[^code-down] | ~20 ms (1%) | no | no |
| **RR CDU counter theft** (the bug) | hardware counter steal | 12,800/s continuous | **0.30 s (15%)**[^cherry-15pct][^eyles-15pct] | no — time only, zero memory | no |

Only **SERVICER, MAKEPLAY (flashing form), HIGATJOB** — plus P66's RODCOMP — ever take a
VAC.[^cherry-3vac] Everything else is NOVAC. In a P63 cycle with the monitor up, roughly
**six of the seven allocation requests are NOVAC** (display job, MONDO, LRH, LRV, gyro)
and can only fail as 1202.[^code-novac]

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
| P63 before LR lock | < 85%[^eyles-margins] | ~100% | quiet knife edge — no alarms for ~5 min |
| P63 + radar locked | ~87%[^eyles-margins] | ~102% | leaking slowly |
| P63 + V16N68 keyed | ≥ 90%[^eyles-margins] | **~105%** | 1202 ~12 s after each monitor start[^tillman-p2] |
| P64 (redesignation, no monitor) | > 90%[^eyles-p64] | **> 105%** | unsheddable: 1201 + 1202 + 1202 in 40 s[^eyles-p64] |
| P66 (ROD; SERVICER ~0.9 s profile) | lower | < 100% | **zero alarms**, 2 min 20 s to touchdown[^eyles-atthold] |

The design requirement was a tolerable **10% unknown loss** ("TLOSS"), verified with a
monitor verb running; the RR bug stole ~13–15%.[^cherry-tloss][^tillman-tloss]

## Why the flight threw 1202 four times and 1201 once

Documented facts (not conjecture):

1. **No typing was involved.** The monitor is keyed once; MONREQ then spawns a NOVAC
   MONDO job every second on its own.[^code-monreq] Both P63 alarms came "after 12
   seconds of a monitor verb"; the three P64 alarms came "**with no Crew DSKY
   activity**".[^tillman-p2] No 1969 source attributes any alarm to keystrokes.
2. **The alarm code names whichever pool the failing request found empty**, and NOVAC
   requests skip the VAC scan.[^adler-rule][^adler-skip]
3. **The display pipeline is the swing factor.** P63/P66 display jobs are core-set-only;
   early P64's *flashing* V06N64 takes a core set + VAC and **sleeps holding both until
   the crew keys PRO**[^code-makeplay][^code-1206] — which Cherry's event log puts at
   +568, *after* the 1201 (+554) and *before* the two P64 1202s (+578, +594).[^cherry-events]
   HIGATJOB also parks on a VAC (≤22 s) around P64 entry[^code-higat] and is not rebuilt
   after the restart.[^cherry-restart]

Reconstruction consistent with all of the above (the 1969 sources never named the
failing request per alarm — Cherry's deduction stops at "some job like SERVICER was
scheduled two and possibly three or four times"[^cherry-multiple]):

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
   immediately if the display is busy[^code-monbusy]), while the job that really did
   sleep holding memory — the display job — is absent.
3. **The `exec-tui/RESEARCH.md` "typing" mechanism is a workaround, not history**: it
   manufactures CHARIN keystroke pairs mid-cycle to supply the 8th core-set holder that
   MAKEPLAY should have been. Tillman's downlist shows no DSKY activity at the P64
   1202s.[^tillman-p2]
4. Minor: HIGATJOB sleep ~8 s (real: ≤22 s[^code-higat]), LRH gate 80 ms (real
   ~95 ms[^code-lrh95]).

A millisecond-per-tick, instruction-level rebuild addressing all four gaps —
the display pipeline included — lives in [`msim/`](msim/README.md): its two
generated timelines ([baseline](msim/timelines/p63-baseline.md), [with the
1668 monitor](msim/timelines/p63-monitor-1668.md)) reproduce the two P63
1202s at the flight offsets, with all eight core sets held and a VAC still
free at the failing request, and no 1201 anywhere in P63.

## Sources

[^cherry-table]: Cherry, *Exegesis of the 1201 and 1202 Alarms* (MIT, 4 Aug 1969), Table of Jobs and Priorities, [pp. 11–12](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=11).
[^cherry-lr]: Cherry, job table note on LRHJOB/LRVJOB — "run for a millisecond or so and then sleep for about 80 milliseconds", [p. 11](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=11).
[^cherry-3vac]: Cherry — "only three of these are jobs which use a VAC area", [p. 6](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=6).
[^cherry-multiple]: Cherry — "some job like SERVICER was scheduled two and possibly three or four times", [p. 6](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=6).
[^cherry-ineluctable]: Cherry — "The clock … ineluctably counts down to the time for the next repetition of a job to begin whether the previous repetition is complete or not", [pp. 1–2](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=1).
[^cherry-tloss]: Cherry — "We insisted on a tolerable TLOSS of about 10% during landing … (with a monitor verb running). Unfortunately, the RR ECDU's stole about 15%", [p. 3](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=3).
[^cherry-15pct]: Cherry — "12.8 × 10³ × 11.72 × 10⁻⁶ = 0.15 seconds/second or 15% of the LGC time", [pp. 7–8](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=7).
[^cherry-events]: Cherry, "Important Events Occurring During Lunar Landing" (PDI-relative event log incl. V16N68 at +304, alarms at +316/+358/+554/+578/+594, PRO to FLV06N64 at +568), [pp. 13–14](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=13).
[^cherry-restart]: Cherry — restart benefits: "the backlog, the logjam, is cleaned up … monitor verbs or extended verbs are not automatically restarted", [pp. 5–6](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=5).
[^tillman-p1]: Tillman memo (Grumman, 31 Jul 1969) — "The EXECUTIVE has available for assignment to Jobs 8 core sets and 5 VAC areas…", [p. 1](https://ibiblio.org/apollo/Documents/Memo-Tillman690731_text.pdf#page=1).
[^tillman-p2]: Tillman — alarm narrative: "in P63 there were two alarms in each case after 12 seconds of a monitor verb … There was no Crew DSKY activity related to these [P64 alarms]", [p. 2](https://ibiblio.org/apollo/Documents/Memo-Tillman690731_text.pdf#page=2).
[^tillman-tloss]: Tillman — "about .15 seconds per second or a 15% real time overhead … above the 10% TLOSS used in verification", [pp. 3–4](https://ibiblio.org/apollo/Documents/Memo-Tillman690731_text.pdf#page=3).
[^eyles-coresets]: Eyles, *Tales From the Lunar Module Guidance Computer* (AAS 04-064) — ["an array of eight such 'core sets' of 12 registers each"](https://www.doneyles.com/LM/Tales.html#:~:text=array%20of%20eight%20such).
[^eyles-vacs]: Eyles — ["five such 'Vector Accumulator (VAC) areas'"](https://www.doneyles.com/LM/Tales.html#:~:text=Vector%20Accumulator%20%28VAC%29%20areas).
[^eyles-interrupts]: Eyles — ["Interrupts were dedicated to particular functions including the digital autopilot, uplink and downlink, and keyboard operation"](https://www.doneyles.com/LM/Tales.html#:~:text=Interrupts%20were%20dedicated%20to%20particular%20functions).
[^eyles-servicer]: Eyles — ["first performed average-G navigation, then guidance equations, then throttle and attitude output, and then the updating of displays"](https://www.doneyles.com/LM/Tales.html#:~:text=average%2DG%20navigation%2C%20then%20guidance%20equations).
[^eyles-punctual]: Eyles — ["it was SERVICER that had not yet reached its conclusion when the next READACCS, running punctually, scheduled SERVICER again"](https://www.doneyles.com/LM/Tales.html#:~:text=running%20punctually).
[^eyles-margins]: Eyles — ["the duty-cycle margin was over 15% … lowered the margin to perhaps 13% … the margin shrank again, to 10% or less"](https://www.doneyles.com/LM/Tales.html#:~:text=margin%20shrank%20again).
[^eyles-p64]: Eyles — ["the essential software by itself left a duty-cycle margin of less than 10%. The alarms kept coming … could not shed load"](https://www.doneyles.com/LM/Tales.html#:~:text=could%20not%20shed%20load).
[^eyles-15pct]: Eyles — ["the RR CDU counters consumed approximately 15% of the available computation time"](https://www.doneyles.com/LM/Tales.html#:~:text=approximately%2015%25%20of%20the%20available).
[^eyles-atthold]: Eyles — ["switched the autopilot from AUTO to ATT HOLD mode, easing the computational burden … After 2 minutes and 20 seconds spent maneuvering in P66 without alarms, the LM landed"](https://www.doneyles.com/LM/Tales.html#:~:text=easing%20the%20computational%20burden).
[^adler-rule]: Adler, "Apollo 11 Program Alarms" (NASA ALSJ) — ["the core sets got filled up and a 1202 alarm was generated. The 1201 … was because the scheduling request that caused the actual overflow was one that had requested a VAC area"](https://www.nasa.gov/wp-content/uploads/static/history/alsj/a11/a11.1201-pa.html#:~:text=core%20sets%20got%20filled%20up).
[^adler-skip]: Adler — ["Scanning for a VAC area would be skipped if the scheduling request specified 'NOVAC'"](https://www.nasa.gov/wp-content/uploads/static/history/alsj/a11/a11.1201-pa.html#:~:text=would%20be%20skipped%20if%20the%20scheduling%20request).
[^code-findvac2]: Flight code — `FINDVAC2` scans the five `VACnUSE` flags first: [`Luminary099/EXECUTIVE.agc` L141–L161](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/EXECUTIVE.agc#L141-L161).
[^code-novac]: Flight code — `NOVAC` entry goes straight to the core-set scan: [`Luminary099/EXECUTIVE.agc` L37–L48](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/EXECUTIVE.agc#L37-L48).
[^code-1201]: Flight code — `TC BAILOUT1 / OCT 1201  # NO VAC AREAS`: [`Luminary099/EXECUTIVE.agc` L161](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/EXECUTIVE.agc#L161).
[^code-1202]: Flight code — `TC BAILOUT1 / OCT 1202` after the eight-set scan (`NO.CORES DEC 7` = one probe + seven repeats): [`Luminary099/EXECUTIVE.agc` L190–L249](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/EXECUTIVE.agc#L190-L249).
[^code-servicer]: Flight code — `CA PRIO20 / TC FINDVAC / 2CADR SERVICER`: [`Luminary099/SERVICER.agc` L120–L123](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/SERVICER.agc#L120-L123).
[^code-goreadax]: Flight code — `GOREADAX … CA 2SECS / TC VARDELAY` (unconditional 2 s re-arm): [`Luminary099/SERVICER.agc` L80–L81](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/SERVICER.agc#L80-L81).
[^code-dispexit]: Flight code — `DISPEXIT` → `P63DISPS` / `P64DISPS` / `VERTDISP` (a display job every guidance pass): [`Luminary099/LUNAR_LANDING_GUIDANCE_EQUATIONS.agc` L835–L889](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/LUNAR_LANDING_GUIDANCE_EQUATIONS.agc#L835-L889).
[^code-makeplay]: Flight code — static display → `TC NOVAC / 2CADR MAKEPLAY`; flashing display → `VACDSP … TC SPVAC` (core set **+ VAC**): [`Luminary099/DISPLAY_INTERFACE_ROUTINES.agc` L836–L856](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/DISPLAY_INTERFACE_ROUTINES.agc#L836-L856).
[^code-1206]: Flight code — display jobs sleep holding resources (`ENDIDLE`/`NVSBWAIT`); a second simultaneous display sleeper aborts with octal 1206: [`Luminary099/PINBALL_GAME_BUTTONS_AND_LIGHTS.agc` L3159–L3168](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/PINBALL_GAME_BUTTONS_AND_LIGHTS.agc#L3159-L3168).
[^code-monreq]: Flight code — `MONREQ` re-enlists itself every `MONDEL` = 1.00 s and spawns a fresh NOVAC `MONDO`: [`Luminary099/PINBALL_GAME_BUTTONS_AND_LIGHTS.agc` L2373–L2395](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/PINBALL_GAME_BUTTONS_AND_LIGHTS.agc#L2373-L2395).
[^code-monbusy]: Flight code — `MONDO … CCS DSPLOCK / TC MONBUSY` → `MONBUSY TC RELDSPON / TC ENDOFJOB` (no sleep): [`Luminary099/PINBALL_GAME_BUTTONS_AND_LIGHTS.agc` L2397–L2403](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/PINBALL_GAME_BUTTONS_AND_LIGHTS.agc#L2397-L2403), [L2451](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/PINBALL_GAME_BUTTONS_AND_LIGHTS.agc#L2451).
[^code-charin]: Flight code — each keystroke/uplink word: `CAF CHRPRIO / TC NOVAC / 2CADR CHARIN`: [`Luminary099/KEYRUPT_UPRUPT.agc` L50](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/KEYRUPT_UPRUPT.agc#L50).
[^code-lrhtask]: Flight code — LRHTASK fires "50 MS PRIOR TO THE NEXT READACCS TASK": [`Luminary099/SERVICER.agc` L697–L727](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/SERVICER.agc#L697-L727).
[^code-lrh]: Flight code — `CA PRIO32 / TC NOVAC / 2CADR LRHJOB`: [`Luminary099/SERVICER.agc` L727](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/SERVICER.agc#L727).
[^code-lrh95]: Flight code — "LRHJOB IS SET BY LRHTASK WHEN LEM IS BELOW 25000 FT … GOES TO SLEEP WHILE THE SAMPLING IS DONE -- ABOUT 95 MS": [`Luminary099/SERVICER.agc` L1567–L1570](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/SERVICER.agc#L1567-L1570).
[^code-lrv]: Flight code — `2CADR LRVJOB` (NOVAC, PRIO32): [`Luminary099/SERVICER.agc` L1437](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/SERVICER.agc#L1437).
[^code-lrv500]: Flight code — "5 VELOCITY SAMPLES AND GOES TO SLEEP WHILE THE SAMPLING IS DONE -- ABOUT 500 MS": [`Luminary099/SERVICER.agc` L1510–L1527](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/SERVICER.agc#L1510-L1527).
[^code-higatask]: Flight code — "HIGATASK IS ENTERED APPROXIMATELY 6 SECS PRIOR TO HIGATE": [`Luminary099/SERVICER.agc` L740–L755](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/SERVICER.agc#L740-L755).
[^code-higat]: Flight code — HIGATJOB (FINDVAC) sleeps until the position-2 discrete, 22-second window: [`Luminary099/SERVICER.agc` L1657–L1664](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/SERVICER.agc#L1657-L1664); Cherry's table note "Sleeps until position #2 discrete is received", [p. 12](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=12).
[^code-gyro]: Flight code — `CA PRIO21 … TC NOVAC / 2CADR 1/GYRO`: [`Luminary099/IMU_COMPENSATION_PACKAGE.agc` L110](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/IMU_COMPENSATION_PACKAGE.agc#L110).
[^code-rod]: Flight code — `RODTASK CAF PRIO22 / TC FINDVAC / 2CADR RODCOMP`, re-armed every `1SEC`: [`Luminary099/LUNAR_LANDING_GUIDANCE_EQUATIONS.agc` L934–L953](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/LUNAR_LANDING_GUIDANCE_EQUATIONS.agc#L934-L953).
[^code-r10]: Flight code — `R10,R11` task re-arms with `OCT31` = 0.25 s: [`Luminary099/P70-P71.agc` L36–L47](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/P70-P71.agc#L36-L47).
[^code-dap]: Flight code — "THE NOMINAL TIME BETWEEN THE P-AXIS RUPTS IS 100 MS": [`Luminary099/P-AXIS_RCS_AUTOPILOT.agc` L41](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/P-AXIS_RCS_AUTOPILOT.agc#L41).
[^code-t4]: Flight code — "MONITORED EVERY 120 MILLISECONDS": [`Luminary099/T4RUPT_PROGRAM.agc` L144](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/T4RUPT_PROGRAM.agc#L144).
[^code-down]: Flight code — "AT 50 TIMES PER SEC (EVERY 20 MS)": [`Luminary099/DOWN_TELEMETRY_PROGRAM.agc` L43](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/DOWN_TELEMETRY_PROGRAM.agc#L43).
