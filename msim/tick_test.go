package msim

import (
	"testing"
)

// ---------- the 100 µs tick ----------
//
// The machine advances 100 µs per tick for maximum accuracy: waitlist
// dispatches, sleeps, and instruction completions resolve on a 100 µs
// lattice instead of a millisecond one, while every millisecond-level
// quantity (the theft waveform, the samples, the reports) is preserved
// exactly.

func TestSubMillisecondTaskPunctuality(t *testing.T) {
	// happy: a task due at 2.0003 s fires at exactly 2.0003 s — not rounded
	// up to the next millisecond
	e := NewEngine(Config{})
	var fired Nanos = -1
	e.ScheduleTask(2*Second+300*Microsecond, "PROBE", 0, func(en *Engine) {
		fired = en.Now()
	})
	e.RunMS(2_010)
	if fired != 2*Second+300*Microsecond {
		t.Fatalf("PROBE fired at %d ns, want exactly %d — the 100 µs lattice", fired, 2*Second+300*Microsecond)
	}
}

func TestOffLatticeDueQuantizesToNextTick(t *testing.T) {
	// unhappy: a due at 2.00035 s (off the 100 µs lattice) fires at the
	// next tick boundary, 2.0004 s — quantization is explicit, never early
	e := NewEngine(Config{})
	var fired Nanos = -1
	e.ScheduleTask(2*Second+350*Microsecond, "PROBE", 0, func(en *Engine) {
		fired = en.Now()
	})
	e.RunMS(2_010)
	if fired != 2*Second+400*Microsecond {
		t.Fatalf("PROBE fired at %d ns, want %d — off-lattice dues wait for the next tick", fired, 2*Second+400*Microsecond)
	}
}

func TestInstructionCompletesAt100MicrosecondResolution(t *testing.T) {
	// happy: a 250 µs instruction on an otherwise idle machine completes at
	// exactly 250 µs — inside the third tick, not at a millisecond boundary
	e := NewEngine(Config{})
	var done Nanos = -1
	e.Spawn(JobSpec{Name: "J", Prio: 20, Script: Script{
		{Section: "S", Op: "BASIC", Cost: 250 * Microsecond,
			Then: func(en *Engine) { done = en.Now() }},
	}})
	e.RunMS(2)
	if done != 250*Microsecond {
		t.Fatalf("J completed at %d ns, want exactly %d", done, 250*Microsecond)
	}
}

func TestCompletionConservationWithTheftAt100Micro(t *testing.T) {
	// unhappy: with the RR bug skimming every tick, the same instruction's
	// completion instant still satisfies done == cost + TheftNsBefore(done)
	// — the sub-millisecond skim never loses a nanosecond
	e := NewEngine(Config{RadarBug: true})
	var done Nanos = -1
	e.Spawn(JobSpec{Name: "J", Prio: 20, Script: Script{
		{Section: "S", Op: "BASIC", Cost: 250 * Microsecond,
			Then: func(en *Engine) { done = en.Now() }},
	}})
	e.RunMS(2)
	if done < 0 {
		t.Fatalf("J never completed")
	}
	if done != 250*Microsecond+e.TheftNsBefore(done) {
		t.Fatalf("J completed at %d, want cost(250µs) + theft(%d) = %d — conservation at tick grain",
			done, e.TheftNsBefore(done), 250*Microsecond+e.TheftNsBefore(done))
	}
}

func TestTheftPerMillisecondExactAcrossTicks(t *testing.T) {
	// happy: the per-millisecond skim total equals the documented waveform
	// value exactly — the ten 100 µs slices telescope without remainder
	e := NewEngine(Config{RadarBug: true})
	e.RunMS(50)
	for ms := Nanos(0); ms < 50; ms++ {
		got := e.TheftNsBefore((ms+1)*Millisecond) - e.TheftNsBefore(ms*Millisecond)
		if got != theftAtMs(ms) {
			t.Fatalf("ms %d skim = %d, want %d exactly", ms, got, theftAtMs(ms))
		}
	}
	// unhappy guard: no single tick may hoard the millisecond's skim —
	// every tick's slice stays within a nanosecond-rounded tenth
	for ms := Nanos(0); ms < 5; ms++ {
		v := theftAtMs(ms)
		for k := Nanos(0); k < 10; k++ {
			slice := e.TheftNsBefore(ms*Millisecond+(k+1)*100*Microsecond) -
				e.TheftNsBefore(ms*Millisecond+k*100*Microsecond)
			lo, hi := v/10, v/10+1
			if slice < lo || slice > hi {
				t.Fatalf("ms %d tick %d slice = %d, want %d..%d", ms, k, slice, lo, hi)
			}
		}
	}
}
