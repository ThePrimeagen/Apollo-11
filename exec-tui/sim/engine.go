// Package sim simulates the Apollo Guidance Computer's Executive and
// Waitlist (Luminary 099) at the scheduler level: where the CPU time goes,
// who holds the 8 core sets and 5 VAC areas, and how the 1201/1202 alarms
// of Apollo 11 develop and recover.
//
// Every constant is sourced in ../RESEARCH.md. This is not a CPU emulator;
// it is a faithful model of the scheduler economics.
package sim

import "fmt"

// ---------------------------------------------------------------------------
// Constants (sourced; see RESEARCH.md)
// ---------------------------------------------------------------------------

const (
	NumCoreSets = 8 // EXECUTIVE.agc: ERASE +83D "EIGHT SETS OF 12 REGISTERS EACH"
	NumVACs     = 5 // VAC1USE..VAC5USE

	CyclePeriodMs = 2000.0 // SERVICER.agc GOREADAX: CA 2SECS / TC VARDELAY

	// DefaultWallToAGC: 1000ms of wall time shows 50ms of AGC time (20x slow).
	DefaultWallToAGC = 0.05

	// stepMs is the engine's quantum of AGC time.
	stepMs = 0.1

	// RadarBugStealFraction: 2 CDU counters x 6400 pulses/s x 11.72us.
	// Eyles: "approximately 15% of the available computation time."
	RadarBugStealFraction = 2 * 6400 * 11.72e-6

	// miscCounterFraction: PIPA and other counter traffic in powered flight.
	miscCounterFraction = 0.005

	// Costs, in AGC milliseconds.
	servicerBaseMs = 1320.0 // P63 braking-phase SERVICER per 2s cycle (~66%)
	servicerP64Ms  = 60.0   // landing-site redesignation processing (P64)
	servicerP66Ms  = 900.0  // rate-of-descent mode: much lighter

	// servicerLRMs: body->nav radar conversion once LR is locked. Eyles puts
	// the LR-lock cost at ~2% (margin 15%->13%) against the flight's ~13%
	// steal; the sim steals the theoretical 15.0%, so the conversion carries
	// the difference to keep the aggregate faithful: P63+LR+bug lands just
	// UNDER 100% demand — the flight's quiet pre-monitor knife edge (no
	// alarms for ~5 minutes with the theft active). See RESEARCH.md.
	servicerLRMs = 70.0

	dapPeriodMs     = 100.0 // digital autopilot interrupt cadence
	dapPoweredMs    = 12.0  // ~12% duty in powered flight
	dapAttHoldMs    = 5.0   // ATT HOLD / P66: eased burden
	t4PeriodMs      = 120.0 // T4RUPT_PROGRAM.agc: 120MS
	t4CostMs        = 0.96  // ~0.8% housekeeping
	t4DsptabExtraMs = 0.5   // one DSPTAB relay word per pass
	downPeriodMs    = 20.0  // DOWNRUPT: 50 telemetry words/s
	downCostMs      = 0.2   // ~1%
	gyroPeriodMs    = 1000.0
	gyroCostMs      = 7.0 // priority-21 gyro compensation

	// gyroPhaseMs offsets the 1Hz gyro tick from the 2s READACCS boundary,
	// the same anti-collision phasing the flight code used (SERVICER.agc
	// deliberately offsets READACCS ~70ms from the DAP rupt). Without it a
	// descent started exactly on a timer mark starves the almost-finished
	// SERVICER at every boundary — an artifact, not flight behavior.
	gyroPhaseMs = 137.0

	// Landing-radar reads (Cherry job table; SERVICER.agc). Jobs of a
	// millisecond or so that SLEEP while the radar gates data in — holding
	// their core set the whole time. LRH fires 50ms before each READACCS
	// ("50 MS PRIOR TO THE NEXT READACCS TASK") and its 80ms gate straddles
	// the cycle boundary; LRV takes 5 samples over ~500ms mid-cycle.
	lrhPhaseMs = 1950.0
	lrhHeadMs  = 1.0
	lrhSleepMs = 80.0
	lrhTailMs  = 1.0
	lrvPhaseMs = 1000.0
	lrvHeadMs  = 1.0
	lrvSleepMs = 500.0
	lrvTailMs  = 1.0

	monitorPeriodMs = 1000.0 // PINBALL: monitors update once per second
	// MONDO refresh: 30ms CPU (~3% at 1Hz; margin 13% -> ~10%), split
	// around a display-wait sleep (PINBALL: display users go to sleep
	// holding their core set; duration est.).
	monitorHeadMs  = 15.0
	monitorSleepMs = 250.0
	monitorTailMs  = 15.0

	// CHARIN keystroke job: 5ms CPU split around the DSPTAB echo wait
	// (PINBALL sleep semantics; duration est.). The sleep is why heavy DSKY
	// activity pressured core sets on Apollo 11.
	charinHeadMs  = 3.0
	charinSleepMs = 150.0
	charinTailMs  = 2.0

	// HIGATJOB (Cherry job table): at high gate, command the LR antenna to
	// position 2, then sleep — holding a VAC — until the position discrete
	// arrives (antenna slew time est. seconds).
	higatHeadMs  = 2.0
	higatSleepMs = 8000.0
	higatTailMs  = 2.0

	keyruptCostMs    = 0.1  // the interrupt itself
	readaccsCostMs   = 1.0  // "deliberately short" waitlist task
	rrReadCostMs     = 80.0 // 'ping the radar' one-shot burst
	restartCostMs    = 20.0 // BAILOUT -> ENEMA software restart overhead
	rereadacDelayMs  = 20.0 // phase-table rebuild lag before REREADAC
	dsptabPerKey     = 2    // relay words queued per keystroke
	dsptabPerMonitor = 3    // relay words queued per monitor refresh

	windowBuckets = 200  // accounting window: 200 x 10ms = one 2s cycle
	BucketMs      = 10.0 // history bucket size
	historySize   = 1200 // 12s of history for the timeline rows
)

