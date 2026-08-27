package msim

import "sort"

// activity is CPU work in interrupt context (a RUPT service or a waitlist
// task's cost). Activities preempt the job layer mid-instruction and are
// consumed FIFO in arrival order.
type activity struct {
	name      string
	remaining Nanos
}

// cadence is a recurring hardware interrupt.
type cadence struct {
	name   string
	period Nanos
	cost   Nanos
	next   Nanos
	fires  []Nanos
}

// wtask is one waitlist entry (T3RUPT dispatch).
type wtask struct {
	at   Nanos
	seq  int
	name string
	cost Nanos
	fire func(*Engine)
}

// Engine is the single-CPU machine. Time advances one millisecond per tick;
// within a tick the order is: waitlist/interrupt fires (timestamped at the
// tick boundary), the RR theft skim, interrupt-context CPU, then the job
// layer under the Executive.
type Engine struct {
	cfg Config

	now     Nanos // current tick start
	subTick Nanos // consumed inside the current tick

	theftNs    Nanos
	softwareNs Nanos
	idleNs     Nanos

	cadences []*cadence
	tasks    []*wtask
	hardware []*wtask // crew/radar events: survive a software restart
	taskSeq  int
	taskGen  int // bumped by the restart flush: due fires from the old waitlist stop
	active   []activity

	exec *Executive

	events   []Event
	samples  []Sample
	alarms   []Alarm
	restarts []Nanos

	maxCores int
	maxVACs  int

	// monitorOn is the MONSAVE1 killer bit, inverted: true while a monitor
	// verb's MONREQ chain is allowed to re-enlist.
	monitorOn bool
}

// NewEngine builds a machine. With cfg.Interrupts the three hardware
// cadences are installed: DAP every 100 ms costing 12 ms (P-AXIS_RCS_
// AUTOPILOT.agc L41; phased +70 ms after the 2 s boundary per SERVICER.agc
// L95-L104), T4RUPT every 120 ms costing 0.96 ms (T4RUPT_PROGRAM.agc L144),
// DOWNRUPT every 20 ms costing 0.2 ms (DOWN_TELEMETRY_PROGRAM.agc L43).
func NewEngine(cfg Config) *Engine {
	e := &Engine{cfg: cfg}
	e.exec = newExecutive(e)
	if cfg.Interrupts {
		e.cadences = []*cadence{
			{name: "DAP", period: 100 * Millisecond, cost: 12 * Millisecond, next: 70 * Millisecond},
			{name: "T4RUPT", period: 120 * Millisecond, cost: 960 * Microsecond, next: 0},
			{name: "DOWNRUPT", period: 20 * Millisecond, cost: 200 * Microsecond, next: 0},
		}
	}
	return e
}

// Now is the exact machine time, including the position inside the tick.
func (e *Engine) Now() Nanos { return e.now + e.subTick }

// ScheduleTask enters a waitlist task: at time `at` the callback fires
// (allocations, re-arms) and `cost` nanoseconds of T3RUPT-context CPU are
// consumed. The dispatch is hardware-punctual: load never delays it.
func (e *Engine) ScheduleTask(at Nanos, name string, cost Nanos, fire func(*Engine)) {
	if at < e.now {
		at = e.now
	}
	e.taskSeq++
	e.tasks = append(e.tasks, &wtask{at: at, seq: e.taskSeq, name: name, cost: cost, fire: fire})
}

// ScheduleHardware enters an event that is NOT on the waitlist — a crew
// keystroke, a radar gate — and therefore survives a software restart.
func (e *Engine) ScheduleHardware(at Nanos, name string, cost Nanos, fire func(*Engine)) {
	if at < e.now {
		at = e.now
	}
	e.taskSeq++
	e.hardware = append(e.hardware, &wtask{at: at, seq: e.taskSeq, name: name, cost: cost, fire: fire})
}

// Spawn enters a job request (FINDVAC or NOVAC per spec.VAC). A nil return
// is success; otherwise the allocation failed with 1201/1202.
func (e *Engine) Spawn(spec JobSpec) *Alarm { return e.exec.allocate(spec) }

// Wake awakens the first dormant job with the given name (JOBWAKE). Waking a
// job that is not asleep, or does not exist, is a no-op.
func (e *Engine) Wake(name string) { e.exec.wakeByName(name) }

// tickNs is the machine's processing grain: 100 µs per tick. Dispatches,
// sleeps, and completions resolve on this lattice; millisecond quantities
// (the theft waveform, the samples) are preserved exactly across the ten
// slices of each millisecond.
const tickNs Nanos = 100 * Microsecond

