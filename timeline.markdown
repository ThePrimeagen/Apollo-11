# Apollo 11 Powered Descent — Exact Timeline of the 1201/1202 Alarms

This is a second-by-second reconstruction of the five guidance-computer program alarms
during the Apollo 11 lunar landing on **20 July 1969**, including the exact air-to-ground
voice exchanges, altitudes, and descent rates.

For the causal/source-code reading order, start with
[`table_of_contents.md`](table_of_contents.md). This file is the mission-time companion to the
three focused technical explanations.

## How to read the times

- **GET** = *Ground Elapsed Time* (mission time), format `HH:MM:SS`, counted from launch
  (16 July 1969, 13:32:00 UTC). This is the primary clock used in all NASA transcripts.
- **UTC** is on 20 July 1969. The conversion for descent is `UTC = GET − 82:28:00`
  (anchored to the official timeline: Powered Descent ignition GET 102:33:05 = 20:05:05 UTC).
- **T+PDI** = seconds after Powered Descent Initiation (engine ignition), the reference MIT
  used in its post-flight analysis. PDI = GET 102:33:05.01.
- **Speed-of-light delay:** the Moon was ~384,000 km away, so radio took **~1.3 seconds each
  way** (~2.6 s round trip). Transcript timestamps are logged at Houston, so a crew
  transmission was *spoken ~1.3 s earlier* than its logged time, and a Houston reply
  *reached the crew ~1.3 s after* its logged time. This matters below: the gap the crew
  actually experienced between asking about the 1202 and hearing "Go" was a couple of seconds
  longer than the raw transcript difference.

## Master timeline

| GET (HH:MM:SS) | UTC (20 Jul) | T+PDI | Event | Altitude | Descent rate |
| :------------- | :----------- | :---- | :---- | :------- | :----------- |
| 102:33:05.01 | 20:05:05 | +0 s | **PDI** — descent engine ignites at 10% throttle (P63, braking phase) | 49,971 ft | 2.2 ft/s |
| 102:33:31 | 20:05:31 | +26 s | Throttle up to maximum (~9,870 lb); descent guidance enabled | ~49,000 ft | — |
| ~102:37:53 | ~20:09:53 | ~+288 s | Landing radar "data good"; Delta‑H ≈ −2,900 ft | ~35,000 ft | — |
| ~102:38:10 | ~20:10:10 | ~+305 s | Aldrin keys **V16 N68** (DELTA‑H monitor) — the extra load that tipped the computer over | ~34,000 ft | — |
| **102:38:22** | **20:10:22** | **+317 s** | **ALARM 1 — 1202** (Executive overflow, no core sets); PROG light on, DSKY reverts to V06 N63 | **~33,500 ft** | ~120 ft/s |
| 102:38:42 | 20:10:42 | +337 s | Armstrong (heard at Houston): *"Give us a reading on the 1202 Program Alarm."* | ~32,000 ft | — |
| 102:38:53 | 20:10:53 | +348 s | Duke (transmitted from Houston): *"We're Go on that alarm."* Crew hears it ~102:38:54 | ~31,000 ft | — |
| 102:39:14 | 20:11:14 | +369 s | Aldrin: *"Same alarm, and it appears to come up when we have a 1668 up."* | ~27,000 ft | — |
| **102:39:02** | **20:11:02** | **+357 s** | **ALARM 2 — 1202** (per Apollo 11 Mission Report, Table 5‑I) | ~29,000 ft | ~125 ft/s |
| 102:39:35 | 20:11:35 | +390 s | **Throttle down** — right on time; strong confidence builder | ~24,000 ft | — |
| 102:41:32 | 20:13:32 | +507 s | **High Gate** reached; **P64** (approach/visibility phase) begins, LM pitches forward | 7,400 ft | 125 ft/s |
| 102:42:10 | 20:14:10 | +545 s | Duke: *"Eagle, Houston. You're Go for landing."* | ~3,500 ft | — |
| **102:42:18** | **20:14:18** | **+553 s** | **ALARM 3 — 1201** (Executive overflow, no VAC areas) | **~3,000 ft** | ~60 ft/s |
| 102:42:19 | 20:14:19 | +554 s | Aldrin: *"Program Alarm. 1201."* → 102:42:24 Armstrong: *"1201."* | ~3,000 ft | — |
| 102:42:25 | 20:14:25 | +560 s | Duke: *"Roger. 1201 alarm. We're Go. Same type. We're Go."* | ~2,900 ft | — |
| **102:42:43** | **20:14:43** | **+578 s** | **ALARM 4 — 1202** | **~2,000 ft** | ~50 ft/s |
| **102:42:58** | **20:14:58** | **+593 s** | **ALARM 5 — 1202** (last alarm) | **~770 ft** | 27 ft/s |
| 102:43:01 | 20:15:01 | +596 s | Duke: *"Roger. 1202. We copy it."* (acknowledging the 5th alarm) | 750 ft | 23 ft/s |
| 102:43:08 | 20:15:08 | +603 s | Armstrong switches **AUTO → ATT HOLD**, taking semi-manual control (sheds computer load; no further alarms) | ~650 ft | ~20 ft/s |
| 102:43:20 | 20:15:20 | +615 s | Armstrong enters **P66** (Rate-of-Descent mode) via the ROD switch | ~430 ft | ~15 ft/s |
| 102:44:45 | 20:16:45 | +700 s | Low-fuel quantity light latches; ~5.6% propellant left ("Bingo" countdown starts) | 100 ft | 3.5 ft/s |
| **102:45:40** | **20:17:40** | **+755 s** | **TOUCHDOWN** — *Eagle* lands in the Sea of Tranquility | 0 ft | 0 |
| 102:45:58 | 20:17:58 | +773 s | Armstrong: *"Houston, Tranquility Base here. **The Eagle has landed.**"* | — | — |

