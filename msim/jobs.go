package msim

// ---------- MONDO / MONREQ: the V16N68 monitor ----------
//
// PINBALL_GAME_BUTTONS_AND_LIGHTS.agc L2373-L2395: MONREQ is a waitlist task.
// Each fire re-enlists itself MONDEL = 1.00 s later and enters a NOVAC
// request for MONDO at CHRPRIO. MONDO itself (L2397+) updates the display —
// and never sleeps: a busy display exits immediately through MONBUSY.
//
// MondoCost is the monitor's per-refresh CPU: the N68 DELTAH fetch, scaling,
// and three five-digit decimal conversions through UPDATNN/NVSUB. The
// documented envelope is 30-60 ms (see msim/RESEARCH.md).
const MondoCost Nanos = 30 * Millisecond

// MonreqCost: LODSAMPT + the two enlistments — a task-context sliver.
const MonreqCost Nanos = 200 * Microsecond

func mondoSpec() JobSpec {
	n := int(MondoCost / (5 * Millisecond))
	s := make(Script, 0, n)
	var left = MondoCost
	for left > 0 {
		c := Nanos(5 * Millisecond)
		if c > left {
			c = left
		}
		s = append(s, Instr{Section: "MONDO", Op: "BASIC",
			Ref: "PINBALL_GAME_BUTTONS_AND_LIGHTS.agc:2397", Cost: c})
		left -= c
	}
	return JobSpec{Name: "MONDO", Prio: 30, VAC: false, Script: s}
}

// StartMonitor keys the monitor up. The ENTR keystroke's own CHARIN paints
// the first frame; MONREQ is enlisted with MONDEL, so the first MONDO
// refresh runs 1.000 s after the ENTR and every second thereafter, until
// StopMonitor (V57/KILLMON) or a software restart (monitor verbs are not
// restarted — Cherry pp. 5-6). A future `at` is the crew's ENTR — a
// hardware event, immune to restarts before it.
func StartMonitor(e *Engine, at Nanos) {
	if at > e.Now() {
		e.ScheduleHardware(at, "V16N68-ENTR", 0, func(en *Engine) {
			en.monitorOn = true
			armMonreq(en, en.Now()+Second)
		})
		return
	}
	e.monitorOn = true
	armMonreq(e, at+Second)
}

func armMonreq(e *Engine, at Nanos) {
	e.ScheduleTask(at, "MONREQ", MonreqCost, func(en *Engine) {
		if !en.monitorOn {
			return // KILLMON: no request, no re-enlistment
		}
		armMonreq(en, en.Now()+Second)
		en.Spawn(mondoSpec())
	})
}

// StopMonitor is V57E / any KILLMON path: the chain dies permanently.
func StopMonitor(e *Engine) { e.monitorOn = false }

// ---------- MAKEPLAY: the display job ----------
//
// DISPLAY_INTERFACE_ROUTINES.agc L836-L856: the display job runs one
// priority above its user (GODSPRS1). A static display (P63's V06N63,
// P66's V06N60) is entered TC NOVAC and ends after pasting its digits.
// A flashing display (early P64's V06N64) branches to VACDSP → TC SPVAC:
// core set + VAC, and it sleeps holding both until the crew responds.

type DisplayKind int

const (
	DisplayStatic DisplayKind = iota
	DisplayFlashing
)

// DisplayCost: MAKEPLAY's relay-word assembly for a three-register noun —
// three five-digit decimal conversions plus DSPTAB queueing.
const DisplayCost Nanos = 8 * Millisecond

// SpawnDisplayJob enters the display job for a user at the given priority.
//
// The static form pastes and ends — UNLESS a monitor verb owns the DSKY.
// A blocked display user sleeps holding its resources (ENDIDLE/NVSBWAIT;
// PINBALL_GAME_BUTTONS_AND_LIGHTS.agc L3159-L3168 aborts 1206 only for a
// SECOND simultaneous sleeper), and the next display request wakes-and-kills
// the previous sleeper. While V16N68 runs, every guidance pass therefore
// leaves one core set parked on the display pipeline — the monitor's load
// beyond its own CPU.
func SpawnDisplayJob(e *Engine, kind DisplayKind, userPrio int) *Alarm {
	switch kind {
	case DisplayFlashing:
		e.Wake("MAKEPLAY") // wake-and-kill any previous display sleeper
		return e.Spawn(JobSpec{Name: "MAKEPLAY", Prio: userPrio + 1, VAC: true, Script: Script{
			{Section: "MAKEPLAY", Op: "BASIC", Ref: "DISPLAY_INTERFACE_ROUTINES.agc:851",
				Cost: DisplayCost / 2, SleepNs: SleepUntilWake},
			{Section: "MAKEPLAY", Op: "BASIC", Ref: "DISPLAY_INTERFACE_ROUTINES.agc:858",
				Cost: DisplayCost / 2},
		}})
	default:
		e.Wake("MAKEPLAY") // wake-and-kill any previous display sleeper
		if e.monitorOn {
			// DSPLOCK is the monitor's: attempt the paste, then NVSBWAIT
			return e.Spawn(JobSpec{Name: "MAKEPLAY", Prio: userPrio + 1, VAC: false, Script: Script{
				{Section: "MAKEPLAY", Op: "BASIC", Ref: "PINBALL_GAME_BUTTONS_AND_LIGHTS.agc:3159",
					Cost: DisplayCost / 4, SleepNs: SleepUntilWake},
				{Section: "MAKEPLAY", Op: "BASIC", Ref: "PINBALL_GAME_BUTTONS_AND_LIGHTS.agc:3168",
					Cost: DisplayCost / 8},
			}})
		}
		return e.Spawn(JobSpec{Name: "MAKEPLAY", Prio: userPrio + 1, VAC: false, Script: Script{
			{Section: "MAKEPLAY", Op: "BASIC", Ref: "DISPLAY_INTERFACE_ROUTINES.agc:844",
				Cost: DisplayCost / 2},
			{Section: "MAKEPLAY", Op: "BASIC", Ref: "DISPLAY_INTERFACE_ROUTINES.agc:858",
				Cost: DisplayCost / 2},
		}})
	}
}

