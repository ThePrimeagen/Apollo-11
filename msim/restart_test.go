package msim

import (
	"testing"
)

// ---------- BAILOUT → software restart ----------
//
// ALARM_AND_ABORT.agc / FRESH_START_AND_RESTART.agc: the restart frees all
// eight PRIORITY words and five VACnUSE flags, wipes the waitlist, and the
// restart tables rebuild exactly one READACCS chain and one SERVICER.
// Monitor verbs are NOT restarted (Cherry pp. 5-6).

func TestRestartFlushesAndRebuilds(t *testing.T) {
	// happy: pools full → alarm → restart frees everything and the
	// RestartHook rebuilds the guidance chain; the monitor chain is gone
	e := NewEngine(Config{AutoRestart: true, RestartHook: func(en *Engine) {
		en.Spawn(job("SERVICER", 20, true, 100))
	}})
	StartMonitor(e, 0)
	// choke the pools: 8 endless NOVAC holders, then one more spawn fails
	for i := 0; i < 8; i++ {
		e.Spawn(JobSpec{Name: "H", Prio: 25, VAC: false, Script: endlessScript(Second)})
	}
	if a := e.Spawn(job("X", 30, false, 10)); a == nil || a.Code != 1202 {
		t.Fatalf("choke spawn = %+v, want 1202", a)
	}
	if e.RestartCount() != 1 {
		t.Fatalf("RestartCount = %d, want 1", e.RestartCount())
	}
	e.RunMS(200)
	// after the restart: only the hook's SERVICER should hold memory
	if c := e.CoresHeld(); c > 1 {
		t.Fatalf("cores after restart = %d, want <= 1 — the flush frees all eight", c)
	}
	mondoAfter := 0
	for _, ev := range e.Events() {
		if ev.Kind == "spawn" && ev.Job == "MONDO" && ev.At > e.RestartAt(0) {
			mondoAfter++
		}
	}
	if mondoAfter != 0 {
		t.Fatalf("MONDO spawned %d times after restart — monitor verbs are not restarted", mondoAfter)
	}
}

func TestNoAutoRestartFreezesForInspection(t *testing.T) {
	// unhappy: with AutoRestart off the alarm is recorded, pools stay frozen
	// (nothing is flushed) and no restart happens — the debugging view
	e := NewEngine(Config{AutoRestart: false})
	for i := 0; i < 8; i++ {
		e.Spawn(JobSpec{Name: "H", Prio: 25, VAC: false, Script: endlessScript(Second)})
	}
	a := e.Spawn(job("X", 30, false, 10))
	if a == nil || a.Code != 1202 {
		t.Fatalf("alarm = %+v, want 1202", a)
	}
	if e.RestartCount() != 0 {
		t.Fatalf("RestartCount = %d, want 0 with AutoRestart off", e.RestartCount())
	}
	if c := e.CoresHeld(); c != 8 {
		t.Fatalf("cores = %d, want 8 — frozen for inspection", c)
	}
	if n := len(e.Alarms()); n != 1 {
		t.Fatalf("alarms = %d, want exactly 1 recorded", n)
	}
}
