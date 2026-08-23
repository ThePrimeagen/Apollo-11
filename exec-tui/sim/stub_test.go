package sim

// t22-t24 — abandoned-stub visibility. Under >100% demand each READACCS
// schedules a fresh SERVICER while the old copy is still holding its core set
// and VAC area. Those superseded copies ("stubs", Eyles) ARE the memory leak;
// the engine must report them so the UI can show the pools being eaten.

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// t22 — SlotState.Stub marks superseded SERVICER copies
// ---------------------------------------------------------------------------

func TestStubSlotMarking(t *testing.T) {
	t.Run("happy: overload marks old copies as stubs, exactly one live SERVICER", func(t *testing.T) {
		e := overloadedP63(t)
		e.AdvanceAGC(6500) // >=3 overloaded cycles: >=2 stubs + 1 live copy

		stubCores, liveCores, stubVacs := 0, 0, 0
		for _, c := range e.CoreSets() {
			if !c.Busy || c.Owner != "SERVICER" {
				continue
			}
			if c.Stub {
				stubCores++
			} else {
				liveCores++
			}
		}
		for _, v := range e.VACs() {
			if v.Busy && v.Owner == "SERVICER" && v.Stub {
				stubVacs++
			}
		}
		if stubCores < 2 {
			t.Fatalf("want >=2 stub core sets after 3 overloaded cycles, got %d", stubCores)
		}
		if stubVacs < 2 {
			t.Fatalf("want >=2 stub VACs after 3 overloaded cycles, got %d", stubVacs)
		}
		if liveCores != 1 {
			t.Fatalf("exactly one live (non-stub) SERVICER should exist, got %d", liveCores)
		}
	})
	t.Run("unhappy: healthy descent never marks a stub", func(t *testing.T) {
		e := New()
		e.StartDescent()
		for step := 0; step < 40; step++ {
			e.AdvanceAGC(500)
			for _, c := range e.CoreSets() {
				if c.Stub {
					t.Fatalf("healthy cycle marked a stub at t=%.0fms: %+v", e.AGCTimeMs(), c)
				}
			}
			for _, v := range e.VACs() {
				if v.Stub {
					t.Fatalf("healthy cycle marked a stub VAC at t=%.0fms: %+v", e.AGCTimeMs(), v)
				}
			}
		}
	})
}

// ---------------------------------------------------------------------------
// t23 — StubCount
// ---------------------------------------------------------------------------

func TestStubCount(t *testing.T) {
	t.Run("happy: overload accumulates one stub per cycle", func(t *testing.T) {
		e := overloadedP63(t)
		e.AdvanceAGC(6500)
		if n := e.StubCount(); n < 2 {
			t.Fatalf("want >=2 stubs after 3 overloaded cycles, got %d", n)
		}
	})
	t.Run("unhappy: healthy descent has zero stubs", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AdvanceAGC(20000)
		if n := e.StubCount(); n != 0 {
			t.Fatalf("healthy: want 0 stubs, got %d", n)
		}
	})
	t.Run("unhappy: the BAILOUT restart flushes every stub", func(t *testing.T) {
		e := overloadedP63(t)
		runUntilAlarm(t, e, 60000)
		e.AdvanceAGC(100) // REREADAC rebuilds one fresh SERVICER
		if n := e.StubCount(); n != 0 {
			t.Fatalf("restart must flush stubs, got %d", n)
		}
	})
}

// ---------------------------------------------------------------------------
// t24 — the leak is narrated in the event log
// ---------------------------------------------------------------------------

func leakEvents(e *Engine) []Event {
	var out []Event
	for _, ev := range e.Events() {
		if ev.Kind == EvLeak {
			out = append(out, ev)
		}
	}
	return out
}

func TestLeakEvents(t *testing.T) {
	t.Run("happy: each overloaded cycle logs the growing stub count", func(t *testing.T) {
		e := overloadedP63(t)
		e.AdvanceAGC(6500)
		evs := leakEvents(e)
		if len(evs) < 2 {
			t.Fatalf("want >=2 leak events after 3 overloaded cycles, got %d", len(evs))
		}
		for _, ev := range evs {
			if !strings.Contains(ev.Text, "stub") {
				t.Fatalf("leak event should mention stubs, got %q", ev.Text)
			}
		}
	})
	t.Run("unhappy: healthy descent logs no leak events", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AdvanceAGC(20000)
		if evs := leakEvents(e); len(evs) != 0 {
			t.Fatalf("healthy: want no leak events, got %+v", evs)
		}
	})
}
