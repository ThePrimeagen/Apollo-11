package msim

// The two timelines. Window start t0 = PDI+290 s ≈ GET 102:37:55, just after
// landing-radar lock (Tillman Table I: "data good" 102:37:49-53). Flight
// anchors, expressed as ms offsets from t0 (Cherry pp. 13-14, PDI-relative):
//
//	V16N68 keyed  +304 s → keys end (ENTR) at t0+15.8 s
//	ALARM 1202 #1 +316 s → t0+26 s   (~12 s after the monitor started)
//	V57E          +338 s → t0+48 s
//	V16N68 again  +346 s → ENTR at t0+57.8 s
//	ALARM 1202 #2 +356/8 → t0+66-68 s
//	third V16N68  +374 s → ENTR t0+84 s; KEY REL +380 s → t0+90 s; no alarm
// Sub-second phases are free parameters (the 1969 event log has one-second
// resolution). The .995 phase puts the monitor's 1 Hz refresh request just
// before the guidance boundary — where the radar gates and the stub pile
// hold their cores — so MONDO's own NOVAC request is the one that finds the
// core wall: "the 1668's running load — not the keystrokes — caused the two
// P63 core-set overflows" (see msim/RESEARCH.md).
const (
	Monitor1EntrMS = 15_985
	V57EntrMS      = 47_985
	Monitor2EntrMS = 56_985
	Monitor3EntrMS = 83_985
	Monitor3RelMS  = 89_985
)

// T0GET is the window's ground-elapsed-time origin for rendering.
const T0GET = "102:37:55"

// Result is one scenario run.
type Result struct {
	Title      string
	DurationMS int
	Alarms     []Alarm
	Events     []Event
	Samples    []Sample
	SoftwareNs Nanos
	TheftNs    Nanos
	IdleNs     Nanos
	ElapsedNs  Nanos
	MaxCores   int
	MaxVACs    int
	Restarts   int
}

// descent wires the P63 radar-locked machine: the READACCS/SERVICER chain,
// the R10/R11 display task, the DAP/T4RUPT/DOWNRUPT cadences, the RR theft,
// and the restart tables.
type descent struct {
	e          *Engine
	servicer   Script
	lastGyroNs Nanos
	// stopped kills the READACCS/R10,R11 re-arm chains — the command
	// screen's DESCENT switch. The flight scenarios never set it.
	stopped bool
}

func newDescent(radarBug bool) *descent {
	d := &descent{lastGyroNs: -Second}
	script, err := ServicerScript(P63Locked)
	if err != nil {
		panic(err)
	}
	d.servicer = script.WithHooks(Hooks{
		Pipa:     d.pipaHook,
		Dispexit: d.dispexitHook,
	})
	d.e = NewEngine(Config{
		RadarBug:    radarBug,
		Interrupts:  true,
		AutoRestart: true,
		RestartHook: d.restartTables,
	})
	d.armReadaccs(0)
	d.armR10R11(250 * Millisecond)
	return d
}

// pipaHook is 1/PIPA's gyro-compensation gate: two pulses accumulate in
// about a second, then PRIO21 TC NOVAC 1/GYRO (IMU_COMPENSATION_PACKAGE.agc
// L107-L110).
func (d *descent) pipaHook(e *Engine) {
	if e.Now()-d.lastGyroNs >= Second {
		d.lastGyroNs = e.Now()
		e.Spawn(gyroSpec())
	}
}

// dispexitHook is P63DISPS: the static V06N63 display request — a NOVAC
// MAKEPLAY one priority above SERVICER (DISPLAY_INTERFACE_ROUTINES.agc
// L836-L847).
func (d *descent) dispexitHook(e *Engine) {
	SpawnDisplayJob(e, DisplayStatic, 20)
}


// armReadaccs is GOREADAX: CA 2SECS / TC VARDELAY — unconditional, punctual,
// and blind to whether the previous SERVICER finished (SERVICER.agc L80-L81).
// Each fire reads the PIPAs and FINDVACs a brand-new PRIO20 SERVICER
// (L120-L123), and sets the cycle's radar reads: both are timed so their
// data is fresh when the NEXT pass integrates it — the altitude gate fires
// 50 ms before the next READACCS (L697-L701), the five velocity samples
// (~500 ms, L1508-L1510) run through the cycle's second half and finish
// right at the boundary.
func (d *descent) armReadaccs(at Nanos) {
	d.e.ScheduleTask(at, "READACCS", Millisecond, func(en *Engine) {
		if d.stopped {
			return // the DESCENT switch: the chain dies here
		}
		d.armReadaccs(en.Now() + 2*Second)
		en.ScheduleTask(en.Now()+1500*Millisecond, "LRVTASK", 200*Microsecond, func(en2 *Engine) {
			en2.Spawn(lrvSpec())
		})
		en.ScheduleTask(en.Now()+1950*Millisecond, "LRHTASK", 200*Microsecond, func(en2 *Engine) {
			en2.Spawn(lrhSpec())
		})
		en.Spawn(JobSpec{Name: "SERVICER", Prio: 20, VAC: true, Script: d.servicer})
	})
}

