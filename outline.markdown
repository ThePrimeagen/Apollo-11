# Repository Outline: Apollo-11

## What this repository is

This is the **original Apollo 11 Guidance Computer (AGC) source code** — the actual assembly
code that flew humans to the Moon in July 1969. It was written at the MIT Instrumentation
Laboratory under Margaret H. Hamilton (Colossus Programming Leader), digitized from hardcopy
printouts held by the MIT Museum, and transcribed by the [Virtual AGC](http://www.ibiblio.org/apollo/)
project. The code is in the public domain.

The repository is an **archival/transcription project**, not an active software project:
pull requests are accepted to fix discrepancies between these transcriptions and the original
source scans (typos in comments, formatting, missed pages), *not* to "fix" or modernize the code.

## Top-level structure

| Path | Description |
| :--- | :---------- |
| `Comanche055/` | Command Module (CM) AGC program **Comanche 055** (Colossus 2A), 86 `.agc` files, ~65,000 lines. Assembled Apr 1, 1969. |
| `Luminary099/` | Lunar Module (LM) AGC program **Luminary 099** (Luminary 1A), 91 `.agc` files, ~65,000 lines. Assembled Jul 14, 1969. |
| `Translations/` | README and CONTRIBUTING translated into ~40 languages. |
| `README.md` | Project overview, attribution, and the original NASA contract/approval sign-offs. |
| `CONTRIBUTING.md` | How to contribute (proofreading against the original scans). |
| `.github/` | Issue templates (including proofreading templates for each program), PR template, labeler, and CI workflows. |
| `package.json` / `bun.lockb` | Only dev tooling: `markdownlint-cli2` for linting the markdown docs (`npm run lint`). |

## The two flight programs

Both programs run on the same hardware — the Apollo Guidance Computer, a 2 kHz-clocked machine
with ~36K words of read-only "rope" memory and ~2K words of erasable memory — and share a large
common core (executive, interpreter, navigation, display). They are written in AGC assembly
language, assembled today with the `yaYUL` assembler from Virtual AGC.

### `Comanche055/` — Command Module (Colossus 2A)

Flew in the Command Module *Columbia* with Michael Collins. Notable areas:

- **Entry point / assembly order**: `MAIN.agc` lists every file in assembly order — it is the
  best "table of contents" for reading the code.
- **Operating system**: `EXECUTIVE.agc` (priority-based job scheduler), `WAITLIST.agc`
  (timed task scheduler), `INTERRUPT_LEAD_INS.agc`, `FRESH_START_AND_RESTART.agc` and
  `RESTARTS_ROUTINE.agc` (the famous restart-protection design).
- **Interpreter**: `INTERPRETER.agc` — a virtual machine providing pseudo-instructions for
  vector/matrix and double-precision math on top of the bare hardware.
- **Astronaut interface (DSKY)**: `PINBALL_GAME_BUTTONS_AND_LIGHTS.agc` and
  `PINBALL_NOUN_TABLES.agc` — the verb/noun display-keyboard system.
- **Mission programs** (`P` files): `P11.agc` (launch), `P20-P25.agc` (rendezvous navigation),
  `P30-P37.agc` (targeting), `P40-P47.agc` (powered flight), `P51-P53.agc` (IMU alignment),
  `P61-P67.agc` (entry), plus `REENTRY_CONTROL.agc` and `CM_ENTRY_DIGITAL_AUTOPILOT.agc` for
  atmospheric reentry — unique to the CM.
- **Autopilots**: `RCS-CSM_DIGITAL_AUTOPILOT.agc` (reaction control) and the `TVC*.agc` files
  (thrust-vector control of the main engine).
- **Navigation & math**: `CONIC_SUBROUTINES.agc`, `ORBITAL_INTEGRATION.agc`,
  `MEASUREMENT_INCORPORATION.agc` (Kalman-style filtering), `STAR_TABLES.agc`,
  `LUNAR_AND_SOLAR_EPHEMERIDES_SUBROUTINES.agc`.

### `Luminary099/` — Lunar Module (Luminary 1A)

Flew in the Lunar Module *Eagle* with Armstrong and Aldrin. Shares most infrastructure with
Comanche; the LM-specific highlights are the reason this repo is famous:

- **`BURN_BABY_BURN--MASTER_IGNITION_ROUTINE.agc`** — engine ignition, named after DJ
  Magnificent Montague's catchphrase.
- **`THE_LUNAR_LANDING.agc`** and **`LUNAR_LANDING_GUIDANCE_EQUATIONS.agc`** — the powered
  descent guidance that landed on the Moon (P63–P67 logic).
- **`P70-P71.agc`** — descent/ascent abort programs.
- **`ASCENT_GUIDANCE.agc`** and `P12.agc` — lunar liftoff.
- **`PINBALL_GAME_BUTTONS_AND_LIGHTS.agc`** — contains the famous
  `TEMPORARY, I HOPE HOPE HOPE` comment and other transcribed humor.
- **`ALARM_AND_ABORT.agc`** and `EXECUTIVE.agc` — the machinery behind the 1201/1202 program
  alarms during the Apollo 11 landing.
- **LM-specific control**: `P-AXIS_RCS_AUTOPILOT.agc`, `Q_R-AXIS_RCS_AUTOPILOT.agc`,
  `TJET_LAW.agc` (jet firing-time law), `TRIM_GIMBAL_CONTROL_SYSTEM.agc`,
  `THROTTLE_CONTROL_ROUTINES.agc`, `KALMAN_FILTER.agc`.
- **Sensors**: `RADAR_LEADIN_ROUTINES.agc` and `RADARUPT` handling (landing/rendezvous radar),
  `AOTMARK.agc` (alignment optical telescope), `LANDING_ANALOG_DISPLAYS.agc`.
- **AGS interface**: `AGS_INITIALIZATION.agc` — hand-off data for the backup Abort Guidance System.

## Anatomy of an `.agc` file

Each file begins with a standard transcription header noting its purpose, the assembler
(`yaYUL`), the contact (Ron Burkey), and the page numbers of the original scan it was
transcribed from. The body preserves the original 1969 code and comments, including original
typos (which are annotated rather than fixed, per the contributing rules).

## How the pieces fit together (reading guide)

1. Start with `Comanche055/MAIN.agc` or `Luminary099/MAIN.agc` for the full assembly order.
2. `ASSEMBLY_AND_OPERATION_INFORMATION.agc` explains verbs, nouns, alarm codes, and how the
   crew operated the DSKY.
3. `ERASABLE_ASSIGNMENTS.agc` and `FLAGWORD_ASSIGNMENTS.agc` (LM) are the "memory map" —
   variable declarations for the 2K words of RAM.
4. `EXECUTIVE.agc` + `WAITLIST.agc` are the cooperative multitasking kernel.
5. `INTERPRETER.agc` is the math virtual machine most guidance code is written against.
6. Then pick a mission phase: the `P*.agc` program files are the top-level mission logic,
   and `R*.agc` files are supporting routines.

## Tooling and CI

- **No build in this repo.** To assemble and run the code, use the
  [Virtual AGC](https://github.com/virtualagc/virtualagc) project (`yaYUL` assembler +
  `yaAGC` emulator).
- **Markdown lint**: `npm run lint` (or `bun`) runs `markdownlint-cli2` over the docs;
  CI runs it via `.github/workflows/markdownlint.yml`.
- **Labeler**: `.github/workflows/label.yml` auto-labels PRs by path (Comanche vs. Luminary
  vs. translations).

## Ideas for exploring further

- Compare a shared file (e.g. `EXECUTIVE.agc`) between `Comanche055/` and `Luminary099/` to
  see how the CM and LM builds diverged.
- Read `LUNAR_LANDING_GUIDANCE_EQUATIONS.agc` alongside an account of the Apollo 11
  1202 alarms.
- Grep the comments for humor: `BURN, BABY, BURN`, `HOPE HOPE HOPE`, `OFF TO SEE THE WIZARD`,
  `ASTRONAUT: PLEASE CRANK THE SILLY THING AROUND`.
