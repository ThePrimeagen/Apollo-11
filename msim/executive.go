package msim

// jobRec is one Executive job: a core set holder, possibly a VAC holder.
type jobRec struct {
	name    string
	prio    int
	vacIdx  int // -1: NOVAC
	script  Script
	ip      int
	rem     Nanos // remaining cost of the current instruction
	started bool
	dormant bool // JOBSLEEP'd: negative priority, invisible to EJSCAN
}

// word is the full PRIORITY word the Executive actually compares: the
// priority bits PLUS the VAC-area address in the low bits (VACFOUND,
// EXECUTIVE.agc L170-L174: "STORE THE ADDRESS OF THE FIRST WORD OF IT IN
// THE LOW NINE BITS OF THE PRIORITY WORD"). SETLOC (L224-L234) and EJ1
// (L492-L499) compare these full words, so among equal priorities the
// higher-addressed VAC wins — the newest copy. NOVAC jobs carry the
// FAKEPRET constant: equal-priority NOVAC words are identical.
func (r *jobRec) word() int {
	return r.prio*8 + (r.vacIdx + 1)
}

// Executive mirrors EXECUTIVE.agc:
//   - 8 core sets; the RUNNING job always occupies slot 0 (CHANJOB swaps,
//     L251-L318)
//   - allocation scans slots upward and takes the first free (NOVAC2/NOVAC3/
//     NEXTCORE, L183-L191); FINDVAC scans the 5 VACs first (FINDVAC2,
//     L141-L161) and claims one BEFORE the core scan
//   - SETLOC posts NEWJOB only for a STRICTLY greater priority (L224-L234)
//   - EJSCAN walks slots 1..7 ascending; a tie keeps the earlier find (EJ1,
//     L492-L499) — equal-priority stubs parked in higher slots starve
//   - job switches happen at DANZIG (INTERPRETER.agc L74-L82): between
//     instructions, never inside one
type Executive struct {
	e       *Engine
	slots   [8]*jobRec
	vacs    [5]*jobRec
	pending int // slot index posted in NEWJOB; -1 = none
	known   map[string]bool
}

func newExecutive(e *Engine) *Executive {
	return &Executive{e: e, pending: -1, known: map[string]bool{}}
}

func (x *Executive) coresHeld() int {
	n := 0
	for _, s := range x.slots {
		if s != nil {
			n++
		}
	}
	return n
}

func (x *Executive) vacsHeld() int {
	n := 0
	for _, v := range x.vacs {
		if v != nil {
			n++
		}
	}
	return n
}

func (x *Executive) runningName() string {
	if r := x.slots[0]; r != nil && !r.dormant {
		return r.name
	}
	return ""
}

// allocate is FINDVAC/NOVAC. The pool snapshot for a failure is taken at
// request entry (before the transient VAC claim), which is what the alarm
// telemetry names.
func (x *Executive) allocate(spec JobSpec) *Alarm {
	snapCores, snapVACs := x.coresHeld(), x.vacsHeld()
	rec := &jobRec{name: spec.Name, prio: spec.Prio, vacIdx: -1, script: spec.Script}

	if spec.VAC {
		found := false
		for i := range x.vacs {
			if x.vacs[i] == nil {
				x.vacs[i] = rec // VACFOUND claims immediately (L170-L174)
				rec.vacIdx = i
				found = true
				break
			}
		}
		if !found {
			a := Alarm{Code: 1201, At: x.e.Now(), Requester: spec.Name,
				CoresHeld: snapCores, VACsHeld: snapVACs}
			x.e.bailout(a)
			return &a
		}
	}

	slot := -1
	for i := range x.slots {
		if x.slots[i] == nil {
			slot = i
			break
		}
	}
	if slot < 0 {
		// all eight PRIORITY words busy → 1202, even for a FINDVAC request
		// (its claimed VAC stays claimed; the restart frees it)
		a := Alarm{Code: 1202, At: x.e.Now(), Requester: spec.Name,
			CoresHeld: snapCores, VACsHeld: snapVACs}
		x.e.bailout(a)
		return &a
	}
	x.slots[slot] = rec
	x.known[spec.Name] = true
	x.e.event("spawn", spec.Name, spec.Prio, "")
	x.e.trackPools()

	// SETLOC: post NEWJOB only if strictly greater than the incumbent
	// (the pending NEWJOB if set, else the runner)
	x.postIfGreater(slot)
	return nil
}

// postIfGreater applies the SETLOC comparison for the job in the given slot.
func (x *Executive) postIfGreater(slot int) {
	rec := x.slots[slot]
	if rec == nil || rec.dormant {
		return
	}
	runner := x.slots[0]
	if runner == nil || runner.dormant {
		// nothing running: select now (the ADVAN dispatch path)
		x.select_()
		return
	}
	incumbent := runner
	if x.pending >= 0 && x.slots[x.pending] != nil {
		incumbent = x.slots[x.pending]
	}
	if rec.word() > incumbent.word() {
		x.pending = slot
	}
}

