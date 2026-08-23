package sim

// t42 — starting the descent the way the crew actually did: V37E 63E on the
// DSKY (Eyles: "The crew keyed in Verb 37 Noun 63 to select P63"). The
// landing radar then acquires on its own during the braking phase ("data
// good" — scripted flavor timing), so one toggle gives the full healthy
// descent: READACCS + SERVICER + radar reads.

import "testing"

func typeKeys(e *Engine, keys string, gapMs float64) {
	for _, k := range []byte(keys) {
		e.PressKey(k)
		e.AdvanceAGC(gapMs)
	}
}

func TestV37E63EStartsDescent(t *testing.T) {
	t.Run("happy: V37E63E enters P63 and the LR acquires on its own", func(t *testing.T) {
		e := New()
		typeKeys(e, "V37E63E", 200)
		if got := e.Phase(); got != P63 {
			t.Fatalf("V37E63E must select P63, got %v", got)
		}
		if e.LandingRadarAcquired() {
			t.Fatal("landing radar must not be locked at ignition")
		}
		e.AdvanceAGC(8000)
		if !e.LandingRadarAcquired() {
			t.Fatal("landing radar should report data good within ~6s of PDI")
		}
	})
	t.Run("happy: the keystrokes appear on the DSKY as they are typed", func(t *testing.T) {
		e := New()
		typeKeys(e, "V37", 100)
		if got := e.DSKY().Verb; got != "37" {
			t.Fatalf("verb display should read 37 mid-entry, got %q", got)
		}
	})
	t.Run("unhappy: V37 with a wrong program number stays idle", func(t *testing.T) {
		e := New()
		typeKeys(e, "V37E99E", 100)
		if got := e.Phase(); got != P00 {
			t.Fatalf("V37E99E must not start anything, got %v", got)
		}
	})
	t.Run("unhappy: V37E63E while already descending is a no-op", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AdvanceAGC(3000)
		before := e.CycleCount()
		typeKeys(e, "V37E63E", 100)
		e.AdvanceAGC(2000)
		if e.Phase() != P63 {
			t.Fatal("phase must remain P63")
		}
		// exactly one READACCS chain: one new cycle start per 2s, no double
		if got := e.CycleCount() - before; got > 2 {
			t.Fatalf("re-selecting P63 must not double the READACCS chain, %d new cycles in 2.7s", got)
		}
	})
}
