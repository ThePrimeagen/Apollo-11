# exec-tui — the Apollo 11 show, one module

Everything lives here now: the scene components, the screenplay that
plays them, and every editor, demo, and config tuner. Running the
launcher opens a menu of all of it.

```bash
cd exec-tui
go run .
```

## Layout

```
assets/       every lunar atlas in one folder: lm-1.json … lm-4.json + lm.json
components/   everything a scene puts together
  sprite/     the pixel model: Sprite = width × height cells (rune + fg/bg)
  particle/   the particle engine
  gunfire/    the one-shot Doom-shotgun blast — owns its config.json
  fire/       flame, booster, compass — owns its config.json
  stars/      the four-layer starfield — owns its config.json
  lander/     the Apollo LM: baked atlas art, the assets/ loader, the Ship component
    descent/  the legacy descent-view renderer (used by the sim UI)
  moon/       Moon (the reusable pixelated disc) + Orbit (the craft circling it)
  rocket/     the size-4 rocket over a down-firing booster
  title/      banner cards set in terminal-fonts
  dsky/       the DSKY panel as a scene component (right-edge dock wipe)
screenplay/   Screenplay → Scene → Component; the lip gloss Screen; Bill + Compose
shows/        composable bills, one package per show
  moonshow/   the moon screenplay: the bare moon, then a ship's fast arrival into orbit
  lunarcloseup/  02. Walkthrough: pause, fly-in, fire, north-facing fall, landing on a huge moon horizon
cmd/          every runnable: editors, demos, tuners
  premiere/   the four-scene screenplay (arrival, DSKY dock, descent orbit, THE END)
  lunarcloseup/  runs 02. Walkthrough (space past the last scene ends it)
  lander/     the continuous-descent demo
  moon/       runs the moon screenplay (space cuts; past the last scene it ends)
  stars/      the starfield strategy browser
  preview/    atlas / fire / rocket previews and tapes
  editor/     the vim-ish ASCII editor — point it at any folder of atlases
              (default assets/); C-p quick-opens across its files
  gunfire/    the one-shot shotgun demo (space fires)
  adjustflame/  tunes components/fire/config.json
  adjuststars/  tunes components/stars/config.json
  adjustgunfire/  tunes components/gunfire/config.json
menu/ sim/ ui/  the launcher and the legacy Executive sim
```

Components live one lifecycle (see `screenplay/README.md`): `Start(w, h)`
allocates for the stage, `Update(dt)` runs the clock, `Render()` returns a
stage-sized `sprite.Sprite`, `Stop()` frees — and `Start` may come again.
Each component's tuning file sits beside its code, so the tuners and the
premiere read the same home. The lunar atlases are the one exception:
they all live in `assets/`, the folder the editor opens by default.

## The legacy Executive sim

An interactive, real-time TUI simulation of the Lunar Module guidance
computer's **Executive** (its operating system) during the powered descent
— the LEGACY EXEC entry in the launcher. You don't fly the spacecraft —
you fly the *computer*: start the 2-second guidance cycles, type on the
DSKY (every keystroke costs real compute), flip on the rendezvous-radar
bug that stole ~15% of the machine, and watch the 1201/1202 alarms of
July 20, 1969 develop, fire, and recover — exactly the way they did at
33,000 feet.

Built for an educational video. Every number is sourced: see
[`RESEARCH.md`](RESEARCH.md). Design and controls: [`ROADMAP.md`](ROADMAP.md).

Best at ≥140×45. Time runs at 20× slow motion by default (1 wall second =
50 ms of AGC time) so you can watch individual preemptions; `]` speeds it up.

### The screen

- **Top**: how much **free compute** is left (the star of the show), duty,
  counter steal, deficit, PROG lamp, FAILREG alarm codes.
- **2s CYCLE bar**: progress through the current READACCS/SERVICER guidance
  period, plus the DSKY (verb/noun/R3).
- **Left, long lines**: the tasks that need computing — one scrolling execution
  timeline per job/interrupt (SERVICER, MONITOR, CHARIN, DAP, T4RUPT, …), plus the
  invisible **RR STEAL** row and the shrinking **IDLE** row.
- **Right, two box columns**: the **8 core sets** and **5 VAC areas**. When they fill,
  the Executive has nowhere to put the next job: 1201 (no VACs) / 1202 (no core sets).
  Abandoned SERVICER copies show as red **STUB** boxes — superseded, starving, never
  reaching ENDOFJOB. Those are the leak; the stats line counts the words never freed.
  When demand is only a hair over 100% a stub can win the race: it keeps the CPU
  (equal priority never preempts), finishes a few ms late and frees its pair — the
  log pairs the red `LEAK:` line with a green `RECOVERED:` line. Stubs only pile up
  once the backlog grows past what the old copy can finish before a higher-priority
  job takes the CPU and the newest copy wins the rescan.
- **Bottom dashes**: your controls.

### Controls

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

### Reproduce July 20, 1969

1. `d` — PDI. Watch a healthy 2-second cycle: SERVICER finishes with room to spare.
2. `l` — landing radar locks. Margin shrinks. **Don't skip this one**: without the
   LR conversion load, `n` + `r` leave demand at ~99% — the computer rides the knife
   edge forever and nothing leaks. All three loads together are what tipped Apollo 11.
3. `n` — Buzz keys up the DELTAH monitor. Margin ≈ 10%.
4. `r` — the bug. 15% of the computer disappears (nothing shows *why*: PINC/MINC
   cycle stealing is invisible to software). Demand is now ~103% — watch `deficit`
   in the header go red.
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
