# Screenplay Memory-Cycle Stealing — How the Radar Took 15% Without Running One Instruction

The other documents state the theft as a fact: "those requests stole processor memory
cycles" ([`radar_problem.md`](radar_problem.md)), "hardware steals exactly one cycle each,
invisibly" ([`operations_and_timing.md`](operations_and_timing.md)). This file explains the
machinery — what a stolen cycle physically is, how the CPU "knew" to yield one, and why no
software anywhere could have seen it happen. Job-level consequences in
[`screenplay_outline.markdown`](screenplay_outline.markdown); mission-time context in
[`screenplay_descent_timeline.markdown`](screenplay_descent_timeline.markdown).

Every claim carries a bracketed source link: the 1969 alarm analysis, the AGC hardware
manuals (paragraph-numbered), the original 1963 logical-design report, and the flight code.

## 1. There is no CPU separate from the memory cycle

The modern mental model — a CPU that computes while memory fetches — does not apply. The
AGC's unit of execution *is* the memory cycle: a fixed 12-beat ritual (time pulses T01–T12,
0.977 µs each, 11.72 µs total, one **MCT**).[^agcis30-tpg] Core memory reads destructively,
so every cycle is read-then-rewrite: the address lands in the S register at T01, the word is
flushed out of the ferrite cores into the G register by T06–T07, the CPU's gates route it
through the buses and adder during T07–T10, and after T10 it is written back into
core.[^wiki-mct] Computing and memory access are two halves of one choreographed cycle,
driven by the same clock beats.

Consequences that matter for the theft:

- **Instructions are chains of whole MCTs** — 1–3 for most, 6 for divide — with perfectly
  deterministic timing. No cache, no wait states, no variability.[^wiki-mct]
- **One datapath, one memory port.** One S register, one set of buses, one adder. Even the
  central registers (A, L, Q, Z) are memory-mapped onto it. Nothing overlaps anything; the
  only fold is that the fetch of the next instruction occupies the final counted MCT of the
  current one (at T12, the word already in B is written into the sequence register
  SQ).[^r393-fetch]
- **The machine's total capacity is therefore a flat number**: 1 s ÷ 11.72 µs ≈ **85,300
  cycles per second, ever** — and every single one is spent on exactly one thing: a step of
  a program instruction, an interrupt's instructions, or a counter update.[^cherry-mct]

So "the radar stole a cycle" means: for one 11.72-µs beat, the shared datapath ran an
errand for the radar interface instead of the program. One pulse = one displaced program
cycle, as an arithmetic identity. There was no architectural slack anywhere to absorb it.

## 2. A counter is a RAM address with wires soldered to it

Erasable locations 0024–0060 (octal) are **counters**: ordinary memory words that external
hardware can nudge by pulsing dedicated input lines.[^agcis30-counters] The pulse carries no
data — it is a bare "+1" or "−1" request. The nudge itself is performed by the CPU, using
its own adder, in one cycle. Nothing outside the computer ever writes into core.

The block, in the LM's Block II address map — priority is simply address order, lowest
served first:[^agcis30-counters][^agcis8-prio]

| Address (octal) | Counter | Fed by | Hardwired sequence |
| :--- | :--- | :--- | :--- |
| 0024–0031 | TIME2, TIME1, TIME3, TIME4, TIME5, TIME6 | the 100-Hz scaler / T6 clock | `PINC` (`DINC` for T6)[^wiki-scaler] |
| 0032–0034 | CDUX, CDUY, CDUZ | IMU gimbal-angle CDUs | `PCDU` / `MCDU`[^agcis30-cdu] |
| **0035** | **CDUT** (`TCDU`) — RR trunnion | RR ECDU lines TRNP / TRNM | `PCDU` / `MCDU`[^agcis30-cdu][^code-cdut] |
| **0036** | **CDUS** (`SCDU`) — RR shaft | RR ECDU lines SHAFTP / SHAFTM | `PCDU` / `MCDU`[^agcis30-cdu][^code-cdut] |
| 0037–0041 | PIPAX, PIPAY, PIPAZ | accelerometer pulses | `PINC` / `MINC` |
| 0042–0060 | RHC counters, uplink, downlink, radar data-in, … | various | `SHINC`, `SHANC`, … |

`PINC`, `MINC`, `PCDU`, `MCDU`, `DINC`, `SHINC` are **counter instructions** — also called
**involuntary instructions** or **unprogrammed sequences**. They are hardwired control-pulse
patterns living in the same gate logic that implements real instructions, but they have no
opcode, cannot appear in a program listing, and are triggered only by hardware
requests.[^agcis30-invol] The CDU counters get the `PCDU`/`MCDU` variants because CDU angles
count in cyclic two's complement rather than the machine's native ones'
complement.[^agcis30-cdu]

