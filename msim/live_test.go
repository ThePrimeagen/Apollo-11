package msim

import (
	"testing"
)

// ---------- introspection for the command screen ----------

func TestSlotViewsExposeTheEightCoreSets(t *testing.T) {
	// happy: a running FINDVAC job, a parked copy, and a sleeper are all
	// visible with name, priority, state, VAC index and script progress
	e := NewEngine(Config{})
	e.Spawn(JobSpec{Name: "OLD", Prio: 20, VAC: true, Script: endlessScript(40 * Millisecond)})
	e.ScheduleTask(6*Millisecond, "SP", 0, func(en *Engine) {
		en.Spawn(JobSpec{Name: "NEWER", Prio: 20, VAC: true, Script: endlessScript(40 * Millisecond)})
	})
	e.Spawn(JobSpec{Name: "GATE", Prio: 32, VAC: false, Script: Script{
		{Section: "G", Op: "BASIC", Cost: Millisecond, SleepNs: SleepUntilWake},
		{Section: "G", Op: "BASIC", Cost: Millisecond},
	}})
	e.RunMS(20)
	views := e.SlotViews()
	if len(views) != 8 {
		t.Fatalf("SlotViews returned %d entries, want always 8", len(views))
	}
	byName := map[string]SlotView{}
	occupied := 0
	for _, v := range views {
		if v.Name == "" {
			continue
		}
		occupied++
		byName[v.Name] = v
	}
	if occupied != 3 {
		t.Fatalf("occupied slots = %d, want 3 (OLD, NEWER, GATE)", occupied)
	}
	if v := byName["NEWER"]; v.State != JobRunning || v.VAC < 0 {
		t.Fatalf("NEWER view = %+v, want running with a VAC index", v)
	}
	if v := byName["OLD"]; v.State != JobParked || v.VAC < 0 || v.IP <= 0 {
		t.Fatalf("OLD view = %+v, want parked mid-script holding its VAC", v)
	}
	if v := byName["GATE"]; v.State != JobSleeping || v.VAC != -1 {
		t.Fatalf("GATE view = %+v, want sleeping NOVAC (VAC -1)", v)
	}
	for _, v := range byName {
		if v.Len <= 0 || v.IP > v.Len {
			t.Fatalf("view %+v has a bad progress pair", v)
		}
	}
}

func TestSlotViewsFreeSlotsAreEmpty(t *testing.T) {
	// unhappy: an idle machine reports 8 slots, all nameless
	e := NewEngine(Config{})
	e.RunMS(5)
	for i, v := range e.SlotViews() {
		if v.Name != "" || v.State != JobUnknown {
			t.Fatalf("slot %d = %+v, want empty on an idle machine", i, v)
		}
	}
}

func TestActivityTracking(t *testing.T) {
	// happy: LastRan records the last instant a named job consumed CPU;
	// TaskFires/LastFired record waitlist dispatches by name
	e := NewEngine(Config{})
	e.Spawn(job("BLIP", 30, false, 3))
	e.ScheduleTask(10*Millisecond, "PING", 100*Microsecond, func(*Engine) {})
	e.ScheduleTask(20*Millisecond, "PING", 100*Microsecond, func(*Engine) {})
	e.RunMS(30)
	if got := e.LastRan("BLIP"); got < 2*Millisecond || got > 4*Millisecond {
		t.Fatalf("LastRan(BLIP) = %d, want ~3 ms (its completion)", got)
	}
	if got := e.TaskFires("PING"); got != 2 {
		t.Fatalf("TaskFires(PING) = %d, want 2", got)
	}
	if got := e.LastFired("PING"); got != 20*Millisecond {
		t.Fatalf("LastFired(PING) = %d, want exactly 20 ms", got)
	}
}

func TestActivityUnknownNames(t *testing.T) {
	// unhappy: names that never ran/fired report -1 and 0
	e := NewEngine(Config{})
	e.RunMS(5)
	if got := e.LastRan("NOBODY"); got != -1 {
		t.Fatalf("LastRan(NOBODY) = %d, want -1", got)
	}
	if got := e.TaskFires("NOBODY"); got != 0 {
		t.Fatalf("TaskFires(NOBODY) = %d, want 0", got)
	}
	if got := e.LastFired("NOBODY"); got != -1 {
		t.Fatalf("LastFired(NOBODY) = %d, want -1", got)
	}
}

// ---------- the Live controller (the command screen's engine) ----------

