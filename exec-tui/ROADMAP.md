# exec-tui — Roadmap and Complete Design

An interactive, real-time simulation of the Apollo 11 Lunar Module Guidance Computer's
**Executive** (job scheduler) during the powered descent, built for an educational video.
You fly the computer, not the spacecraft: you add load, steal time, type on the DSKY, and
watch the machine march toward — and recover from — the famous 1201/1202 alarms.

Everything in the simulation is sourced. See [`RESEARCH.md`](RESEARCH.md) for every number
and its citation.

---

## 1. The wireframe, decoded

```
free compute text ────────►  ┌──────────────────────────────────────────────────┐
                             │ FREE COMPUTE: 14.2%   TLOSS 15.0%   P63  ALARMS  │
long lines =                 ├───────────────────────────────┬───────┬──────────┤
tasks that need computing ─► │ SERVICER  ████████░░████──    │ CORE  │  VAC     │
(scrolling execution         │ DAP       █─█─█─█─█─█─█─      │ ┌───┐ │ ┌────┐   │
 timelines, one row per      │ MONITOR   █────█────█──       │ │ 1 │ │ │ 1  │   │
 job/interrupt/steal)        │ CHARIN    ──█────────█─       │ ├───┤ │ ├────┤   │
                             │ RR STEAL  ▒▒▒▒▒▒▒▒▒▒▒▒▒      │ │...│ │ │... │   │
first box column =           │ IDLE      ░░──░───────       │ │ 8 │ │ │ 5  │   │
8 core sets ────────────────►│                               │ └───┘ │ └────┘   │
second box column =          ├───────────────────────────────┴───────┴──────────┤
5 VAC areas ────────────────►│ 2s CYCLE ▕█████████░░░░░░▏ 1.24s   DSKY V16 N68  │
                             ├──────────────────────────────────────────────────┤
dashes = keys you press ────►│ [d]escent [l]radar [n]eil types [r]bug [p]ing …  │
                             └──────────────────────────────────────────────────┘
```

- **Top line**: how much free compute is left, updated live. This is the star of the show.
- **Left, long lines**: the actual work that needs computing. Each row is one competitor
  for CPU time (jobs, interrupts, counter theft, idle). Time scrolls left; a filled cell
  means "this thing owned the CPU during that slice."
- **Right, first box column**: the **8 core sets** (12 words each) — every job needs one.
- **Right, second box column**: the **5 VAC areas** (44 words each) — jobs doing
  vector/matrix math via the Interpreter need one of these too.
- **Bottom dashes**: your controls. Each key injects a historically real event.

## 2. Time model

- **1000 ms of wall time = 50 ms of AGC time** (20× slow motion, the default; `[` / `]`
  change it, `space` pauses). At this rate one guidance cycle (2.000 s of AGC time) plays
  out over 40 seconds — slow enough to watch individual preemptions happen.
