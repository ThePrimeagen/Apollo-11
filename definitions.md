# Definitions — Everything You Need to Read the Apollo 11 Alarm Trace

This is the first required document in the
[`table_of_contents.md`](table_of_contents.md) reading order. It is the glossary for all three
focused traces: [`radar_problem.md`](radar_problem.md), [`memory_leak.md`](memory_leak.md), and
[`alarm_recovery.md`](alarm_recovery.md). If you hit a word, acronym, assembly instruction, or
hard-coded number you don't recognize, it should be in one of the tables below.

---

## 0. Open Questions

Were verbs / nouns typed in by the astronaughts as "display altitude" or was it really
"V16 N68".  Did the astronaughts have to memorize all of the different codes?

| **Memory cycle (MCT)** | Memory Cycle Time | The fundamental time unit:
**11.72 µs**, the time to read/modify/write one word. Every instruction and
every hardware counter increment costs whole MCTs. |

Does this mean that a single cpu instruction cannot execute faster than 11.72us?

What does "Every instruction and every hardware counter increment costs whole MCTs."

GOJAM - What did the astronaughts see?  Did they see the 1201 as a number
printed? red light?  What was it and what happened when the machine would
cycle?  Would they lose control for some period of time?  If so, how long?

Manual landing... does that mean Aldrin couldn't see the ground, he had to go
off instruments and what the computer was outputing?  How did the restarting
effect this?

Nextcore -
The algorithm, could you write it in plain Odin such that we can have the
simplest version of the algorithm.

Where is the data being stored?  Can you give me the offsets into the RAM that VACs and CoreSets are at?

Lets create an animation that lines up with the code so i can step them through one at a time.

T3RUPT runs in perfect timing regardless of load... how?

## 1. Core concepts (the "what is VAC?" table)