// Priorities (FIXED_FIXED_CONSTANT_POOL / repo job table).
const (
	prioServicer = 20
	prioGyro     = 21
	prioCharin   = 30 // CHRPRIO OCT 30000
	prioMonitor  = 30
	prioRadar    = 32
)

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

// Phase is the DSKY program phase.
type Phase int

const (
	P00 Phase = iota // idle
	P63              // braking
	P64              // approach / visibility
	P66              // rate-of-descent (ATT HOLD entered)
)

func (p Phase) String() string {
	switch p {
	case P63:
		return "P63"
	case P64:
		return "P64"
	case P66:
		return "P66"
	default:
		return "P00"
	}
}

// Consumer identifies who received a slice of CPU time.
type Consumer int

const (
	CIdle Consumer = iota
	CServicer
	CReadAccs
	CDAP
	CT4Rupt
	CDownrupt
	CGyro
	CLRRead
	CMonitor
	CCharin
	CRRRead
	COther
	CKeyRupt
	CSteal
	CPipa
	CRestart
	numConsumers
)

func (c Consumer) String() string {
	switch c {
	case CIdle:
		return "IDLE"
	case CServicer:
		return "SERVICER"
	case CReadAccs:
		return "READACCS"
	case CDAP:
		return "DAP"
	case CT4Rupt:
		return "T4RUPT"
	case CDownrupt:
		return "DOWNLINK"
	case CGyro:
		return "GYRO"
	case CLRRead:
		return "LR READ"
	case CMonitor:
		return "MONITOR"
	case CCharin:
		return "CHARIN"
	case CRRRead:
		return "RR READ"
	case CKeyRupt:
		return "KEYRUPT"
	case CSteal:
		return "RR STEAL"
	case CPipa:
		return "PIPA CTR"
	case CRestart:
		return "RESTART"
	default:
		return "JOB"
	}
}

// SlotState reports one core set or VAC area. Stub marks a slot held by a
// superseded job copy — a newer copy of the same job exists, so this one has
// lost the equal-priority tie and starves while still holding its memory.
type SlotState struct {
	Busy  bool
	Owner string
	Prio  int
	Stub  bool
}

// Alarm is one executive-overflow program alarm.
type Alarm struct {
	Code      string // "1201" (no VAC areas) or "1202" (no core sets)
	AGCTimeMs float64
}

// Accounting decomposes the trailing 2-second window of AGC time.
type Accounting struct {
	JobsPct       float64
	InterruptsPct float64
	StealPct      float64
	RestartPct    float64
	IdlePct       float64
	DeficitPct    float64 // growth rate of outstanding, unfinished job work
}

// EventKind tags entries of the event log.
type EventKind int

const (
	EvCycleStart EventKind = iota
	EvAlarm
	EvRestart
	EvKey
	EvMonitorOn
	EvMonitorOff
	EvPhase
	EvBug
	EvPing
	EvLRLock
	EvLeak
	EvRecover
	EvHint
)

// Event is one entry of the event log.
type Event struct {
	AGCTimeMs float64
	Kind      EventKind
	Text      string
}

// Bucket is one 10ms slice of history: which consumers ran, and who dominated.
type Bucket struct {
	Mask     uint32
	Dominant Consumer
}

// DSKYState mirrors the display and keyboard unit.
type DSKYState struct {
	Verb, Noun string
	R1, R2, R3 string
	ProgLamp   bool
}

// ---------------------------------------------------------------------------
// Internal types
// ---------------------------------------------------------------------------

type job struct {
	name      string
	prio      int
	remaining float64 // CPU left in the current segment
	needsVAC  bool
	core      int
	vac       int
	seq       int // scheduling order; ties on priority favor the newest
	consumer  Consumer

	// Sleep segment (JOBSLEEP/JOBWAKE): after the head segment the job
	// sleeps for sleepMs — holding its core set and VAC — then wakes to run
	// tailMs more. Radar gates and display waits live here.
	sleepMs  float64
	tailMs   float64
	sleeping bool
	wakeAt   float64

	// superseded is set once a newer copy of the same job is scheduled.
	// It stays set even if the newer copies finish first (LIFO unwind), so
	// a late finish is always recognized as a recovered double-booking.
	superseded bool
}

type wtask struct {
	name string
	due  float64
	fire func(e *Engine)
}

type intrWork struct {
	c         Consumer
	remaining float64
}

