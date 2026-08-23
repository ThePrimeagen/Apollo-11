# Memory Timeline — How Scratchpad Exhaustion Froze the Landing Computer

This document is the **visual causal timeline**: exact erasable addresses, ASCII memory maps,
and the code that moved each bit of state. It answers “where did the memory go, what was
running, and how did we run out?”

For mission clock / air-to-ground times, see [`timeline.markdown`](timeline.markdown).
For prose explanations, see [`memory_leak.md`](memory_leak.md) and
[`radar_problem.md`](radar_problem.md).
For a talk-through in simplified C (same MEMORY_LEAK1..9 steps), see [`timeline.c`](timeline.c):

```bash
gcc -o timeline timeline.c && ./timeline
```

For an interactive Rose Pine TUI (step leaks + play the 2s cycle bars), see [`timeline-tui/`](timeline-tui/):

```bash
cd timeline-tui && go run .
```

Addresses below are **octal**, computed from Luminary 099
`ERASABLE_ASSIGNMENTS.agc` (`SETLOC 67` → core sets → VAC pool).

---

## The pool that ran dry

```text
Erasable RAM (2,048 words total)
│
├── 035 / 036 ………… CDUT / CDUS   ← radar steals TIME here (0 words allocated)
├── 026 ………………… TIME3         ← timer still perfect; fires every 2.00 s
│
├── 154 … 313 ………… 8 CORE SETS   (96 words)   ← every job needs one
│     each: 12 words; free when PRIORITY = −0
│
└── 400 … 733 ………… 5 VAC AREAS   (220 words)  ← Interpreter jobs need one too
      each: 1 use-flag + 43 workspace = 44 words
      free when VACnUSE > 0; busy when VACnUSE = 0
```

One unfinished `SERVICER` holds **55 words** of workspace (12 + 43), plus its busy VAC flag.
Five concurrent copies can empty the VAC pool; eight can empty the core-set pool. Either
failure is what the crew saw as a brief freeze: PROG light, alarm code, software restart.

---

## Absolute address map (exact locations)

### Core sets — `MPAC` … `PRIORITY`, stride `COREINC = 12`

```text
   octal
   154  ┌──────────────────────────────────────────────────────────────┐
        │ CORE SET 0                                                   │
   154  │ MPAC..MPAC+6                                                 │
   163  │ MODE                                                         │
   164  │ LOC                                                          │
   165  │ BANKSET                                                      │
   166  │ PUSHLOC                                                      │
   167  │ PRIORITY   ← free = −0 ; busy = positive priority (+ VAC ptr)│
   170  ├──────────────────────────────────────────────────────────────┤
        │ CORE SET 1   MPAC=170  …  PRIORITY=203                       │
   204  ├──────────────────────────────────────────────────────────────┤
        │ CORE SET 2   MPAC=204  …  PRIORITY=217                       │
   220  ├──────────────────────────────────────────────────────────────┤
        │ CORE SET 3   MPAC=220  …  PRIORITY=233                       │
   234  ├──────────────────────────────────────────────────────────────┤
        │ CORE SET 4   MPAC=234  …  PRIORITY=247                       │
   250  ├──────────────────────────────────────────────────────────────┤
        │ CORE SET 5   MPAC=250  …  PRIORITY=263                       │
   264  ├──────────────────────────────────────────────────────────────┤
        │ CORE SET 6   MPAC=264  …  PRIORITY=277                       │
   300  ├──────────────────────────────────────────────────────────────┤
        │ CORE SET 7   MPAC=300  …  PRIORITY=313                       │
   314  └──────────────────────────────────────────────────────────────┘
```

### VAC areas — `VACnUSE` then 43-word workspace

```text
   400  VAC1USE ──► 401 … 453  VAC1   (43 words)
   454  VAC2USE ──► 455 … 527  VAC2
   530  VAC3USE ──► 531 … 603  VAC3
   604  VAC4USE ──► 605 … 657  VAC4
   660  VAC5USE ──► 661 … 733  VAC5
```

### Related hardware words (no allocation, only time)

```text
   026  TIME3     WAITLIST clock (overflow → T3RUPT)
   035  CDUT      RR trunnion counter  } each PINC/MINC = 11.72 µs stolen
   036  CDUS      RR shaft counter     } ~15% CPU when both at max rate
   375  FAILREG   alarm codes (crew reads with V05 N09)
```

---

## How to read each step

Each step shows:

1. **what operation ran**, and **which file/label**;
2. a **memory board** after that step (`·` free, `S0`/`S1`/… = which SERVICER copy owns it);
3. the **code that caused that board change**.

Legend for the board:

```text
  PRIORITY row:  · = −0 (free)     S0 = owned by SERVICER copy 0, etc.
  VAC USE row:   · = positive/free  0  = claimed (busy)
```

---

## T = 0 — Healthy landing cycle (baseline)

One `SERVICER` owns one core set and one VAC. Plenty of spare slots. Timer re-armed for +2 s.

```text
CORE PRIORITY @ 167 203 217 233 247 263 277 313
                │   │   │   │   │   │   │   │
                S0  ·   ·   ·   ·   ·   ·   ·     (other jobs may also occupy sets)

VAC USE @       400 454 530 604 660
                │   │   │   │   │
                0   ·   ·   ·   ·                 VAC1 claimed by S0
```

No freeze. Copy `S0` will finish, hit `ENDOFJOB`, and free both slots before the next cycle.

---

## Step 1 — Re-arm demand for exactly 2.00 seconds

**File:** `Luminary099/SERVICER.agc` · `GOREADAX` · MEMORY_LEAK1

**Operation:** schedule the next `READACCS` task. No check that the old job finished.
**Memory moved:** none. Only WAITLIST time-list entries change.

```text
BEFORE / AFTER (erasable pool unchanged)
CORE  S0 · · · · · · ·
VAC   0  · · · ·
```

```agc
GOREADAX  TC      GNUTFAZ5
          CA      2SECS          # exactly 2.00 seconds
          TC      VARDELAY       # no "is old SERVICER done?" test
```

---

## Step 2 — Hardware timer fires on time (CPU load irrelevant)

**File:** `Luminary099/WAITLIST.agc` · `T3RUPT` · MEMORY_LEAK2

**Operation:** `TIME3` overflow dispatches the due WAITLIST task (`READACCS`).
**Memory moved:** none in the core/VAC pools. Radar stole instruction cycles earlier; it did
**not** slow this counter.

```text
TIME3 @ 026  ──overflow──►  T3RUPT  ──dispatch──►  READACCS

CORE / VAC still whatever the unfinished jobs hold
(radar only burned wall-clock time at CDUT/CDUS 035/036)
```

```agc
T3RUPT    EXTEND
          ROR     SUPERBNK
          TS      BANKRUPT
          ...
T3RUPT2   CAF     NEG1/2         # DISPATCH WAITLIST TASK
          ...
          ADS     TIME3
```

Mismatch created here:

```text
  supply:  ~85% of expected CPU  (15% stolen at CDUT/CDUS)
  demand:  still one new cycle every 2.00 s
```

---

## Step 3 — Short task starts another guidance cycle

**File:** `Luminary099/SERVICER.agc` · `READACCS` · MEMORY_LEAK3

**Operation:** read PIPA counters; prepare to allocate a new `SERVICER`.
**Memory moved:** none yet (task, not a VAC job).

```text
CORE  S0 · · · · · · ·     ← old copy still unfinished under overload
VAC   0  · · · ·
```

```agc
READACCS  CS      OCT37771
          AD      TIME5
          ...
          TC      PIPASR          # READ THE PIPAS
```

---

## Step 4 — Request a brand-new `SERVICER` (never reuses the old one)

**File:** `Luminary099/SERVICER.agc` · after `PIPSDONE` · MEMORY_LEAK4

**Operation:** `FINDVAC` = “make a priority-20 Interpreter job with its **own** VAC + core set.”
**Memory moved:** about to claim the next free pair. Old `S0` keeps what it already holds.

```text
REQUEST:  new job SERVICER @ PRIO20, needs VAC + core set
HELD:     S0 still owns its pair (has not reached ENDOFJOB)
```

```agc
          CA      PRIO20
          TC      FINDVAC
          EBANK=  DVCNTR
          2CADR   SERVICER        # brand-new copy, not a resume of S0
```

---

## Step 5 — Claim a VAC area (`VACnUSE ← 0`)

**File:** `Luminary099/EXECUTIVE.agc` · `VACFOUND` · MEMORY_LEAK5

**Operation:** zero the free use-flag; pack VAC address into the low 9 bits of `NEWPRIO`.
**Memory moved:** one 44-word VAC slot leaves the free pool.

```text
AFTER claiming VAC2 for new copy S1:

CORE PRIORITY   S0  ·  ·  ·  ·  ·  ·  ·     (core claim is next step)
VAC USE @400…   0   0  ·  ·  ·
                │   │
                S0  S1   ← two VAC slots now busy
```