| Term | Full name | What it actually means |
| :--- | :-------- | :--------------------- |
| **AGC** | Apollo Guidance Computer | The flight computer. ~2 KHz effective speed, 15-bit words, 36K words fixed (ROM) + 2K words erasable (RAM). |
| **LGC** | LM Guidance Computer | The name for the AGC when it's the one in the Lunar Module. Same hardware; different program (LUMINARY). |
| **LUMINARY 099** | — | The specific software build (revision 99) that flew Apollo 11's Lunar Module. This repo's `Luminary099/` folder. |
| **Core set** | — | A block of **12 words** of erasable memory that *every* job needs as its private scratchpad/registers while it runs. There are **8** of them. Think: one "thread control block." |
| **VAC** | **V**ector **AC**cumulator area | A **larger** block (44 words) of erasable memory that a job needs *in addition* to a core set **if** it does vector/matrix math via the Interpreter. There are only **5**. "VAC area" = the workspace for double-precision vector arithmetic. This is the resource whose exhaustion caused the **1201** alarm. |
| **Job** | — | A unit of work with a **priority**. The Executive runs the highest-priority ready job. Jobs can be suspended and resumed. Needs a core set (and maybe a VAC area). Example: `SERVICER`. |
| **Task** | — | A short piece of work scheduled to run **at a specific time** (not a priority). Managed by WAITLIST, run from a timer interrupt. Must be very brief. Example: `READACCS`. |
| **Priority** | — | A number attached to a job; higher wins the CPU. `SERVICER` runs at 20 (low); radar/keyboard jobs at 30–32 (higher), so they interrupt `SERVICER`. |
| **Executive** | — | The AGC's job scheduler (a mini operating system). Hands out core sets/VAC areas and picks which job runs. Lives in `EXECUTIVE.agc`. |
| **WAITLIST** | — | The AGC's timer-based task scheduler. Runs tasks when their countdown hits zero. Lives in `WAITLIST.agc`. |
| **Interpreter** | — | A software "virtual machine" providing vector/matrix and double-precision pseudo-instructions. Powerful but slow (a vector cross-product ≈ 5 ms). Jobs using it need a VAC area. |
| **DSKY** | **D**i**sk**a**y** (Display & Keyboard) | The astronaut's console: numeric displays, a keyboard, and warning lights (including the **PROG** light). Operated with **Verb**/**Noun** codes. |
| **Verb / Noun** | — | The DSKY command language. A Verb is an action ("display", "monitor"), a Noun is the data it acts on. E.g. **V16 N68** = "monitor (V16) the altitude-difference data (N68)"; **V05 N09** = "display alarm codes." |
| **PROG light** | Program alarm light | The DSKY warning lamp lit when the software raises a program alarm. This is what came on during the 1201/1202 events. |
| **PIPA** | **P**ulsed **I**ntegrating **P**endulous **A**ccelerometer | The IMU's accelerometers. They report velocity increments into hardware counters (`PIPAX/Y/Z`). |
| **IMU** | Inertial Measurement Unit | The gyro/accelerometer platform that gives the computer attitude and velocity. |
| **CDU** | Coupling Data Unit | Hardware that converts an analog angle (resolver) into digital counts the computer can read, by incrementing a counter. |
| **ECDU** | Electronic CDU | The rendezvous radar's version of a CDU. Its malfunction (spurious counting) is the root cause of the alarms. |
| **RR** | Rendezvous Radar | The radar used to find the Command Module during rendezvous. Left in the wrong mode (`AUTO`/`SLEW`) during the landing, it flooded the computer with fake counter increments. |
| **ATCA** | Attitude & Translation Control Assembly | Grumman-built box whose 800 Hz signal, out of phase with the computer's reference, drove the spurious ECDU counting. |
| **DAP** | Digital Autopilot | The software that controls the spacecraft's attitude by firing thrusters / gimbaling the engine. |
| **RCS / DPS** | Reaction Control System / Descent Propulsion System | The small attitude thrusters, and the main throttleable descent engine. |
| **PGNS / AGS** | Primary Guidance & Navigation System / Abort Guidance System | The main guidance system (the AGC) and the simpler backup computer. |
| **Memory cycle (MCT)** | Memory Cycle Time | The fundamental time unit: **11.72 µs**, the time to read/modify/write one word. Every instruction and every hardware counter increment costs whole MCTs. |
| **TLOSS** | Time loss | MIT's term for CPU time stolen by an unknown source. Designed to tolerate ~10%; the radar stole ~15%. |
| **Restart / GOJAM** | — | A full software reset that wipes the scheduler queues and rebuilds work from "phase tables." Triggered by hardware faults *or* by the software itself (`BAILOUT`). This is what saved the landing. |
| **Phase table** | — | Pre-planned checkpoints saying "if we restart now, resume these jobs/tasks from these points." Consulted during a restart. |
| **PDI** | Powered Descent Initiation | Ignition of the descent engine, the start (T=0) of the landing burn. |
| **P63 / P64 / P66** | Program 63 / 64 / 66 | Mission phases: P63 = braking, P64 = approach/visibility (landing-site redesignation), P66 = manual rate-of-descent. |
| **High Gate / Low Gate** | — | Trajectory milestones: High Gate (~7,400 ft) is where P63→P64 hands over and the LM pitches up so the crew can see the site. |

---

## 2. The scheduler's key routines (labels you'll jump through)

