package sim

// t32 — the event log must tell the leak STORY, not spam its steady state.
// At the knife edge (demand ≈ 100%) every cycle produces an identical
// LEAK/RECOVERED pair; repeats within a cycle window are throttled so the
// log stays quiet until something CHANGES (escalation, re-onset, alarm).
// A persistent hint after a P63 restart explains why the leaking stopped.

import (
	"strings"
	"testing"
)

// knifeEdge is the user's post-restart regime built directly: READACCS
// boundaries offset from the 1Hz job marks, LR + bug on, no monitor —
// demand ≈ 100%: every cycle overruns slightly and recovers.
func knifeEdge(t *testing.T) *Engine {
	t.Helper()
	e := New()
	e.AdvanceAGC(170) // desync the 2s boundary from the 1Hz marks
	e.StartDescent()
	e.AcquireLandingRadar()
	e.SetRadarBug(true)
	return e
}

func hintEvents(e *Engine) []Event {
	var out []Event
	for _, ev := range e.Events() {
		if ev.Kind == EvHint {
			out = append(out, ev)
		}
	}
	return out
}

func TestKnifeEdgeLogThrottling(t *testing.T) {
	t.Run("happy: steady overrun/recovery logs once, not every cycle", func(t *testing.T) {
		e := knifeEdge(t)
		e.AdvanceAGC(40000) // 20 cycles
		if n := len(e.Alarms()); n != 0 {
			t.Fatalf("knife edge must not alarm on its own, got %d alarms", n)
		}
		if n := len(leakEvents(e)); n < 1 || n > 2 {
			t.Fatalf("steady LEAK should be logged once (maybe twice), got %d", n)
		}
		if n := len(recoverEvents(e)); n < 1 || n > 2 {
			t.Fatalf("steady RECOVERED should be logged once (maybe twice), got %d", n)
		}
		if !e.RecoveredRecently(5000) {
			t.Fatal("engine must still report recent recoveries for the UI indicator")
		}
	})
	t.Run("happy: escalation is never throttled — every new depth logs", func(t *testing.T) {
		e := overloadedP63(t)
		alarmAt := runUntilAlarm(t, e, 60000)
		var pre []string
		for _, ev := range leakEvents(e) {
			if ev.AGCTimeMs <= alarmAt {
				pre = append(pre, ev.Text)
			}
		}
		if len(pre) < 3 {
			t.Fatalf("want >=3 escalating LEAK logs before the alarm, got %d", len(pre))
		}
		seen := map[string]bool{}
		for _, txt := range pre {
			if seen[txt] {
				t.Fatalf("escalating LEAK logs must all differ, repeated: %q", txt)
			}
			seen[txt] = true
		}
	})
	t.Run("unhappy: healthy descent logs nothing and reports no recoveries", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AdvanceAGC(20000)
		if len(leakEvents(e))+len(recoverEvents(e))+len(hintEvents(e)) != 0 {
			t.Fatal("healthy descent must log no leak/recover/hint events")
		}
		if e.RecoveredRecently(20000) {
			t.Fatal("healthy descent must not report recoveries")
		}
	})
}

func TestPostRestartHint(t *testing.T) {
	t.Run("happy: a P63 restart explains how to overload again", func(t *testing.T) {
		e := overloadedP63(t)
		runUntilAlarm(t, e, 60000)
		hints := hintEvents(e)
		if len(hints) == 0 {
			t.Fatal("P63 restart must log a re-load hint")
		}
		if !strings.Contains(hints[0].Text, "n") || !strings.Contains(hints[0].Text, "P64") {
			t.Fatalf("hint should mention the n and P64 controls, got %q", hints[0].Text)
		}
	})
	t.Run("unhappy: P64 restarts stay hintless — the load is protected", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AcquireLandingRadar()
		e.SetRadarBug(true)
		e.EnterP64()
		runUntilAlarm(t, e, 60000)
		if n := len(hintEvents(e)); n != 0 {
			t.Fatalf("P64 restart must not suggest shedding worked, got %d hints", n)
		}
	})
}
