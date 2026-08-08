# Part 3 — Detecting Exhaustion, Reporting the Alarm, and Recovering

This is the third causal chain. It begins when a new job asks the Executive for memory after
unfinished jobs have claimed all compatible resources.

Read [`definitions.md`](definitions.md), [`radar_problem.md`](radar_problem.md), and
[`memory_leak.md`](memory_leak.md) first. Open the matching Vim tour with
`:luafile alarm_recovery.lua`.

## Two alternative detection branches, one shared recovery

The first four markers are not four sequential instructions from one alarm:

- **VAC exhaustion branch:** ALARM_RECOVERY1 → ALARM_RECOVERY2 → ALARM_RECOVERY5
- **core-set exhaustion branch:** ALARM_RECOVERY3 → ALARM_RECOVERY4 → ALARM_RECOVERY5

Only one branch occurs for a given failed allocation. Both join at `BAILOUT1`, after which the
reporting and restart path is shared.

## Source trail

Run:

```bash
grep -rn "ALARM_RECOVERY[0-9]" Luminary099/*.agc
```

### ALARM_RECOVERY1 — Alternative A: scan five VAC areas

File: `Luminary099/EXECUTIVE.agc`, label `FINDVAC2`

```text
FINDVAC2  TS      EXECTEM1
          CCS     VAC1USE
          TCF     VACFOUND
          ...
          CCS     VAC5USE
          TCF     VACFOUND
```

A job that uses the Interpreter needs a VAC area. Each `CCS VACnUSE` checks one use-word:

- positive: free; branch to `VACFOUND`;
- zero/busy: continue scanning.

Falling through all five tests proves no VAC area is free.

### ALARM_RECOVERY2 — Alternative A: raise 1201

File: `Luminary099/EXECUTIVE.agc`

```text
          LXCH    EXECTEM1
          CA      Q
          TC      BAILOUT1
          OCT     1201           # NO VAC AREAS
```

`TC BAILOUT1` is followed by an inline data word. `BAILOUT1` will inspect the caller's return
address and read that following word as the alarm code: octal `1201`.

Continue at ALARM_RECOVERY5.

### ALARM_RECOVERY3 — Alternative B: scan eight core sets

File: `Luminary099/EXECUTIVE.agc`, labels `NOVAC2` / `NOVAC3`

```text
NOVAC2    CAF     ZERO
          TS      LOCCTR
          CAF     NO.CORES
NOVAC3    TS      EXECTEM2
          INDEX   LOCCTR
          CCS     PRIORITY
          TCF     NEXTCORE
NO.CORES  DEC     7
```

Every job needs a core set. The Executive tests one `PRIORITY` word, then repeats seven times,
for eight total sets. Minus zero means free. A positive priority means active/busy.

The nearby original comment saying "seven sets of eleven registers" is stale. Independent
source evidence proves Luminary 099 has eight sets of twelve words:

- `COREINC DEC 12`;
- `ERASE +83D` labeled "EIGHT SETS OF 12 REGISTERS EACH";
- the restart code clears eight `PRIORITY` words, 12 words apart.

### ALARM_RECOVERY4 — Alternative B: raise 1202

File: `Luminary099/EXECUTIVE.agc`, after `NEXTCORE`

```text
          LXCH    EXECTEM1
          CA      Q
          TC      BAILOUT1        # NO CORE SETS AVAILABLE
          OCT     1202
```

Reaching this path proves all eight core sets are occupied. The inline alarm data is octal
`1202`.

Continue at ALARM_RECOVERY5.

### ALARM_RECOVERY5 — Join both branches and catch the code

File: `Luminary099/ALARM_AND_ABORT.agc`, label `BAILOUT1`

```text
BAILOUT1  INHINT
          DXCH    ALMCADR
          CAF     ADR40400
BOTHABRT  TS      ITEMP1
          INDEX   Q
          CAF     0
          TS      L
          TCF     CHKFAIL1
```

`Q` contains the return address from `TC BAILOUT1`. `INDEX Q / CAF 0` reads the word at that
address: the caller's `OCT 1201` or `OCT 1202`. The code is placed in `L`.

