# Part 1 — The Rendezvous-Radar Problem

This is the first causal chain. It explains the **external hardware condition** that silently
removed about 15% of the Lunar Module Guidance Computer's execution time.

> This was the **rendezvous radar**, not the voice radio. The famous conversation with Houston
> happened later, after the radar problem caused a memory backlog and the computer reported an
> alarm.

Before reading this file, read [`definitions.md`](definitions.md). Then open the matching Vim
tour with `:luafile radar_problem.lua`.

## Outcome

The crew checklist put the rendezvous-radar mode switch in `AUTO`/`SLEW` during descent so the
radar would be warm if an abort required rendezvous. In those modes:

1. the ATCA excited the radar resolvers with one 800-Hz signal;
2. the radar CDUs compared that signal against another 800-Hz reference;
3. the signals were frequency-locked but not phase-synchronized;
4. near a 90° or 270° phase difference, both CDUs generated counter requests at their maximum
   rate;
5. those requests stole processor memory cycles without executing ordinary software.

The exact loss was approximately:

```text
2 counters × 6,400 requests/second × 11.72 microseconds/request
    = 0.150 seconds stolen per second
    = about 15% of the computer
```

The flight software had been tested for roughly 10% unexplained time loss (`TLOSS`), not 15%.

## Source trail

Run:

```bash
grep -rn "RADAR_PROBLEM[0-9]" Luminary099/*.agc
```

### RADAR_PROBLEM1 — The software/hardware control wires

File: `Luminary099/INPUT_OUTPUT_CHANNEL_BIT_DESCRIPTIONS.agc`

```text
# CHANNEL 12; OUTPUT CHANNEL
# BIT 1  ZERO RR CDU
# BIT 2  ENABLE CDU RADAR ERROR COUNTERS
```

These bits are the boundary between software and radar hardware:

- bit 1 tells the hardware to hold/zero the radar angle counters;
- bit 2 enables the radar error-counter machinery.

Software could turn those functions on or off. It did **not** generate each radar angle count
itself and could not inspect the phase relationship inside the external electronics.

### RADAR_PROBLEM2 — Software sees a valid radar mode

File: `Luminary099/T4RUPT_PROGRAM.agc`, label `RRAUTCHK`

```text
RRAUTCHK  CA      RADMODES
          EXTEND
          RXOR    CHAN33
          MASK    AUTOMBIT
          EXTEND
          BZF     RRCDUCHK
```

Every 480 ms, `RRAUTCHK` compared the remembered radar mode with channel 33's
`RR AUTO/POWER ON` input. It could detect a mode transition and perform normal turn-on,
zeroing, and failure checks.

What it could **not** see was the true fault: two nominally valid 800-Hz signals with an unsafe
phase difference. Therefore, every software-visible discrete could look correct while the
external ECDUs were about to generate useless counts.

### RADAR_PROBLEM3 — Normal zeroing ends

File: `Luminary099/P20-P25.agc`, label `RRZEROSB`

```text
RRZEROSB  EXTEND
          QXCH    RRRET
          CAF     BIT1
          EXTEND
          WOR     CHAN12       # TURN ON ZERO RR CDU
          ...
          CS      ONE
          EXTEND
          WAND    CHAN12       # REMOVE ZEROING BIT
```

`RRZEROSB` performs normal radar startup:

1. assert the radar-CDU zero command;
2. clear the software-visible angle counters;
3. remove the zero command;
4. allow the external radar CDUs to report real angles.

The bug was outside this logic. Once zeroing was removed, Apollo 11's phase-mismatched ECDUs
were free to hammer the counters. Luminary 099 contains no check for that electrical phase
condition because the interface documentation never told the software team it was possible.

### RADAR_PROBLEM4 — The counters that receive the pulses

File: `Luminary099/ERASABLE_ASSIGNMENTS.agc`

```text
CDUT      EQUALS  35           # REND RADAR TRUNNION CDU
CDUS      EQUALS  36           # REND RADAR SHAFT CDU
```

`CDUT` and `CDUS` are special low-memory hardware counters:

- `CDUT`: trunnion (elevation-like) radar angle;
- `CDUS`: shaft (rotation-like) radar angle.

Each external `PINC`/`MINC` request directly changes one counter and occupies one 11.72-µs
memory cycle. The current instruction pauses; there is no software interrupt and no call stack
showing where the time went.

This is the last software location in this trail. The component creating the false requests is
electrical hardware, so no line in this repository says "generate a bogus pulse." That absence
is important evidence, not a missing step.

## Handoff to Part 2

At this point:

- the timer hardware still keeps accurate time;
- ordinary jobs receive only about 85% of the CPU they expected;
- the next `READACCS` deadline still arrives every two seconds.

Continue with [`memory_leak.md`](memory_leak.md), which shows how missing that deadline turns
lost CPU time into exhausted core sets and VAC areas.

## Primary sources

- George W. Cherry, *Exegesis of the 1201 and 1202 Alarms Which Occurred During the Mission G
  Lunar Landing*: <https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf>
- Don Eyles, *Tales From the Lunar Module Guidance Computer*:
  <http://www.klabs.org/history/apollo_11_alarms/eyles_2004/eyles_2004.htm>
