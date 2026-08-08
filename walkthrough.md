# High-Level Walkthrough — What Actually Happened to the Apollo 11 Computer

This is the plain-English overview. No assembly required. For the intended learning sequence,
start at [`table_of_contents.md`](table_of_contents.md), then read
[`definitions.md`](definitions.md), [`radar_problem.md`](radar_problem.md),
[`memory_leak.md`](memory_leak.md), and [`alarm_recovery.md`](alarm_recovery.md). For the
minute-by-minute mission timeline, see [`timeline.markdown`](timeline.markdown).

---

## The one-paragraph version

As Apollo 11's Lunar Module descended toward the Moon on 20 July 1969, its guidance computer
flashed alarms **1202** (four times) and **1201** (once) and kept restarting itself. The cause
was **not** a software bug and **not** a computer failure. A rendezvous-radar switch was in the
wrong position, which — because of an unrelated wiring/phase quirk — made the radar bombard the
computer with meaningless "counter increment" pulses. Each pulse silently stole a sliver of
processing time; together they stole about **15%** of the computer's capacity. That was just
enough to push the computer over its limit: its most important job could no longer finish in
its allotted 2 seconds, work piled up, the computer ran out of scratchpad memory, and it raised
an alarm. But the computer had been **designed to survive exactly this**: each alarm triggered a
fast restart that threw away the backlog and rebuilt only the essential work. Navigation and
steering never stopped. Mission Control recognized the alarms as survivable and said "Go," and
*Eagle* landed safely.

---

## The cast (what these pieces are)

- **The computer (LGC).** Tiny by modern standards: about 2 KHz of usable speed and 2,048
  words of scratchpad memory. It ran a little operating system called the **Executive**.
- **Jobs and the Executive.** Work was divided into **jobs**, each with a **priority**. The
  Executive always ran the highest-priority job and could pause a lower one to run a higher
  one. To pause and resume a job it needed to save that job's state in a small block of
  scratchpad memory.
- **Two kinds of scratchpad memory.** Every job needs a **core set** (a small 12-word block).
  Jobs that do heavy vector math *also* need a **VAC area** (a bigger 44-word block). There
  were only **8 core sets and 5 VAC areas** in the entire machine. Running out of either is
  what the 1201/1202 alarms report.
- **SERVICER.** The single most important job during the landing. Once every **2 seconds** it
  did the whole navigation-and-guidance cycle: read the accelerometers, update position and
  velocity, run the guidance equations, command the engine throttle and attitude, and update
  the crew's displays. It was the *lowest priority* job (so everything else could interrupt
  it) and the *longest* — a dangerous combination under load.
- **The rendezvous radar (RR).** Used to find the Command Module during rendezvous. During the
  landing it was switched on "just in case," in a mode (`AUTO`/`SLEW`) that, unknown to almost
  everyone, made its electronics feed the computer a torrent of bogus data.

---

## What went wrong, in five beats

### 1. The radar quietly stole time

The radar reports its antenna angles by nudging two **hardware counters** inside the computer.
Nudging a counter costs one **memory cycle** (11.72 millionths of a second) and briefly freezes
the running program — with no interrupt, nothing the software can see. Because two 800-cycle
electrical signals were out of phase (a documentation error years earlier), the radar's
electronics thought the angles were constantly wrong and tried to "correct" them at the maximum
rate: **6,400 nudges per second, on each of two counters**. That's 12,800 stolen memory cycles
every second — about **15% of the whole computer**, doing nothing useful. MIT's term for
mystery time-loss like this was **TLOSS**; the software was only tested to tolerate about 10%.

### 2. The clock kept perfect time while the work fell behind

`SERVICER` was launched every 2 seconds by a timer. Timers on this machine are also hardware
counters, so they were **immune** to the theft — they kept perfect time. But the *work* was
now running ~15% slow. So `SERVICER` started missing its 2-second deadline: the timer would
launch a **new** `SERVICER` before the **previous** one had finished.

### 3. The memory leak