- The engine advances in **100 µs AGC steps**. Each step is charged to exactly one
  consumer, in hardware priority order:
  1. **Counter increments** (PINC/MINC cycle stealing — the radar bug lives here). These
     are unprogrammed sequences: invisible to software, they just make everything late.
  2. **Interrupts** (T3RUPT waitlist dispatch, T4RUPT DSKY/housekeeping every 120 ms,
     DAP autopilot every 100 ms, KEYRUPT on keystrokes, DOWNRUPT telemetry).
  3. **The highest-priority ready job** (the Executive's whole scheduling rule).
  4. **DUMMYJOB** (idle) if nothing is ready — this is "free compute."
- **T3RUPT keeps perfect time no matter the load** — that punctuality is precisely what
  turns a CPU shortage into a memory exhaustion.

## 3. The cast (what can show up and take time)

| Row | What | When | Cost model (sourced, see RESEARCH.md) |
| :-- | :--- | :--- | :--- |
| SERVICER | Priority-20 job, needs core set + VAC | every 2.000 s during powered flight | ~1.30 s/cycle base; +40 ms with landing radar; +grows in P64 |
| READACCS | Waitlist task | every 2.000 s | ~1 ms; schedules SERVICER, re-arms itself |
| DAP | Autopilot interrupt | every 100 ms in powered flight | ~12 ms per fire (≈12% duty) |
| T4RUPT | Housekeeping interrupt | every 120 ms always | ~1 ms; +0.5 ms per queued DSKY digit change |
| MONITOR | V16N68 display job, priority 30 | once per second while keyed up | ~60 ms per refresh (≈3% duty) |
| CHARIN | Keystroke job, priority 30, core set only | per DSKY keypress | ~5 ms per key + display traffic |
| RADAR READ | Priority-32 one-shot job (your "ping") | on demand | ~80 ms burst |
| GYRO COMP | Priority-21 job | every 1 s | ~7 ms |
| DOWNRUPT | Telemetry interrupt | 50/s always | ~0.2 ms per fire (≈1%) |
| PIPA/other counters | Normal counter traffic | powered flight | ≈0.5% |
| **RR STEAL** | The bug: 2 CDU counters × 6400 pulses/s × 11.72 µs | when you enable it | **≈15.0% of everything** |
| IDLE | DUMMYJOB | whenever nothing is ready | = free compute |

## 4. The failure mechanism (what the video teaches)

1. Baseline descent (P63) uses <85% → margin >15%. Everything finishes; pools stay at
   1–2 boxes busy; free-compute number is healthy.
2. Landing radar lock (+~2%), then Aldrin keys **V16 N68** (+~3%) → margin ~10%.
3. You flip the **radar bug** on: 15% of every second silently vanishes. Demand ≈ 105%.
4. SERVICER (lowest priority, longest job) absorbs the entire shortfall — it's still
   running when the next READACCS fires punctually and FINDVACs a *new* SERVICER. The old
   copy legitimately keeps its core set + VAC. **The deficit is paid in allocations.**
5. Watch the box columns fill: one extra core set + VAC pair per overloaded cycle.
6. A request finds no VAC → **1201**; no core set → **1202**. BAILOUT: code into FAILREG,
   **PROG lamp lights**, software restart wipes both box columns clean, rebuilds exactly
   one SERVICER + one REREADAC, and **drops the unprotected V16N68 monitor** — the load
   sheds itself and the DSKY snaps back to N63. That's why Houston could say "go."
7. In **P64** the extra redesignation load is protected — restarts can't shed it, so the
   alarms just keep coming (historically: three in 40 seconds) until the pilot reduces
   the computer's job (ATT HOLD / P66).

## 5. Controls (the dashes)

| Key | Action |
| :-- | :----- |
| `d` | Start powered descent: READACCS/SERVICER 2-second cycles + DAP + counters (P63) |
| `l` | Landing radar acquires ("data good") — adds the radar-data conversion load |
| `n` | **Neil/Buzz types**: auto-keys `V 1 6 N 6 8 ENTR` with human inter-key delays; every keystroke is a real KEYRUPT+CHARIN that costs compute; monitor starts refreshing at 1 Hz |
| `t` | Typing mode: **your own keys** become DSKY keystrokes (each one costs compute time and display updates); `esc` leaves |
| `r` | Toggle the **Apollo 11 rendezvous radar bug** (RR mode switch to AUTO/SLEW with the 800 Hz phase mismatch): +15.0% TLOSS |
| `p` | **Ping the radar**: one-shot priority-32 radar read burst |
| `6` | Advance P63 → P64 (high gate: redesignation logic; restart can no longer shed load) |
| `a` | ATT HOLD / P66: shed autopilot + guidance load (how Armstrong stopped the alarms) |
| `space` | Pause / resume |
| `[` `]` | Slower / faster time scale (default 20× slow motion) |
| `x` | Reset to idle |
| `q` | Quit |

## 6. Build phases

- **Phase 0 — Research** (done): every constant traced to the Luminary 099 source in this
  repo, Don Eyles AAS 04-064, or the Cherry memo. Output: `RESEARCH.md`.
- **Phase 1 — Broken tests first**: the full test suite (engine + UI) written against the
  not-yet-existing API; suite must fail before any implementation exists. Every test has a
  happy and an unhappy path.
- **Phase 2 — Engine** (`sim/`): pure Go, deterministic, no I/O. Executive (8 core sets,
  5 VACs, priority scheduling, FINDVAC/NOVAC), Waitlist (punctual T3RUPT), counter
  stealing, interrupts, DSKY keystroke pipeline, BAILOUT + restart with phase tables.
- **Phase 3 — TUI** (`ui/` + `main.go`): bubbletea/lipgloss. Header (free compute),
  scrolling timelines, core/VAC box columns, 2-second cycle bar, DSKY panel, event log,
  instruction bar. 30 FPS wall-clock ticks feeding the engine.
- **Phase 4 — Scenario polish**: the historical Apollo 11 sequence reproducible by hand
  (d → l → n → r … alarms … recovery), tuned so numbers match Eyles' duty-cycle table.
- **Phase 5 — Demo video**: long recorded session operating every control, including
  faked human typing on the DSKY, alarms firing and recovering, P64 alarm storm, and the
  ATT HOLD save.

## 7. Non-goals

- Not a cycle-accurate CPU emulator (Virtual AGC exists for that). This simulates the
  *scheduler economics*: where time goes, when memory runs out, and why the design saved
  the landing.
- No trajectory/physics simulation — altitude readouts are scripted flavor, not dynamics.
