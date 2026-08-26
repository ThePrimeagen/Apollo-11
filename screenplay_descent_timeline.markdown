# Screenplay Descent Timeline — What Ran in Each Region

The four regions — pre-descent, P63, P64, P66 — with the 10% throttle phase broken out
on its own. PDI = 102:33:05 GET.[^sp4029] Every dated claim carries a bracketed source
link. Job intervals/costs and the full source-linked job table in
[`screenplay_outline.markdown`](screenplay_outline.markdown).

## Region 1 — After undocking, before the engine (100:12 → 102:33)

Very little was running: no descent program work until P63 was called up, and no heavy
work until average-G started ~30 s before ignition.

| GET | Event | What the computer ran |
| :--- | :--- | :--- |
| 100:12 | *Eagle* undocks[^sp4029] | Coasting flight: T4RUPT (120 ms), DOWNRUPT (50/s), DAP attitude hold, DSKY on demand. **No SERVICER** — the 2 s guidance loop isn't running. Duty cycle low. |
| ~descent prep | RR mode switch → SLEW/AUTO (checklist)[^alsj-slew][^cherry-checklist] | **The ~15% counter theft begins here** — harmless while demand is low; nobody notices. |
| ~102:27 (PDI−6 min) | Crew keys **V37E 63E** — P63 | Ignition algorithm computes TIG/attitude once, then waits (BURNBABY ignition chain runs as timed tasks).[^code-burnbaby] |
| 102:32:35 (PDI−30 s) | **Average-G on** | READACCS + SERVICER 2-second loop starts.[^code-preread] From here the machine is loaded. |
| 102:32:58 (PDI−7.5 s) | Ullage — RCS settles propellant | |
| 102:33:00 (PDI−5 s) | Aldrin keys PRO | Crew go for ignition. |

## Region 2 — The 10% throttle phase (PDI+0 → +26 s)

Not a *test* — the DPS lights at 10% so the engine gimbal can be trimmed through the
LM's center of mass before full thrust is committed.

Running: **P63 core loop only** — SERVICER + READACCS (every 2 s, one core set + one
VAC), the per-pass display job, DAP (100 ms), T4RUPT (120 ms), DOWNRUPT (20 ms), R10/R11
displays (250 ms), throttle logic inside SERVICER, RR theft (~15%).

Not running yet: guidance steering (enabled at throttle-up +26 s[^sp4029-thr]),
**landing-radar jobs** (no lock until ~+288 s[^tillman-t1]), **V16N68 monitor** (keyed
+304 s[^cherry-events]), P64 redesignation, HIGATJOB.

So: all the *core* programs, yes — none of the extras. Load ≈ <85% known
software[^eyles-margins] + 15% theft ≈ ~100% — a quiet knife edge.

## Region 3 — P63 full throttle (+26 s → +506 s): the 1668 and alarms 1–2

| GET | T+PDI | Alt | Event |
| :--- | :--- | :--- | :--- |
| 102:33:31 | +26 s | ~49,000 ft | Throttle to max; descent guidance enabled[^sp4029-thr] |
| 102:36:55 | +230 s | — | Yaw face-up[^cherry-events] |
| ~102:37:53 | +288 s | ~35,000 ft | LR "data good"[^tillman-t1] — nav-frame conversion adds ~2% inside SERVICER (margin 15→13%)[^eyles-margins] |
| **~102:38:04** | **+304 s** | **~34,000 ft** | **Aldrin keys 1668 (V16 N68, DELTAH monitor)**[^cherry-events] — margin → ≤10%;[^eyles-margins] demand with theft ~105% |
| **102:38:22** | **+316 s** | ~33,500 ft | **ALARM 1: 1202** (no core sets)[^sp4029][^cherry-events] — 12 s into the monitor.[^tillman-p2] Restart sheds the monitor; guidance never stops[^cherry-restart] |
| ~102:38:55 | +350 s | ~30,000 ft | Houston "Go"; Aldrin keys V57 (+338), re-keys V16 N68 (+346)[^cherry-events] |
| **102:39:02** | **+356 s** | ~29,000 ft | **ALARM 2: 1202**[^sp4029] — again 12 s into the monitor[^tillman-p2] |
| 102:39:31 | +386 s | ~24,500 ft | Throttle down, on time[^alsj-thrdown] |

**When did Buzz type the 1668?** At ~102:38:04 (PDI+304 s, ~34,000 ft), mid-P63, right
after radar lock — and again at ~102:38:55.[^cherry-events] He typed it **once each
time**; the monitor then refreshes itself every second (MONREQ respawns the MONDO job at
1 Hz) with no further typing.[^code-monreq] Each monitor start was followed ~12 s later
by a **1202**; a third use (~+374, exited after ~10 s) caused no
alarm.[^tillman-p2][^cherry-events] The 1668's *running load* — not the keystrokes —
caused the two P63 core-set overflows.[^eyles-1668] It did **not** cause the 1201.

