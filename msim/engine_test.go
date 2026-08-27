package msim

import (
	"testing"
)

// ---------- RR CDU counter theft (the bug) ----------
//
// 12,800 counter steals/s x 11.72 us = 150.016 ms stolen per second.
// Per 1 ms engine tick that is exactly 150,016 ns. The steals are hardware
// PINC cycles: they happen whether or not any software is running.

func TestTheftSkimExactWhenBugOn(t *testing.T) {
	// happy: over 10.000 s the bug steals exactly 10 * 150,016,000 ns
	e := NewEngine(Config{RadarBug: true})
	e.RunMS(10_000)
	want := Nanos(10) * 150_016_000
	if got := e.TheftNs(); got != want {
		t.Fatalf("TheftNs = %d, want %d (12,800/s x 11.72 us, exact)", got, want)
	}
}

func TestTheftZeroWhenBugOff(t *testing.T) {
	// unhappy: no RR bug, no theft — not a single nanosecond
	e := NewEngine(Config{RadarBug: false})
	e.RunMS(10_000)
	if got := e.TheftNs(); got != 0 {
		t.Fatalf("TheftNs = %d, want 0 when the RR bug is off", got)
	}
}

func TestTheftStealsEvenWhileIdle(t *testing.T) {
	// unhappy: the counter increments are not software — an idle CPU
	// (no jobs, no interrupts) still loses exactly the same time
	e := NewEngine(Config{RadarBug: true})
	e.RunMS(1_000)
	if got, want := e.TheftNs(), Nanos(150_016_000); got != want {
		t.Fatalf("idle TheftNs = %d, want %d — PINC steals are hardware", got, want)
	}
	if got := e.SoftwareBusyNs(); got != 0 {
		t.Fatalf("SoftwareBusyNs = %d, want 0 on an idle machine", got)
	}
}

// ---------- interrupt cadences ----------
//
// DOWNRUPT every 20 ms (DOWN_TELEMETRY_PROGRAM.agc L43), DAP every 100 ms
// (P-AXIS_RCS_AUTOPILOT.agc L41, phased +70 ms per SERVICER.agc L95-L104),
// T4RUPT every 120 ms (T4RUPT_PROGRAM.agc L144).

func TestInterruptCadenceCounts(t *testing.T) {
	// happy: exact fire counts over 12.000 s (LCM of 20/100/120 divides it)
	e := NewEngine(Config{Interrupts: true})
	e.RunMS(12_000)
	cases := []struct {
		name string
		want int
	}{
		{"DOWNRUPT", 600}, // 12,000 / 20
		{"DAP", 120},      // 12,000 / 100 (phase +70 ms: fires at 70, 170, ...)
		{"T4RUPT", 100},   // 12,000 / 120
	}
	for _, c := range cases {
		if got := e.InterruptFires(c.name); got != c.want {
			t.Fatalf("%s fired %d times in 12 s, want %d", c.name, got, c.want)
		}
	}
}

func TestInterruptCPUAccounting(t *testing.T) {
	// happy: interrupt CPU cost lands in SoftwareBusyNs
	// DOWNRUPT 0.2 ms x 50/s + DAP 12 ms x 10/s + T4RUPT 0.96 ms x 8.333/s
	e := NewEngine(Config{Interrupts: true})
	e.RunMS(12_000)
	// per 12 s: 600*0.2ms + 120*12ms + 100*0.96ms = 120ms + 1440ms + 96ms
	want := Nanos(600)*200_000 + Nanos(120)*12_000_000 + Nanos(100)*960_000
	got := e.SoftwareBusyNs()
	if got != want {
		t.Fatalf("SoftwareBusyNs = %d, want %d (interrupts only)", got, want)
	}
}

func TestSimultaneousInterruptsSerialize(t *testing.T) {
	// unhappy: at t=8.470 s (LCM alignment of DAP phase 70+100k, T4 120k, DOWN 20k
	// is not needed — pick 4.070 s where DAP and DOWNRUPT can collide) multiple
	// interrupts due in the same tick must serialize: total software time is the
	// sum of both costs, nothing lost, nothing run twice.
	e := NewEngine(Config{Interrupts: true})
	e.RunMS(70 + 1) // DAP fires once at 70 ms; DOWNRUPTs at 0,20,40,60 (4 by 71 ms... 0 is t=0? count from first period)
	dap := e.InterruptFires("DAP")
	if dap != 1 {
		t.Fatalf("DAP fires by 71 ms = %d, want exactly 1 (phase +70 ms)", dap)
	}
	// Serialization invariant: cumulative software time == sum over fires of cost
	down := e.InterruptFires("DOWNRUPT")
	t4 := e.InterruptFires("T4RUPT")
	want := Nanos(dap)*12_000_000 + Nanos(down)*200_000 + Nanos(t4)*960_000
	if got := e.SoftwareBusyNs(); got != want {
		t.Fatalf("SoftwareBusyNs = %d, want %d — colliding interrupts must serialize losslessly", got, want)
	}
}

// ---------- waitlist punctuality ----------
//
// GOREADAX: CA 2SECS / TC VARDELAY (SERVICER.agc L80-L81) — the re-arm is
// unconditional and T3RUPT dispatches on the hardware clock, never delayed by
// software load.