// select_ is EJSCAN + CHANJOB: find the highest active PRIORITY WORD in
// slots 1..7 and swap it into slot 0. An exactly-equal word keeps the
// EARLIER find: EJ1's ones'-complement compare of equal magnitudes yields
// minus zero, and CCS on -0 takes the fourth branch — "PROCEED WITH SEARCH"
// (EXECUTIVE.agc L492-L499). Only identical words can tie (equal-priority
// NOVAC jobs); FINDVAC words always differ by their VAC address.
func (x *Executive) select_() {
	best := -1
	for i := 1; i < len(x.slots); i++ {
		s := x.slots[i]
		if s == nil || s.dormant {
			continue
		}
		if best < 0 || s.word() > x.slots[best].word() {
			best = i
		}
	}
	if best < 0 {
		return // DUMMYJOB: idle (a dormant slot-0 occupant may remain)
	}
	x.chanjob(best)
}

// chanjob swaps slot 0 with the given slot (CHANJOB L251+).
func (x *Executive) chanjob(slot int) {
	if slot == 0 {
		return
	}
	x.slots[0], x.slots[slot] = x.slots[slot], x.slots[0]
	if x.pending == slot {
		x.pending = -1
	}
}

// runFor gives the job layer up to `budget` CPU nanoseconds. Returns what was
// consumed; zero means idle (no active job).
func (x *Executive) runFor(budget Nanos) Nanos {
	var consumed Nanos
	for consumed < budget {
		r := x.slots[0]
		if r == nil || r.dormant {
			x.select_()
			r = x.slots[0]
			if r == nil || r.dormant {
				return consumed
			}
		}
		// A pending NEWJOB is honored only between instructions — rem == 0
		// means the next instruction has not begun (a DANZIG-boundary state).
		// Mid-instruction (rem > 0) the pending switch waits.
		if r.rem == 0 {
			if x.pending >= 0 && x.slots[x.pending] != nil && !x.slots[x.pending].dormant &&
				x.slots[x.pending].word() > r.word() {
				x.chanjob(x.pending)
				continue
			}
			x.pending = -1
		}

		r.started = true
		if r.rem == 0 {
			r.rem = r.script[r.ip].Cost
		}
		c := r.rem
		if c > budget-consumed {
			c = budget - consumed
		}
		r.rem -= c
		consumed += c
		x.e.subTick += c
		x.e.softwareNs += c
		x.e.lastRan[r.name] = x.e.Now()
		if r.rem > 0 {
			return consumed // budget exhausted mid-instruction
		}

		// ---- the DANZIG boundary: instruction complete ----
		in := r.script[r.ip]
		r.ip++
		if in.Then != nil {
			in.Then(x.e)
		}
		if in.SleepNs != 0 && r.ip < len(r.script) {
			// JOBSLEEP: dormant, memory held, invisible to EJSCAN
			r.dormant = true
			x.e.event("sleep", r.name, r.prio, "")
			if in.SleepNs > 0 {
				rec := r
				x.e.ScheduleTask(x.e.Now()+in.SleepNs, "WAKE:"+r.name, 0, func(en *Engine) {
					en.exec.wakeRec(rec)
				})
			}
			x.select_()
			continue
		}
		if r.ip >= len(r.script) {
			// ENDOFJOB: free the core set and the VAC, then EJSCAN
			x.endJob(r)
			continue
		}
		// NEWJOB test (INTERPRETER.agc L81: CCS NEWJOB / TCF CHANG2)
		if x.pending >= 0 && x.slots[x.pending] != nil && !x.slots[x.pending].dormant {
			x.chanjob(x.pending)
		}
		x.pending = -1
	}
	return consumed
}

func (x *Executive) endJob(r *jobRec) {
	for i := range x.slots {
		if x.slots[i] == r {
			x.slots[i] = nil
			break
		}
	}
	if r.vacIdx >= 0 {
		x.vacs[r.vacIdx] = nil
	}
	x.select_()
}

// wakeRec is JOBWAKE for a specific sleeper.
func (x *Executive) wakeRec(rec *jobRec) {
	if rec == nil || !rec.dormant {
		return // not asleep: no-op (JOBWAKE2's LOCCTR=-1 exit)
	}
	// the sleeper must still be in a slot (a restart may have flushed it)
	slot := -1
	for i := range x.slots {
		if x.slots[i] == rec {
			slot = i
			break
		}
	}
	if slot < 0 {
		return
	}
	rec.dormant = false
	x.e.event("wake", rec.name, rec.prio, "")
	x.postIfGreater(slot)
}

func (x *Executive) wakeByName(name string) {
	for i := range x.slots {
		if s := x.slots[i]; s != nil && s.dormant && s.name == name {
			x.wakeRec(s)
			return
		}
	}
}

func (x *Executive) jobState(name string) JobState {
	// prefer the most-alive instance
	bestState := JobUnknown
	rank := func(s JobState) int {
		switch s {
		case JobRunning:
			return 4
		case JobParked:
			return 3
		case JobWaiting:
			return 2
		case JobSleeping:
			return 1
		}
		return 0
	}
	for i, s := range x.slots {
		if s == nil || s.name != name {
			continue
		}
		var st JobState
		switch {
		case s.dormant:
			st = JobSleeping
		case i == 0:
			st = JobRunning
		case s.started:
			st = JobParked
		default:
			st = JobWaiting
		}
		if rank(st) > rank(bestState) {
			bestState = st
		}
	}
	if bestState != JobUnknown {
		return bestState
	}
	if x.known[name] {
		return JobDone
	}
	return JobUnknown
}

// flush is the restart: every PRIORITY word and every VACnUSE freed.
func (x *Executive) flush() {
	for i := range x.slots {
		x.slots[i] = nil
	}
	for i := range x.vacs {
		x.vacs[i] = nil
	}
	x.pending = -1
}
