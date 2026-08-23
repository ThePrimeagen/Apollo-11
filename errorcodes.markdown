# The 1201 & 1202 Program Alarms — Apollo 11 Landing

## Brief outline

- **The story is true.** During Apollo 11's powered descent to the Moon on **20 July 1969**,
  the Lunar Module *Eagle*'s guidance computer (the LGC) repeatedly flashed **program alarms**
  — codes **1202** and **1201**. There were **five alarms total: four 1202s and one 1201**,
  at these times (from the MIT post-flight analysis; PDI = Powered Descent Initiation):

  | Alarm | Time | Program |
  | :---- | :--- | :------ |
  | 1202 | PDI + 316 s | P63 (braking phase) |
  | 1202 | PDI + 356 s | P63 |
  | 1201 | PDI + 552 s | P64 (visibility phase) |
  | 1202 | PDI + 578 s | P64 |
  | 1202 | PDI + 594 s | P64 |

- **What the codes mean** (both are "Executive overflow" — more jobs requested than the
  scheduler had memory for):
  - **1202 — "EXECUTIVE OVERFLOW - NO CORE SETS"**: all **8 core sets** (12-word blocks of
    erasable memory that every job needs) were in use.
  - **1201 — "EXECUTIVE OVERFLOW - NO VAC AREAS"**: all **5 VAC areas** (44-word Vector
    Accumulator workspaces needed by jobs doing interpretive vector math) were in use.
- **Root cause: a hardware/documentation interface problem, not a software bug.** The
  rendezvous radar (RR) mode switch was in `AUTO TRACK`/`SLEW` per the crew checklist. In
  those modes the radar angle resolvers were excited by an 800 Hz signal whose phase was
  random relative to the computer's 800 Hz reference, so the radar's coupling data units
  (CDUs) bombarded two hardware counters in the LGC with spurious increment pulses at the
  maximum rate — stealing ~15% of all processor time.
- **Why the landing continued:** each alarm triggered a software **restart** (`BAILOUT` →
  `ENEMA`) that wiped the scheduler queues and rebuilt only the essential jobs from
  pre-registered restart tables. Guidance never stopped steering. Mission control (Steve
  Bales, with Jack Garman in the back room) called **"Go"** on every alarm, and *Eagle*
  landed safely.

The rest of this document is a full trace of the exact execution path through the code in
this repository (`Luminary099/`, the actual flight software), with the timing arithmetic.

## Focused source trails

This document is the older consolidated technical reference. For learning the source, use the
new three-part order in [`table_of_contents.md`](table_of_contents.md):

1. [`radar_problem.md`](radar_problem.md) + `radar_problem.lua`:
   `RADAR_PROBLEM1…4`;
2. [`memory_leak.md`](memory_leak.md) + `memory_leak.lua`:
   `MEMORY_LEAK1…9`;
3. [`alarm_recovery.md`](alarm_recovery.md) + `alarm_recovery.lua`:
   `ALARM_RECOVERY1…12`.

Each marker identifies one precise source location. The three separate quickfix lists prevent
the initiating radar fault, the job-resource leak, and the alarm/restart response from being
mixed into a single traversal.

---

## 1. The clock: what a "memory cycle" actually is

Everything in the AGC is measured in **Memory Cycle Times (MCT)**:

- The AGC's oscillator ran at **2.048 MHz**, divided to a **1.024 MHz** clock. One memory
  cycle — one read-modify-write of a 15-bit (+ parity) word of core memory — took **12 clock
  ticks ≈ 11.72 µs** (Eyles rounds to 11.7 µs; the MIT memos use 11.72 µs).
- Every machine instruction is built from MCTs: `TC` (jump) takes 1 MCT, `CA` (load
  accumulator) 2 MCTs, `CCS` (compare/branch) 2 MCTs, etc. So the CPU executed roughly
  **43,000–85,000 instructions per second** — about 85 kHz of memory cycles.
- The **Interpreter** (the virtual machine used for vector math) was far slower: a single
  double-precision vector cross product cost about **5 ms** (~425 memory cycles).