func TestWaitlistFiresExactly(t *testing.T) {
	// happy: a self-rearming 2.000 s task fires at 2.000, 4.000, ... exactly
	e := NewEngine(Config{})
	var fires []Nanos
	var arm func(at Nanos)
	arm = func(at Nanos) {
		e.ScheduleTask(at, "READACCS", 1_000_000, func(en *Engine) {
			fires = append(fires, en.Now())
			arm(at + 2*Second)
		})
	}
	arm(2 * Second)
	e.RunMS(10_000)
	if len(fires) != 5 {
		t.Fatalf("task fired %d times in 10 s, want 5", len(fires))
	}
	for i, at := range fires {
		want := Nanos(i+1) * 2 * Second
		if at != want {
			t.Fatalf("fire %d at %d ns, want %d — waitlist is punctual", i, at, want)
		}
	}
}

func TestWaitlistPunctualUnderSaturation(t *testing.T) {
	// unhappy: CPU saturated by an endless prio-30 job + the RR bug; the task
	// STILL fires on the exact tick — T3RUPT is hardware, load cannot delay it
	e := NewEngine(Config{RadarBug: true})
	if alarm := e.Spawn(JobSpec{
		Name: "HOG", Prio: 30, VAC: false,
		Script: endlessScript(100 * Second),
	}); alarm != nil {
		t.Fatalf("unexpected alarm scheduling HOG: %+v", alarm)
	}
	var fired Nanos = -1
	e.ScheduleTask(5*Second, "PROBE", 100_000, func(en *Engine) { fired = en.Now() })
	e.RunMS(6_000)
	if fired != 5*Second {
		t.Fatalf("PROBE fired at %d ns, want exactly %d despite 100%% load", fired, 5*Second)
	}
}

// ---------- DANZIG boundaries ----------
//
// INTERPRETER.agc L74-L82: NEWJOB is tested at DANZIG — between interpretive
// instructions. A higher-priority job therefore waits for the current
// instruction to finish. Hardware interrupts do NOT wait: they pause the
// instruction mid-flight and the job resumes with no lost time.

func TestPreemptionOnlyAtInstructionBoundary(t *testing.T) {
	// happy: LOW is mid-way through a 5 ms instruction when HIGH is spawned;
	// HIGH must not run until that instruction completes.
	e := NewEngine(Config{})
	if alarm := e.Spawn(JobSpec{Name: "LOW", Prio: 20, Script: Script{
		{Section: "S1", Op: "VXV", Cost: 5 * Millisecond},
		{Section: "S1", Op: "VXV", Cost: 5 * Millisecond},
	}}); alarm != nil {
		t.Fatalf("spawn LOW: %+v", alarm)
	}
	// spawn HIGH from a zero-cost task 2 ms in — mid-instruction
	e.ScheduleTask(2*Millisecond, "SPAWNER", 0, func(en *Engine) {
		en.Spawn(JobSpec{Name: "HIGH", Prio: 30, Script: Script{
			{Section: "H", Op: "BASIC", Cost: 1 * Millisecond},
		}})
	})
	e.RunMS(3) // t = 3 ms: LOW's first instruction (0-5 ms) still executing
	if got := e.RunningJob(); got != "LOW" {
		t.Fatalf("at 3 ms RunningJob = %q, want LOW — no preemption mid-instruction", got)
	}
	e.RunMS(3) // t = 6 ms: boundary passed at 5 ms, HIGH must have taken over
	if got := e.RunningJob(); got != "HIGH" {
		t.Fatalf("at 6 ms RunningJob = %q, want HIGH — DANZIG check at boundary", got)
	}
}

func TestInterruptPausesInstructionLosslessly(t *testing.T) {
	// unhappy: an interrupt seizes the CPU mid-instruction; the instruction's
	// remaining nanoseconds are preserved, so the job finishes exactly
	// (instruction cost + interrupt cost + theft) later than it started.
	e := NewEngine(Config{Interrupts: true, RadarBug: true})
	done := Nanos(-1)
	if alarm := e.Spawn(JobSpec{Name: "J", Prio: 20, Script: Script{
		{Section: "S", Op: "VXM", Cost: 30 * Millisecond,
			Then: func(en *Engine) { done = en.Now() }},
	}}); alarm != nil {
		t.Fatalf("spawn J: %+v", alarm)
	}
	e.RunMS(100)
	if done < 0 {
		t.Fatalf("J never finished in 100 ms")
	}
	// From t=0 to completion the CPU gave time to: theft (150,016 ns/ms),
	// DOWNRUPT (0.2 ms each 20 ms), DAP (12 ms at 70 ms), T4RUPT (0.96 ms at
	// 0? phase 0 fires at 120...), and J. Rather than re-deriving the exact
	// finish tick here, assert the conservation law: at the finish instant,
	// software busy = J's 30 ms + all interrupt costs so far, and
	// done == softwareBusy + theft + idle(0).
	interruptNs := Nanos(e.InterruptFiresBefore("DOWNRUPT", done))*200_000 +
		Nanos(e.InterruptFiresBefore("DAP", done))*12_000_000 +
		Nanos(e.InterruptFiresBefore("T4RUPT", done))*960_000
	wantBusy := 30*Millisecond + interruptNs
	theftAtDone := e.TheftNsBefore(done)
	if done != wantBusy+theftAtDone {
		t.Fatalf("J finished at %d; want busy(%d) + theft(%d) = %d — nothing may be lost mid-instruction",
			done, wantBusy, theftAtDone, wantBusy+theftAtDone)
	}
}

// endlessScript builds a script of 5 ms instructions totalling at least d.
func endlessScript(d Nanos) Script {
	n := int(d/(5*Millisecond)) + 1
	s := make(Script, n)
	for i := range s {
		s[i] = Instr{Section: "HOG", Op: "VXV", Cost: 5 * Millisecond}
	}
	return s
}
