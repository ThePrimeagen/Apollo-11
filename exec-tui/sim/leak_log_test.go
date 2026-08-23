package sim

// t32 — the event log must tell the leak STORY, not spam a steady state.
// The knife edge (theft active, no monitor, demand just under 100%) is the
// flight's QUIET regime: ~5 minutes with the theft active and no alarms, no
// overruns — so the log must stay silent there. Under real overload the LEAK
// escalation logs every new depth. A persistent hint after a P63 restart
// explains why the leaking stopped.

import (
	"strings"
	"testing"
)

// knifeEdge is the flight's pre-monitor regime: LR + bug on, no monitor —
// demand just under 100%: margin gone, nothing overruns.
func knifeEdge(t *testing.T) *Engine {
	t.Helper()
	e := New()
	e.AdvanceAGC(170) // arbitrary start offset, as in interactive use
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
	t.Run("happy: the quiet knife edge logs no leaks, no recoveries, no alarms", func(t *testing.T) {
		e := knifeEdge(t)
		e.AdvanceAGC(40000) // 20 cycles
		if n := len(e.Alarms()); n != 0 {
			t.Fatalf("knife edge must not alarm on its own (flight: quiet ~5min), got %d alarms", n)
		}
		if n := len(leakEvents(e)); n != 0 {
			t.Fatalf("quiet knife edge must not log LEAK events, got %d", n)
		}
		if n := len(recoverEvents(e)); n != 0 {
			t.Fatalf("quiet knife edge must not log RECOVERED events, got %d", n)
		}
		if !e.KnifeEdge() {
			t.Fatal("engine must report the knife edge for the UI indicator")
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