const ticksPerMs = int(Millisecond / tickNs)

// tickSkim is the theft taken in slice k (0..9) of millisecond ms: the
// Bresenham split of the waveform value, telescoping to it exactly.
func tickSkim(ms, k Nanos) Nanos {
	v := theftAtMs(ms)
	return v*(k+1)/Nanos(ticksPerMs) - v*k/Nanos(ticksPerMs)
}

// RunMS advances the machine n milliseconds (n x 10 ticks).
func (e *Engine) RunMS(n int) {
	for i := 0; i < n*ticksPerMs; i++ {
		e.tick()
	}
}

func (e *Engine) tick() {
	e.subTick = 0

	// 1) hardware dispatches at the tick boundary — timestamped before any
	// consumption so punctuality is exact. Order: T5 (DAP), T3 (waitlist),
	// T4RUPT, DOWNRUPT — the RUPT priority order.
	for _, c := range e.cadences {
		if c.name == "DAP" {
			e.fireCadence(c)
		}
	}
	e.drainDue(&e.tasks, true)
	e.drainDue(&e.hardware, false)
	for _, c := range e.cadences {
		if c.name != "DAP" {
			e.fireCadence(c)
		}
	}

	// 2) the RR CDU counter theft: pure hardware, skims the front of the tick
	budget := tickNs
	kInMs := (e.now % Millisecond) / tickNs
	if e.cfg.RadarBug {
		skim := tickSkim(e.now/Millisecond, kInMs)
		e.theftNs += skim
		e.subTick = skim
		budget -= skim
	}

	// 3) interrupt-context CPU, then the job layer
	for budget > 0 {
		if len(e.active) > 0 {
			a := &e.active[0]
			c := a.remaining
			if c > budget {
				c = budget
			}
			a.remaining -= c
			budget -= c
			e.subTick += c
			e.softwareNs += c
			if a.remaining == 0 {
				e.active = e.active[1:]
			}
			continue
		}
		consumed := e.exec.runFor(budget)
		if consumed == 0 {
			e.idleNs += budget
			e.subTick += budget
			budget = 0
			break
		}
		budget -= consumed
	}

	// one occupancy sample per millisecond, at its final slice
	if kInMs == Nanos(ticksPerMs-1) {
		e.samples = append(e.samples, Sample{
			AtMs:    int(e.now / Millisecond),
			Cores:   e.CoresHeld(),
			VACs:    e.VACsHeld(),
			Running: e.RunningJob(),
		})
	}
	e.now += tickNs
	e.subTick = 0
}

func (e *Engine) fireCadence(c *cadence) {
	for c.next <= e.now {
		c.fires = append(c.fires, c.next)
		e.active = append(e.active, activity{name: c.name, remaining: c.cost})
		c.next += c.period
	}
}

// drainDue fires everything due at the current tick boundary, in (time,
// insertion) order. The pending list is settled BEFORE firing because a fire
// may re-arm into the same list. A fire may also trigger a BAILOUT restart,
// which flushes the waitlist: nothing further from the OLD waitlist may run
// (isWaitlist), while crew/radar hardware events are not the computer's to
// lose.
func (e *Engine) drainDue(list *[]*wtask, isWaitlist bool) {
	var due, rest []*wtask
	for _, t := range *list {
		if t.at <= e.now {
			due = append(due, t)
		} else {
			rest = append(rest, t)
		}
	}
	if len(due) == 0 {
		return
	}
	sort.SliceStable(due, func(i, j int) bool {
		if due[i].at != due[j].at {
			return due[i].at < due[j].at
		}
		return due[i].seq < due[j].seq
	})
	*list = rest
	gen := e.taskGen
	for _, t := range due {
		if isWaitlist && e.taskGen != gen {
			break // the restart flushed the waitlist these fires came from
		}
		if t.fire != nil {
			t.fire(e)
		}
		if t.cost > 0 {
			e.active = append(e.active, activity{name: t.name, remaining: t.cost})
		}
	}
}

// ---------- accounting ----------

func (e *Engine) TheftNs() Nanos      { return e.theftNs }
func (e *Engine) SoftwareBusyNs() Nanos { return e.softwareNs }
func (e *Engine) IdleNs() Nanos       { return e.idleNs }