**The crucial hardware detail:** the LGC's low erasable addresses include *hardware
counters*. Peripherals don't raise an interrupt to update them; instead the hardware
performs an **unprogrammed sequence** (`PINC`/`MINC` — "plus/minus increment") that seizes
the memory bus for **exactly one MCT (11.72 µs)** and increments the counter directly. The
running program is simply *paused*: no interrupt, no context switch, nothing visible to
software except that wall-clock time passes faster than instructions execute. The two
rendezvous radar angle counters are declared here:

```
Luminary099/ERASABLE_ASSIGNMENTS.agc, lines 120-121

CDUT		EQUALS	35		# REND RADAR TRUNNION CDU
CDUS		EQUALS	36		# REND RADAR SHAFT CDU
```

## 2. The theft: 12,800 stolen cycles per second

With the RR mode switch in `AUTO TRACK` or `SLEW`, the ATCA (a Grumman-built assembly)
excited the radar's shaft/trunnion resolvers with an 800 Hz signal of **random phase**
relative to the 800 Hz reference used by the CDUs (an interface control document said
"frequency locked" but never "phase synchronized"). Reading a meaningless signal, each CDU
slewed its counter at the **maximum rate of 6,400 pulses per second**, futilely trying to
null an error that could not be nulled. Apollo 11 happened to power up near the worst-case
phase angle (≈90°/270°).

The arithmetic, exactly as George Cherry wrote it in August 1969:

```
2 counters × 6,400 pulses/sec = 12,800 PINC/MINC sequences per second
12,800 × 11.72 µs = 0.15 seconds stolen per second  =  15% of the CPU
```

Per 2-second guidance cycle (~170,600 MCTs available), roughly **25,600 memory cycles
(0.3 s) were stolen** doing nothing useful. MIT called unexplained drain like this
**TLOSS**. The software had been verified to tolerate ~10% TLOSS. It was getting 13–15%.

This is "phase 0" of the failure: **no code in this repository executes it at all.** That is
what made it so mysterious — the thief never appears in any listing.

## 3. The 2-second heartbeat: `READACCS` → `SERVICER`

Powered-flight navigation ran on a fixed 2-second cycle, built from a waitlist *task* and an
Executive *job*:

1. **`T3RUPT`** (`Luminary099/WAITLIST.agc`, line 380) — the waitlist clock interrupt fires
   when hardware counter `TIME3` (incremented every 10 ms) overflows, and dispatches the due
   task. Critically, `TIME3` is a *hardware counter*: the radar theft slowed job execution
   but **the clock kept perfect time**, so tasks kept firing punctually.

2. **`READACCS`** (`Luminary099/SERVICER.agc`, line 79) — the dispatched task. It reads the
   accelerometers (`TC PIPASR`, line 90), then schedules the big navigation/guidance job:

```
Luminary099/SERVICER.agc, lines 97-100

		CA	PRIO20
		TC	FINDVAC
		EBANK=	DVCNTR
		2CADR	SERVICER	# SET UP SERVICER JOB
```

3. …and re-schedules **itself** two seconds ahead, unconditionally (lines 71-74,
   `CA 2SECS / TC VARDELAY`, reached via `MAKEACCS`, line 138). Nothing checks whether the
   *previous* `SERVICER` ever finished. As Cherry wrote: "The clock … ineluctably counts
   down to the time for the next repetition of a job to begin whether the previous
   repetition is complete or not."