// KeyPRO is the crew's PRO: it wakes the flashing display's sleeper.
func KeyPRO(e *Engine) { e.Wake("MAKEPLAY") }

// ---------- 1/GYRO: gyro compensation ----------
//
// IMU_COMPENSATION_PACKAGE.agc L107-L110: CA PRIO21 / TC NOVAC / 2CADR
// 1/GYRO — entered from 1/PIPA inside SERVICER when the accumulated
// compensation reaches two pulses, about once per second.

const GyroCost Nanos = 7 * Millisecond

func gyroSpec() JobSpec {
	return JobSpec{Name: "1/GYRO", Prio: 21, VAC: false, Script: Script{
		{Section: "1/GYRO", Op: "BASIC", Ref: "IMU_COMPENSATION_PACKAGE.agc:248", Cost: 3500 * Microsecond},
		{Section: "1/GYRO", Op: "BASIC", Ref: "IMU_COMPENSATION_PACKAGE.agc:360", Cost: 3500 * Microsecond},
	}}
}

// ---------- CHARIN: one DSKY keystroke ----------
//
// KEYRUPT_UPRUPT.agc L47-L50: each key fires KEYRUPT1, which enters CHARIN
// at CHRPRIO, NOVAC. ~5 ms of decoding and DSPTAB updates per key.

const CharinCost Nanos = 5 * Millisecond

func charinSpec() JobSpec {
	return JobSpec{Name: "CHARIN", Prio: 30, VAC: false, Script: Script{
		{Section: "CHARIN", Op: "BASIC", Ref: "KEYRUPT_UPRUPT.agc:50", Cost: CharinCost},
	}}
}

// Keystroke schedules the KEYRUPT for one key at time `at`. Keys are crew
// hardware: a software restart cannot unpress them.
func Keystroke(e *Engine, at Nanos, key string) {
	e.ScheduleHardware(at, "KEYRUPT", 100*Microsecond, func(en *Engine) {
		en.Note("key", "DSKY", key)
		en.Spawn(charinSpec())
	})
}

// ---------- the landing-radar read gates ----------
//
// Cherry's job table (Exegesis pp. 11-12) and the outline's cycle map put
// both reads in the locked P63 cycle: LRHJOB (PRIO32, NOVAC — SERVICER.agc
// L724-L727) runs ~1 ms, then sleeps ~95 ms through the radar's altitude
// gate HOLDING ITS CORE SET (L1567-L1570), fired by LRHTASK "50 MS PRIOR TO
// THE NEXT READACCS" (L697-L701) so the hold straddles the cycle boundary.
// LRVJOB (PRIO32, NOVAC — L1437) sleeps ~500 ms through five velocity
// samples (L1508-L1510) mid-cycle. These sleeps are the per-cycle NOVAC
// core-set pressure of the locked configuration.

func lrhSpec() JobSpec {
	return JobSpec{Name: "LRHJOB", Prio: 32, VAC: false, Script: Script{
		{Section: "LRHJOB", Op: "BASIC", Ref: "SERVICER.agc:1572", Cost: Millisecond,
			SleepNs: 95 * Millisecond},
		{Section: "LRHJOB", Op: "BASIC", Ref: "SERVICER.agc:1590", Cost: Millisecond},
	}}
}

func lrvSpec() JobSpec {
	return JobSpec{Name: "LRVJOB", Prio: 32, VAC: false, Script: Script{
		{Section: "LRVJOB", Op: "BASIC", Ref: "SERVICER.agc:1516", Cost: Millisecond,
			SleepNs: 500 * Millisecond},
		{Section: "LRVJOB", Op: "BASIC", Ref: "SERVICER.agc:1533", Cost: Millisecond},
	}}
}