The radar's two counters sit mid-chain, behind the clocks and the IMU's CDUs — nothing
about their position is special. What was special on Apollo 11 was their *rate*: the
phase-mismatched ECDUs pulsed TRNP/TRNM and SHAFTP/SHAFTM at the interface ceiling, 6,400
counts per second each.[^cherry-15pct]

## 3. How the computer "knew": it didn't — a latch remembered

A pulse on a counter line does not alert anything. It **sets a flip-flop** in the **Counter
Priority Control** — one request cell per counter. The cells accept pulses for several
counters simultaneously; each has a holding flip-flop in front of the sampling gate, so a
pulse arriving at an awkward instant is never lost.[^agcis8-prio][^r393-test] The request
just sits there, in hardware, until it is serviced. No software is aware a request exists;
no software *could* be — there is no addressable register that exposes the request cells.

## 4. The steal, beat by beat

```text
                    ┌──────────── one MCT, 11.72 µs ────────────┐
program              CA RADMODES — cycle 1 of 2                  (radar pulse arrives,
                                                                  latch sets, nothing
                                                                  else happens yet)
T12 test             any request cell set?  → yes: INKL asserted
stolen cycle         PCDU 0036: read CDUS → adder +1 → write back
T12 test             any request cell set?  → CDUT too: INKL stays up
stolen cycle         PCDU 0035: read CDUT → adder +1 → write back
T12 test             any request cell set?  → no: INKL drops
program              CA RADMODES — cycle 2 of 2                  (resumes mid-instruction;
                                                                  nothing was saved because
                                                                  nothing was disturbed)
```

Step by step:

1. **At time pulse 12 of *every* cycle** — busy or not — combinational logic examines all
   the request cells at once. The test is free: it is just gating clocked by the beat that
   was happening anyway. No list is walked; no time is spent deciding.[^r393-test]
2. If any cell is set, the priority control asserts a signal named **INKL**, which forces
   the sequence generator's SQ register and state decoders into a **hold state**. The
   opcode and step-state of the instruction in flight are not saved anywhere — they are
   simply frozen in place, unchanged.[^agcis5-inkl]
3. The priority control jams the winning counter's address into the S register and selects
   the counter instruction; the next 12 beats execute it: destructive read pulls the word
   out of core, the adder applies ±1 in passing, the rewrite half of the same cycle puts
   the result back.[^r393-insert] Exactly one MCT — the same shape as any other memory
   cycle, which is the entire trick.
4. The serviced request cell resets, and the same T12 test runs at the end of the stolen
   cycle. **Pending requests are serviced back-to-back**, one cycle each, in descending
   priority, until none remain — on Apollo 11, CDUT and CDUS were routinely both
   waiting.[^r393-chain]
5. INKL drops and the frozen instruction continues from mid-flight — same opcode, same
   step, program counter (Z) untouched, A/L/Q untouched.[^agcis5-inkl]