| Label | File | Role in the story |
| :---- | :--- | :---------------- |
| `READACCS` | `SERVICER.agc` | Timer **task**: reads accelerometers, schedules `SERVICER`, re-arms itself every 2 s. |
| `SERVICER` | `SERVICER.agc` | The big low-priority **job**: navigation → guidance → throttle/attitude → displays. The one that overran. |
| `REREADAC` | `SERVICER.agc` | Re-entry point used after a restart to re-read the accelerometers (so no data is lost). |
| `FINDVAC` | `EXECUTIVE.agc` | "Schedule a job that needs a VAC area." Entry that scans VAC areas then core sets. |
| `NOVAC` | `EXECUTIVE.agc` | "Schedule a job that needs only a core set." |
| `VACFOUND` | `EXECUTIVE.agc` | Marks a chosen VAC area busy (the **claim**). |
| `CORFOUND` | `EXECUTIVE.agc` | Marks a chosen core set busy (the **claim**). |
| `NEXTCORE` | `EXECUTIVE.agc` | Loop step that advances to the next core set; falls through to the **1202** alarm when none are left. |
| `ENDOFJOB` / `ENDJOB1` | `EXECUTIVE.agc` | The **release**: frees this job's core set (`PRIORITY := -0`) and VAC area. The overrun job never reaches it in time. |
| `BAILOUT` / `BAILOUT1` | `ALARM_AND_ABORT.agc` | Called when the Executive can't allocate. Stores the alarm code and triggers a restart. |
| `CHKFAIL1` | `ALARM_AND_ABORT.agc` | Stores the alarm code into a free `FAILREG` slot. |
| `PROGLARM` | `ALARM_AND_ABORT.agc` | Lights the PROG lamp. |
| `WHIMPER` | `ALARM_AND_ABORT.agc` | Forces the software restart (jumps to `ENEMA`). |
| `ENEMA` | `FRESH_START_AND_RESTART.agc` | Software-restart entry point. |
| `GOPROG3` / `NXTRST` | `FRESH_START_AND_RESTART.agc` | Verifies phase tables and rebuilds the surviving jobs/tasks. |
| `T3RUPT` | `WAITLIST.agc` | Timer interrupt that dispatches the next due task (keeps perfect time regardless of CPU load). |

---

## 3. Assembly instructions, thoroughly (yaYUL / AGC Block II)

This section covers **every instruction, pseudo-instruction, and instruction-like call** that
appears in the three tours and in the code snippets of the companion documents.

### 3.1 The registers (and why `TS Q` and `DXCH Z` make sense)

The AGC's "registers" are **memory-mapped**: they live at the lowest erasable addresses, so
any instruction that takes a memory operand can operate on them directly.

| Register | Address (octal) | Role |
| :------- | :-------------- | :--- |
| `A` | 0 | The accumulator. Almost every instruction reads or writes it. |
| `L` | 1 | "Low" register: second word of double-precision values; low half of products. |
| `Q` | 2 | Holds the return address after `TC`. Between calls it is fair game as a scratch register — code in the trace really does `TS Q`. |
| `EB` | 3 | Erasable-bank selector. |
| `FB` | 4 | Fixed-bank selector. |
| `Z` | 5 | **The program counter.** Because it is addressable, writing to `Z` *is* a jump — see the `DXCH Z` idiom below. |
| `BB` | 6 | Both-banks register (EB and FB together, saved/restored across calls). |
| `ZERO` | 7 | Always reads as +0. |

Interrupt handling has a parallel save set: `ARUPT`, `LRUPT`, `QRUPT`, `ZRUPT`, `BBRUPT`, and
**`BRUPT`** — the word that will be *executed as the first instruction* when `RESUME` returns
from the interrupt. `WHIMPER` exploits exactly that (idiom 4 below).

### 3.2 Basic machine instructions

Timing is in memory cycles (1 MCT ≈ 11.72 µs), from the standard Block II instruction set
documentation. Instructions marked **ext.** must be preceded by `EXTEND`.

**Loading and storing:**

| Mnemonic | Name | MCT | Plain meaning |
| :------- | :--- | :-- | :------------ |
| `CA k` | Clear and Add | 2 | Copy the contents of `k` into A. |
| `CAF k` | Clear and Add Fixed | 2 | Same as `CA`; the yaYUL spelling when `k` is in fixed (ROM) memory — used for constants. |
| `CAE k` | Clear and Add Erasable | 2 | Same as `CA`; the spelling when `k` is in erasable (RAM) memory. |
| `CS k` | Clear and Subtract | 2 | Copy the ones'-complement (negation) of `k` into A. `CS ZERO` yields −0, the "free" sentinel. |
| `DCA k` | Double Clear and Add (ext.) | 3 | Copy the 2-word value at `k`,`k+1` into A,L. |
| `DCS k` | Double Clear and Subtract (ext.) | 3 | Copy the negated 2-word value at `k`,`k+1` into A,L. |
| `TS k` | Transfer to Storage | 2 | Copy A into `k`. (On overflow it also skips one instruction — not relied on in our trace.) |
| `XCH k` | Exchange | 2 | Swap A and `k`. |
| `LXCH k` | L Exchange | 2 | Swap L and `k`. |
| `QXCH k` | Q Exchange (ext.) | 2 | Swap Q and `k` — how subroutines save their return address (`RRZEROSB` does `QXCH RRRET`). |
| `DXCH k` | Double Exchange | 3 | Swap A,L with `k`,`k+1`. `DXCH Z` is a jump (idiom 3). |
| `ZL` | Zero L | 2 | Set L to +0 (it is `LXCH ZERO`). |