func TestLiveDescentToggleStopsAndResumesTheChain(t *testing.T) {
	// happy: descent off → the READACCS chain dies and no new SERVICER
	// spawns; on again → the chain resumes on the 2 s lattice
	l := NewLive()
	l.SetRadar(false) // margin: this test is about the chain, not the leak
	l.StepMS(5_000)
	before := countSpawns(l.Engine(), "SERVICER")
	if before < 2 {
		t.Fatalf("SERVICER spawned %d times in 5 s with descent on, want >= 2", before)
	}
	l.SetDescent(false)
	l.StepMS(6_000)
	afterOff := countSpawns(l.Engine(), "SERVICER")
	if afterOff > before+1 {
		t.Fatalf("SERVICER spawns went %d -> %d while descent was OFF — the chain must die", before, afterOff)
	}
	l.SetDescent(true)
	l.StepMS(5_000)
	afterOn := countSpawns(l.Engine(), "SERVICER")
	if afterOn < afterOff+2 {
		t.Fatalf("SERVICER spawns went %d -> %d after descent back ON — the chain must resume", afterOff, afterOn)
	}
	// the resumed chain sits on the 2 s lattice
	var lastSpawn Nanos
	for _, ev := range l.Engine().Events() {
		if ev.Kind == "spawn" && ev.Job == "SERVICER" {
			lastSpawn = ev.At
		}
	}
	if lastSpawn%(2*Second) > 2*Millisecond {
		t.Fatalf("resumed SERVICER spawn at %d ns is off the 2 s lattice", lastSpawn)
	}
}

func TestLiveDescentToggleIdempotent(t *testing.T) {
	// unhappy: double-on must not double the chain; double-off must not panic
	l := NewLive()
	l.SetRadar(false)
	l.SetDescent(true) // already on
	l.StepMS(4_100)
	spawns := countSpawns(l.Engine(), "SERVICER")
	if spawns > 3 {
		t.Fatalf("SERVICER spawned %d times in 4.1 s — double-on must not double the chain", spawns)
	}
	l.SetDescent(false)
	l.SetDescent(false)
	l.StepMS(2_000)
}

func TestLiveMonitorToggle(t *testing.T) {
	// happy: 1668 on → MONDO refreshes at 1 Hz; off (KEY REL) → they stop.
	// The monitor is the crew's: it runs even with descent off.
	l := NewLive()
	l.SetRadar(false)
	l.SetDescent(false)
	l.StepMS(3_000)
	l.SetMonitor(true)
	l.StepMS(3_500)
	on := countSpawns(l.Engine(), "MONDO")
	if on < 2 || on > 4 {
		t.Fatalf("MONDO spawned %d times in 3.5 s with 1668 up, want ~3 (1 Hz from ENTR+1)", on)
	}
	l.SetMonitor(false)
	l.StepMS(4_000)
	off := countSpawns(l.Engine(), "MONDO")
	if off != on {
		t.Fatalf("MONDO spawns went %d -> %d after KEY REL — the chain must die", on, off)
	}
}

func TestLiveMonitorToggleIdempotent(t *testing.T) {
	// unhappy: keying 1668 twice must not double the refresh rate
	l := NewLive()
	l.SetRadar(false)
	l.SetDescent(false)
	l.SetMonitor(true)
	l.SetMonitor(true)
	l.StepMS(4_100)
	if got := countSpawns(l.Engine(), "MONDO"); got > 5 {
		t.Fatalf("MONDO spawned %d times in 4.1 s — double-keying must not double the chain", got)
	}
}

func TestLiveRadarToggleFreezesTheft(t *testing.T) {
	// happy: radar steal off mid-run → TheftNs stops growing; on → resumes
	l := NewLive()
	l.StepMS(2_000)
	t1 := l.Engine().TheftNs()
	if t1 <= 0 {
		t.Fatalf("theft = %d after 2 s with the steal on, want > 0", t1)
	}
	l.SetRadar(false)
	l.StepMS(2_000)
	t2 := l.Engine().TheftNs()
	if t2 != t1 {
		t.Fatalf("theft grew %d -> %d while the steal was OFF", t1, t2)
	}
	l.SetRadar(true)
	l.StepMS(2_000)
	if t3 := l.Engine().TheftNs(); t3 <= t2 {
		t.Fatalf("theft frozen at %d after the steal came back ON", t3)
	}
}

func TestDescentOffCancelsThePendingChain(t *testing.T) {
	// happy: switching descent off at t=0 removes the pre-armed chain
	// entries — READACCS and R10,R11 never fire, never burn task cost
	l := NewLive()
	l.SetRadar(false)
	l.SetDescent(false)
	l.StepMS(3_000)
	e := l.Engine()
	for _, name := range []string{"READACCS", "R10,R11", "LRHTASK", "LRVTASK"} {
		if got := e.TaskFires(name); got != 0 {
			t.Fatalf("%s fired %d times with descent off from t=0, want 0", name, got)
		}
		if got := e.BusyNs(name); got != 0 {
			t.Fatalf("%s burned %d ns with descent off from t=0, want 0", name, got)
		}
	}
	// unhappy: cancelling names that do not exist is a no-op
	e.CancelTasks("NOBODY", "NOTHING")
	l.StepMS(100)
}

func TestLiveDefaultsMatchTheFlight(t *testing.T) {
	// unhappy-guard: the screen opens in the flight's P63 state — descent
	// on, monitor off, radar steal on
	l := NewLive()
	if !l.DescentOn() || l.MonitorOn() || !l.RadarOn() {
		t.Fatalf("defaults = descent %v, monitor %v, radar %v; want on/off/on",
			l.DescentOn(), l.MonitorOn(), l.RadarOn())
	}
}