4. **`SERVICER`** — priority 20, a **VAC job**, the *lowest*-priority and *longest* job in
   the landing. In one pass it does average-G navigation → guidance equations → throttle and
   attitude (DAP) commands → display updates, then releases its core set and VAC area via
   `ENDOFJOB` (`Luminary099/EXECUTIVE.agc`, line 117). The jobs active during landing
   (from Cherry's memo): SERVICER (VAC, prio 20), MAKEPLAY (VAC, 20), 1/GYRO (NOVAC, 21),
   CHARIN — DSKY keystrokes (NOVAC, 30), MONDO — the V16N68 monitor (NOVAC, ~30),
   LRHJOB/LRVJOB — landing radar reads (NOVAC, 32), HIGATJOB (VAC, 32).

## 4. The allocator: `FINDVAC` scans, and the two exact failure points

`FINDVAC` (`Luminary099/EXECUTIVE.agc`, line 52) enters the Executive bank at `FINDVAC2`
and scans the five VAC-area "use" flags. **Failure point #1** — all five in use:

```
Luminary099/EXECUTIVE.agc, lines 133-147

FINDVAC2	TS	EXECTEM1	# (SAVE CALLER'S BANK FIRST.)
		CCS	VAC1USE
		TCF	VACFOUND
		CCS	VAC2USE
		TCF	VACFOUND
		CCS	VAC3USE
		TCF	VACFOUND
		CCS	VAC4USE
		TCF	VACFOUND
		CCS	VAC5USE
		TCF	VACFOUND
		LXCH	EXECTEM1
		CA	Q
		TC	BAILOUT1
		OCT	1201		# NO VAC AREAS.
```

If a VAC area is found (or the caller was `NOVAC`), it falls into `NOVAC2` and scans the
core sets' `PRIORITY` registers, spaced 12 words apart (a free set holds −0). The loop
counter `NO.CORES DEC 7` gives one initial probe plus seven more — **eight core sets**
(the "SEVEN SETS" comment in the code is stale; see §7). **Failure point #2** — all eight
in use:

```
Luminary099/EXECUTIVE.agc, lines 201-208

NEXTCORE	CAF	COREINC
		ADS	LOCCTR
		CCS	EXECTEM2
		TCF	NOVAC3
		LXCH	EXECTEM1
		CA	Q
		TC	BAILOUT1	# NO CORE SETS AVAILABLE.
		OCT	1202
```

Note the ordering: a VAC job checks VAC areas *first*. That is why the P64 alarm burst
contained a **1201** — at that failing request the five VAC areas ran out before the eight
core sets did.

## 5. The squeeze: how the queue actually filled up

Now put §2, §3 and §4 together. This is the precise loop that generated each alarm:

1. `READACCS` fires at T+0, reads accelerometers, calls `FINDVAC` → SERVICER copy **A**
   gets core set #n and VAC area #m, and starts running at priority 20.
2. The radar theft means copy A executes only ~85% as many instructions per wall-clock
   second as designed, and it keeps being preempted by the higher-priority jobs (radar
   reads at prio 32, DSKY keystroke processing at 30, the crew's V16N68 DELTAH monitor
   at ~30, gyro compensation at 21) — all of which are also running ~15% slow.
3. At T+2.00 exactly (hardware clock, unaffected by TLOSS), `READACCS` fires again and
   calls `FINDVAC` → SERVICER copy **B** gets a *second* core set and VAC area, because
   copy A **has not reached `ENDOFJOB` and still owns its memory**. Copy A is now a stale
   "stub": newer work is more relevant, but the old copy remains queued and retains its
   resources (and under some later scheduling conditions could resume).
4. Each cycle the deficit repeats. With the landing load at ~87–90% + 15% stolen ≈ 102–105%
   of real time, every 2-second cycle leaks roughly one more held-but-unfinished allocation.
   Within tens of seconds the free pool (8 core sets / 5 VAC areas, minus those legitimately
   held by the other jobs) is gone.
5. Some request — the next `READACCS`→`FINDVAC`, or a radar/display/monitor job — asks for
   the ninth core set (→ **1202**) or the sixth VAC area (→ **1201**). The Executive cannot
   comply, and jumps to `BAILOUT1` with the alarm code inline after the call.

This matches the observed spacing: the first 1202 came at PDI+316 s, only ~12–18 s (six to
nine 2-second cycles) after the crew keyed up the V16N68 monitor at ~PDI+300 s (Aldrin even
noticed the correlation: "It appears to come up when we have a 1668 up").

## 6. The alarm and the save: `BAILOUT` → `WHIMPER` → `ENEMA` → phase-table restart

**Step 1 — record the alarm.** `BAILOUT1` (`Luminary099/ALARM_AND_ABORT.agc`, line 189)
picks up the `OCT 1201`/`OCT 1202` word that follows the caller's `TC BAILOUT1` (via
`INDEX Q / CAF 0`), stores it in the first free slot of the three `FAILREG` registers
(`CHKFAIL1`, line 66), and lights the **PROG** lamp by setting the alarm bit in the DSKY
lamp table (`PROGLARM`, line 85). This is the moment Aldrin saw the yellow light. The crew
read the code back with Verb 05 Noun 09 (the Tillman memo: "displayed on the DSKY (by
V5N9E)").