## The famous 1202 exchange, verbatim (P63)

From the corrected Apollo 11 Lunar Surface Journal transcript. Onboard = intercom
(not transmitted); the rest is on the air-to-ground loop.

```
102:38:26  Armstrong:  (slight urgency) Program Alarm.
102:38:30  Armstrong:  (to Houston) It's a 1202.
102:38:32  Aldrin:     1202.  (Pause)
102:38:42  Armstrong:  (onboard, to Buzz) What is it? Let's incorporate (the landing radar).
                       (to Houston) Give us a reading on the 1202 Program Alarm.
102:38:53  Duke:       Roger. We got you...(some urgency) We're Go on that alarm.
102:38:59  Armstrong:  Roger.  [onboard: "P30 ... Okay"]
102:39:14  Aldrin:     Same alarm, and it appears to come up when we have a 1668 up.
102:39:17  Duke:       Roger. Copy.
```

- **The wait the crew felt:** Armstrong's request was logged at Houston at 102:38:42, so he
  *spoke* it at about 102:38:40.7 (subtracting ~1.3 s uplink... i.e. downlink light-time).
  Duke's "We're Go" was transmitted at 102:38:53 and reached the crew at about 102:38:54.3.
  So from Armstrong finishing his question to hearing the answer was roughly **13–14 seconds**
  — about 11 s of it Steve Bales/Jack Garman deliberating on the ground, plus the round-trip
  light delay. (Eyles recalls "half a minute elapsed between the alarm and the 'go'," measuring
  from the alarm itself at 102:38:22 rather than from Armstrong's spoken request.)
- Aldrin's "1668" is **Verb 16 Noun 68** — the DELTA‑H monitor he had keyed up. He correctly
  intuited that requesting that display was pushing the already-loaded computer over the edge.

## The 1201 exchange, verbatim (P64)

```
102:42:10  Duke:       Eagle, Houston. You're Go for landing. Over.
102:42:17  Aldrin:     Roger. Understand. Go for landing. 3000 feet. Program Alarm.
102:42:19  Aldrin:     1201.
102:42:24  Armstrong:  1201.  [onboard: "Okay, 2000 at 50."]
102:42:25  Duke:       Roger. 1201 alarm. We're Go. Same type. We're Go.
102:42:31  Aldrin:     2000 feet. 2000 feet.  [Into the AGS, 47 degrees.]
102:42:41  Duke:       Eagle, looking great. You're Go.
102:42:58  Duke:       Roger. 1202. We copy it.
```

This is the busiest, most dangerous stretch: three alarms (1201, 1202, 1202) inside
**40 seconds** while descending from 3,000 ft to under 800 ft. Armstrong's heart rate rose
from 120 to 150 bpm. Because P64 had *no* dispensable load to shed, the restarts could not
cure the overload — it only stopped when Armstrong took manual attitude control at 102:43:08.

## Notes, discrepancies, and sourcing

- **Number of alarms:** five total — **four 1202** and **one 1201** — split two in P63,
  three in P64. This is consistent across the NASA SP‑4029 timeline, the Apollo 11 Mission
  Report (Table 5‑I), Cherry's MIT memo, and Eyles' account.
- **Alarm-time source spread:** the official NASA SP‑4029 *Apollo 11 Timeline* lists the
  alarms at GET 102:38:22, 102:39:02, 102:42:18, 102:42:43, 102:42:58. MIT's Cherry memo
  lists them relative to PDI at +316, +356, +552, +578, +594 s — which map to 102:38:21,
  102:39:01, 102:42:17, 102:42:43, 102:42:59. These agree to within one second. Where the
  transcript's *spoken* callout of an alarm differs from the logged computer time (e.g. the
  Mission Report puts the 1201 at 102:41:32 while Aldrin's voice call is at 102:42:19), the
  computer/telemetry time is used above and the difference noted.
- **First-alarm time caveat:** the Apollo 11 Mission Report (Table 5‑I) gives the *first*
  1202 as 102:39:02; the SP‑4029 timeline and most secondary sources use 102:38:22. The table
  above lists both events; treat ±40 s uncertainty on the very first alarm's label.
- **Altitudes** are from the ALSJ narrative annotations and the crew's own callouts
  (33,500 ft at the first alarm; 7,400 ft at High Gate/P64; 3,000 ft, 2,000 ft, and 770 ft at
  the three P64 alarms) and from Eyles (770 ft / 27 ft/s at the final alarm). Values between
  fixed callouts are interpolated and marked with "~".
- **Touchdown** at GET 102:45:40 (20:17:40 UTC) is the engine-off/contact time; the famous
  "The Eagle has landed" transmission followed at 102:45:58.

## Sources

1. NASA, *Apollo by the Numbers / SP‑4029 Apollo 11 Timeline*: authoritative GET and UTC
   for PDI and each alarm.
   <https://web.archive.org/web/20190817230133/https://history.nasa.gov/SP-4029/Apollo_11i_Timeline.htm>
2. NASA Apollo 11 Lunar Surface Journal, *The First Lunar Landing* (corrected air-to-ground
   transcript with altitude annotations).
   <https://www.nasa.gov/wp-content/uploads/static/history/alsj/a11/a11.landing.html>
3. Don Eyles, *Tales From the Lunar Module Guidance Computer* (AAS 04‑064, 2004): altitudes
   and descent rates at the P64 alarms, and the P63-vs-P64 load analysis.
   <http://www.klabs.org/history/apollo_11_alarms/eyles_2004/eyles_2004.htm>
4. George W. Cherry (MIT/IL), *Exegesis of the 1201 and 1202 Alarms* (4 Aug 1969): the five
   alarm times relative to PDI.
   <https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf>
5. Apollo 11 Technical Air‑to‑Ground Voice Transcription (GOSS NET 1), NASA, July 1969.
   <http://apollo-11.schwagmeier.net/data/document/a11trscr_tec_html/a11trscr_tec.htm>

For the underlying cause and the exact code path, see [`errorcodes.markdown`](errorcodes.markdown).
