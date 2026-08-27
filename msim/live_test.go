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

func TestLiveDefaultsMatchTheFlight(t *testing.T) {
	// unhappy-guard: the screen opens in the flight's P63 state — descent
	// on, monitor off, radar steal on
	l := NewLive()
	if !l.DescentOn() || l.MonitorOn() || !l.RadarOn() {
		t.Fatalf("defaults = descent %v, monitor %v, radar %v; want on/off/on",
			l.DescentOn(), l.MonitorOn(), l.RadarOn())
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