// Engine is the simulation.
type Engine struct {
	t         float64 // AGC time, ms
	wallToAGC float64

	phase   Phase
	lrAcq   bool
	bug     bool
	monitor bool

	cores [NumCoreSets]*job
	vacs  [NumVACs]*job
	jobs  []*job
	cur   *job
	seq   int

	waitlist []wtask
	intrQ    []intrWork

	stealDebt float64
	pipaDebt  float64

	dapNext, t4Next, downNext float64
	gyroNext, monNext         float64

	dsptab int

	alarms   []Alarm
	failreg  []string
	progLamp bool
	restarts int
	events   []Event

	// repeat-throttle state for chatty event kinds (LEAK/RECOVERED): a
	// steady per-cycle chorus logs once; changes always log.
	throttleText map[EventKind]string
	throttleAt   map[EventKind]float64

	lastRecoverAt float64 // AGC time of the last stub recovery, log or not
	lastRestartAt float64 // AGC time of the last software restart
	compActy      bool    // last accounted step was non-idle (COMP ACTY lamp)

	runningJob string

	// DSKY entry state
	verbBuf, nounBuf string
	entering         byte // 'V', 'N' or 0
	progSel          bool // V37E seen: the next nn E selects program nn

	// accounting rings
	bucketUse   [historySize][numConsumers]float32 // ms per consumer per bucket
	bucketMask  [historySize]uint32
	bucketOut   [historySize]float32 // outstanding job work at bucket close
	bucketHead  int                  // next bucket index to write
	bucketCount int                  // closed buckets in the ring (caps at historySize)
	closedTotal int                  // closed buckets ever (monotonic)
	curUse      [numConsumers]float32
	curFill     float64
}

// New returns an idle engine at AGC time zero.
func New() *Engine {
	e := &Engine{
		wallToAGC:     DefaultWallToAGC,
		throttleText:  map[EventKind]string{},
		throttleAt:    map[EventKind]float64{},
		lastRecoverAt: -1e18,
		lastRestartAt: -1e18,
	}
	e.t4Next = t4PeriodMs
	e.downNext = downPeriodMs
	e.dapNext = dapPeriodMs
	e.gyroNext = gyroPeriodMs + gyroPhaseMs
	e.monNext = monitorPeriodMs
	return e
}

// ---------------------------------------------------------------------------
// Time
// ---------------------------------------------------------------------------

// SetWallToAGC sets how much AGC time one wall millisecond represents.
func (e *Engine) SetWallToAGC(r float64) {
	if r > 0 {
		e.wallToAGC = r
	}
}

// WallToAGC reports the current time scale.
func (e *Engine) WallToAGC() float64 { return e.wallToAGC }

// AdvanceWall advances the simulation by wall-clock milliseconds.
func (e *Engine) AdvanceWall(wallMs float64) {
	if wallMs <= 0 {
		return
	}
	e.AdvanceAGC(wallMs * e.wallToAGC)
}

// AdvanceAGC advances the simulation by AGC milliseconds.
func (e *Engine) AdvanceAGC(agcMs float64) {
	if agcMs <= 0 {
		return
	}
	steps := int(agcMs/stepMs + 0.5)
	for i := 0; i < steps; i++ {
		e.step()
	}
}

// AGCTimeMs is the AGC clock.
func (e *Engine) AGCTimeMs() float64 { return e.t }

// ---------------------------------------------------------------------------
// The heart: one 100us step
// ---------------------------------------------------------------------------

func (e *Engine) step() {
	e.t += stepMs

	// Hardware timers fire punctually no matter what (T3RUPT et al.).
	e.fireWaitlist()
	e.fireRecurrences()
	e.wakeSleepers()

	// 1. Counter increments (PINC/MINC cycle stealing) pause everything.
	if e.bug {
		e.stealDebt += RadarBugStealFraction * stepMs
		if e.stealDebt >= stepMs {
			e.stealDebt -= stepMs
			e.account(CSteal)
			return
		}
	}
	if e.phase != P00 {
		e.pipaDebt += miscCounterFraction * stepMs
		if e.pipaDebt >= stepMs {
			e.pipaDebt -= stepMs
			e.account(CPipa)
			return
		}
	}

	// 2. Interrupt-level work (T4RUPT, DAP, DOWNRUPT, KEYRUPT, tasks, restart).
	if len(e.intrQ) > 0 {
		w := &e.intrQ[0]
		w.remaining -= stepMs
		c := w.c
		if w.remaining <= 1e-9 {
			e.intrQ = e.intrQ[1:]
		}
		e.account(c)
		return
	}

	// 3. The highest-priority ready job.
	if e.cur == nil {
		e.cur = e.selectJob()
	}
	if j := e.cur; j != nil {
		j.remaining -= stepMs
		e.runningJob = j.name
		e.account(j.consumer)
		if j.remaining <= 1e-9 {
			if j.sleepMs > 0 {
				// JOBSLEEP: head done; hold the memory, give up the CPU.
				j.sleeping = true
				j.wakeAt = e.t + j.sleepMs
				j.remaining = j.tailMs
				j.sleepMs, j.tailMs = 0, 0
				e.cur = e.selectJob()
			} else {
				e.endOfJob(j)
			}
		}
		return
	}

	// 4. DUMMYJOB: free compute.
	e.runningJob = ""
	e.account(CIdle)
}

func (e *Engine) fireWaitlist() {
	for i := 0; i < len(e.waitlist); {
		if e.waitlist[i].due <= e.t {
			task := e.waitlist[i]
			e.waitlist = append(e.waitlist[:i], e.waitlist[i+1:]...)
			task.fire(e)
			i = 0 // fire may mutate the list
			continue
		}
		i++
	}
}