**Step 2 — pull the ripcord.** Control falls into `WHIMPER`:

```
Luminary099/ALARM_AND_ABORT.agc, lines 144-149

WHIMPER		CA	TWO
		AD	Z
		TS	BRUPT
		RESUME
		TC	POSTJUMP	# RESUME SENDS CONTROL HERE
		CADR	ENEMA
```

`ENEMA` (`Luminary099/FRESH_START_AND_RESTART.agc`, line 281) is the **software restart** —
the same recovery path a hardware glitch would take through `GOPROG` (line 206, entered
from fixed address 4000 on a real "GOJAM"), but skipping the erasable-memory integrity
check.

**Step 3 — annihilate the queues.** `ENEMA` runs `STARTSB1`/`STARTSB2` (lines 399/428),
which flow into the initialization that wipes both schedulers — this single block of code
is what killed every zombie SERVICER stub *and* shed the non-essential load:

```
Luminary099/FRESH_START_AND_RESTART.agc, lines 475-529 (abridged)

		CAF	NEG1/2		# INITIALIZE WAITLIST DELTA-TS.
		TS	LST1 +7
		...
		CS	ZERO		# MAKE ALL EXECUTIVE REGISTER SETS
		TS	PRIORITY	# AVAILABLE.
		TS	PRIORITY +12D
		...
		TS	PRIORITY +84D
		...
		CAF	VAC1ADRC	# MAKE ALL VAC AREAS AVAILABLE.
		TS	VAC1USE
		...
		TS	VAC5USE
```

(Those eight `PRIORITY +0…+84D` stores, 12 apart, are the eight core sets.) Note what it
deliberately does **not** touch: engine-on/off state, IMU modes, gyro enables — the
spacecraft keeps flying on its last commands during the ~fraction-of-a-second rebuild.

**Step 4 — rebuild only what the phase tables say.** `GOPROG3` (line 290) first verifies
each restart group's phase against its complement (`PHASE1` vs `-PHASE1`; a mismatch means
corrupted state → alarm 1107 and a full fresh start — that never happened). Then the
`NXTRST` loop (line 323) walks the six restart groups and, for each active one, calls
`RESTARTS` (`Luminary099/RESTARTS_ROUTINE.agc`, line 35), which re-issues `FINDVAC`/
`NOVAC`/`WAITLIST` calls from `RESTART_TABLES.agc`. The landing's group 5, phase 5.4 —
registered by `READACCS` itself just before scheduling SERVICER — rebuilds exactly this:

```
Luminary099/RESTART_TABLES.agc, lines 257-263

5.4SPOT		DEC	200
		EBANK=	DVCNTR
	       -2CADR	REREADAC

		OCT	20000
		EBANK=	DVCNTR
		2CADR	SERVICER
```

That is: **one** waitlist task `REREADAC` due in 200 centiseconds (2.00 s), plus **one**
SERVICER job at priority 20 (`OCT 20000`). `REREADAC` (`Luminary099/SERVICER.agc`, line
594) re-reads the accelerometer counters — which, being *hardware counters*, had kept
integrating velocity right through the restart — so **no navigation data was lost**.

**Step 5 — load shedding, for free.** Everything *not* in the phase tables simply never
comes back: all the accumulated SERVICER stubs, and the crew's V16N68 DELTAH monitor job
(MONDO), which was classified as dispensable. That is why, after each P63 alarm, the DSKY
display snapped back from Noun 68 to Noun 63. In P63 this actually *cured* the overload for
a while — the restart was an accidental but effective load-shedding mechanism.