```agc
VACFOUND  AD      TWO
          ZL
          INDEX   A
          LXCH    0 -1            # write 0 into VACnUSE (claim)
          ADS     NEWPRIO         # pack VAC address into priority word
```

---

## Step 6 — Claim a core set (`PRIORITY ← +prio`)

**File:** `Luminary099/EXECUTIVE.agc` · `CORFOUND` · MEMORY_LEAK6

**Operation:** write positive priority into the chosen set’s `PRIORITY` word.
**Memory moved:** one 12-word core set leaves the free pool. `S1` now holds **both**.

```text
AFTER CORFOUND for S1 (illustrative: core set 1 @ PRIORITY 203):

     addr: 167 203 217 233 247 263 277 313
CORE       S0  S1  ·   ·   ·   ·   ·   ·

     addr: 400 454 530 604 660
VAC        0   0   ·   ·   ·
           S0  S1
```

```agc
CORFOUND  CA      NEWPRIO
          INDEX   LOCCTR
          TS      PRIORITY        # positive = busy
          MASK    LOW9
          INDEX   LOCCTR
          TS      PUSHLOC
```

Words held by two unfinished copies: `2 × 55 = 110` (+ two busy flags).

---

## Step 7 — Long priority-20 job runs (and gets preempted)

**File:** `Luminary099/SERVICER.agc` · `SERVICER` · MEMORY_LEAK7

**Operation:** average-G, guidance, throttle/attitude, displays. Priority **20** — lowest of
the landing jobs — so radar (~32), keyboard (~30), monitor (~30), gyro bias (21) all cut in.
**Memory moved:** none released. Ownership unchanged while it works.

```text
CPU under radar theft (~15% gone):

  ┌─ 2.00 s wall clock ─────────────────────────────────────────┐
  │████ radar PINC/MINC ████│▓▓ higher-prio jobs ▓▓│░░ S1 ░░│… │
  │        ~0.30 s          │                      │ late…  │   │
  └─────────────────────────────────────────────────────────────┘
         S0 may still be queued unfinished from the previous cycle
```

```agc
SERVICER  TC      PHASCHNG
          OCT     16035
          OCT     20000
          EBANK=  DVCNTR
          ...
          TC      INTPRET         # needs the VAC claimed in step 5
```

---

## Step 8 — The finish line it must reach (often missed)

**File:** `Luminary099/SERVICER.agc` · `SERVEXIT` · MEMORY_LEAK8

**Operation:** intended exit into `ENDOFJOB`. Under overload, the **next** Step 1–4 can run
**before** this line executes for the old copy.

```text
HEALTHY:     S0 reaches here → Step 9 frees memory → pool stable
OVERLOAD:    S0 still above this line when S1 (then S2…) is allocated
             ┌──────────────┐
             │ stub S0 holds│ 55 words forever (until restart)
             └──────────────┘
```

```agc
SERVEXIT  TC      PHASCHNG
          OCT     00035
 +2       TCF     ENDOFJOB        # only path that returns the pool
```

---

## Step 9 — Actual release (the line overload prevents in time)

**File:** `Luminary099/EXECUTIVE.agc` · `ENDJOB1` · MEMORY_LEAK9

**Operation:** `PRIORITY ← −0` (free core set); restore `VACnUSE` to a positive address (free VAC).
**Memory moved:** 55 words (+ flag) return to the pool — **only if this runs**.

```text
AFTER a successful ENDOFJOB for S0:

CORE  ·  S1 · · · · · ·
VAC   ·  0  · · ·
         S1 only
```

```agc
ENDJOB1   INHINT
          CS      ZERO
          TS      BUF +1
          XCH     PRIORITY        # deposits −0 → core set free
          MASK    LOW9
          TS      L
          ...
          INDEX   A
          TS      0               # restore VACnUSE → VAC free
```

If Step 9 loses the race to Steps 1–6 every cycle, the board grows:

---

## Compounding — one extra held pair every overloaded 2 s

Illustrative growth after V16 N68 tipped duty cycle past 100% (≈40 s to first 1202):