func (e *Engine) fireRecurrences() {
	if e.t >= e.t4Next {
		e.t4Next += t4PeriodMs
		cost := t4CostMs
		if e.dsptab > 0 {
			e.dsptab--
			cost += t4DsptabExtraMs
		}
		e.intrQ = append(e.intrQ, intrWork{CT4Rupt, cost})
	}
	if e.t >= e.downNext {
		e.downNext += downPeriodMs
		e.intrQ = append(e.intrQ, intrWork{CDownrupt, downCostMs})
	}
	if e.t >= e.dapNext {
		e.dapNext += dapPeriodMs
		if e.phase == P63 || e.phase == P64 {
			e.intrQ = append(e.intrQ, intrWork{CDAP, dapPoweredMs})
		} else if e.phase == P66 {
			e.intrQ = append(e.intrQ, intrWork{CDAP, dapAttHoldMs})
		}
	}
	if e.t >= e.gyroNext {
		e.gyroNext += gyroPeriodMs
		if e.phase != P00 {
			e.scheduleJobInternal("GYRO COMP", prioGyro, gyroCostMs, false, CGyro)
		}
	}
	if e.t >= e.monNext {
		e.monNext += monitorPeriodMs
		if e.monitor {
			e.dsptab += dsptabPerMonitor
			e.scheduleSegJob("MONITOR", prioMonitor,
				monitorHeadMs, monitorSleepMs, monitorTailMs, false, CMonitor)
		}
	}
}