## Region 4 — P64 (+506 s → +603 s): alarms 3–5, including the 2,000 ft one

P64 added landing-point-designator/redesignation processing to every SERVICER pass
(essential, unsheddable),[^eyles-p64] put up the **flashing V06N64** display (a job that
holds a core set + VAC while it sleeps awaiting the crew's PRO),[^tillman-t1][^code-makeplay]
and HIGATJOB parked on a VAC (≤22 s, from ~6 s before high gate).[^code-higat] Per the
Tillman downlist, all three P64 alarms occurred **with no crew DSKY
activity**.[^tillman-p2] Demand was > 105% on essentials alone, so restarts cured
nothing.[^eyles-p64]

| GET | T+PDI | Alt | Event |
| :--- | :--- | :--- | :--- |
| 102:41:32 | +506 s | 7,400 ft | High gate — **P64**, pitchover, flashing V06N64 up, LPD active[^tillman-t1] |
| **102:42:18** | **+552 s** | ~3,000 ft | **ALARM 3: 1201** (no VAC areas)[^sp4029] — flashing-display VAC + HIGATJOB VAC + stub VACs hit the 5-VAC wall |
| 102:42:33 | +568 s | — | Crew keys **PRO** — V06N64 goes static (its VAC hold ends); HIGATJOB gone since the restart[^cherry-events] |
| **102:42:43** | **+578 s** | **~2,000 ft** | **ALARM 4: 1202**[^sp4029] — this is the ~2,000 ft alarm. **Still P64, not P66.** Core-first regime again |
| **102:42:58** | **+594 s** | 770 ft | **ALARM 5: 1202** (last)[^sp4029][^eyles-770] |
| 102:43:08 | +603 s | ~650 ft | Armstrong: AUTO → ATT HOLD — sheds auto-steering load; alarms stop[^eyles-atthold] |

## Region 5 — P66 (+615 s → touchdown): zero alarms

**No alarm ever occurred in P66.** The last alarm (102:42:58) was 22 s before P66 entry
(102:43:20, ~430 ft, via the ROD switch).[^tillman-t1] P66 *reduced* load — Armstrong
flew attitude by hand while SERVICER dropped to a lighter vertical-channel profile
(average-G + throttle only; redesignation gone, no monitor).[^eyles-atthold] P66 did add
one small loop of its own — `RODTASK` every 1 s spawning `RODCOMP` (prio 22, takes a
core set + VAC briefly) to fold Armstrong's ROD-switch clicks (±1 ft/s each) into the
commanded descent rate[^code-rod] — but total demand stayed under 100% even with the
theft still running, and *Eagle* flew 2 min 20 s alarm-free to touchdown at
102:45:40.[^sp4029][^eyles-atthold]

## The five alarms, one line

1202 (+316 s, P63) · 1202 (+356 s, P63) · 1201 (+552 s, P64) · 1202 (+578 s, P64) ·
1202 (+594 s, P64)[^cherry-p1][^sp4029] — four core-set overflows, one VAC; two
triggered by the 1668 monitor's load, three by P64's own load; none in P66.

## Sources

[^sp4029]: NASA SP-4029, *Apollo 11 Timeline* — undocking, PDI, the five alarms (102:38:22, 102:39:02, 102:42:18, 102:42:43, 102:42:58), touchdown: [archived timeline](https://web.archive.org/web/20190817230133/https://history.nasa.gov/SP-4029/Apollo_11i_Timeline.htm#:~:text=LM%201202%20alarm).
[^sp4029-thr]: Throttle-up at PDI+26 s (102:33:31): ALSJ landing transcript and annotations, [a11.landing.html](https://www.nasa.gov/wp-content/uploads/static/history/alsj/a11/a11.landing.html); NASA SP-4029, [archived timeline](https://web.archive.org/web/20190817230133/https://history.nasa.gov/SP-4029/Apollo_11i_Timeline.htm).
[^cherry-p1]: Cherry, *Exegesis* — the five alarms with PDI offsets (+316/+356/+552/+578/+594), [p. 1](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=1).
[^cherry-events]: Cherry, "Important Events Occurring During Lunar Landing" — V16N68 at +304, alarm +316, V57E +338, V16N68E +346, alarm +358, V16N68E +374 / Key Release +380, throttle down +384, P64 +506, alarm +554, PRO to FLV06N64 +568, alarms +578/+594, ATT HOLD +606, P66 +618, touchdown +760: [pp. 13–14](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=13).
[^cherry-restart]: Cherry — restart flushes the backlog; "monitor verbs or extended verbs are not automatically restarted", [pp. 5–6](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=5).
[^cherry-checklist]: Cherry — "The Apollo 11 crew checklist specified that the RR switch be in AUTO TRACK immediately before calling P63", [p. 8](https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf#page=8).
[^tillman-p2]: Tillman memo — "in P63 there were two alarms in each case after 12 seconds of a monitor verb. One other use of the monitor for 9 or 10 seconds did not induce an alarm … There was no Crew DSKY activity related to these [P64 alarms]", [p. 2](https://ibiblio.org/apollo/Documents/Memo-Tillman690731_text.pdf#page=2).
[^tillman-t1]: Tillman memo, Table I (LGC downlist) — LR "data good" 102:37:49, flashing V06N64 at P64 entry 102:41:31, ROD/P66 selected 102:43:21: [Table I](https://ibiblio.org/apollo/Documents/Memo-Tillman690731_text.pdf#page=5).
[^eyles-margins]: Eyles, *Tales* — [">15% before radar lock, ~13% after, ≤10% with the monitor"](https://www.doneyles.com/LM/Tales.html#:~:text=margin%20shrank%20again).
[^eyles-1668]: Eyles — ["When a monitor display such as Verb 16 Noun 68 was added, the margin shrank again"](https://www.doneyles.com/LM/Tales.html#:~:text=monitor%20display%20such%20as%20Verb%2016%20Noun%2068); Aldrin at 102:39:14: ["it appears to come up when we have a 16/68 up"](https://www.nasa.gov/wp-content/uploads/static/history/alsj/a11/a11.landing.html#:~:text=when%20we%20have%20a%2016%2F68%20up).
[^eyles-p64]: Eyles — ["the essential software by itself left a duty-cycle margin of less than 10% … the software restart flushed the Executive queue but could not shed load"](https://www.doneyles.com/LM/Tales.html#:~:text=could%20not%20shed%20load).
[^eyles-770]: Eyles — ["at 770 feet with a sink rate of 27 ft/sec, yet another 1202"](https://www.doneyles.com/LM/Tales.html#:~:text=770%20feet).
[^eyles-atthold]: Eyles — ["switched the autopilot from AUTO to ATT HOLD mode, easing the computational burden, and then entered semi-manual mode P66, where the burden was still lighter. After 2 minutes and 20 seconds spent maneuvering in P66 without alarms, the LM landed"](https://www.doneyles.com/LM/Tales.html#:~:text=easing%20the%20computational%20burden).
[^alsj-slew]: ALSJ landing transcript, Fjeld annotation at 102:33:00 — ["Neil puts the Rendezvous Radar mode switch in Slew — an action that leads to the Program Alarms"](https://www.nasa.gov/wp-content/uploads/static/history/alsj/a11/a11.landing.html#:~:text=Rendezvous%20Radar%20mode%20switch%20in%20Slew).
[^alsj-thrdown]: ALSJ landing transcript — throttle-down calls at 102:39:31, ["Throttle down on time!"](https://www.nasa.gov/wp-content/uploads/static/history/alsj/a11/a11.landing.html#:~:text=Throttle%20down%20on%20time).
[^code-monreq]: Flight code — `MONREQ` re-enlists every `MONDEL` = 1.00 s and spawns a fresh NOVAC `MONDO`: [`Luminary099/PINBALL_GAME_BUTTONS_AND_LIGHTS.agc` L2373–L2395](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/PINBALL_GAME_BUTTONS_AND_LIGHTS.agc#L2373-L2395).
[^code-makeplay]: Flight code — flashing display job allocated with a VAC (`VACDSP … TC SPVAC`), sleeps holding it: [`Luminary099/DISPLAY_INTERFACE_ROUTINES.agc` L836–L856](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/DISPLAY_INTERFACE_ROUTINES.agc#L836-L856).
[^code-higat]: Flight code — HIGATJOB (FINDVAC) sleeps on the position-2 discrete, 22-second window, entered ~6 s before high gate: [`Luminary099/SERVICER.agc` L740–L755](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/SERVICER.agc#L740-L755), [L1657–L1664](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/SERVICER.agc#L1657-L1664).
[^code-rod]: Flight code — `RODTASK CAF PRIO22 / TC FINDVAC / 2CADR RODCOMP`, re-armed every second; ROD clicks arrive via `RODCOUNT`: [`Luminary099/LUNAR_LANDING_GUIDANCE_EQUATIONS.agc` L934–L953](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/LUNAR_LANDING_GUIDANCE_EQUATIONS.agc#L934-L953).
[^code-burnbaby]: Flight code — the P63 ignition chain (TIG−35 → TIG−30 → TIG−5 → ignition) runs as timed tasks: [`Luminary099/BURN_BABY_BURN--MASTER_IGNITION_ROUTINE.agc`](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/BURN_BABY_BURN--MASTER_IGNITION_ROUTINE.agc).
[^code-preread]: Flight code — `PREREAD` starts average-G (arms READACCS at 2 s): [`Luminary099/SERVICER.agc` L44–L66](https://github.com/ThePrimeagen/Apollo-11/blob/master/Luminary099/SERVICER.agc#L44-L66).
