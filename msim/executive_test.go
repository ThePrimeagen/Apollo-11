package msim

import (
	"testing"
)

// ---------- the two pools and the two alarm codes ----------
//
// EXECUTIVE.agc: FINDVAC2 (L141-L161) scans the five VACnUSE flags and falls
// into TC BAILOUT1 / OCT 1201; NOVAC2..NEXTCORE (L183-L249) scans the eight
// PRIORITY words and falls into OCT 1202. NOVAC enters at NOVAC2 — it never
// looks at the VAC pool, so a NOVAC request can only ever raise 1202.

func job(name string, prio int, vac bool, ms int) JobSpec {
	return JobSpec{Name: name, Prio: prio, VAC: vac,
		Script: Script{{Section: name, Op: "VXV", Cost: Nanos(ms) * Millisecond}}}
}

func TestFindvacTakesVACAndCore(t *testing.T) {
	// happy: one FINDVAC job holds exactly one core set and one VAC area
	e := NewEngine(Config{})
	if alarm := e.Spawn(job("A", 20, true, 50)); alarm != nil {
		t.Fatalf("spawn: %+v", alarm)
	}
	if c, v := e.CoresHeld(), e.VACsHeld(); c != 1 || v != 1 {
		t.Fatalf("cores=%d vacs=%d, want 1 and 1", c, v)
	}
}

func TestNovacTakesCoreOnly(t *testing.T) {
	// happy: a NOVAC job holds a core set and no VAC
	e := NewEngine(Config{})
	if alarm := e.Spawn(job("A", 20, false, 50)); alarm != nil {
		t.Fatalf("spawn: %+v", alarm)
	}
	if c, v := e.CoresHeld(), e.VACsHeld(); c != 1 || v != 0 {
		t.Fatalf("cores=%d vacs=%d, want 1 and 0", c, v)
	}
}

func TestSixthFindvacRaises1201(t *testing.T) {
	// unhappy: five FINDVAC jobs exhaust the VAC pool; the sixth dies 1201
	e := NewEngine(Config{})
	for i := 0; i < 5; i++ {
		if alarm := e.Spawn(job(string(rune('A'+i)), 20+i, true, 1000)); alarm != nil {
			t.Fatalf("spawn %d: %+v", i, alarm)
		}
	}
	alarm := e.Spawn(job("F", 26, true, 1000))
	if alarm == nil || alarm.Code != 1201 {
		t.Fatalf("sixth FINDVAC alarm = %+v, want code 1201 (NO VAC AREAS)", alarm)
	}
	if alarm.VACsHeld != 5 {
		t.Fatalf("alarm snapshot VACsHeld = %d, want 5", alarm.VACsHeld)
	}
}

func TestNinthCoreRaises1202(t *testing.T) {
	// unhappy: eight NOVAC jobs exhaust the core sets; the ninth dies 1202
	e := NewEngine(Config{})
	for i := 0; i < 8; i++ {
		if alarm := e.Spawn(job(string(rune('A'+i)), 20+i, false, 1000)); alarm != nil {
			t.Fatalf("spawn %d: %+v", i, alarm)
		}
	}
	alarm := e.Spawn(job("I", 30, false, 1000))
	if alarm == nil || alarm.Code != 1202 {
		t.Fatalf("ninth NOVAC alarm = %+v, want code 1202 (NO CORE SETS)", alarm)
	}
	if alarm.CoresHeld != 8 {
		t.Fatalf("alarm snapshot CoresHeld = %d, want 8", alarm.CoresHeld)
	}
}

func TestFindvacWithFreeVACButNoCoreRaises1202(t *testing.T) {
	// unhappy (the subtle one): FINDVAC scans VACs FIRST and claims one, then
	// falls into the core scan — with 8 cores held but a VAC still free the
	// failing request reports 1202, not 1201. This is why a FINDVAC request
	// can raise the "core sets" alarm.
	e := NewEngine(Config{})
	// 4 FINDVAC jobs: 4 cores + 4 VACs
	for i := 0; i < 4; i++ {
		if alarm := e.Spawn(job(string(rune('A'+i)), 20+i, true, 1000)); alarm != nil {
			t.Fatalf("spawn vac %d: %+v", i, alarm)
		}
	}
	// 4 NOVAC jobs: 4 more cores → 8/8 cores, 4/5 VACs
	for i := 0; i < 4; i++ {
		if alarm := e.Spawn(job(string(rune('P'+i)), 25+i, false, 1000)); alarm != nil {
			t.Fatalf("spawn novac %d: %+v", i, alarm)
		}
	}
	alarm := e.Spawn(job("X", 32, true, 1000))
	if alarm == nil || alarm.Code != 1202 {
		t.Fatalf("FINDVAC with free VAC but no core = %+v, want 1202", alarm)
	}
	if alarm.CoresHeld != 8 || alarm.VACsHeld < 4 {
		t.Fatalf("snapshot cores=%d vacs=%d, want 8 and >=4", alarm.CoresHeld, alarm.VACsHeld)
	}
}