// wakeSleepers is JOBWAKE: sleeping jobs whose gate has passed become
// runnable again and compete by priority (like a fresh schedule, a woken job
// only preempts a strictly lower priority).
func (e *Engine) wakeSleepers() {
	for _, j := range e.jobs {
		if j.sleeping && e.t >= j.wakeAt {
			j.sleeping = false
			if e.cur == nil || j.prio > e.cur.prio {
				e.cur = j
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Executive: allocation, selection, release, BAILOUT
// ---------------------------------------------------------------------------

func consumerFor(name string) Consumer {
	switch name {
	case "SERVICER":
		return CServicer
	case "CHARIN":
		return CCharin
	case "MONITOR":
		return CMonitor
	case "RR READ":
		return CRRRead
	case "LR READ", "LRHJOB", "LRVJOB", "HIGATJOB":
		return CLRRead
	case "GYRO COMP":
		return CGyro
	default:
		return COther
	}
}

// ScheduleJob enters a job request, exactly like NOVAC/FINDVAC. It returns
// false when the Executive had to BAILOUT with 1201 or 1202 instead.
func (e *Engine) ScheduleJob(name string, prio int, costMs float64, needsVAC bool) bool {
	return e.scheduleJobInternal(name, prio, costMs, needsVAC, consumerFor(name))
}

func (e *Engine) scheduleJobInternal(name string, prio int, costMs float64, needsVAC bool, c Consumer) bool {
	return e.scheduleSegJob(name, prio, costMs, 0, 0, needsVAC, c)
}

// scheduleSegJob schedules a job whose head segment is followed by a
// resource-holding sleep and a tail segment (radar gate / display wait).
func (e *Engine) scheduleSegJob(name string, prio int, headMs, sleepMs, tailMs float64, needsVAC bool, c Consumer) bool {
	j := &job{name: name, prio: prio, remaining: headMs, sleepMs: sleepMs, tailMs: tailMs,
		needsVAC: needsVAC, core: -1, vac: -1, consumer: c}

	// FINDVAC scans the five VAC use-words first (EXECUTIVE.agc FINDVAC2).
	if needsVAC {
		v := -1
		for i := 0; i < NumVACs; i++ {
			if e.vacs[i] == nil {
				v = i
				break
			}
		}
		if v < 0 {
			e.bailout("1201")
			return false
		}
		j.vac = v
	}
	// Then the eight core sets (NOVAC2/NOVAC3/NEXTCORE).
	cs := -1
	for i := 0; i < NumCoreSets; i++ {
		if e.cores[i] == nil {
			cs = i
			break
		}
	}
	if cs < 0 {
		e.bailout("1202")
		return false
	}
	j.core = cs
	e.cores[cs] = j
	if j.vac >= 0 {
		e.vacs[j.vac] = j
	}
	e.seq++
	j.seq = e.seq
	for _, k := range e.jobs {
		if k.name == j.name {
			k.superseded = true
		}
	}
	e.jobs = append(e.jobs, j)

	// SETLOC rule: a new job preempts only a strictly lower priority.
	if e.cur == nil || j.prio > e.cur.prio {
		e.cur = j
	}
	return true
}

// selectJob is EJSCAN: highest active priority among runnable (non-sleeping)
// jobs. Ties favor the most recently scheduled copy — which is how old
// SERVICER stubs starved on Apollo 11 and leaked one core set + VAC pair per
// overloaded cycle (see RESEARCH.md).
func (e *Engine) selectJob() *job {
	var best *job
	for _, j := range e.jobs {
		if j.sleeping {
			continue
		}
		if best == nil || j.prio > best.prio || (j.prio == best.prio && j.seq > best.seq) {
			best = j
		}
	}
	return best
}

func (e *Engine) endOfJob(j *job) {
	// A superseded copy that still got CPU (equal priority never preempts a
	// running job, and the LIFO unwind can hand old copies the CPU back)
	// reaches ENDOFJOB late and gives its memory back: the double-booking
	// healed itself. Narrate it, or the earlier LEAK event looks like a
	// stub silently vanishing.
	if j.name == "SERVICER" && j.superseded {
		e.lastRecoverAt = e.t
		e.logThrottled(EvRecover,
			"RECOVERED: superseded SERVICER finished late — core set + VAC freed",
			CyclePeriodMs+200)
	}
	if j.core >= 0 {
		e.cores[j.core] = nil
	}
	if j.vac >= 0 {
		e.vacs[j.vac] = nil
	}
	for i, x := range e.jobs {
		if x == j {
			e.jobs = append(e.jobs[:i], e.jobs[i+1:]...)
			break
		}
	}
	if e.cur == j {
		e.cur = e.selectJob()
	}
}

// bailout is BAILOUT1 -> FAILREG -> PROGLARM -> WHIMPER -> ENEMA.
func (e *Engine) bailout(code string) {
	e.alarms = append(e.alarms, Alarm{Code: code, AGCTimeMs: e.t})
	if len(e.failreg) < 3 {
		e.failreg = append(e.failreg, code)
	} else {
		e.failreg[2] = code
	}
	e.progLamp = true
	e.restarts++
	e.lastRestartAt = e.t
	what := "NO CORE SETS"
	if code == "1201" {
		what = "NO VAC AREAS"
	}
	e.logEvent(EvAlarm, fmt.Sprintf("PROG ALARM %s — %s", code, what))
	e.logEvent(EvRestart, "BAILOUT: software restart (ENEMA)")

	// The restart wipes queues and frees every resource.
	e.jobs = nil
	e.cur = nil
	e.runningJob = ""
	for i := range e.cores {
		e.cores[i] = nil
	}
	for i := range e.vacs {
		e.vacs[i] = nil
	}
	e.waitlist = nil
	e.intrQ = []intrWork{{CRestart, restartCostMs}}
	e.dsptab = 0
	if e.monitor {
		e.monitor = false
		e.logEvent(EvMonitorOff, "restart drops V16N68 (not restart-protected)")
	}
	// The restart flushed the stubs and shed the sheddable load; a P63
	// machine is back on its margin. Tell the operator how the flight
	// escalated from here, or the sudden quiet reads as a bug.
	if e.phase == P63 {
		e.logEvent(EvHint, "margin restored — press n (V16N68) or 6 (P64) to overload again")
	}
	e.throttleText = map[EventKind]string{}
	e.throttleAt = map[EventKind]float64{}
	e.verbBuf, e.nounBuf, e.entering, e.progSel = "", "", 0, false

	// Phase tables (5.4SPOT): rebuild one REREADAC task + one SERVICER.
	if e.phase != P00 {
		// Aldrin: the display "switched back to Verb 06 Noun 63."
		e.verbBuf, e.nounBuf = "06", "63"
		due := e.t + rereadacDelayMs
		e.waitlist = append(e.waitlist, wtask{"REREADAC", due, func(e *Engine) {
			e.intrQ = append(e.intrQ, intrWork{CReadAccs, readaccsCostMs})
			if e.scheduleJobInternal("SERVICER", prioServicer, e.servicerCost(), true, CServicer) {
				e.armReadaccs(e.t + CyclePeriodMs)
			}
		}})
	}
}

// ---------------------------------------------------------------------------
// Waitlist: READACCS every 2.000 seconds
// ---------------------------------------------------------------------------

func (e *Engine) armReadaccs(due float64) {
	e.waitlist = append(e.waitlist, wtask{"READACCS", due, func(e *Engine) {
		e.logEvent(EvCycleStart, "READACCS: read PIPAs, schedule SERVICER")
		e.intrQ = append(e.intrQ, intrWork{CReadAccs, readaccsCostMs})
		// GOREADAX: CA 2SECS / TC VARDELAY. If the allocation instead ended
		// in BAILOUT, the restart wiped this chain; REREADAC now owns the
		// rebuild, so re-arming here would double the cycle demand.
		if e.scheduleJobInternal("SERVICER", prioServicer, e.servicerCost(), true, CServicer) {
			e.armReadaccs(e.t + CyclePeriodMs)
			e.armLandingRadarReads()
			// The copy that lost the equal-priority tie is now abandoned —
			// this line IS the memory leak of July 20, 1969.
			if n := e.StubCount(); n > 0 {
				e.logThrottled(EvLeak, fmt.Sprintf(
					"LEAK: %d unfinished SERVICER stub(s) hold %d core set(s) + %d VAC(s)", n, n, n),
					CyclePeriodMs+200)
			}
		}
	}})
}

// armLandingRadarReads schedules this cycle's radar read jobs (Cherry's job
// table): LRVJOB mid-cycle (5 samples, ~500ms gate), and LRHTASK -> LRHJOB
// 50ms before the next READACCS so its 80ms gate straddles the boundary.
// Both are millisecond jobs that sleep holding a core set.
func (e *Engine) armLandingRadarReads() {
	if !e.lrAcq || e.phase == P00 {
		return
	}
	e.waitlist = append(e.waitlist, wtask{"LRVTASK", e.t + lrvPhaseMs, func(e *Engine) {
		if e.lrAcq && e.phase != P00 {
			e.scheduleSegJob("LRVJOB", prioRadar, lrvHeadMs, lrvSleepMs, lrvTailMs, false, CLRRead)
		}
	}})
	e.waitlist = append(e.waitlist, wtask{"LRHTASK", e.t + lrhPhaseMs, func(e *Engine) {
		if e.lrAcq && e.phase != P00 {
			e.scheduleSegJob("LRHJOB", prioRadar, lrhHeadMs, lrhSleepMs, lrhTailMs, false, CLRRead)
		}
	}})
}

func (e *Engine) servicerCost() float64 {
	switch e.phase {
	case P64:
		c := servicerBaseMs + servicerP64Ms
		if e.lrAcq {
			c += servicerLRMs
		}
		return c
	case P66:
		return servicerP66Ms
	default:
		c := servicerBaseMs
		if e.lrAcq {
			c += servicerLRMs
		}
		return c
	}
}

// ---------------------------------------------------------------------------
// Flight controls
// ---------------------------------------------------------------------------

// StartDescent enters P63: the 2-second READACCS/SERVICER cycle begins.
func (e *Engine) StartDescent() {
	if e.phase != P00 {
		return
	}
	e.phase = P63
	e.logEvent(EvPhase, "P63: powered descent — READACCS every 2.000s")
	e.armReadaccs(e.t)
}

// AcquireLandingRadar marks landing-radar "data good".
func (e *Engine) AcquireLandingRadar() {
	if e.lrAcq {
		return
	}
	e.lrAcq = true
	e.logEvent(EvLRLock, "landing radar data good — conversion load added")
}

// LandingRadarAcquired reports the LR lock state.
func (e *Engine) LandingRadarAcquired() bool { return e.lrAcq }

// EnterP64 begins the approach phase (redesignation logic; restart cannot
// shed this load). HIGATJOB (Cherry's table) commands the LR antenna to
// position 2 and sleeps on a VAC until the position discrete arrives — the
// VAC pressure behind the flight's 1201 at 102:42:17.
func (e *Engine) EnterP64() {
	if e.phase != P63 {
		return
	}
	e.phase = P64
	e.logEvent(EvPhase, "P64: high gate — redesignation load (protected)")
	e.scheduleSegJob("HIGATJOB", prioRadar, higatHeadMs, higatSleepMs, higatTailMs, true, CLRRead)
}

// AttHold is Armstrong's move: AUTO -> ATT HOLD, then P66.
func (e *Engine) AttHold() {
	if e.phase != P63 && e.phase != P64 {
		return
	}
	e.phase = P66
	e.logEvent(EvPhase, "ATT HOLD / P66: computational burden shed")
}

// Reset returns to the idle state, preserving the time scale.
func (e *Engine) Reset() {
	scale := e.wallToAGC
	*e = *New()
	e.wallToAGC = scale
}

// SetRadarBug enables or disables the rendezvous-radar counter theft.
func (e *Engine) SetRadarBug(on bool) {
	if e.bug == on {
		return
	}
	e.bug = on
	if on {
		e.logEvent(EvBug, "RR mode AUTO/SLEW: ECDU counters steal ~15% (TLOSS)")
	} else {
		e.logEvent(EvBug, "RR mode LGC: counter theft ends")
	}
}

// RadarBug reports whether the theft is active.
func (e *Engine) RadarBug() bool { return e.bug }

// PingRadar schedules a one-shot priority-32 rendezvous-radar read burst.
func (e *Engine) PingRadar() {
	e.logEvent(EvPing, "RR READ: one-shot radar read burst (prio 32)")
	e.ScheduleJob("RR READ", prioRadar, rrReadCostMs, false)
}

// ---------------------------------------------------------------------------
// DSKY
// ---------------------------------------------------------------------------

// PressKey is one DSKY keystroke: KEYRUPT1 fires, CHARIN (priority 30) is
// scheduled, and DSPTAB display updates are queued for T4RUPT to drain.
// Keys: 'V', 'N', 'E' (ENTR), 'C' (CLR), '0'..'9'.
func (e *Engine) PressKey(k byte) {
	e.logEvent(EvKey, "KEYRUPT: "+string(k))
	e.intrQ = append(e.intrQ, intrWork{CKeyRupt, keyruptCostMs})
	e.dsptab += dsptabPerKey
	if !e.scheduleSegJob("CHARIN", prioCharin, charinHeadMs, charinSleepMs, charinTailMs, false, CCharin) {
		return // the keystroke's job died in the overflow it caused
	}
	switch {
	case k == 'V':
		e.entering = 'V'
		e.verbBuf = ""
		e.progSel = false
	case k == 'N':
		e.entering = 'N'
		e.nounBuf = ""
		e.progSel = false
	case k == 'C':
		e.entering = 0
		e.progSel = false
	case k == 'E':
		e.entering = 0
		if e.progSel {
			// V37E nnE — program change (Eyles: "The crew keyed in Verb 37
			// Noun 63 to select P63").
			e.progSel = false
			if e.nounBuf == "63" && e.phase == P00 {
				e.StartDescent()
				// Landing radar "data good" arrives on its own during the
				// braking phase (scripted flavor timing, cf. flight
				// PDI+262s; compressed so one toggle tells the story).
				e.waitlist = append(e.waitlist, wtask{"LRDATA", e.t + 6000, func(e *Engine) {
					if e.phase != P00 {
						e.AcquireLandingRadar()
					}
				}})
			}
			break
		}
		if e.verbBuf == "37" {
			e.progSel = true
			e.entering = 'N'
			e.nounBuf = ""
			break
		}
		if e.verbBuf == "16" && e.nounBuf == "68" {
			if !e.monitor {
				e.monitor = true
				e.monNext = e.t + monitorPeriodMs
				e.logEvent(EvMonitorOn, "V16N68: DELTAH monitor up (1Hz refresh)")
			}
		} else if e.verbBuf == "34" {
			if e.monitor {
				e.monitor = false
				e.logEvent(EvMonitorOff, "V34: monitor terminated")
			}
		}
	case k >= '0' && k <= '9':
		switch e.entering {
		case 'V':
			if len(e.verbBuf) < 2 {
				e.verbBuf += string(k)
			}
		case 'N':
			if len(e.nounBuf) < 2 {
				e.nounBuf += string(k)
			}
		}
	}
}

// MonitorActive reports whether the V16N68 monitor is refreshing.
func (e *Engine) MonitorActive() bool { return e.monitor }

// PendingDSPTAB is the number of queued display relay words.
func (e *Engine) PendingDSPTAB() int { return e.dsptab }

// DSKY returns the display state.
func (e *Engine) DSKY() DSKYState {
	d := DSKYState{Verb: e.verbBuf, Noun: e.nounBuf, ProgLamp: e.progLamp}
	if e.phase != P00 {
		// Scripted flavor values (educational, not dynamics): P63 starts at
		// 49,971 ft sinking toward high gate at 7,400 ft.
		alt := 49971.0 - 84.0*(e.t/1000.0)
		if alt < 0 {
			alt = 0
		}
		d.R1 = fmt.Sprintf("%+06.0f", 5559.7-6.0*(e.t/1000.0))
		d.R2 = fmt.Sprintf("%+06.0f", -84.0)
		d.R3 = fmt.Sprintf("%+06.0f", alt)
		if e.monitor {
			d.R3 = "-02900" // DELTAH as Aldrin saw it
		}
	}
	return d
}

// ---------------------------------------------------------------------------
// Introspection
// ---------------------------------------------------------------------------

// isStub reports whether a newer live copy of the same job exists. With the
// newest-wins tie-break, such a superseded copy never runs again under
// sustained overload — it just sits on its core set (and VAC) forever.
func (e *Engine) isStub(j *job) bool {
	for _, k := range e.jobs {
		if k != j && k.name == j.name && k.seq > j.seq {
			return true
		}
	}
	return false
}

// StubCount is the number of superseded SERVICER copies still holding memory.
func (e *Engine) StubCount() int {
	n := 0
	for _, j := range e.jobs {
		if j.name == "SERVICER" && e.isStub(j) {
			n++
		}
	}
	return n
}

// CoreSets reports the eight core sets.
func (e *Engine) CoreSets() [NumCoreSets]SlotState {
	var out [NumCoreSets]SlotState
	for i, j := range e.cores {
		if j != nil {
			out[i] = SlotState{Busy: true, Owner: j.name, Prio: j.prio, Stub: e.isStub(j)}
		}
	}
	return out
}

// VACs reports the five vector accumulator areas.
func (e *Engine) VACs() [NumVACs]SlotState {
	var out [NumVACs]SlotState
	for i, j := range e.vacs {
		if j != nil {
			out[i] = SlotState{Busy: true, Owner: j.name, Prio: j.prio, Stub: e.isStub(j)}
		}
	}
	return out
}

// Alarms lists every program alarm so far.
func (e *Engine) Alarms() []Alarm { return e.alarms }

// FailReg returns the up-to-three stored alarm codes (V05 N09).
func (e *Engine) FailReg() []string { return e.failreg }

// ProgLamp reports the PROG lamp.
func (e *Engine) ProgLamp() bool { return e.progLamp }

// RestartCount is the number of software restarts (REDOCTR).
func (e *Engine) RestartCount() int { return e.restarts }

// Phase is the current program phase.
func (e *Engine) Phase() Phase { return e.phase }

// RunningJob names the job-level work most recently given CPU time.
func (e *Engine) RunningJob() string { return e.runningJob }

// ServicerCopies counts live SERVICER incarnations (stubs included).
func (e *Engine) ServicerCopies() int {
	n := 0
	for _, j := range e.jobs {
		if j.name == "SERVICER" {
			n++
		}
	}
	return n
}

// Events returns the event log.
func (e *Engine) Events() []Event { return e.events }

// CycleElapsedMs is the AGC time since the last READACCS fired.
func (e *Engine) CycleElapsedMs() float64 {
	last := -1.0
	for i := len(e.events) - 1; i >= 0; i-- {
		if e.events[i].Kind == EvCycleStart {
			last = e.events[i].AGCTimeMs
			break
		}
	}
	if last < 0 {
		return 0
	}
	return e.t - last
}

// CycleCount is the number of READACCS dispatches so far.
func (e *Engine) CycleCount() int {
	n := 0
	for _, ev := range e.events {
		if ev.Kind == EvCycleStart {
			n++
		}
	}
	return n
}

func (e *Engine) logEvent(k EventKind, text string) {
	e.events = append(e.events, Event{AGCTimeMs: e.t, Kind: k, Text: text})
}

// logThrottled suppresses a repeat of the same kind+text within windowMs.
// Every attempt refreshes the window, so an unchanged per-cycle chorus logs
// exactly once; any change of text (escalation) or a quiet gap logs again.
func (e *Engine) logThrottled(k EventKind, text string, windowMs float64) {
	repeat := e.throttleText[k] == text && e.t-e.throttleAt[k] <= windowMs
	e.throttleText[k] = text
	e.throttleAt[k] = e.t
	if !repeat {
		e.logEvent(k, text)
	}
}

// RecoveredRecently reports whether a superseded SERVICER finished (and
// freed its pair) within the trailing window — logged or throttled.
func (e *Engine) RecoveredRecently(windowMs float64) bool {
	return e.t-e.lastRecoverAt <= windowMs
}

// RestartRecently reports whether a software restart happened within the
// trailing window — drives the DSKY RESTART lamp.
func (e *Engine) RestartRecently(windowMs float64) bool {
	return e.t-e.lastRestartAt <= windowMs
}

// CompActy reports whether the last accounted step did real work — the
// DSKY COMP ACTY lamp.
func (e *Engine) CompActy() bool { return e.compActy }

// KnifeEdge reports the flight's quiet pre-monitor regime: the theft is
// active and has consumed the entire margin (free compute pinned near zero)
// but nothing has overrun yet — one more straw (the monitor, P64) breaks it.
func (e *Engine) KnifeEdge() bool {
	if !e.bug || e.phase == P00 || e.StubCount() > 0 {
		return false
	}
	return e.Accounting().IdlePct < 2
}

// ---------------------------------------------------------------------------
// Accounting
// ---------------------------------------------------------------------------

func (e *Engine) outstanding() float64 {
	total := 0.0
	for _, j := range e.jobs {
		total += j.remaining
	}
	return total
}

func (e *Engine) account(c Consumer) {
	e.compActy = c != CIdle
	e.curUse[c] += stepMs
	e.curFill += stepMs
	if e.curFill+1e-9 >= BucketMs {
		mask := uint32(0)
		dominant, best := CIdle, float32(-1)
		for ci := Consumer(0); ci < numConsumers; ci++ {
			if e.curUse[ci] > 0 {
				mask |= 1 << uint(ci)
				if e.curUse[ci] > best {
					best = e.curUse[ci]
					dominant = ci
				}
			}
		}
		e.bucketUse[e.bucketHead] = e.curUse
		e.bucketMask[e.bucketHead] = mask
		e.bucketOut[e.bucketHead] = float32(e.outstanding())
		e.bucketHead = (e.bucketHead + 1) % historySize
		if e.bucketCount < historySize {
			e.bucketCount++
		}
		e.closedTotal++
		_ = dominant
		e.curUse = [numConsumers]float32{}
		e.curFill = 0
	}
}

// BucketsClosed is the monotonic count of history buckets closed since boot.
// The UI anchors its 2-buckets-per-cell pairing to this count's parity so
// cells never re-pair (and flicker) between frames.
func (e *Engine) BucketsClosed() int { return e.closedTotal }

// History returns the most recent n closed buckets, oldest first.
func (e *Engine) History(n int) []Bucket {
	if n > e.bucketCount {
		n = e.bucketCount
	}
	out := make([]Bucket, 0, n)
	for i := n; i >= 1; i-- {
		idx := (e.bucketHead - i + historySize) % historySize
		use := e.bucketUse[idx]
		dominant, best := CIdle, float32(-1)
		for ci := Consumer(0); ci < numConsumers; ci++ {
			if use[ci] > best {
				best = use[ci]
				dominant = ci
			}
		}
		out = append(out, Bucket{Mask: e.bucketMask[idx], Dominant: dominant})
	}
	return out
}

func (e *Engine) windowUse() ([numConsumers]float64, float64) {
	n := windowBuckets
	if n > e.bucketCount {
		n = e.bucketCount
	}
	var sums [numConsumers]float64
	total := 0.0
	for i := 1; i <= n; i++ {
		idx := (e.bucketHead - i + historySize) % historySize
		for ci := Consumer(0); ci < numConsumers; ci++ {
			sums[ci] += float64(e.bucketUse[idx][ci])
			total += float64(e.bucketUse[idx][ci])
		}
	}
	return sums, total
}

// FreeComputePercent is the idle share of the trailing 2-second window.
func (e *Engine) FreeComputePercent() float64 {
	return e.Accounting().IdlePct
}

// Accounting decomposes the trailing 2-second window.
func (e *Engine) Accounting() Accounting {
	sums, total := e.windowUse()
	if total <= 0 {
		return Accounting{IdlePct: 100}
	}
	pct := func(cs ...Consumer) float64 {
		s := 0.0
		for _, c := range cs {
			s += sums[c]
		}
		return s / total * 100
	}
	a := Accounting{
		JobsPct:       pct(CServicer, CGyro, CLRRead, CMonitor, CCharin, CRRRead, COther),
		InterruptsPct: pct(CReadAccs, CDAP, CT4Rupt, CDownrupt, CKeyRupt),
		StealPct:      pct(CSteal, CPipa),
		RestartPct:    pct(CRestart),
		IdlePct:       pct(CIdle),
	}
	// Deficit: growth of outstanding, unfinished job work over exactly one
	// full 2-second cycle, so the SERVICER sawtooth compares in phase.
	if e.bucketCount >= windowBuckets+1 {
		newest := (e.bucketHead - 1 + historySize) % historySize
		oldest := (e.bucketHead - 1 - windowBuckets + historySize) % historySize
		growth := float64(e.bucketOut[newest] - e.bucketOut[oldest])
		if growth > 0 {
			a.DeficitPct = growth / (float64(windowBuckets) * BucketMs) * 100
		}
	}
	return a
}