// armR10R11 is the 0.25 s tape-meter/cross-pointer task (P70-P71.agc
// L36-L47, OCT31 = 0.25 s).
func (d *descent) armR10R11(at Nanos) {
	d.e.ScheduleTask(at, "R10,R11", 3000*Microsecond, func(en *Engine) {
		if d.stopped {
			return
		}
		d.armR10R11(en.Now() + 250*Millisecond)
	})
}

// restartTables mirrors RESTART_TABLES.agc 5.4SPOT: the flush is followed by
// one REREADAC (readaccs chain re-armed) and one fresh SERVICER; group-2
// protection restores the R10/R11 chain. The monitor is NOT restored. The
// rebuilt chain stays on the PIPTIME lattice — the phase tables carry the
// recorded time base, so READACCS keeps firing on the same 2 s grid.
func (d *descent) restartTables(e *Engine) {
	if d.stopped {
		return
	}
	next := ((e.Now()+20*Millisecond)/(2*Second) + 1) * (2 * Second)
	d.armReadaccs(next)
	d.armR10R11(e.Now() + 250*Millisecond)
}

func (d *descent) run(title string, durMS int) *Result {
	d.e.RunMS(durMS)
	return &Result{
		Title:      title,
		DurationMS: durMS,
		Alarms:     d.e.Alarms(),
		Events:     d.e.Events(),
		Samples:    d.e.Samples(),
		SoftwareNs: d.e.SoftwareBusyNs(),
		TheftNs:    d.e.TheftNs(),
		IdleNs:     d.e.IdleNs(),
		ElapsedNs:  Nanos(durMS) * Millisecond,
		MaxCores:   d.e.MaxCores(),
		MaxVACs:    d.e.MaxVACs(),
		Restarts:   d.e.RestartCount(),
	}
}

// RunBaselineP63 is timeline one: radar locked, RR bug stealing, nobody
// touches the DSKY.
func RunBaselineP63(durMS int) *Result {
	d := newDescent(true)
	return d.run("P63 baseline — radar locked, RR bug on, no monitor", durMS)
}

// RunBaselineP63NoBug is the control: the same software with the RR mode
// switch where it belonged.
func RunBaselineP63NoBug(durMS int) *Result {
	d := newDescent(false)
	return d.run("P63 control — RR bug off", durMS)
}

// keyV16N68 types V-1-6-N-6-8-ENTR ending at entr, ~0.45 s per key
// (seven keys over ~2.7 s), and starts the monitor chain at the ENTR.
func keyV16N68(e *Engine, entr Nanos, label string) {
	keys := []string{"VERB", "1", "6", "NOUN", "6", "8", "ENTR"}
	for i, k := range keys {
		at := entr - Nanos(len(keys)-1-i)*450*Millisecond
		Keystroke(e, at, k+" ("+label+")")
	}
	StartMonitor(e, entr)
	e.ScheduleHardware(entr, "NOTE", 0, func(en *Engine) {
		en.Note("note", "DSKY", "V16N68 ENTR — DELTAH monitor up ("+label+")")
	})
}

// RunMonitor1668 is timeline two: the same machine with Aldrin's V16N68 at
// the flight offsets.
func RunMonitor1668(durMS int) *Result {
	d := newDescent(true)
	e := d.e

	keyV16N68(e, Monitor1EntrMS*Millisecond, "first")

	// V57E — four keys; by now the restart has already killed the monitor
	for i, k := range []string{"VERB", "5", "7", "ENTR"} {
		Keystroke(e, (V57EntrMS-Nanos(3-i)*300)*Millisecond, k+" (V57)")
	}
	e.ScheduleHardware(V57EntrMS*Millisecond, "V57", 0, func(en *Engine) {
		StopMonitor(en)
		en.Note("note", "DSKY", "V57E — enable LR updates")
	})

	keyV16N68(e, Monitor2EntrMS*Millisecond, "second")
	keyV16N68(e, Monitor3EntrMS*Millisecond, "third")

	// KEY REL: the third monitor is dropped after ~6 s — too short to leak
	// the pools dry (flight: no alarm from this one)
	e.ScheduleHardware(Monitor3RelMS*Millisecond, "KEYREL", 0, func(en *Engine) {
		StopMonitor(en)
		en.Note("note", "DSKY", "KEY REL — monitor dropped")
	})

	return d.run("P63 with V16N68 — the DELTAH monitor at the flight offsets", durMS)
}
