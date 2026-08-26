# Rough Descent Timeline — Four Regions, What Ran in Each

Program state of the LGC in four regions: before ignition, the 10% throttle phase, P63/P64
under alarms, and P66. PDI = 102:33:05 GET. Full second-by-second record in
[`timeline.markdown`](timeline.markdown); job intervals/costs in
[`rough_outline.markdown`](rough_outline.markdown).

## Region 1 — After undocking, before the engine (100:12 → 102:33)

Your assumption is **correct: very little was running.** No descent program work until
P63 was called up, and no heavy work until average-G started 30 s before ignition.

| GET | Event | What the computer ran |
| :--- | :--- | :--- |
| 100:12 | *Eagle* undocks | Coasting flight: T4RUPT (120 ms), DOWNRUPT (50/s), DAP attitude hold, DSKY on demand. **No SERVICER** — the 2 s guidance loop isn't running. Duty cycle low. |
| ~descent prep | RR mode switch → SLEW/AUTO (checklist) | **The 15% counter theft begins here** — harmless while demand is low; nobody notices. |
| ~102:27 (PDI−6 min) | Crew keys **V37E 63E** — P63 | Ignition algorithm computes TIG/attitude once, then waits. Brief pre-PDI **alarm 500** (LR antenna position), cleared. |
| 102:32:30 (PDI−35 s) | Ignition sequence (BURNBABY) | Display blanks 5 s. |
| 102:32:35 (PDI−30 s) | **Average-G on** | READACCS + SERVICER 2-second loop starts. From here the machine is loaded. |
| 102:32:58 (PDI−7.5 s) | Ullage | RCS settles propellant. |
| 102:33:00 (PDI−5 s) | Aldrin keys PRO | Crew go for ignition. |

## Region 2 — The 10% throttle phase (PDI+0 → +26 s)

One correction: this wasn't a *test* — the DPS lights at 10% so the engine gimbal can be
trimmed through the LM's center of mass before full thrust is committed.

Running: **P63 core loop only** — SERVICER + READACCS (every 2 s, one core set + one VAC),
DAP (100 ms), T4RUPT (120 ms), DOWNRUPT (20 ms), R10/R11 displays (250 ms), throttle logic
inside SERVICER, RR theft (15%).

Not running yet: guidance steering (enabled at throttle-up +26 s — the first 26 s fly
attitude-hold), **landing-radar jobs** (no lock until ~+288 s), **V16N68 monitor** (keyed
+299 s), P64 redesignation, HIGATJOB.

So: all the *core* programs, yes — none of the extras. Load ≈ <85% known software + 15%
theft ≈ ~100% — a quiet knife edge. No spare margin, but nothing tipping it over.

## Region 3 — P63 full throttle (+26 s → +506 s): the 1668 and alarms 1–2

| GET | T+PDI | Alt | Event |
| :--- | :--- | :--- | :--- |
| 102:33:31 | +26 s | ~49,000 ft | Throttle to max; descent guidance enabled |
| 102:36:55 | +230 s | — | Yaw face-up |
| ~102:37:53 | +288 s | ~35,000 ft | LR "data good" — nav-frame conversion adds ~2% inside SERVICER (margin 15→13%) |
| **~102:38:04** | **+299 s** | **~34,000 ft** | **Aldrin keys 1668 (V16 N68, DELTAH monitor)** — margin → ≤10%; demand with theft ~105% |
| **102:38:22** | **+316 s** | ~33,500 ft | **ALARM 1: 1202** (no core sets) — ~6 cycles after the monitor. Restart sheds the monitor; guidance never stops |
| ~102:38:55 | +350 s | ~30,000 ft | Houston "Go"; Aldrin keys V57, re-keys V16 N68 |
| **102:39:02** | **+356 s** | ~29,000 ft | **ALARM 2: 1202** — same pattern, ~12 s after re-key |
| 102:39:31 | +386 s | ~24,500 ft | Throttle down, on time |

**When did Buzz type the 1668?** At ~102:38:04 (PDI+299 s, ~34,000 ft), mid-P63, right
after radar lock — and again at ~102:38:55. Each keying was followed ~12 s (≈6 guidance
cycles) later by a **1202**. So the 1668 caused the two P63 **core-set** overflows only.
It did **not** cause the 1201 — that came in P64 with no monitor up at all.

## Region 4 — P64 (+506 s → +603 s): alarms 3–5, including the 2,000 ft one

P64 added landing-point-designator/redesignation processing to every SERVICER pass
(essential, unsheddable) and HIGATJOB parked on a VAC ~8 s at entry. No monitor needed —
demand was > 105% on essentials alone, so restarts cured nothing.

| GET | T+PDI | Alt | Event |
| :--- | :--- | :--- | :--- |
| 102:41:32 | +506 s | 7,400 ft | High gate — **P64**, pitchover, LPD active |
| **102:42:18** | **+552 s** | ~3,000 ft | **ALARM 3: 1201** (no VAC areas) — stub VACs + HIGATJOB's held VAC hit the 5-VAC wall |
| **102:42:43** | **+578 s** | **~2,000 ft** | **ALARM 4: 1202** — this is the ~2,000 ft alarm. **Still P64, not P66** |
| **102:42:58** | **+594 s** | 770 ft | **ALARM 5: 1202** (last) |
| 102:43:08 | +603 s | ~650 ft | Armstrong: AUTO → ATT HOLD — sheds auto-steering load; alarms stop |

## Region 5 — P66 (+615 s → touchdown): zero alarms

Correction to the premise: **no alarm ever occurred in P66.** The last alarm (102:42:58)
was 22 s before P66 entry (102:43:20, ~430 ft, via the ROD switch). P66 *reduced* load —
Armstrong flew attitude by hand while SERVICER kept only average-G + the vertical
(rate-of-descent) channel; redesignation logic gone, no monitor. Demand dropped below
100% even with the theft still running, and *Eagle* flew 2 min 20 s alarm-free to
touchdown at 102:45:40.

## The five alarms, one line

1202 (+316 s, P63) · 1202 (+356 s, P63) · 1201 (+552 s, P64) · 1202 (+578 s, P64) ·
1202 (+594 s, P64) — four core-set overflows, one VAC; two caused by the 1668 monitor,
three by P64's own load; none in P66.