// ---------- slot-true scheduling ----------
//
// EXECUTIVE.agc: the running job always occupies core set 0 (CHANJOB swaps,
// L251-L318); allocation scans slots upward taking the first free (NOVAC2,
// L183-L191); SETLOC sets NEWJOB only for a STRICTLY greater priority
// (L224-L234, BZMF ENDFIND); EJSCAN walks slots 1..7 ascending and a tie
// keeps the earlier find (EJ1, L492-L499). Together: an equal-priority copy
// parked in a higher slot starves — Eyles' "uncompleted SERVICER stubs".

func TestHigherPriorityPreemptsAtBoundary(t *testing.T) {
	// happy: HIGH(30) spawned mid-LOW(20) takes over at the next boundary,
	// LOW parks (keeps its core set), then resumes after HIGH ends
	e := NewEngine(Config{})
	e.Spawn(JobSpec{Name: "LOW", Prio: 20, VAC: true, Script: Script{
		{Section: "L", Op: "VXV", Cost: 2 * Millisecond},
		{Section: "L", Op: "VXV", Cost: 2 * Millisecond},
		{Section: "L", Op: "VXV", Cost: 2 * Millisecond},
	}})
	e.ScheduleTask(3*Millisecond, "SP", 0, func(en *Engine) {
		en.Spawn(job("HIGH", 30, false, 2))
	})
	e.RunMS(5) // HIGH spawned at 3 ms, LOW's 2nd instr ends at 4 ms, HIGH runs 4-6
	if got := e.RunningJob(); got != "HIGH" {
		t.Fatalf("at 5 ms running %q, want HIGH", got)
	}
	if st := e.JobState("LOW"); st != JobParked {
		t.Fatalf("LOW state = %v, want parked (still holding its core set)", st)
	}
	if c := e.CoresHeld(); c != 2 {
		t.Fatalf("cores held = %d, want 2 — preempted LOW keeps its core set", c)
	}
	e.RunMS(3) // HIGH done at 6 ms, LOW resumes
	if got := e.RunningJob(); got != "LOW" {
		t.Fatalf("after HIGH ends running %q, want LOW resumed", got)
	}
}

func TestEqualPriorityNeverPreempts(t *testing.T) {
	// unhappy: an equal-priority job must wait for the runner to finish —
	// SETLOC's test is strictly-greater (BZMF falls through on zero)
	e := NewEngine(Config{})
	e.Spawn(job("FIRST", 20, true, 10))
	e.ScheduleTask(2*Millisecond, "SP", 0, func(en *Engine) {
		en.Spawn(job("SECOND", 20, true, 10))
	})
	e.RunMS(9)
	if got := e.RunningJob(); got != "FIRST" {
		t.Fatalf("at 9 ms running %q, want FIRST — a tie must not preempt", got)
	}
	e.RunMS(3) // FIRST ends at 10 ms
	if got := e.RunningJob(); got != "SECOND" {
		t.Fatalf("after FIRST ends running %q, want SECOND", got)
	}
}

