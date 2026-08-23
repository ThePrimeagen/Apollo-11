package sim

// t30 — stub recovery must be narrated. When demand is only a hair over
// 100%, the old SERVICER is still running at the READACCS re-arm (LEAK event
// fires) but nothing preempts it at that instant, so it finishes late and
// FREES its pair. Without a RECOVERED event the log claims a stub exists
// while the pools show it gone — which reads as stubs "not propagating".

import (
	"strings"
	"testing"
)

func recoverEvents(e *Engine) []Event {
	var out []Event
	for _, ev := range e.Events() {
		if ev.Kind == EvRecover {
			out = append(out, ev)
		}
	}
	return out
}

func TestStubRecovery(t *testing.T) {
	t.Run("happy: superseded copy that keeps the CPU finishes and recovers", func(t *testing.T) {
		e := New()
		e.ScheduleJob("SERVICER", 20, 50, true) // old copy
		e.AdvanceAGC(10)                        // running: ~40ms left
		e.ScheduleJob("SERVICER", 20, 1000, true)
		if n := e.StubCount(); n != 1 {
			t.Fatalf("old copy should be a stub right after supersession, got %d", n)
		}
		e.AdvanceAGC(60) // equal prio does not preempt: old copy finishes
		if n := e.StubCount(); n != 0 {
			t.Fatalf("finished stub must release, StubCount = %d", n)
		}
		cores, vacs := countBusy(e)
		if cores != 1 || vacs != 1 {
			t.Fatalf("only the new copy should hold a pair, cores=%d vacs=%d", cores, vacs)
		}
		evs := recoverEvents(e)
		if len(evs) != 1 {
			t.Fatalf("want exactly one RECOVERED event, got %d", len(evs))
		}
		if !strings.Contains(evs[0].Text, "RECOVERED") {
			t.Fatalf("recovery event should say RECOVERED, got %q", evs[0].Text)
		}
	})
	t.Run("unhappy: a starved stub never logs a recovery", func(t *testing.T) {
		e := New()
		e.ScheduleJob("SERVICER", 20, 50, true) // old copy
		e.AdvanceAGC(10)
		e.ScheduleJob("BLOCKER", 25, 100, false) // takes the CPU
		e.ScheduleJob("SERVICER", 20, 1e9, true) // new copy queued behind
		e.AdvanceAGC(200)                        // BLOCKER ends; rescan picks the NEWEST SERVICER
		if n := e.StubCount(); n != 1 {
			t.Fatalf("old copy must starve while the newest runs, StubCount = %d", n)
		}
		core, vac := ownerHeld(e, "SERVICER")
		if !core || !vac {
			t.Fatal("starved stub must still hold its core set and VAC")
		}
		if evs := recoverEvents(e); len(evs) != 0 {
			t.Fatalf("starved stub must not log a recovery, got %+v", evs)
		}
	})
	t.Run("happy: post-restart marginal regime narrates LEAK then RECOVERED", func(t *testing.T) {
		e := overloadedP63(t)
		alarmAt := runUntilAlarm(t, e, 60000)
		e.AdvanceAGC(10000) // ~5 marginal cycles after the restart
		found := false
		for _, ev := range recoverEvents(e) {
			if ev.AGCTimeMs > alarmAt {
				found = true
			}
		}
		if !found {
			t.Fatal("post-restart cycles should log stub recoveries")
		}
		if len(e.Alarms()) != 1 {
			t.Fatalf("no second alarm expected within 10s of the restart, got %d", len(e.Alarms()))
		}
	})
	t.Run("unhappy: healthy descent logs neither LEAK nor RECOVERED", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AdvanceAGC(20000)
		if evs := recoverEvents(e); len(evs) != 0 {
			t.Fatalf("healthy: want no recover events, got %+v", evs)
		}
		if evs := leakEvents(e); len(evs) != 0 {
			t.Fatalf("healthy: want no leak events, got %+v", evs)
		}
	})
}
