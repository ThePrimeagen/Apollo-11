# msim RESEARCH — the model, its calibration, and its sources

Companion to [`README.md`](README.md). Primary sources as in the repository
root: **[Cherry]** *Exegesis of the 1201 and 1202 Alarms* (MIT, Aug 1969),
**[Tillman]** Grumman memo (31 Jul 1969), **[Eyles]** *Tales From the Lunar
Module Guidance Computer* (AAS 04-064), **[L099]** the `Luminary099/` listing
in this repository, **[Outline]** `screenplay_outline.markdown` /
`screenplay_descent_timeline.markdown`.

## The machine model

- One CPU, 100 µs per tick, nanosecond bookkeeping inside the tick; the
  theft waveform and the occupancy samples stay millisecond-exact across
  each millisecond's ten slices.
- Order inside a tick: hardware dispatches are timestamped at the boundary
  (T5/DAP, then the waitlist, then hardware crew/radar events, then T4RUPT,
  DOWNRUPT), the RR theft skims the front of the tick, interrupt-context CPU
  drains, the job layer gets the rest.
- Interrupts pause the running instruction mid-flight and it resumes with
  nothing lost. Job switches happen ONLY between instructions — the DANZIG
  boundary, where NEWJOB is tested ([L099] `INTERPRETER.agc` L74-L82).

## The Executive (`executive.go`)

Everything mirrors `EXECUTIVE.agc`:

| Mechanism | Source |
| :--- | :--- |
| 8 core sets; the runner always occupies slot 0; CHANJOB swaps | L251-L318 |
| Allocation scans slots upward, first free | NOVAC2/NOVAC3/NEXTCORE L183-L191 |
| FINDVAC scans the 5 VACs first and CLAIMS one before the core scan | FINDVAC2 L141-L161, VACFOUND L170-L174 |
| VAC pool empty at a FINDVAC → 1201; core scan empty → 1202 (even for FINDVAC) | L161, L246-L249 |
| A NOVAC request never scans VACs — it can only raise 1202 | L37-L48 |
| **The PRIORITY word carries the VAC-area address in its low bits** | VACFOUND: "STORE THE ADDRESS ... IN THE LOW NINE BITS OF THE PRIORITY WORD" |
| SETLOC and EJ1 compare FULL words → among equal priorities the higher-addressed VAC wins, i.e. the NEWEST copy preempts and wins every scan | SETLOC L224-L234, EJ1 L492-L499 |
| JOBSLEEP negates the word: dormant, invisible, memory held | JOBSLP1 L322-L332 |
| ENDOFJOB frees the core set and VAC, then EJSCAN | ENDJOB1 L420-L437 |