func TestTieGoesToLowerSlotAndStubStarves(t *testing.T) {
	// unhappy (the leak in miniature): OLD(20) is running with WAITER(20)
	// queued in a lower slot than where OLD will park. A prio-21 blip preempts
	// OLD; when the blip ends, EJSCAN finds WAITER first (lower slot) and OLD
	// starves — parked, holding core set + VAC, forever.
	e := NewEngine(Config{})
	// OLD occupies slot 0 (runner); WAITER lands slot 1; BLIP lands slot 2.
	e.Spawn(JobSpec{Name: "OLD", Prio: 20, VAC: true, Script: endlessScript(40 * Millisecond)})
	e.ScheduleTask(1*Millisecond, "SP1", 0, func(en *Engine) {
		en.Spawn(JobSpec{Name: "WAITER", Prio: 20, VAC: true, Script: endlessScript(40 * Millisecond)})
	})
	e.ScheduleTask(6*Millisecond, "SP2", 0, func(en *Engine) {
		en.Spawn(job("BLIP", 21, false, 2)) // gyro-like: brief, prio 21
	})
	e.RunMS(20)
	// BLIP preempted OLD at its 10 ms boundary, OLD swapped into slot 2;
	// BLIP ended at ~12 ms; EJSCAN slots 1..7: WAITER (slot 1) wins the tie.
	if got := e.RunningJob(); got != "WAITER" {
		t.Fatalf("running %q, want WAITER — tie must go to the lower slot", got)
	}
	if st := e.JobState("OLD"); st != JobParked {
		t.Fatalf("OLD state = %v, want parked — the starved stub", st)
	}
	if c, v := e.CoresHeld(), e.VACsHeld(); c != 2 || v != 2 {
		t.Fatalf("cores=%d vacs=%d, want 2/2 — the stub still holds its pair", c, v)
	}
	// run long: OLD must STILL be starved (WAITER endless, ties never flip)
	e.RunMS(30)
	if st := e.JobState("OLD"); st != JobParked {
		t.Fatalf("OLD state after 50 ms = %v, want still parked", st)
	}
}

// ---------- JOBSLEEP / JOBWAKE ----------
//
// EXECUTIVE.agc JOBSLP1 (L322-L332) negates PRIORITY — the dormant job is
// invisible to EJSCAN but its core set (and VAC) stay claimed. JOBWAKE2
// (L351-L394) scans for the sleeper and re-posts it through SETLOC.

func TestSleepHoldsCoreSetThroughSleep(t *testing.T) {
	// happy: LRH-style gate — 1 ms head, sleep 95 ms, 1 ms tail;
	// while dormant the core set is held and the CPU is free for others
	e := NewEngine(Config{})
	e.Spawn(JobSpec{Name: "LRH", Prio: 32, VAC: false, Script: Script{
		{Section: "LRH", Op: "BASIC", Cost: 1 * Millisecond, SleepNs: 95 * Millisecond},
		{Section: "LRH", Op: "BASIC", Cost: 1 * Millisecond},
	}})
	e.RunMS(50) // mid-sleep
	if st := e.JobState("LRH"); st != JobSleeping {
		t.Fatalf("LRH state = %v, want sleeping", st)
	}
	if c := e.CoresHeld(); c != 1 {
		t.Fatalf("cores=%d, want 1 — the sleeper holds its core set", c)
	}
	if got := e.RunningJob(); got != "" {
		t.Fatalf("running %q, want idle while LRH sleeps", got)
	}
	e.RunMS(50) // wake at 96 ms, tail runs 96-97, done
	if st := e.JobState("LRH"); st != JobDone {
		t.Fatalf("LRH state = %v, want done after wake+tail", st)
	}
	if c := e.CoresHeld(); c != 0 {
		t.Fatalf("cores=%d, want 0 after ENDOFJOB", c)
	}
}

func TestWakeOfMissingJobIsNoop(t *testing.T) {
	// unhappy: JOBWAKE2 exits via LOCCTR=-1 when no sleeper matches — no
	// state change, no crash
	e := NewEngine(Config{})
	e.Spawn(job("A", 20, false, 5))
	e.Wake("NOBODY")
	e.RunMS(10)
	if st := e.JobState("A"); st != JobDone {
		t.Fatalf("A state = %v, want done — a stray wake must not disturb anything", st)
	}
}

func TestNeverWokenSleeperHoldsCoreForever(t *testing.T) {
	// unhappy: a sleeper with no hardware wake (SleepNs=0 means "wait for
	// explicit Wake") holds its core set for the rest of the run
	e := NewEngine(Config{})
	e.Spawn(JobSpec{Name: "GHOST", Prio: 25, VAC: true, Script: Script{
		{Section: "G", Op: "BASIC", Cost: 1 * Millisecond, SleepNs: SleepUntilWake},
		{Section: "G", Op: "BASIC", Cost: 1 * Millisecond},
	}})
	e.RunMS(5_000)
	if st := e.JobState("GHOST"); st != JobSleeping {
		t.Fatalf("GHOST state = %v, want still sleeping after 5 s", st)
	}
	if c, v := e.CoresHeld(), e.VACsHeld(); c != 1 || v != 1 {
		t.Fatalf("cores=%d vacs=%d, want 1/1 — dormant memory is never reclaimed", c, v)
	}
}
