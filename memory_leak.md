# Part 2 — The Unfinished-Job Memory Leak

This is the second causal chain. It starts after the rendezvous radar has removed about 15% of
the CPU and explains how **unfinished copies of `SERVICER` keep core sets and VAC areas
claimed**.

This is called a "memory leak" for readability. It is not a lost pointer or heap leak in the
modern C/C++ sense. The Executive knows exactly who owns every resource. The problem is that
an unfinished job legitimately retains its resources while the fixed timer keeps creating new
copies.

Read [`definitions.md`](definitions.md) and [`radar_problem.md`](radar_problem.md) first.
Open the matching Vim tour with `:luafile memory_leak.lua`.

## The resource equation

Each `SERVICER` copy needs:

- one of **8 core sets** (12 words each), and
- one of **5 VAC areas** (44 words each), because it uses the Interpreter.

The only normal release happens after that copy finishes and reaches `ENDOFJOB`.

## Source trail

Run:

```bash
grep -rn "MEMORY_LEAK[0-9]" Luminary099/*.agc
```

### MEMORY_LEAK1 — Re-arm the next cycle for exactly two seconds

File: `Luminary099/SERVICER.agc`, label `GOREADAX`

```text
GOREADAX  TC      GNUTFAZ5
          CA      2SECS
          TC      VARDELAY
```

This schedules the next `READACCS` task for exactly 2.00 seconds later. It does not wait for the
current `SERVICER` job and contains no "is the old copy done?" test.

This fixed demand is safe only while average work finishes faster than new work arrives.

### MEMORY_LEAK2 — The hardware timer dispatches punctually

File: `Luminary099/WAITLIST.agc`, label `T3RUPT`

```text
T3RUPT   EXTEND
         ROR      SUPERBNK
         TS       BANKRUPT
         ...
T3RUPT2  CAF      NEG1/2       # DISPATCH WAITLIST TASK
```

The radar steals instruction/memory cycles, but it does not slow the hardware timer. `TIME3`
still overflows on schedule and `T3RUPT` dispatches `READACCS` every two seconds.

This creates the mismatch:

- supply: only about 85% of expected execution time;
- demand: still one new guidance cycle every 2.00 seconds.

### MEMORY_LEAK3 — `READACCS` starts another cycle

File: `Luminary099/SERVICER.agc`, label `READACCS`

```text
READACCS  CS      OCT37771
          AD      TIME5
          ...
          TC      PIPASR        # READ THE PIPAS
```

`READACCS` is deliberately short. It captures accelerometer data and proceeds to schedule the
large `SERVICER` job. It does not search for older copies because the design assumes the old
copy completed within the two-second period.

### MEMORY_LEAK4 — Request a brand-new `SERVICER`

File: `Luminary099/SERVICER.agc`

```text
          CA      PRIO20
          TC      FINDVAC
          EBANK=  DVCNTR
          2CADR   SERVICER
```

`FINDVAC` means: "create a priority-20 job that needs both a VAC area and a core set." It does
not reuse an old `SERVICER` allocation. A previous unfinished copy remains a separate valid
job with separate private state.

### MEMORY_LEAK5 — Claim a VAC area

File: `Luminary099/EXECUTIVE.agc`, label `VACFOUND`

```text
VACFOUND  AD      TWO
          ZL
          INDEX   A
          LXCH    0 -1
          ADS     NEWPRIO
```

The preceding scan found a free `VACnUSE` word. `VACFOUND` writes zero into that use-word,
marking the associated 44-word Vector Accumulator workspace busy and attaching its address to
the new job's bookkeeping.

That VAC area remains unavailable until this exact job finishes.

### MEMORY_LEAK6 — Claim a core set

File: `Luminary099/EXECUTIVE.agc`, label `CORFOUND`

```text
CORFOUND  CA      NEWPRIO
          INDEX   LOCCTR
          TS      PRIORITY
          MASK    LOW9
          INDEX   LOCCTR
          TS      PUSHLOC
```

`CORFOUND` writes the positive job priority into the selected core set's `PRIORITY` word.
That marks the 12-word core set busy. The new `SERVICER` now owns both resources.

### MEMORY_LEAK7 — The long, low-priority job begins

File: `Luminary099/SERVICER.agc`, label `SERVICER`

```text
SERVICER  TC      PHASCHNG
          OCT     16035
          OCT     20000
          EBANK=  DVCNTR
```

`SERVICER` performs, in order:

1. average-G navigation;
2. guidance equations;
3. mass/throttle calculations;
4. attitude commands to the digital autopilot;
5. display updates.

It is priority 20, lower than radar, keyboard, and several sensor jobs. Those jobs preempt it.
With 15% CPU theft, it can miss the two-second finish line.

### MEMORY_LEAK8 — The finish point it is trying to reach

File: `Luminary099/SERVICER.agc`, label `SERVEXIT`

```text
SERVEXIT  TC      PHASCHNG
          OCT     00035

 +2       TCF     ENDOFJOB
```

A successful copy eventually reaches this branch. If the next two-second task arrives first,
the old copy has **not** executed `TCF ENDOFJOB`; therefore, retaining its memory is correct
from the Executive's perspective.

The old copy is often called a "stub." Newer data makes the newest copy more relevant, but the
Executive has no policy allowing it to guess that the older copy is disposable.

### MEMORY_LEAK9 — The actual release

File: `Luminary099/EXECUTIVE.agc`, label `ENDJOB1`

```text
ENDJOB1  INHINT
         CS       ZERO
         TS       BUF +1
         XCH      PRIORITY
         MASK     LOW9
         TS       L
         ...
         INDEX    A
         TS       0
```

`ENDOFJOB` transfers here:

- `XCH PRIORITY` puts minus zero into the current core-set `PRIORITY`, marking it free;
- for a VAC job, the indexed `TS 0` restores the VAC use-word to a positive/free value.

If an old `SERVICER` has not reached this routine, both claims from MEMORY_LEAK5 and
MEMORY_LEAK6 remain active. Another two-second cycle claims another pair. Eventually:

- all five VAC areas are busy, or
- all eight core sets are busy.

That is the whole leak.

## Handoff to Part 3

The Executive now receives a new allocation request with no compatible resource left. Continue
with [`alarm_recovery.md`](alarm_recovery.md), which traces how it proves exhaustion, selects
1201 or 1202, reports the alarm, wipes the backlog, and resumes guidance.