**The steal lands between any two memory cycles, including inside a multi-cycle
instruction.** The gate-level documents are explicit — "a Counter Instruction can be
executed after any Action 12"; the incrementing cycle is inserted "before proceeding with
the next 'normal' cycle called for by the instruction being executed" — even though
higher-level summaries (and the Block II manual's own overview paragraph) loosely say
"between instructions."[^agcis5-inkl][^r393-insert][^agcis30-invol] What resumes after the
theft is usually the *middle* of the interrupted instruction, not the next one.

## 5. Why nothing could see it

Compare the machine's three ways of changing what runs next:

| | Counter steal | Interrupt (RUPT) | Job dispatch (Executive) |
| :--- | :--- | :--- | :--- |
| Granted at | any memory-cycle boundary | instruction boundaries only | software decision |
| Mechanism | freeze via INKL, one hardwired cycle | save Z/B into ZRUPT/BRUPT, jump to vector, `RESUME` | priority scan, core-set/VAC bookkeeping |
| Cost | exactly 1 MCT | tens of cycles + handler | milliseconds |
| Software trace | **none** | handler runs, state saved | queue entries, alarms |

The steal executes zero instructions. No vector, no register save, no queue entry. The only
observable effects in the entire machine: the counter word changed, and the wall clock
advanced 11.72 µs.[^cherry-mct] That is why `PINC`/`PCDU` never appear in the Luminary
listing — and why that absence is evidence, not a gap
([`radar_problem.md`](radar_problem.md), RADAR_PROBLEM4).

The design is deliberate and, for its inputs, correct. The AGC lives on pulse trains —
accelerometer counts, uplink bits, radar angles, and its own clocks: TIME3, TIME4, TIME5
are themselves counters that the 100-Hz scaler bumps with a `PINC` every 10 ms, and
`T3RUPT` fires only when TIME3 *overflows* after many PINCs.[^wiki-scaler] Taking an
interrupt per pulse, at tens of cycles each, would consume the machine; one stolen cycle is
the cheapest bookkeeping possible. It is DMA-style cycle stealing — except the AGC has no
separate DMA engine, so the CPU's own sequencer runs the errand. The scheme's one blind
spot is exactly what Apollo 11 flew into: it silently bills the program for the peripheral's
enthusiasm, with no rate limit and no meter.[^cherry-15pct]

It also explains the paradox in [`memory_leak.md`](memory_leak.md): the theft slows *jobs*
but not *time*. TIME3 is serviced by the same steal mechanism at fixed hardware rate, so
WAITLIST deadlines arrive punctually while the work between them runs 15% late — a punctual
clock scheduling new SERVICERs on top of unfinished ones.[^cherry-robbery]

## 6. The arithmetic, restated at cycle level

```text
2 counters × 6,400 pulses/s               = 12,800 stolen cycles/s
12,800 × 11.72 µs                         = 0.150 s stolen per second
machine total                             ≈ 85,300 cycles/s
12,800 ÷ 85,300                           ≈ 15.0% of everything, gone
```

Cherry's own line: "each involuntary 'counter' increment takes a memory cycle (one memory
cycle = 11.72 microseconds)" — 12.8 × 10³ × 11.72 × 10⁻⁶ = 0.15 seconds per
second.[^cherry-15pct] The clock never slowed; the machine simply executed ~15% fewer
program cycles each second, one invisible 12-beat errand at a time.

## 7. Common misreadings

| You might say | What actually happened |
| :--- | :--- |
| "The radar wrote data into memory" | It sent bare ±1 pulses; the CPU itself did the read-±1-write. No external data path into core exists. |
| "The CPU noticed the request and switched over" | Nothing noticed anything. A latch held the request until the free end-of-cycle test found it.[^r393-test] |
| "The CPU checked a priority list" | The check is combinational gating at T12 — it costs zero cycles.[^r393-test] |
| "Then it ran the next instruction in the program" | It resumed the frozen *middle* of the current instruction. The steal splits instructions.[^agcis5-inkl] |
| "It's a kind of interrupt" | No vector, no saved state, no instructions executed. Cherry's word was "robbery," not "interruption."[^cherry-robbery] |
| "The stolen cycles filled up memory" | Time only, zero words. The memory exhaustion came later, from unfinished jobs holding core sets and VACs ([`memory_leak.md`](memory_leak.md)). |

## 8. Say it in one breath

A radar increment request sets a latch and waits. At the end of every 11.72-µs memory
cycle, hardwired logic looks at all the counter latches at once; if any are set, the
highest-priority one — fixed by address order — steals the next cycle for a read-±1-write
of its counter word, with the running instruction frozen mid-flight, and pending requests
steal following cycles one by one until none remain. Then the frozen instruction resumes as
if nothing happened — because, as far as any software could ever tell, nothing did. At
12,800 latches per second, "nothing" was 15% of the computer.

## Sources

