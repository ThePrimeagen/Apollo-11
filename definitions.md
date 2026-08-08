# Definitions — Everything You Need to Read the Apollo 11 Alarm Trace

This is the first required document in the
[`table_of_contents.md`](table_of_contents.md) reading order. It is the glossary for all three
focused traces: [`radar_problem.md`](radar_problem.md), [`memory_leak.md`](memory_leak.md), and
[`alarm_recovery.md`](alarm_recovery.md). If you hit a word, acronym, assembly instruction, or
hard-coded number you don't recognize, it should be in one of the tables below.

---

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

## 3. Assembly instructions encountered (yaYUL / AGC Block II)

The AGC accumulator is called **A**; **L** and **Q** are other special registers; **Z** is the
program counter. Times are in memory cycles (MCT ≈ 11.72 µs).

| Mnemonic | Name | Plain meaning |
| :------- | :--- | :------------ |
| `CA k` | Clear and Add | Load the contents of memory `k` into A. (Copy `k` → A.) |
| `CAF k` | Clear and Add Fixed | Same as `CA`, but `k` is in fixed (ROM) memory. Commonly used to load constants. |
| `CS k` | Clear and Subtract | Load the *complement* (negation) of `k` into A. `CS ZERO` yields −0. |
| `TS k` | Transfer to Storage | Store A into memory `k`. (Copy A → `k`.) |
| `XCH k` | Exchange | Swap the contents of A and `k`. |
| `LXCH k` | L Exchange | Swap the contents of L and `k`. |
| `DXCH k` | Double Exchange | Swap a 2-word (double-precision) value between A,L and `k`,`k+1`. |
| `QXCH k` | Q Exchange | Swap the contents of Q and `k`. |
| `AD k` | Add | Add contents of `k` to A. |
| `ADS k` | Add to Storage | Add A to `k`, storing the sum back in `k` (and A). |
| `MASK k` | Mask (logical AND) | Bitwise-AND A with `k`. Used to isolate bit fields. |
| `TC k` | Transfer Control | Call/jump to `k`, saving the return address in Q. (A subroutine call.) |
| `TCF k` | Transfer Control to Fixed | Jump to `k` in fixed memory (no return saved). (A `goto`.) |
| `CCS k` | Count, Compare, and Skip | The AGC's branch: examine `k`'s sign/value and take one of four following instructions (the classic "+ / +0 / − / −0" 4-way branch). Also loads `abs(k)−1` into A. |
| `BZF k` | Branch Zero to Fixed | Branch to `k` if A = 0. (Extended instruction.) |
| `BZMF k` | Branch Zero or Minus to Fixed | Branch to `k` if A ≤ 0. (Extended.) |
| `INDEX k` | Index | Add the contents of `k` to the *next* instruction before executing it (computed addressing / jump tables). |
| `EXTEND` | Extend | Prefix that switches the next opcode to the "extended" instruction set (e.g. `DCA`, `MP`, `BZF`, `ROR`). |
| `INHINT` | Inhibit Interrupts | Disable interrupts (enter a critical section). |
| `RELINT` | Release Interrupts | Re-enable interrupts. |
| `RESUME` | Resume | Return from an interrupt to the interrupted program (via the `BRUPT`/`ZRUPT` save regs). |
| `DCA k` | Double Clear and Add | Load a 2-word value from `k`,`k+1` into A,L. (Extended.) |
| `ZL` | Zero L | Set the L register to +0. |
| `COM` | Complement | Bitwise-complement (negate) A. |
| `DOUBLE` | Double | Add A to itself (multiply by 2). |
| `ROR ch` | Read OR | OR A with the contents of I/O channel `ch`. (Extended; here used to read the "superbank".) |
| `WOR ch` | Write OR | OR A into I/O channel `ch` (set output bits). (Extended.) |
| `WAND ch` | Write AND | AND A into I/O channel `ch` (clear output bits). (Extended.) |
| `RAND ch` | Read AND | AND A with I/O channel `ch`. (Extended.) |
| `RXOR ch` | Read XOR | XOR A with I/O channel `ch`. (Extended.) |

### Assembler directives (not CPU instructions — they tell yaYUL how to build the program)

| Directive | Meaning |
| :-------- | :------ |
| `EQUALS` / `=` | Define a symbol to equal a value or another symbol (like `#define`). E.g. `CDUT EQUALS 35`. |
| `ERASE` | Reserve erasable (RAM) words. `ERASE +83D` reserves 84 words. |
| `DEC n` | Emit a decimal numeric constant. `NO.CORES DEC 7`. |
| `OCT n` | Emit an octal numeric constant. `OCT 1201` is the literal alarm code. |
| `2CADR` / `CADR` | Emit a 2-word / 1-word address-of-code reference (so `TC`/`FINDVAC` know the target and its bank). |
| `ADRES` / `GENADR` / `FCADR` | Other forms of address constants. |
| `EBANK= / FBANK / BBANK` | Set/track the erasable-bank / fixed-bank / both-bank so the 15-bit machine can reach its full memory via banking. |
| `BANK` / `SETLOC` | Select a fixed-memory bank / set the assembly location. |
| `COUNT*` | Bookkeeping directive that tags following code to a named log section (`$$/EXEC`, `$$/SERV`, …). |
| `BLOCK` | Selects a memory block for the following code. |

---

## 4. Hard-coded memory locations and constants that matter here

| Symbol / literal | Value | Meaning |
| :--------------- | :---- | :------ |
| `CDUT` | address **35** (octal) | Rendezvous-radar **trunnion** angle counter — one of the two hardware counters the radar spammed. |
| `CDUS` | address **36** (octal) | Rendezvous-radar **shaft** angle counter — the other spammed counter. |
| `TIME3` | address **26** | The WAITLIST hardware clock counter; its overflow fires `T3RUPT`. |
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