Physical source order is unusual: `CHKFAIL1` and `PROGLARM` are above `BAILOUT1` in the file.
Follow marker numbers, not line-number direction.

### ALARM_RECOVERY6 — Store the code

File: `Luminary099/ALARM_AND_ABORT.agc`, label `CHKFAIL1`

```text
CHKFAIL1  CCS     FAILREG
          TCF     CHKFAIL2
          LXCH    FAILREG
          TCF     PROGLARM
```

The program scans three alarm registers (`FAILREG`, `FAILREG+1`, `FAILREG+2`) and stores the
new code in the first empty slot. The crew can read these with Verb 05 Noun 09.

### ALARM_RECOVERY7 — Light the PROG lamp

File: `Luminary099/ALARM_AND_ABORT.agc`, label `PROGLARM`

```text
PROGLARM  CS      DSPTAB +11D
          MASK    OCT40400
          ADS     DSPTAB +11D
```

This sets the program-alarm bit in the DSKY lamp table. This is the source location that makes
the yellow **PROG** light visible to Aldrin.

### ALARM_RECOVERY8 — Trigger a software restart

File: `Luminary099/ALARM_AND_ABORT.agc`, label `WHIMPER`

```text
WHIMPER   CA      TWO
          AD      Z
          TS      BRUPT
          RESUME
          TC      POSTJUMP
          CADR    ENEMA
```

`WHIMPER` arranges for `RESUME` to land at `POSTJUMP`, which transfers to `ENEMA`. This enters
the restart machinery cleanly from software.

### ALARM_RECOVERY9 — Enter restart code

File: `Luminary099/FRESH_START_AND_RESTART.agc`, label `ENEMA`

```text
ENEMA     INHINT
          TC      STARTSB1
          TCF     GOPROG2A
```

The restart reinitializes scheduling state while deliberately preserving flight-critical
hardware state, including engine on/off and IMU/gyro operation. The LM keeps following its
last valid steering commands during the short rebuild.

### ALARM_RECOVERY10 — Wipe queues and free all resources

File: `Luminary099/FRESH_START_AND_RESTART.agc`

```text
          CS      ZERO
          TS      PRIORITY
          TS      PRIORITY +12D
          ...
          TS      PRIORITY +84D

          CAF     VAC1ADRC
          TS      VAC1USE
          ...
          TS      VAC5USE
```

This clears:

- the WAITLIST task queue;
- all eight core-set `PRIORITY` words;
- all five VAC-area use-words.

Every unfinished `SERVICER` stub disappears. Nonessential work not protected by a restart
phase (including the crew's V16 N68 monitor) also disappears, shedding load.

### ALARM_RECOVERY11 — Rebuild one current cycle

File: `Luminary099/RESTART_TABLES.agc`, label `5.4SPOT`

```text
5.4SPOT   DEC     200
          EBANK=  DVCNTR
         -2CADR   REREADAC

          OCT     20000
          EBANK=  DVCNTR
          2CADR   SERVICER
```

The phase table recreates exactly:

- one `REREADAC` timer task due in 200 centiseconds;
- one priority-20 `SERVICER` job.

It does **not** recreate the accumulated old copies.

### ALARM_RECOVERY12 — Re-read sensors and continue

File: `Luminary099/SERVICER.agc`, label `REREADAC`

```text
REREADAC  CCS     PIPAGE
          TCF     READACCS
          ...
          CCS     DELVZ
```

The PIPA accelerometer counters are hardware counters. They continued accumulating velocity
increments while software restarted. `REREADAC` captures those increments, so navigation loses
no acceleration data.

The system is now back to:

- one current guidance job;
- one timer for the next cycle;
- current sensor data;
- clean core-set and VAC pools.

## Why Houston could say "Go"

The alarms meant "the scheduler's finite workspace is full," not "guidance calculations are
wrong." The software immediately:

1. preserved the code for diagnosis;
2. warned the crew;
3. discarded the backlog;
4. restarted only essential flight work.

MIT had deliberately tested restart protection. Mission Control could continue as long as the
trajectory remained correct and alarms did not recur faster than recovery could handle.

For the exact mission times, altitudes, and voice exchanges, continue to
[`timeline.markdown`](timeline.markdown).