**Arithmetic and logic** (ones'-complement, 15-bit words):

| Mnemonic | Name | MCT | Plain meaning |
| :------- | :--- | :-- | :------------ |
| `AD k` | Add | 2 | A := A + contents of `k`. |
| `ADS k` | Add to Storage | 2 | `k` := `k` + A; the sum is also left in A. Used to set bits: `ADS DSPTAB +11D`. |
| `DAS k` | Double Add to Storage | 3 | 2-word add of A,L into `k`,`k+1`. |
| `INCR k` | Increment | 2 | `k` := `k` + 1 (e.g. `INCR REDOCTR` counts restarts). |
| `COM` | Complement | 2 | Negate A (it is `CS A`). |
| `DOUBLE` | Double | 2 | A := 2 × A (it is `AD A`). |
| `MASK k` | Mask | 2 | A := A AND `k` — isolate bit fields. |
| `MP k` | Multiply (ext.) | 3 | A,L := A × `k` (double-length product). |
| `DV k` | Divide (ext.) | 6 | A := (A,L) ÷ `k`, remainder in L. The slowest basic instruction. |

**Control flow:**

| Mnemonic | Name | MCT | Plain meaning |
| :------- | :--- | :-- | :------------ |
| `TC k` | Transfer Control | 1 | Call `k`: jump there and leave the return address in Q. Also used for plain jumps when no return is needed. |
| `TCF k` | Transfer Control to Fixed | 1 | Jump to `k` **without** touching Q — a pure `goto`. |
| `CCS k` | Count, Compare, and Skip | 2 | The AGC's conditional branch. See the dedicated explanation below. |
| `BZF k` | Branch Zero to Fixed (ext.) | 1 or 2 | Jump to `k` if A is ±0 (1 MCT when the branch is taken, 2 when not). |
| `BZMF k` | Branch Zero or Minus to Fixed (ext.) | 1 or 2 | Jump to `k` if A is zero or negative. |
| `INDEX k` | Index | 2 | Add the contents of `k` to the **next instruction's word** before executing it — computed addressing (idioms 1 and 2). |
| `EXTEND` | Extend | 1 | Prefix selecting the extended opcode set for the next instruction. |
| `INHINT` | Inhibit Interrupts | 1 | Disable interrupts (enter a critical section). |
| `RELINT` | Release Interrupts | 1 | Re-enable interrupts. |
| `RESUME` | Resume | 2 | Return from an interrupt: restore `Z` from `ZRUPT` and execute the word in `BRUPT` as the first instruction back. |

**I/O channel instructions** (all extended; channels are a separate small address space for
hardware wires — see `CHAN12`/`CHAN30`/`CHAN33` in section 4):

| Mnemonic | Name | MCT | Plain meaning |
| :------- | :--- | :-- | :------------ |
| `ROR ch` | Read and OR | 2 | A := A OR channel `ch` (read input bits). |
| `RAND ch` | Read and AND | 2 | A := A AND channel `ch`. |
| `RXOR ch` | Read and XOR | 2 | A := A XOR channel `ch` — how `RRAUTCHK` detects a *changed* radar-mode bit. |
| `WOR ch` | Write OR | 2 | Channel `ch` := `ch` OR A (set output bits, e.g. "zero the RR CDUs"). |
| `WAND ch` | Write AND | 2 | Channel `ch` := `ch` AND A (clear output bits). |

### `CCS`, the four-way branch, in full

`CCS k` is the strangest and most important instruction in the trace. It examines `k`, stores
the **diminished absolute value** (`abs(k) − 1`, minimum +0) into A, and then continues at one
of the **four following instruction slots** depending on what `k` was:

```text
        CCS  k
slot 1:  ...          taken if k was  > +0   (positive, nonzero)
slot 2:  ...          taken if k was  = +0
slot 3:  ...          taken if k was  <  0   (negative, nonzero)
slot 4:  ...          taken if k was  = -0
```

Three uses matter here:

- **Resource scanning.** `CCS VAC1USE / TCF VACFOUND`: a free VAC area holds a positive value,
  so slot 1 (`TCF VACFOUND`) is taken; a busy one holds +0, so slot 2 — the *next `CCS`* — runs
  instead. Five of these in a row make the whole 1201 scan. In the core-set scan, a free set
  holds **−0** (slot 4 falls through into `CORFOUND`) and a busy one holds a positive priority
  (slot 1, `TCF NEXTCORE`).
- **Loop counting.** `CCS EXECTEM2 / TCF NOVAC3`: because A receives `abs(k) − 1`, storing A
  back each pass counts 7, 6, 5 … 0 — which is how `NO.CORES DEC 7` yields exactly eight
  probes.
- **Testing the accumulator.** `CCS A` is the idiom for "branch on what is currently in A"
  (e.g. "is this flag bit set?").

### 3.3 Interpreter pseudo-instructions (the vector virtual machine)

Some trace-adjacent code (inside `SERVICER` and `NORMLIZE`) is not machine code at all.
`TC INTPRET` hands control to the **Interpreter**, which reads *packed pseudo-instructions*
(up to two per word, operands on the following lines) and executes vector/matrix math using a
push-down stack in the job's **VAC area** — this is exactly why `SERVICER` needs one. `EXIT`
returns to native code, with results left in `MPAC`.

Pseudo-ops you will see near the tour stops:

| Pseudo-op | Meaning |
| :-------- | :------ |
| `VLOAD x` | Load vector `x` into the multi-purpose accumulator (`MPAC`). |
| `ABVAL` | Replace the vector in `MPAC` with its length (absolute value). |
| `STORE x` | Store `MPAC` into `x`. |
| `STOVL x` | `STORE`, then `VLOAD` the next operand. |
| `STCALL x` | `STORE`, then call an interpretive subroutine. |
| `PUSH` | Push `MPAC` onto the VAC-area stack. |
| `DSU x` | Double-precision subtract: `MPAC` := `MPAC` − `x`. |
| `VXV x` | Vector cross product. |
| `MXV x` | Matrix × vector. |
| `VSL6` | Shift the vector left 6 bit positions (rescaling). |
| `BOFF x, y` | Branch to `y` if flag bit `x` is off. |
| `CLEAR x` | Clear flag bit `x`. |
| `EXIT` | Leave the Interpreter; resume native instructions. |

A single interpretive vector operation costs on the order of **milliseconds** (a cross product
≈ 5 ms ≈ 425 MCT), which is why `SERVICER` is so long compared to the 1–2 MCT native
instructions around it.

### 3.4 Things that look like instructions but are subroutine calls

These appear as `TC NAME` and are operating-system services, not opcodes. (The scheduler-level
ones — `FINDVAC`, `NOVAC`, `ENDOFJOB`, `WAITLIST` — are described in section 2.)

| Call | What it does |
| :--- | :----------- |
| `VARDELAY` | Schedule the calling task to run again after the delay in A (centiseconds). The 2-second re-arm is `CA 2SECS / TC VARDELAY`. |
| `FIXDELAY` | Same, but the delay is the `DEC` word following the call (`TC FIXDELAY / DEC 100` = wait 1.00 s). |
| `LONGCALL` | Like `WAITLIST`, for delays longer than 162.5 s. |
| `TASKOVER` | Ends a waitlist task (the task's counterpart of `ENDOFJOB`). |
| `PHASCHNG`, `GNUTFAZ5`, `QUIKFAZ5` | Register a restart-protection checkpoint ("if we restart now, resume from here") in the phase tables. The `OCT` words after `PHASCHNG` encode the phase. |
| `BANKCALL` / `IBNKCALL` | Call a routine in another fixed-memory bank (the `CADR` after the call names it); needed because a plain `TC` only reaches the current bank. |
| `POSTJUMP` | Jump (no return) to another bank — how `WHIMPER` reaches `ENEMA`. |
| `SWCALL` / `BANKJUMP` | Other cross-bank call/jump helpers (target address arrives in A). |
| `INTPRET` | Enter the Interpreter (section 3.3). |
| `PIPASR` | Read the three PIPA accelerometer counters. |
| `MAGSUB` / `SETTRKF` | Radar helpers: magnitude test of a CDU angle; update the tracker-fail lamp. |

### 3.5 Idioms — how these combine in the trace

1. **Inline data after a call.** `TC BAILOUT1` is followed by `OCT 1201`. Inside `BAILOUT1`,
   `INDEX Q / CAF 0` computes "load the word at address 0 + Q" — and Q holds the return
   address, which points *at the `OCT` word*. That is how a call site passes its alarm code
   without any argument registers.
2. **Computed go-to.** `MULTEXIT` does `XCH ITEMP1 / INDEX A / TC 1`: the target address sits
   in A, and `INDEX` adds it into the `TC 1`, producing "jump to A + 1." This is how the alarm
   path returns to different continuations for different callers.
3. **Jumping by writing the program counter.** Because `Z` is memory-mapped,
   `DCA AVGEXIT / DXCH Z` loads a 2-word address and swaps it into the program counter —
   `SERVICER` exits through exactly this.
4. **Redirecting an interrupt return.** `WHIMPER` runs with a saved interrupt context.
   `CA TWO / AD Z / TS BRUPT / RESUME` writes "current location + 2" into `BRUPT`, so `RESUME`
   — instead of returning to the interrupted program — executes the `TC POSTJUMP / CADR ENEMA`
   two words down. A controlled hijack of the interrupt-return machinery, used to enter the
   software restart cleanly.
5. **`CCS` scan chains and countdown loops** — see the `CCS` explanation above; this single
   instruction implements both the 1201 and the 1202 detection scans.
6. **Registers as variables.** `CAF DONEADR / TS Q` (in `REREADAC`) forges a return address so
   a shared subroutine will "return" to a place chosen by the caller; `TS Q` elsewhere simply
   uses Q as spare storage between calls.
7. **A constant that is also an instruction.** The word at label `OCT10002` in `SERVICER.agc`
   is the instruction `DV Q` — whose encoding equals octal 10002 — and other code uses that
   same word as a bit mask (`MASK OCT10002`). Memory was scarce enough that instructions did
   double duty as data.
8. **`PINC`/`MINC` are not instructions.** They are hardware bus cycles ("unprogrammed
   sequences") that increment a counter behind the software's back. You will never see them in
   a listing — which is precisely why the radar theft was invisible.

### 3.6 Assembler directives (not CPU instructions — they tell yaYUL how to build the program)

| Directive | Meaning |
| :-------- | :------ |
| `EQUALS` / `=` | Define a symbol to equal a value or another symbol (like `#define`). E.g. `CDUT EQUALS 35`. |
| `ERASE` | Reserve erasable (RAM) words. `ERASE +83D` reserves 84 words (offsets 0–83). |
| `DEC n` | Emit a decimal numeric constant. `NO.CORES DEC 7`. |
| `OCT n` | Emit an octal numeric constant. `OCT 1201` is the literal alarm code. |
| `2CADR` / `CADR` | Emit a 2-word / 1-word "complete address" of a routine (address + bank bits) for schedulers and cross-bank calls. |
| `-2CADR` | The same 2-word address stored **complemented**. In the restart tables the sign is the type flag: negative entries restart **tasks** (via `WAITLIST`), positive entries restart **jobs** (via `FINDVAC`/`NOVAC`). `5.4SPOT` uses one of each. |
| `ADRES` / `GENADR` / `FCADR` | Other forms of single-word address constants (different bank encodings). |
| `EBANK= / FBANK / BBANK` | Set/track the erasable-bank / fixed-bank / both-bank so the 15-bit machine can reach its full memory via banking. |
| `BANK` / `SETLOC` | Select a fixed-memory bank / set the assembly location. |
| `COUNT*` | Bookkeeping directive that tags following code to a named log section (`$$/EXEC`, `$$/SERV`, …). |
| `BLOCK` | Selects a memory block for the following code. |

---

## 4. Hard-coded memory locations and constants that matter here

| Symbol / literal | Value | Meaning |
| :--------------- | :---- | :------ |
| `CDUT` | address **35** (octal) | Rendezvous-radar **trunnion** angle counter — one of the two hardware counters the radar spammed. (Radar code comments sometimes call the pair `OPTY`/`OPTX`.) |
| `CDUS` | address **36** (octal) | Rendezvous-radar **shaft** angle counter — the other spammed counter. |
| `TIME3` | address **26** | The WAITLIST hardware clock counter; its overflow fires `T3RUPT`. |
| `TIME4` | address **27** | Clock for `T4RUPT`, the housekeeping interrupt that drives DSKY updates and the radar monitors (`RRAUTCHK`/`RRCDUCHK` run on a 480 ms cadence). |
| `CHAN12` | output channel 12 | Hardware command bits: bit 1 = zero the RR CDUs, bit 2 = enable RR error counters, bit 14 = RR auto track enable. |
| `CHAN30` | input channel 30 | Hardware status bits: bit 7 = RR CDU fail. |
| `CHAN33` | input channel 33 | Hardware status bits: bit 2 = RR auto/power on — the bit `RRAUTCHK` samples. |
| `RADMODES` | erasable flagword | The radar mode/status word (turn-on in progress, CDUs being zeroed, antenna mode, auto-mode, CDU-fail, etc.). |
| `PIPAX/PIPAY/PIPAZ` | 37 / 40 / 41 | Accelerometer velocity-increment hardware counters (kept counting through restarts). |
| `PRIORITY` | erasable, 8 slots 12 words apart | Each core set's control word. Holds the running job's priority, or **−0** when the set is free. |
| `VAC1USE … VAC5USE` | erasable, 5 flags | The five VAC-area "in use" flags. Positive = free; **0** = claimed/busy. |
| `NO.CORES` | `DEC 7` | Loop counter: 1 initial probe + 7 repeats = **8 core sets** scanned. (The nearby "SEVEN SETS" comment is stale — see below.) |
| `COREINC` | `DEC 12` | Spacing between core sets = **12 words** each. Proves core sets are 12 (not 11) words. |
| `OCT 1201` | literal | Alarm code emitted after the `TC BAILOUT1` in the VAC-exhaustion path — **"no VAC areas."** |
| `OCT 1202` | literal | Alarm code emitted after the `TC BAILOUT1` in the core-set-exhaustion path — **"no core sets."** |
| `PRIO20` / `PRIO22` / `PRIO32` | 20 / 22 / 32 | Priority constants. `SERVICER` runs at 20 (low); higher numbers preempt it. |
| `2SECS` | 200 centiseconds | The fixed 2.00-second guidance period; used to re-arm `READACCS`. |
| `DSPTAB +11D` | erasable | The DSKY lamp/word table entry; setting its alarm bit lights the **PROG** lamp (`PROGLARM`). |
| `FAILREG` (+0,+1,+2) | erasable, 3 words | The three alarm-code registers. Up to three distinct alarms are remembered here for V05 N09. |
| `ALMCADR` | erasable | Stores the address of the code that raised the alarm (for diagnostics/downlink). |
| `LST1 / LST2` | erasable lists | WAITLIST's task time-list and task-address-list; wiped and rebuilt on restart. |
| `5.4SPOT` | in `RESTART_TABLES.agc` | The phase-table recipe for the landing's SERVICER group: rebuild one `REREADAC` task + one `SERVICER` job. |
| `NEG1/2` | constant −1/2 | The filler the restart writes into empty `LST1` delta-time slots ("no task waiting"). |
| `POSMAX` | constant +16383 | The largest positive word; used to park timers (e.g. disable `TIME6`). |
| `ENDTASK` | fixed address | Stored (complemented) into empty `LST2` slots so an empty waitlist slot "runs" a harmless end-of-task. |
| `OCT40400` | bit mask | The DSKY lamp-table bits `PROGLARM` sets: bit 9 (PROG lamp) plus the table's update-request bit 15. |
| `REDOCTR` | erasable counter | Counts restarts (`INCR REDOCTR` in `GOPROG`); telemetry evidence of how many times the computer restarted. |

> **The "eight core sets, twelve registers" clarification.** A comment in `EXECUTIVE.agc`
> reads `# SEVEN SETS OF ELEVEN REGISTERS EACH`. That comment is **stale/incorrect** for this
> build. The authoritative evidence in the code: `COREINC DEC 12` (12 words per set) and the
> erasable reservation `ERASE +83D` labeled `EIGHT SETS OF 12 REGISTERS EACH`, plus the
> restart routine freeing `PRIORITY +0 … +84D` in steps of 12 (eight stores). So: **8 core
> sets of 12 words**, and **5 VAC areas of 44 words**.

---

## 5. What a "word" is, and how big everything is

The AGC does not use bytes. Its native unit is the **word**.

| Property | Value |
| :------- | :---- |
| Data bits per word | **15** |
| Parity bit | 1 (error checking; not usable by programs) |
| Physical bits stored | **16** = 2 bytes |
| Number representation | ones' complement: 1 sign bit + 14 magnitude bits |
| Range of one word | −16383 to +16383, with both `+0` and `−0` |
| Erasable memory (RAM) | 2,048 words = about 4 KB |
| Fixed memory (ROM/rope) | 36,864 words = about 72 KB |

A "double precision" value uses 2 words (28 magnitude bits), which is why vector and matrix
math is expensive and needs the larger VAC workspace.

### Sizes of the two scheduler resources

Verified directly from this repository:

| Resource | Size | Count | Source evidence |
| :------- | :--- | :---- | :-------------- |
| Core set | **12 words** | **8** | `COREINC DEC 12`; `ERASE +83D` marked "EIGHT SETS OF 12 REGISTERS EACH" |
| VAC workspace | **43 words** | **5** | `VACn ERASE +42D` (offsets 0 through 42) |
| VAC use flag | 1 word | 5 | `VACnUSE ERASE` |
| VAC slot total | **44 words** | 5 | 5 slots × 44 = 220, matching the "(220D)" section comment |

So one `SERVICER` copy holds **12 + 43 = 55 words** of working memory (56 words if you also
count the one-word VAC use flag that marks the slot busy).

In modern units, 55 words is about **103 bytes** of data, or **110 bytes** of physical storage
including parity bits. The whole scheduler pool is 96 + 220 = **316 words**, roughly **15% of
all 2,048 words of erasable memory**.

## 6. Number bases and notation

- Numbers written plainly in AGC source are **octal** (base 8). A trailing **`D`** means
  **decimal**: `+83D` = 83 decimal; `+84D` = 84 decimal.
- `-0` (minus zero) is a real, distinct value on this ones'-complement machine and is used as
  a sentinel — e.g. a `PRIORITY` register of `-0` means "this core set is free."
- A leading `+`/`-` on a numeric label (e.g. `+2`, `-CCSPR`) is a relative/negated address.
- Addresses like `PRIORITY +12D` mean "12 decimal words past `PRIORITY`."

---

## 7. The three annotation trails in the source

All are comment-only (ignored by the assembler) and can be listed with `grep`:

| Trail | Grep | What it traces |
| :---- | :--- | :------------- |
| `RADAR_PROBLEM<n>` | `grep -rn "RADAR_PROBLEM[0-9]" Luminary099/*.agc` | External radar control, mode monitoring, zero release, and the counters receiving bogus pulses. |
| `MEMORY_LEAK<n>` | `grep -rn "MEMORY_LEAK[0-9]" Luminary099/*.agc` | Fixed demand → dispatch → allocate → claim VAC/core → late release. |
| `ALARM_RECOVERY<n>` | `grep -rn "ALARM_RECOVERY[0-9]" Luminary099/*.agc` | Resource scans → 1201/1202 → alarm display → restart → recovery. |