[^cherry-mct]: Cherry, *Exegesis of the 1201 and 1202 Alarms* (MIT, 4 Aug 1969) — "each involuntary 'counter' increment takes a memory cycle (one memory cycle = 11.72 microseconds)", [pp. 7–8](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=7).
[^cherry-15pct]: Cherry — "the ECDUs may count at their maximum rate, 6400 cps … 12.8 × 10³ × 11.72 × 10⁻⁶ = 0.15 seconds/second or 15% of the LGC time", [pp. 7–8](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=7).
[^cherry-robbery]: Cherry — "The execution of jobs may be slowed down by RR ECDU memory cycle robbery but the clock which schedules jobs like SERVICER … is scarcely affected", [pp. 1–2](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=1).
[^agcis30-tpg]: *AGC Information Series, Issue 30: Block II AGC* (Raytheon) — ¶30-21: "The Time Pulse Generator generates one time pulse every 0.977 µsec and a sequence of twelve time pulses (T01 through T12) every 11.7 µsec, which is referred to as one Memory Cycle Time (MCT)", [agcis_30](https://ibiblio.org/apollo/Documents/agcis_30_block_ii_agc.pdf).
[^agcis30-counters]: AGCIS 30 — ¶30-66: "Locations 0024 through 0060 are counters"; table 30-4 maps T2 at 0024 through the CDU, PIPA, and I/O counters, [agcis_30](https://ibiblio.org/apollo/Documents/agcis_30_block_ii_agc.pdf).
[^agcis30-invol]: AGCIS 30 — ¶30-25/26: "Involuntary Instructions (Interrupting Instructions and Counter Instructions) are executed at the request of the Priority Control … The Counter Instructions are executed between the execution of Regular Instructions, and each delays the program execution by one MCT", [agcis_30](https://ibiblio.org/apollo/Documents/agcis_30_block_ii_agc.pdf).
[^agcis30-cdu]: AGCIS 30 — table 30-4, counters 0032–0036 (XCDU…TCDU 0035, SCDU 0036): input signals TRNP/TRNM and SHAFTP/SHAFTM request PCDU/MCDU; "All information is angular and stored in cyclic TWO's complement numbers", [agcis_30](https://ibiblio.org/apollo/Documents/agcis_30_block_ii_agc.pdf).
[^agcis8-prio]: *AGC Information Series, Issue 8: Priority Control* — ¶8-5: "Only one counter can be updated at a time. However, the Counter Priority Control will accept incremental pulses for several counters simultaneously. When incremental pulses for more than one counter are received, the counters are updated in the sequence listed" (fixed address-order chain; the document describes the Block I map, where the counter block sits at 0034–0057 — the Block II LM map above starts at 0024), [agcis_8](https://www.ibiblio.org/apollo/Documents/agcis_8_priority_control.pdf).
[^agcis5-inkl]: *AGC Information Series, Issue 5: Timer & Sequence Generator* — ¶5-52/53: "the SQ and state decoders must be forced into a hold state. This is accomplished by signal INKL … A Counter Instruction can be executed after any Action 12 … Note that the content of register SQ has not changed during the execution of a Counter Instruction", [agcis_5](https://ibiblio.org/apollo/Documents/agcis_5_timer_sequence_generator.pdf).
[^r393-test]: Hopkins, Alonso & Blair-Smith, R-393, *Logical Description for the Apollo Guidance Computer (AGC4)* (MIT/IL, 1963) — "When the Sequence Generator reaches the end of a memory cycle, a test is made to see if any of PINC, MINC, or SHINC are requested … Such an incoming pulse is not lost, however, since there are holding flip-flops before the blocking gate inside the P cells", [r-393](https://www.ibiblio.org/apollo/klabs/history/history_docs/r-393.pdf).
[^r393-insert]: R-393 — "The counter to be serviced is then incremented after the end of a memory cycle, essentially by inserting an incrementing memory cycle before proceeding with the next 'normal' cycle called for by the instruction being executed", [r-393](https://www.ibiblio.org/apollo/klabs/history/history_docs/r-393.pdf).
[^r393-chain]: R-393 — "If more than one counter is to be serviced, the counters are incremented in order of descending priority until there are no further increments to be made. The same test made at the end of a 'normal' memory cycle is made at the end of an incrementing cycle", [r-393](https://www.ibiblio.org/apollo/klabs/history/history_docs/r-393.pdf).
[^r393-fetch]: R-393 — "After Time 11 of the last subsequence in each sequence, C(B) is the next instruction. If no unprogrammed sequences are commanded, Time 12 selects the next instruction by writing C(B) into the sequence selection register SQ", [r-393](https://www.ibiblio.org/apollo/klabs/history/history_docs/r-393.pdf).
[^wiki-mct]: Wikipedia, *Apollo Guidance Computer*, "Memory cycle" — address into S at TP1, erasable data in G by TP6, CPU access TP7–TP10, write-back after TP10, [wikipedia](https://en.wikipedia.org/wiki/Apollo_Guidance_Computer).
[^wiki-scaler]: Wikipedia, *Apollo Guidance Computer* — the scaler's 100 Hz F10 stage "was fed back into the AGC to increment the real-time clock and other involuntary counters using Pinc"; "T3rupt and Dsrupt interrupts were produced when their counters, driven by a 100 Hz hardware clock, overflowed after executing many Pinc subsequences", [wikipedia](https://en.wikipedia.org/wiki/Apollo_Guidance_Computer).
[^code-cdut]: Flight code — `CDUT EQUALS 35 # REND RADAR TRUNNION CDU` / `CDUS EQUALS 36 # REND RADAR SHAFT CDU`: [`Luminary099/ERASABLE_ASSIGNMENTS.agc` L131–L132](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/ERASABLE_ASSIGNMENTS.agc#L131-L132).
