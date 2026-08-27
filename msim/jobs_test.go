package msim

import (
	"testing"
)

// ---------- MONDO / MONREQ (the V16N68 monitor) ----------
//
// PINBALL_GAME_BUTTONS_AND_LIGHTS.agc L2373-L2395: MONREQ is a waitlist task
// that re-enlists itself every MONDEL = 1.00 s and spawns a NOVAC MONDO at
// CHRPRIO. MONDO (L2397-L2403) never sleeps: if the display is busy it exits
// through MONBUSY immediately.

func TestMonitorRespawnsAtOneHertz(t *testing.T) {
	// happy: keyed once, MONDO copies appear every 1.000 s, each NOVAC,
	// each holding a core set only briefly
	e := NewEngine(Config{})
	StartMonitor(e, 0)
	e.RunMS(5_500)
	spawns := 0
	for _, ev := range e.Events() {
		if ev.Kind == "spawn" && ev.Job == "MONDO" {
			spawns++
		}
	}
	if spawns != 6 { // t=0,1,2,3,4,5 s
		t.Fatalf("MONDO spawned %d times in 5.5 s, want 6 — MONREQ re-arms every 1.00 s", spawns)
	}
	if v := e.VACsHeld(); v != 0 {
		t.Fatalf("VACs held = %d, want 0 — MONDO is NOVAC", v)
	}
}

func TestMonitorCostWithinDocumentedRange(t *testing.T) {
	// happy: each MONDO run costs 30-60 ms of CPU (the outline's bound)
	if MondoCost < 30*Millisecond || MondoCost > 60*Millisecond {
		t.Fatalf("MondoCost = %d ms, want within [30, 60] ms", MondoCost/Millisecond)
	}
}

func TestMonitorNeverSleeps(t *testing.T) {
	// unhappy: MONDO must not carry a sleep — the shipping engine's 250 ms
	// core-holding sleep is exactly the fidelity bug this sim removes
	e := NewEngine(Config{})
	StartMonitor(e, 0)
	e.RunMS(3_000)
	for _, ev := range e.Events() {
		if ev.Kind == "sleep" && ev.Job == "MONDO" {
			t.Fatalf("MONDO slept at %d ns — real MONDO ends via MONBUSY, it never sleeps", ev.At)
		}
	}
}

func TestMonitorKilledStaysDead(t *testing.T) {
	// unhappy: V57/KILLMON stops the respawn chain; nothing re-arms it
	e := NewEngine(Config{})
	StartMonitor(e, 0)
	e.RunMS(2_500)
	StopMonitor(e)
	before := 0
	for _, ev := range e.Events() {
		if ev.Kind == "spawn" && ev.Job == "MONDO" {
			before++
		}
	}
	e.RunMS(4_000)
	after := 0
	for _, ev := range e.Events() {
		if ev.Kind == "spawn" && ev.Job == "MONDO" {
			after++
		}
	}
	if after != before {
		t.Fatalf("MONDO spawns went %d -> %d after kill — KILLMON must be permanent", before, after)
	}
}

// ---------- MAKEPLAY (the display job) ----------
//
// DISPLAY_INTERFACE_ROUTINES.agc L836-L856: GODSPRS1 makes the display job
// one priority higher than its user; a flashing display branches to VACDSP →
// TC SPVAC (core set + VAC, sleeps until the crew responds); a static
// display goes TC NOVAC / 2CADR MAKEPLAY.

func TestStaticDisplayJobIsNovacAtUserPrioPlusOne(t *testing.T) {
	// happy: P63's V06N63 refresh — NOVAC, prio = user+1, ends without sleeping
	e := NewEngine(Config{})
	SpawnDisplayJob(e, DisplayStatic, 20) // user prio 20 → display 21
	if v := e.VACsHeld(); v != 0 {
		t.Fatalf("VACs held = %d, want 0 — static MAKEPLAY is NOVAC", v)
	}
	found := false
	for _, ev := range e.Events() {
		if ev.Kind == "spawn" && ev.Job == "MAKEPLAY" {
			found = true
			if ev.Prio != 21 {
				t.Fatalf("MAKEPLAY prio = %d, want 21 (user 20 + 1)", ev.Prio)
			}
		}
	}
	if !found {
		t.Fatalf("no MAKEPLAY spawn event")
	}
	e.RunMS(50)
	if st := e.JobState("MAKEPLAY"); st != JobDone {
		t.Fatalf("MAKEPLAY state = %v, want done — static form never sleeps", st)
	}
	if c := e.CoresHeld(); c != 0 {
		t.Fatalf("cores = %d, want 0 after the static display ends", c)
	}
}

func TestFlashingDisplayJobTakesVACAndSleepsUntilPRO(t *testing.T) {
	// unhappy: the early-P64 flashing V06N64 takes a core set + VAC and
	// sleeps holding BOTH until the crew keys PRO
	e := NewEngine(Config{})
	SpawnDisplayJob(e, DisplayFlashing, 20)
	e.RunMS(5_000)
	if st := e.JobState("MAKEPLAY"); st != JobSleeping {
		t.Fatalf("MAKEPLAY state = %v, want sleeping until PRO", st)
	}
	if c, v := e.CoresHeld(), e.VACsHeld(); c != 1 || v != 1 {
		t.Fatalf("cores=%d vacs=%d, want 1/1 — the flashing sleeper holds both", c, v)
	}
	KeyPRO(e)
	e.RunMS(100)
	if st := e.JobState("MAKEPLAY"); st != JobDone {
		t.Fatalf("MAKEPLAY state = %v, want done after PRO", st)
	}
	if c, v := e.CoresHeld(), e.VACsHeld(); c != 0 || v != 0 {
		t.Fatalf("cores=%d vacs=%d, want 0/0 after PRO releases the display", c, v)
	}
}