// theftAtMs is the deterministic dither sweep: the loss depends on the RR
// shaft/trunnion angle geometry (worst near 90/270 deg), so it wanders
// between Grumman's measured floor and the theoretical ceiling, dwelling at
// the floor while the geometry sits far from the worst-case angles. A
// flat-bottomed triangle spanning seven guidance cycles, its floor dwell
// pinned over the keying cycles; period, phase and dwell are the run's
// free parameters (msim/RESEARCH.md).
func theftAtMs(ms Nanos) Nanos {
	const period = 14_000
	const dwell = 1_100 // ms at the floor on each side of the trough
	p := (ms + 12_500) % period
	if p > period/2 {
		p = period - p
	}
	if p < dwell {
		p = 0
	} else {
		p -= dwell
	}
	return TheftMinPerMs + p*(TheftMaxPerMs-TheftMinPerMs)/(period/2-dwell)
}

// TheftNsBefore is the theft accumulated by exact machine time t, mirroring
// the engine's per-tick Bresenham skim.
func (e *Engine) TheftNsBefore(t Nanos) Nanos {
	if !e.cfg.RadarBug {
		return 0
	}
	fullMs := t / Millisecond
	var total Nanos
	for ms := Nanos(0); ms < fullMs; ms++ {
		total += theftAtMs(ms)
	}
	rem := t % Millisecond
	k := rem / tickNs
	off := rem % tickNs
	// full slices of the current millisecond telescope to v*k/10
	total += theftAtMs(fullMs) * k / Nanos(ticksPerMs)
	if s := tickSkim(fullMs, k); off > s {
		off = s
	}
	return total + off
}

func (e *Engine) findCadence(name string) *cadence {
	for _, c := range e.cadences {
		if c.name == name {
			return c
		}
	}
	return nil
}

// InterruptFires is the total dispatch count for one cadence.
func (e *Engine) InterruptFires(name string) int {
	if c := e.findCadence(name); c != nil {
		return len(c.fires)
	}
	return 0
}

// InterruptFiresBefore counts dispatches at or before exact time t.
func (e *Engine) InterruptFiresBefore(name string, t Nanos) int {
	c := e.findCadence(name)
	if c == nil {
		return 0
	}
	n := sort.Search(len(c.fires), func(i int) bool { return c.fires[i] > t })
	return n
}

// ---------- pool / job introspection ----------

func (e *Engine) CoresHeld() int      { return e.exec.coresHeld() }
func (e *Engine) VACsHeld() int       { return e.exec.vacsHeld() }
func (e *Engine) RunningJob() string  { return e.exec.runningName() }
func (e *Engine) JobState(n string) JobState { return e.exec.jobState(n) }

func (e *Engine) Events() []Event { return e.events }
func (e *Engine) Samples() []Sample { return e.samples }
func (e *Engine) Alarms() []Alarm { return e.alarms }
func (e *Engine) RestartCount() int { return len(e.restarts) }
func (e *Engine) RestartAt(i int) Nanos { return e.restarts[i] }
func (e *Engine) MaxCores() int { return e.maxCores }
func (e *Engine) MaxVACs() int  { return e.maxVACs }

// Note records a scenario-level timeline event.
func (e *Engine) Note(kind, job, detail string) {
	e.events = append(e.events, Event{At: e.Now(), Kind: kind, Job: job, Detail: detail})
}

func (e *Engine) event(kind, job string, prio int, detail string) {
	e.events = append(e.events, Event{At: e.Now(), Kind: kind, Job: job, Prio: prio, Detail: detail})
}

func (e *Engine) trackPools() {
	if c := e.exec.coresHeld(); c > e.maxCores {
		e.maxCores = c
	}
	if v := e.exec.vacsHeld(); v > e.maxVACs {
		e.maxVACs = v
	}
}

// ---------- BAILOUT → software restart ----------

func (e *Engine) bailout(a Alarm) {
	e.alarms = append(e.alarms, a)
	e.event("alarm", a.Requester, 0, alarmDetail(a))
	if !e.cfg.AutoRestart {
		return
	}
	e.restarts = append(e.restarts, e.Now())
	e.event("restart", "", 0, "flush: all core sets, all VACs, the waitlist")
	// the flush: every PRIORITY word and VACnUSE freed, the waitlist wiped,
	// MONSAVE zeroed (monitor verbs are not restarted — Cherry pp. 5-6).
	// Crew/radar hardware events are not the computer's to lose.
	e.exec.flush()
	e.tasks = nil
	e.taskGen++
	e.monitorOn = false
	e.active = append([]activity(nil), activity{name: "RESTART", remaining: 20 * Millisecond})
	if e.cfg.RestartHook != nil {
		e.cfg.RestartHook(e)
	}
}

func alarmDetail(a Alarm) string {
	code := "1202 NO CORE SETS"
	if a.Code == 1201 {
		code = "1201 NO VAC AREAS"
	}
	return code
}
