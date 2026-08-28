package msim

// SlotView is one core set as the command screen sees it.
type SlotView struct {
	Slot  int
	Name  string
	Prio  int
	State JobState
	VAC   int // VAC-area index, -1 for a NOVAC holder
	IP    int // instructions completed
	Len   int // script length
}

// SlotViews reports all eight core sets; free slots carry an empty name.
func (e *Engine) SlotViews() []SlotView {
	out := make([]SlotView, len(e.exec.slots))
	for i, r := range e.exec.slots {
		out[i] = SlotView{Slot: i, VAC: -1}
		if r == nil {
			continue
		}
		st := JobWaiting
		switch {
		case r.dormant:
			st = JobSleeping
		case i == 0:
			st = JobRunning
		case r.started:
			st = JobParked
		}
		out[i] = SlotView{Slot: i, Name: r.name, Prio: r.prio, State: st,
			VAC: r.vacIdx, IP: r.ip, Len: len(r.script)}
	}
	return out
}

// LastRan is the last instant the named job consumed CPU, -1 if never.
func (e *Engine) LastRan(name string) Nanos {
	if at, ok := e.lastRan[name]; ok {
		return at
	}
	return -1
}

// TaskFires counts waitlist/hardware dispatches by task name.
func (e *Engine) TaskFires(name string) int { return e.taskFires[name] }

// LastFired is the last dispatch instant for a task or cadence, -1 if never.
func (e *Engine) LastFired(name string) Nanos {
	if at, ok := e.lastFired[name]; ok {
		return at
	}
	return -1
}

// SetRadarBug flips the RR CDU counter theft live (the RR mode switch).
func (e *Engine) SetRadarBug(on bool) { e.cfg.RadarBug = on }

// BusyNs is the cumulative CPU consumed under the given name — jobs via
// the runner, tasks and interrupts via their activity.
func (e *Engine) BusyNs(name string) Nanos { return e.busyByName[name] }

// SpawnCount counts Executive job entries by name.
func (e *Engine) SpawnCount(name string) int {
	n := 0
	for _, ev := range e.events {
		if ev.Kind == "spawn" && ev.Job == name {
			n++
		}
	}
	return n
}

// Live is the command screen's engine: the P63 descent machine with the
// three cockpit switches — DESCENT (the whole job chain), 1668 (the DELTAH
// monitor), RADAR STEAL (the theft) — togglable while it runs.
type Live struct {
	d *descent
}

// NewLive opens in the flight's P63 state: descent on, monitor off, radar
// steal on.
func NewLive() *Live {
	return &Live{d: newDescent(true)}
}

// Engine is the machine underneath.
func (l *Live) Engine() *Engine { return l.d.e }

// StepMS advances the machine n milliseconds.
func (l *Live) StepMS(n int) { l.d.e.RunMS(n) }

// DescentOn reports the P63 chain switch.
func (l *Live) DescentOn() bool { return !l.d.stopped }

// SetDescent flips the whole P63 job chain: off kills the READACCS re-arm
// (the chain drains); on resumes it on the 2 s PIPTIME lattice.
func (l *Live) SetDescent(on bool) {
	if on == !l.d.stopped {
		return
	}
	if !on {
		l.d.stopped = true
		l.d.e.CancelTasks("READACCS", "R10,R11", "LRHTASK", "LRVTASK")
		return
	}
	l.d.stopped = false
	e := l.d.e
	next := (e.Now()/(2*Second) + 1) * (2 * Second)
	l.d.armReadaccs(next)
	l.d.armR10R11(e.Now() + 250*Millisecond)
}

// MonitorOn reports the 1668 switch — the engine's MONSAVE state, so a
// software restart (which kills monitor verbs) drops the switch too.
func (l *Live) MonitorOn() bool { return l.d.e.monitorOn }

// SetMonitor keys V16N68 up (the ENTR keystroke plus the MONREQ chain) or
// drops it with KEY REL.
func (l *Live) SetMonitor(on bool) {
	e := l.d.e
	if on == e.monitorOn {
		return
	}
	if on {
		Keystroke(e, e.Now(), "ENTR (V16N68)")
		StartMonitor(e, e.Now())
		e.Note("note", "DSKY", "V16N68 ENTR — DELTAH monitor up")
		return
	}
	StopMonitor(e)
	e.Note("note", "DSKY", "KEY REL — monitor dropped")
}

// RadarOn reports the RR mode switch.
func (l *Live) RadarOn() bool { return l.d.e.cfg.RadarBug }

// SetRadar flips the RR CDU counter theft.
func (l *Live) SetRadar(on bool) { l.d.e.SetRadarBug(on) }

// ServicerOneShot reports the single-cycle rule.
func (l *Live) ServicerOneShot() bool { return l.d.oneshot }

// SetServicerOneShot lets only the run's FIRST READACCS enter a SERVICER —
// the graphs screen's portrait rule, so the one pass can stretch past its
// own boundary instead of being buried under the 2 s successor. Everything
// else (READACCS, the LR gates, R10/R11, the cadences) keeps its timer.
func (l *Live) SetServicerOneShot(on bool) { l.d.oneshot = on }

// ApproachOn reports the P64 approach switch.
func (l *Live) ApproachOn() bool { return l.d.approach }

// SetApproach flips the machine into early P64 — the approach phase. On:
// every SERVICER pass carries the REDESIG landing-site perturbations, the
// pass's display request becomes the flashing V06N64 (it takes a VAC and
// sleeps holding it until PRO), and HIGATASK enters HIGATJOB to park on its
// own VAC through the antenna slew. Off restores the P63 pass and cancels a
// not-yet-fired HIGATASK; a HIGATJOB already asleep keeps its memory — the
// antenna does not un-slew.
func (l *Live) SetApproach(on bool) {
	d := l.d
	if on == d.approach {
		return
	}
	d.approach = on
	d.loadServicer()
	if on {
		StartHigate(d.e, d.e.Now())
		return
	}
	d.e.CancelTasks("HIGATASK")
}
