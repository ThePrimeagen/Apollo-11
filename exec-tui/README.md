# exec-tui — fly the Apollo 11 guidance computer

An interactive, real-time TUI simulation of the Lunar Module guidance computer's
**Executive** (its operating system) during the powered descent. You don't fly the
spacecraft — you fly the *computer*: start the 2-second guidance cycles, type on the
DSKY (every keystroke costs real compute), flip on the rendezvous-radar bug that
stole ~15% of the machine, and watch the 1201/1202 alarms of July 20, 1969 develop,
fire, and recover — exactly the way they did at 33,000 feet.

Built for an educational video. Every number is sourced: see [`RESEARCH.md`](RESEARCH.md).
Design and controls: [`ROADMAP.md`](ROADMAP.md).

## Run

```bash
cd exec-tui
go run .
```

Best at ≥140×45. Time runs at 20× slow motion by default (1 wall second = 50 ms of
AGC time) so you can watch individual preemptions; `]` speeds it up.

## The screen

- **Top**: how much **free compute** is left (the star of the show), duty, counter
  steal, deficit, PROG lamp, FAILREG alarm codes.
- **2s CYCLE bar**: progress through the current READACCS/SERVICER guidance period,
  plus the DSKY (verb/noun/R3).
- **Left, long lines**: the tasks that need computing — one scrolling execution
  timeline per job/interrupt (SERVICER, MONITOR, CHARIN, DAP, T4RUPT, …), plus the
  invisible **RR STEAL** row and the shrinking **IDLE** row.
- **Right, two box columns**: the **8 core sets** and **5 VAC areas**. When they fill,
  the Executive has nowhere to put the next job: 1201 (no VACs) / 1202 (no core sets).
- **Bottom dashes**: your controls.

## Controls

| Key | Action |
| :-- | :----- |
| `d` | Start the powered descent (P63): 2-second guidance cycles begin |
| `l` | Landing radar "data good" (+ conversion load) |
| `n` | Neil/Buzz types `V16N68` ENTR with human timing — watch CHARIN preempt SERVICER |
| `t` | Typing mode: **your** keys are DSKY keys and cost real compute (`esc` leaves) |
| `r` | Toggle the Apollo 11 rendezvous-radar bug: ~15% of every second vanishes |
| `p` | Ping the radar: one-shot priority-32 read burst |
| `6` | High gate → P64 (redesignation load; restarts can no longer shed it) |
| `a` | ATT HOLD / P66 — Armstrong's move that stopped the alarms |
| `space` | Pause |
| `[` / `]` | Slower / faster time scale |
| `x` | Reset |
| `q` | Quit |

## Reproduce July 20, 1969

1. `d` — PDI. Watch a healthy 2-second cycle: SERVICER finishes with room to spare.
2. `l` — landing radar locks. Margin shrinks.
3. `n` — Buzz keys up the DELTAH monitor. Margin ≈ 10%.
4. `r` — the bug. 15% of the computer disappears (nothing shows *why*: PINC/MINC
   cycle stealing is invisible to software). Demand is now >100%.
5. Watch the box columns fill, one leaked core-set+VAC pair per cycle → **1201/1202**,
   PROG lamp, BAILOUT, restart — pools wiped, one SERVICER rebuilt, monitor silently
   dropped. The computer saves itself. Houston says "go".
6. `6` — P64. The extra load is restart-protected; the alarms keep coming.
7. `a` — ATT HOLD. Load shed. Land.

## Tests

Written before the implementation (see git history), happy + unhappy paths throughout:

```bash
go test ./...
```
