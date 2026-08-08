# The 1201 & 1202 Program Alarms — Apollo 11 Landing

## Brief outline

- **The story is true.** During Apollo 11's powered descent to the Moon on **20 July 1969**,
  the Lunar Module *Eagle*'s guidance computer (the LGC) repeatedly flashed **program alarms**
  — codes **1202** and **1201** — starting at roughly 33,000 feet. There were **five alarms
  total: four 1202s and one 1201.**
- **What the codes mean** (both are "Executive overflow" — the computer was given more work
  than it had memory to schedule):
  - **1202 — "EXECUTIVE OVERFLOW - NO CORE SETS"**: no free *core set* (the small block of
    erasable memory every job needs) was available.
  - **1201 — "EXECUTIVE OVERFLOW - NO VAC AREAS"**: no free *VAC area* (Vector Accumulator, a
    larger workspace needed by heavier interpretive math jobs) was available.
- **Root cause: a hardware/procedure interface problem, not a software bug.** The rendezvous
  radar (RR) mode switch was left in `AUTO TRACK`/`SLEW` instead of `LGC`. This left two 800 cps
  reference signals with a random phase relationship, so the radar's coupling data units (ECDUs)
  fired spurious "counter increment" pulses at up to ~6,400 cps. Each pulse stole a memory cycle,
  robbing roughly **13–15% of the computer's duty cycle**. Under the extra load of P64
  (landing-site redesignation), the Executive ran out of core sets/VAC areas.
- **Why the landing continued anyway:** the alarms were *not* crashes. Each alarm triggered a
  software **restart** (via the `BAILOUT` routine) that flushed the job queue and used
  pre-planned restart points to immediately resume only the essential jobs — guidance, engine
  steering, and the crew display (DSKY). The bogus radar jobs were simply not rescheduled.
  Because MIT had extensively tested this restart protection, the guidance officer (Steve Bales)
  and back-room engineer (Jack Garman) were able to call **"Go"** on each alarm.
- **Outcome:** *Eagle* landed safely at 20:17 UTC. The robust restart-based design of the AGC —
  led by Margaret Hamilton's team at the MIT Instrumentation Laboratory — is widely credited
  with saving the landing.

## The codes in *this* repository

This repo contains the actual Luminary 099 flight code, so you can read the alarm logic
yourself. The two codes are raised by the Executive's job-scheduling routine `FINDVAC`:

```147:147:Luminary099/EXECUTIVE.agc
		OCT	1201		# NO VAC AREAS.
```

```207:208:Luminary099/EXECUTIVE.agc
		TC	BAILOUT1	# NO CORE SETS AVAILABLE.
		OCT	1202
```

The Executive scans five VAC areas and, in this revision, **seven** core sets before giving up:

```155:163:Luminary099/EXECUTIVE.agc
NOVAC2		CAF	ZERO		# NOVAC ENTERS HERE.  FIND A CORE SET.
		TS	LOCCTR
		CAF	NO.CORES	# SEVEN SETS OF ELEVEN REGISTERS EACH.
NOVAC3		TS	EXECTEM2
		INDEX	LOCCTR
		CCS	PRIORITY	# EACH PRIORITY REGISTER CONTAINS -0 IF
		TCF	NEXTCORE	# THE CORESPONDING CORE SET IS AVAILABLE.
NO.CORES	DEC	7
		TCF	NEXTCORE	# AN ACTIVE JOB HAS A POSITIVE PRIORITY
```

> Note: contemporary memos describe *eight* core sets; this shipped Luminary 099 build defines
> seven (`NO.CORES DEC 7`). Either way, once they were exhausted the Executive raised 1202.

The alarm codes are documented in the program's own alarm-code table:

```966:967:Luminary099/ASSEMBLY_AND_OPERATION_INFORMATION.agc
# 01201	      *	EXECUTIVE OVERFLOW - NO VAC AREAS		EXEC
# 01202	      *	EXECUTIVE OVERFLOW - NO CORE SETS		EXEC
```

When a code is raised, control transfers to the `BAILOUT` routine, which stores the alarm and
initiates the protective software restart:

```133:133:Luminary099/ALARM_AND_ABORT.agc
BAILOUT		INHINT
```

## Sources

1. NASA / MIT contemporaneous memo — *Program Alarms in Powered Descent, Apollo 11*
   (Tillman/Draper Lab, 31 Jul 1969): tabulates "4 1202 alarms and 1 1201 alarm" and defines
   core sets vs. VAC areas. <https://ibiblio.org/apollo/Documents/Memo-Tillman690731_text.pdf>
2. E. M. Cherry, *Exegesis of the 1201 and 1202 Alarms Which Occurred During the Mission G
   Lunar Landing* (MIT IL) — details how the rendezvous radar ECDUs stole ~15% of LGC time.
   <https://www.ibiblio.org/apollo/Documents/CherryApollo11Exegesis.pdf>
3. Don Eyles, *Tales From the Lunar Module Guidance Computer* (2004) — first-hand account of
   the alarms and the operating system. <http://www.klabs.org/history/apollo_11_alarms/eyles_2004/eyles_2004.htm>
4. NASA — *Apollo 11 Lunar Surface Journal: Program Alarms* (Hal Laning / Don Eyles material),
   explaining VAC-area vs. core-set scanning and the restart behavior.
   <https://www.nasa.gov/wp-content/uploads/static/history/alsj/a11/a11.1201-pa.html>