**Why P64 was worse:** P64 added landing-site-redesignation processing to every guidance
pass, leaving <10% margin *with no monitor verb at all*. Restarting shed nothing that
mattered; the deficit re-accumulated immediately — hence **three alarms in ~40 seconds**
(1201 at +552 s, 1202s at +578 s and +594 s). The alarms only stopped when Armstrong took
over attitude manually (ATT HOLD, then P66), which removed the auto-guidance redesignation
load. The LM landed 2 minutes 20 seconds later with no further alarms.

## 7. A correction, and the core-set count

An earlier revision of this document said this build had *seven* core sets, based on this
comment:

```
Luminary099/EXECUTIVE.agc, line 157

		CAF	NO.CORES	# SEVEN SETS OF ELEVEN REGISTERS EACH.
```

That comment is **stale** (probably inherited from the earlier CM program — COLOSSUS did
have 7 core sets, per Cherry). Three pieces of evidence in this repository prove Luminary
099 has **eight core sets of twelve registers**:

1. `COREINC DEC 12 # 12 REGISTERS PER CORE SET.` (`EXECUTIVE.agc`, line 318);
2. the erasable declaration `ERASE +83D # EIGHT SETS OF 12 REGISTERS EACH`
   (`ERASABLE_ASSIGNMENTS.agc`, line 383 — 84 words after core set 0's own 12);
3. the restart code freeing `PRIORITY +0` through `PRIORITY +84D` in steps of 12
   (`FRESH_START_AND_RESTART.agc`, lines 507-515) — eight registers.

The `DEC 7` at `NO.CORES` is a loop *counter*: one initial probe plus seven repeats = eight
core sets scanned before declaring 1202.

## 8. Aftermath: the fixes

- **Procedure:** later crews put the RR mode switch in LGC for descent.
- **Software:** PCR 848 ("Prevent RR ECDUs from Stealing LGC Memory Cycles", written by Don
  Eyles on 23 July 1969 — three days after the landing) made Luminary 1B monitor the
  RR-mode discrete and command the radar CDUs to zero whenever the switch wasn't in LGC,
  stopping the counter traffic at the source.
- **Longer term:** MIT prototyped a "variable SERVICER" whose cycle stretched under load
  instead of overlapping (never flown), and V16N68's function was folded into an existing
  job so no separate monitor job was needed.

## Sources

1. NASA/MIT memo — *Program Alarms in Powered Descent, Apollo 11* (Tillman, 31 Jul 1969):
   alarm tabulation, core set/VAC definitions, V5N9E display path.
   <https://ibiblio.org/apollo/Documents/Memo-Tillman690731_text.pdf>
2. George W. Cherry (Luminary Project Director), *Exegesis of the 1201 and 1202 Alarms Which
   Occurred During the Mission G Lunar Landing* (AG# 370-69, 4 Aug 1969): alarm times, the
   15% arithmetic (12,800 × 11.72 µs), job/priority table, restart philosophy.
   <https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf>
3. Don Eyles, *Tales From the Lunar Module Guidance Computer* (AAS 04-064, 2004): the
   800 Hz phase story, READACCS/SERVICER mechanics, restart protection, P63 vs P64 margins.
   <http://www.klabs.org/history/apollo_11_alarms/eyles_2004/eyles_2004.htm>
4. NASA — *Apollo 11 Lunar Surface Journal: Program Alarms*: FINDVAC scanning order
   (VAC areas before core sets) and restart behavior. 
   <https://www.nasa.gov/wp-content/uploads/static/history/alsj/a11/a11.1201-pa.html>
5. The flight code itself, in this repository: `Luminary099/EXECUTIVE.agc`,
   `SERVICER.agc`, `WAITLIST.agc`, `ALARM_AND_ABORT.agc`, `FRESH_START_AND_RESTART.agc`,
   `RESTARTS_ROUTINE.agc`, `RESTART_TABLES.agc`, `ERASABLE_ASSIGNMENTS.agc`.