Here is the heart of it (traced in the source under `MEMORY_LEAK1`…`MEMORY_LEAK9`):

- A job only gives back its core set and VAC area when it **finishes** (reaches a point called
  `ENDOFJOB`).
- An overloaded `SERVICER` hadn't finished — so it still **held** its core set and VAC area.
- The timer launched a fresh `SERVICER` anyway, which grabbed **another** core set and VAC area.
- The old, unfinished copy became a stale "stub" that retained its memory. It could remain
  queued (and under some later load patterns could even resume), but it had not completed the
  current guidance cycle before a newer copy was created.

Every overloaded 2-second cycle leaked one more set of scratchpad memory. It was only a matter
of time before the pool ran dry.

### 4. The alarm

When the next `SERVICER` (or a radar/keyboard/monitor job) asked the Executive for memory and
there was none:

- if no **VAC area** was free, the Executive raised **1201** ("no VAC areas");
- if no **core set** was free, it raised **1202** ("no core sets").

The Executive stored the code, lit the **PROG** warning light on the crew's console, and — key
point — **did not try to guess which job to kill.** By design it refused to make mission
decisions. Instead it pulled a much bigger lever.

### 5. The save: restart as a feature, not a failure

The AGC had **restart protection**, originally built so a power glitch couldn't ruin a
maneuver. A restart:

1. **throws away** both scheduler queues (all the leaked `SERVICER` stubs vanish);
2. rebuilds **only** the work listed in pre-planned "phase tables" — for the landing that was
   exactly **one** fresh `SERVICER` plus its 2-second timer, nothing else;
3. leaves the engine, autopilot, and navigation state untouched, so the spacecraft keeps flying
   the whole time.

Crucially, the accelerometers are hardware counters too, so they kept accumulating velocity
right through the restart — **no navigation data was lost**. The whole reset took under a
second. The alarm was, in effect, the computer saying "I'm overloaded, clearing my desk," and
then immediately getting back to flying.

In the early braking phase (P63) this even *cured* the problem for a while, because the restart
also dropped a non-essential display the crew had requested. In the final approach phase (P64)
there was no slack left to drop, so the alarms repeated — until Neil Armstrong took semi-manual
control, which lightened the computer's load, and the alarms stopped. *Eagle* landed with about
25 seconds of fuel margin.

---

## Why it wasn't a bug (and what got fixed)

Every individual piece worked as designed:

- The radar interface behaved according to its (flawed) documentation.
- The scheduler correctly refused to overcommit or to unilaterally kill jobs.
- The restart system did exactly its job.

The failure was a **systems** failure — a checklist/hardware/documentation gap that no single
team owned — amplified into a computer overload. The fixes were equally systemic:

- **Procedure:** later missions set the radar switch correctly for descent.
- **Software:** a patch (PCR 848, written by Don Eyles three days after the landing) made the
  computer zero out the radar counters whenever the switch wasn't in the computer-controlled
  position, killing the bogus pulses at the source.
- **Process:** MIT created review boards to make sure hardware/software interface quirks were
  documented and simulated in the future.

---

## The three focused trails in the code

The source annotations are divided by concern. They don't change how the program assembles:

| Goal | Command | You'll see |
| :--- | :------ | :--------- |
| Radar/time-theft cause | `grep -rn "RADAR_PROBLEM[0-9]" Luminary099/*.agc` | `RADAR_PROBLEM1` … `RADAR_PROBLEM4` |
| Unfinished-job memory leak | `grep -rn "MEMORY_LEAK[0-9]" Luminary099/*.agc` | `MEMORY_LEAK1` … `MEMORY_LEAK9` |
| Exhaustion detection, alarm, restart, recovery | `grep -rn "ALARM_RECOVERY[0-9]" Luminary099/*.agc` | `ALARM_RECOVERY1` … `ALARM_RECOVERY12` |

Each has a matching quickfix file: `radar_problem.lua`, `memory_leak.lua`, and
`alarm_recovery.lua`. The canonical reading order is in
[`table_of_contents.md`](table_of_contents.md).