func TestLiveServicerOneShot(t *testing.T) {
	// happy: one-shot on — the run's FIRST READACCS enters the only
	// SERVICER; the 2 s lattice itself keeps firing (READACCS, the LR
	// gates, R10/R11 all stay on their timers)
	l := NewLive()
	l.SetRadar(false)
	l.SetServicerOneShot(true)
	if !l.ServicerOneShot() {
		t.Fatalf("SetServicerOneShot(true) did not latch")
	}
	l.StepMS(6_500)
	e := l.Engine()
	if got := countSpawns(e, "SERVICER"); got != 1 {
		t.Fatalf("SERVICER spawned %d times in 6.5 s one-shot, want exactly 1", got)
	}
	if got := e.TaskFires("READACCS"); got < 3 {
		t.Fatalf("READACCS fired %d times in 6.5 s, want >= 3 — the lattice must keep firing", got)
	}
	if got := e.TaskFires("LRVTASK"); got < 3 {
		t.Fatalf("LRVTASK fired %d times in 6.5 s, want >= 3 — the radar gates ride every cycle", got)
	}
	// unhappy: double-set stays one-shot
	l.SetServicerOneShot(true)
	l.StepMS(2_000)
	if got := countSpawns(e, "SERVICER"); got != 1 {
		t.Fatalf("SERVICER spawned %d times after a double-set, want still exactly 1", got)
	}
	// unhappy-guard: the default machine is the flight's — a fresh copy
	// every cycle
	l2 := NewLive()
	l2.SetRadar(false)
	if l2.ServicerOneShot() {
		t.Fatalf("one-shot must default OFF — the flight scenarios respawn every 2 s")
	}
	l2.StepMS(4_100)
	if got := countSpawns(l2.Engine(), "SERVICER"); got < 2 {
		t.Fatalf("default machine spawned SERVICER %d times in 4.1 s, want >= 2", got)
	}
}

func TestLiveApproachP64(t *testing.T) {
	// happy: approach on — HIGATASK enters HIGATJOB once, which sleeps
	// holding a VAC on the antenna position-2 discrete; the pass's display
	// request is the flashing V06N64 (sleeps holding a VAC until PRO); and
	// the pass itself carries the REDESIG load
	l := NewLive()
	l.SetRadar(false)
	l.SetServicerOneShot(true)
	l.SetApproach(true)
	if !l.ApproachOn() {
		t.Fatalf("SetApproach(true) did not latch")
	}
	l.StepMS(2_400)
	e := l.Engine()
	if got := countSpawns(e, "HIGATJOB"); got != 1 {
		t.Fatalf("HIGATJOB spawned %d times, want exactly 1 (the high-gate antenna job)", got)
	}
	if st := e.JobState("HIGATJOB"); st != JobSleeping {
		t.Fatalf("HIGATJOB state %v, want sleeping on the position-2 discrete", st)
	}
	if st := e.JobState("MAKEPLAY"); st != JobSleeping {
		t.Fatalf("MAKEPLAY state %v, want the flashing V06N64 asleep awaiting PRO", st)
	}
	if got := e.VACsHeld(); got < 2 {
		t.Fatalf("VACsHeld = %d, want >= 2 (HIGATJOB + the flashing display)", got)
	}
	if got := e.BusyNs("SERVICER"); got < 1450*Millisecond {
		t.Fatalf("the finished approach pass consumed %d ms, want >= 1450 ms — the REDESIG sections are missing",
			got/Millisecond)
	}
	// unhappy: double-on arms a single HIGATJOB
	l2 := NewLive()
	l2.SetRadar(false)
	l2.SetServicerOneShot(true)
	l2.SetApproach(true)
	l2.SetApproach(true)
	l2.StepMS(2_400)
	if got := countSpawns(l2.Engine(), "HIGATJOB"); got != 1 {
		t.Fatalf("double SetApproach(true) spawned HIGATJOB %d times, want 1", got)
	}
	// unhappy: approach off before the run restores the P63 pass — no
	// HIGATJOB, no REDESIG cost, a static display that finishes
	l3 := NewLive()
	l3.SetRadar(false)
	l3.SetServicerOneShot(true)
	l3.SetApproach(true)
	l3.SetApproach(false)
	if l3.ApproachOn() {
		t.Fatalf("SetApproach(false) did not clear")
	}
	l3.StepMS(2_400)
	e3 := l3.Engine()
	if got := countSpawns(e3, "HIGATJOB"); got != 0 {
		t.Fatalf("HIGATJOB spawned %d times with approach keyed off before the run, want 0", got)
	}
	if got := e3.BusyNs("SERVICER"); got > 1400*Millisecond {
		t.Fatalf("the pass consumed %d ms with approach off, want the P63 pass (<= 1400 ms)", got/Millisecond)
	}
	if st := e3.JobState("MAKEPLAY"); st != JobDone {
		t.Fatalf("MAKEPLAY state %v with approach off, want done — the static V06N63 pastes and ends", st)
	}
}

func countSpawns(e *Engine, name string) int {
	n := 0
	for _, ev := range e.Events() {
		if ev.Kind == "spawn" && ev.Job == name {
			n++
		}
	}
	return n
}