The word-order rule is the engine of the leak: each cycle's fresh SERVICER
(allocated the lowest FREE VAC, which is above every stub's) preempts the
running copy at its next DANZIG and starves it — Eyles' "uncompleted
SERVICER stubs", which the sim parks at ip ≈ 95% of the pass.

The restart (BAILOUT → flush) frees all eight PRIORITY words and five VAC
use-flags, wipes the waitlist (a mid-tick flush stops the rest of that
tick's due waitlist fires too), zeroes MONSAVE (monitor verbs are not
restarted — [Cherry] pp. 5-6), costs 20 ms, and the restart tables rebuild
one READACCS chain + one SERVICER on the SAME 2 s PIPTIME lattice.

## The SERVICER instruction array (`servicer.go`)

One P63 radar-locked pass, transcribed in execution order from the listing:
SERVICER entry and 1/PIPA (SERVICER.agc L206-L263), AVERAGEG → RVBOTH →
MUNRVG with MUNGRAV twice (L265, L1058-L1131), the copy cycle/DVMON/1/ACCS
(L270-L372), the LR nav-frame conversion when locked (UPDATCHK/POSUPDAT
through STORE DELTAH, L1146-L1188), LUNLAND + GUILDENSTERN (LLGE L117-L246),
TTFINCR (L288-L329), RGVGCALC (L442-L480), TTF/8 + ROOTPSRS (L489-L527),
QUADGUID/AFCCALC (L546-L627), CGCALC (L641-L682), EXTLOGIC/EXBRAK
(L692-L820), THROTTLE (THROTTLE_CONTROL_ROUTINES.agc), FINDCDUW (its whole
file), DISPEXIT → P63DISPS (LLGE L835-L854). Packed opcode pairs are two
entries; STODL/STOVL/STCALL split into their two packed operations.

Costs: `opCost`. Relative costs follow instruction class; two documented
anchors pin the absolute scale:

1. an interpretive vector op (VXV) runs ≈ 5 ms — [Eyles]; ops that exceed
   the 5 ms DANZIG grain are decomposed into their real sub-phases (MXV =
   three row dots, UNIT = ABVAL + scale, SQRT = normalize + iterate);
2. the whole pass costs 1.30-1.45 s — 65-72% of the 2 s cycle ([Cherry]'s
   job table; [Eyles]' margins; [Outline]).

`execResidueUS` (36 ms) is the pass's phase-table/bank-switch/pushdown
residue — the explicit calibration constant that places the pass total
(1359 ms locked) inside that band. The prelock variant omits the LR
conversion: −75 ms, inside Eyles' margin step (15% → ~13%).

## The rest of the load

| Component | Value | Source |
| :--- | :--- | :--- |
| DAP | 12 ms per 100 ms, phase +70 ms | [Eyles] 12%; P-AXIS L41; SERVICER.agc L95-L104 |
| T4RUPT | 0.96 ms per 120 ms | T4RUPT_PROGRAM.agc L144 |
| DOWNRUPT | 0.2 ms per 20 ms | DOWN_TELEMETRY L43 |
| R10,R11 → LANDISP | 3 ms per 250 ms, in task context | P70-P71.agc L36-L49 (runs LANDISP in-task) |
| READACCS | 1 ms per 2.000 s, unconditional re-arm | GOREADAX L80-L81 |
| MONDO | 30 ms, 1 Hz, NOVAC, never sleeps; first refresh ENTR+1 s (MONDEL) | PINBALL L2373-L2403; [Outline] 30-60 ms envelope |
| MAKEPLAY static | 8 ms NOVAC at user prio+1 | DISPLAY_INTERFACE L836-L847 |
| MAKEPLAY blocked | sleeps holding its core set until the next display request wakes-and-kills it | ENDIDLE/NVSBWAIT; PINBALL L3159-L3168 (the 1206 logic) |
| LRHJOB | 1 ms + 95 ms core-holding sleep + 1 ms, fired 50 ms before the next READACCS | SERVICER.agc L697-L727, L1567-L1570 |
| LRVJOB | 1 ms + 500 ms core-holding sleep + 1 ms, samples timed to finish at the boundary | L1508-L1510; [Cherry] job table |
| 1/GYRO | 7 ms NOVAC prio 21, ~1/s from 1/PIPA | IMU_COMPENSATION L107-L110 |
| CHARIN | 5 ms NOVAC prio 30 per keystroke | KEYRUPT_UPRUPT L47-L50 |
| RR theft | deterministic sweep 12.8-15.0% | [Cherry] 15% theoretical; [Tillman] ~13.4% measured |

The theft sweep is a flat-bottomed triangle (period 14 s, 1.1 s floor dwell
each side, offset 12.5 s): the dither loss depends on the RR angle geometry
and dwells at the measured floor away from the worst-case angles. Period,
phase and dwell are the run's free parameters, pinned so a floor dwell
covers one full guidance cycle near each monitor keying — the sim's stand-in
for whatever benign dither geometry the real flight happened to be in when
the keying cycles demonstrably completed (the monitor did come up).

## What the timelines show (the mechanism)

1. **Baseline**: the pass fits with tens of milliseconds to spare; at the
   sweep peak the worst 2 s window demands ~98% — the quiet knife edge.
   Zero alarms for as long as you care to run it.
2. **Keying cycle**: the seven keystrokes (~0.45 s apart) cost ~5 ms each;
   the cycle still completes, and its V06N63 display job — now blocked
   behind the monitor's DSKY — **sleeps holding a core set**. One core,
   parked, invisible, until a restart.
3. **The latch**: the next cycle carries the monitor's two 30 ms refreshes
   and misses completion. From then on every boundary's fresh SERVICER
   (higher VAC word) preempts the runner, which parks at ~95% done holding
   its core set + VAC. One pair leaks per cycle. The system is bistable:
   the same demand that fits in the clean regime cannot recover once the
   preemption cascade begins — only the restart's flush unlatches it.
4. **The wall**: with four pairs parked, the boundary FINDVAC finds a free
   VAC (claims it) and then no eighth core set — the pile is 4 pairs + the
   blocked display + LRHJOB (straddling every boundary by design) + LRVJOB
   (its five samples end at the boundary) + MONDO (straddling, ENTR phase
   .985) = 8/8. **1202, NO CORE SETS**, with a VAC free: the flight's code,
   for the flight's reason. Never 1201: the completing keying-cycle freed a
   VAC just before the wall, and NOVAC churn probes the cores first.
5. **Restart**: flush, rebuild, monitor gone. Re-key → the same ten seconds
   → the second 1202. The third use is dropped after six seconds — three
   boundaries, three pairs, seven cores at the worst instant — clean.

Measured against the flight record:

| Anchor | Flight | Sim |
| :--- | :--- | :--- |
| First alarm after keying V16N68 | 1202, ~12 s ([Tillman] p. 2; +304→+316) | 1202, +10.0 s |
| Second alarm after re-keying | 1202, ~10-12 s (+346→+356/8) | 1202, +11.0 s |
| Third, short monitor use | no alarm (+374/+380) | no alarm |
| Any 1201 in P63 | none | none |
| Pool state at the failure | (not recorded in 1969) | cores 8/8, VACs 4/5 |
| SERVICER copies at the flush | "two and possibly three or four" ([Cherry] p. 6) | four pairs + the runner's request |
| Baseline | no alarms without the monitor | clean, 98% peak demand |

## Known deviations and open questions

- **LR gates at ~34,000 ft — resolved in the code's favor**: LRHTASK's
  comment says "below 25,000 ft" (L697-L701), but the actual gate is the
  READLR flag, and R10,R11's 35KCHK tests `ALTCRIT` index 2 = **50KFT**
  (L948): the altitude read was live at the flight's ~34k ft, as [Cherry]'s
  job table and the [Outline]'s locked-cycle map assume. LRVJOB's velocity
  gate (VALTCHK's |V| test) is looser ground; the sim keeps it per the
  [Outline]'s cycle map, timed so its samples end at the boundary.
- **NOR29NOW** (SERVICER.agc L855-L917, the HCALC/RN1/VN1 state rebuild,
  ~28 interpretive ops) is not transcribed entry-by-entry; its cost is
  absorbed by the `execResidueUS` calibration block, which exists for
  exactly this class of un-transcribed housekeeping.
- **MONDO cost** is at the documented envelope's floor (30 ms ≈ Eyles' +3%
  per cycle); the envelope's upper half produces the same alarms earlier.
- The ENTR sub-second phases (.985) and the theft sweep's free parameters
  are tuned inside their documented bands to reproduce the flight trace;
  Cherry's event log has one-second resolution.
- The restart is modeled as a single 20 ms CPU block; the real
  BAILOUT/ENEMA path spreads that work differently.
- MXV/UNIT/SQRT sub-phase splits keep every entry inside the user-visible
  5 ms DANZIG grain; the real interpreter dispatched them as single (longer)
  instructions with internal operand fetches.
- LANDISP runs in task context per P70-P71.agc; its 3 ms is an estimate of
  the tape-meter conversions.

## Source tensions the sim had to arbitrate

Two independent validation passes audited the artifacts and the mechanics;
these are the places where the 1969 sources disagree with each other and
what the sim chose:

- **Keying instants.** The first keystroke lands at PDI+303.3 and the ENTR
  at +306 — consistent with [Cherry]'s "+304" if his event log stamps the
  start of typing. The alarm GETs are pinned to Cherry's offsets
  (+316/+358 → 102:38:21 / 102:39:03); SP-4029 stamps 102:38:22 /
  102:39:02 — the two sources disagree at the ±1 s level and no choice
  matches both simultaneously.
- **Monitor-to-alarm gap.** [Tillman] says "in each case after 12 seconds
  of a monitor verb"; Cherry's own event offsets give ~10-12 s. The sim
  lands at 10.0 s and 11.0 s.
- **Third-use duration.** Cherry's log (+374 keyed, +380 KEY REL) gives
  6 s, which the sim reproduces; Tillman's prose says "9 or 10 seconds".
  Either duration stays clean here (three boundaries, at most seven cores
  at the worst instant).
- **The no-monitor demand.** The [Outline]'s phase table characterizes
  locked-P63-no-monitor as ~102% ("leaking slowly"); the flight record
  shows no alarms in that configuration for minutes. The sim resolves the
  tension through the theft sweep: 13.7% average (Grumman's measured
  region) with 15.0% peaks — 97.7% average demand, ~98% in the worst 2 s
  window, and zero leak until the monitor's load arrives.
- **EJSCAN tie direction.** EJ1's compare is ones'-complement: equal
  magnitudes sum to minus zero and CCS on -0 proceeds with the search,
  keeping the EARLIER find. Only identical words (equal-priority NOVAC
  jobs) can tie; FINDVAC words always differ by VAC address.
- **Strip sampling.** The occupancy strips sample each second's final
  millisecond; boundary transients (READACCS + LRH straddle + a fresh
  SERVICER) can exceed the strip maxima between samples, which is why the
  accounting section's peak line is the authoritative peak.