```text
t+0s   CORE  S0 ·  ·  ·  ·  ·  ·  ·     VAC  0 · · · ·
t+2s   CORE  S0 S1 ·  ·  ·  ·  ·  ·     VAC  0 0 · · ·
t+4s   CORE  S0 S1 S2 ·  ·  ·  ·  ·     VAC  0 0 0 · ·
t+6s   CORE  S0 S1 S2 S3 ·  ·  ·  ·     VAC  0 0 0 0 ·
t+8s   CORE  S0 S1 S2 S3 S4 ·  ·  ·     VAC  0 0 0 0 0   ← VAC pool FULL
       … other jobs also eat core sets …
       CORE  S0 S1 S2 S3 S4 xx xx xx     ← core pool FULL → 1202
```

(`xx` = radar / keyboard / monitor / other jobs that also need core sets.)

VAC is the tighter structural limit (5 vs 8), but **1202 happened four times** and **1201
once** because other jobs share core sets.

---

## The freeze — no compatible memory left

### Path A — no VAC areas → alarm **1201**

**File:** `Luminary099/EXECUTIVE.agc` · `FINDVAC2`

```text
VAC USE @400 454 530 604 660
         0   0   0   0   0     ← every CCS falls through
```

```agc
FINDVAC2  CCS     VAC1USE
          TCF     VACFOUND
          CCS     VAC2USE
          TCF     VACFOUND
          CCS     VAC3USE
          TCF     VACFOUND
          CCS     VAC4USE
          TCF     VACFOUND
          CCS     VAC5USE
          TCF     VACFOUND
          TC      BAILOUT1
          OCT     1201            # NO VAC AREAS
```

### Path B — no core sets → alarm **1202**

**File:** `Luminary099/EXECUTIVE.agc` · `NOVAC3` / after `NEXTCORE`

```text
PRIORITY @167 203 217 233 247 263 277 313
          xx  xx  xx  xx  xx  xx  xx  xx   ← none are −0
```

```agc
NOVAC3    INDEX   LOCCTR
          CCS     PRIORITY
          TCF     NEXTCORE
          ...
NEXTCORE  CAF     COREINC         # +12
          ADS     LOCCTR
          CCS     EXECTEM2
          TCF     NOVAC3
          TC      BAILOUT1
          OCT     1202            # NO CORE SETS
```

### What the crew experienced as the “freeze”

```text
  BAILOUT1 ──► store code in FAILREG @375
           ──► PROGLARM lights PROG on DSKY
           ──► software restart (ENEMA / GOJAM path)
                 │
                 ├─ wipe job/task queues  → all stub SERVICER copies vanish
                 ├─ free core sets / VAC areas
                 └─ rebuild from phase table: ONE fresh READACCS + ONE SERVICER
                      engine / DAP / PIPA counters keep running through it
```

Navigation state was not wiped. The pause was the restart clearing the **scheduler desk**,
not a dead computer. That is why Houston could say “Go” — and why the same overload could
raise the alarm again minutes later in P64.

---

## One-page causal chain (code ↔ memory)

```text
 CDUT/CDUS 035,036          TIME3 026               core 154–313 / VAC 400–733
 ─────────────────          ─────────               ────────────────────────────
 radar PINC/MINC            T3RUPT on time          FINDVAC claims new pair
 steal ~15% CPU      ┐                            ┌  old pair still held
                     │      READACCS every 2s     │  until ENDOFJOB
                     ├────────────────────────────┤
                     │   SERVICER (prio 20) late  │
                     └──────────► stubs pile up ──► 1201 / 1202 ──► restart wipe
```

| Step | Code locus | Erasable effect |
| :--- | :--------- | :-------------- |
| Radar theft | hardware → `CDUT`/`CDUS` | **0 words**; time only |
| Re-arm | `GOREADAX` | WAITLIST only |
| Dispatch | `T3RUPT` | none in pool |
| New job | `TC FINDVAC` | request +55 words |
| Claim VAC | `VACFOUND` | `VACnUSE ← 0` |
| Claim core | `CORFOUND` | `PRIORITY ← +prio` |
| Work | `SERVICER` | holds both |
| Missed exit | before `TCF ENDOFJOB` | leak continues |
| Release | `ENDJOB1` | returns both (if reached) |
| Freeze | `BAILOUT1` + `OCT 1201/1202` | pool empty → restart |

---

## Source navigation

```bash
grep -rn "MEMORY_LEAK[0-9]" Luminary099/*.agc
grep -rn "ALARM_RECOVERY[0-9]" Luminary099/*.agc
:luafile memory_leak.lua
:luafile alarm_recovery.lua
```

Related reading: [`definitions.md`](definitions.md) · [`walkthrough.md`](walkthrough.md) ·
[`alarm_recovery.md`](alarm_recovery.md) · mission times in [`timeline.markdown`](timeline.markdown).
