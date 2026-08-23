package sim

// t34-t40 — flight-fidelity mechanics that the first engine model missed.
// Written FIRST, against the sources:
//
//   [Cherry] job table: LRHJOB/LRVJOB "only run for a millisecond or so and
//   then sleep for about 80 milliseconds while the LR sync pulses are sent
//   out. They are awakened when LR reading is completed and then run for
//   another millisecond or so." HIGATJOB (VAC) "Sleeps until position #2
//   discrete is received."
//   [L099] SERVICER.agc: LRHTASK reads altitude "50 MS PRIOR TO THE NEXT
//   READACCS TASK"; LRVJOB takes "5 VELOCITY SAMPLES AND GOES TO SLEEP WHILE
//   THE SAMPLING IS DONE -- ABOUT 500 MS."
//   [L099] PINBALL: jobs wanting a busy display GO TO SLEEP holding their
//   core set (alarm 1206 exists precisely for a second display sleeper).
//   Flight record: both P63 alarms were 1202 (no core sets) and came during
//   heavy DSKY activity; the first P64 alarm was 1201 (no VAC areas).
//
// Sleeping jobs are the missing core-set/VAC pressure: they hold memory
// without consuming CPU, which is what let the flight hit the 8-core wall.

import (
	"testing"
)

// ---------------------------------------------------------------------------
// t34 — sleep semantics: a sleeping job holds its memory but not the CPU
// ---------------------------------------------------------------------------

func TestSleepingJobHoldsResources(t *testing.T) {
	t.Run("happy: LRVJOB sleeps ~500ms holding a core set, then finishes and frees it", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AcquireLandingRadar()
		// LRV is scheduled mid-cycle; walk to just after its head segment.
		e.AdvanceAGC(1105)
		core, _ := ownerHeld(e, "LRVJOB")
		if !core {
			t.Fatal("LRVJOB should hold a core set during its radar-gate sleep")
		}
		if got := e.RunningJob(); got == "LRVJOB" {
			t.Fatal("a sleeping LRVJOB must not own the CPU")
		}
		e.AdvanceAGC(600) // wake + tail complete
		if core, _ := ownerHeld(e, "LRVJOB"); core {
			t.Fatal("LRVJOB must free its core set after wake+tail")
		}
	})
	t.Run("unhappy: sleeping jobs are not selectable by the Executive scan", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AcquireLandingRadar()
		e.AdvanceAGC(1105) // LRV asleep at prio 32
		if got := e.RunningJob(); got != "SERVICER" {
			t.Fatalf("SERVICER should run while the higher-prio LRVJOB sleeps, got %q", got)
		}
	})
}

// ---------------------------------------------------------------------------
// t35 — LR read cadence: per 2s cycle, LRH straddling the READACCS boundary
// ---------------------------------------------------------------------------

func TestLRReadCadence(t *testing.T) {
	t.Run("happy: LRHJOB is asleep across the cycle boundary (fired 50ms before READACCS)", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AcquireLandingRadar()
		e.AdvanceAGC(2010) // 10ms past the second READACCS
		core, _ := ownerHeld(e, "LRHJOB")
		if !core {
			t.Fatal("LRHJOB (fired at boundary-50ms, 80ms radar gate) must hold a core set across the boundary")
		}
		e.AdvanceAGC(100)
		if core, _ := ownerHeld(e, "LRHJOB"); core {
			t.Fatal("LRHJOB must complete and free its core set ~80ms after the boundary")
		}
	})
	t.Run("unhappy: without LR lock neither LR job ever runs", func(t *testing.T) {
		e := New()
		e.StartDescent()
		for i := 0; i < 16; i++ {
			e.AdvanceAGC(250)
			if c1, _ := ownerHeld(e, "LRHJOB"); c1 {
				t.Fatal("LRHJOB must not run before the landing radar locks")
			}
			if c2, _ := ownerHeld(e, "LRVJOB"); c2 {
				t.Fatal("LRVJOB must not run before the landing radar locks")
			}
		}
	})
}

// ---------------------------------------------------------------------------
// t36 — LR lock costs ~4% duty (conversion inside SERVICER dominates; the
//        read jobs are ~2ms CPU each, not a 20ms burst)
// ---------------------------------------------------------------------------

func TestLRLockDutyCost(t *testing.T) {
	t.Run("happy: LR lock lowers free compute by ~4%", func(t *testing.T) {
		e := New()
		e.AdvanceAGC(170)
		e.StartDescent()
		e.AdvanceAGC(8000)
		before := e.FreeComputePercent()
		e.AcquireLandingRadar()
		e.AdvanceAGC(8000)
		drop := before - e.FreeComputePercent()
		if drop < 2.5 || drop > 5.5 {
			t.Fatalf("LR lock should cost ~4%% duty, got %.2f%%", drop)
		}
	})
	t.Run("unhappy: the LR read jobs alone are tiny — most of the cost is in SERVICER", func(t *testing.T) {
		e := New()
		e.AdvanceAGC(170)
		e.StartDescent()
		e.AcquireLandingRadar()
		e.AdvanceAGC(8000)
		sums, total := e.windowUse()
		if total <= 0 {
			t.Fatal("no accounting window")
		}
		lrPct := float64(sums[CLRRead]) / total * 100
		if lrPct > 1.0 {
			t.Fatalf("LR read jobs should be <1%% CPU (they sleep, not spin), got %.2f%%", lrPct)
		}
	})
}

// ---------------------------------------------------------------------------
// t37 — HIGATJOB: P64 entry claims a VAC and sleeps on the antenna discrete
// ---------------------------------------------------------------------------

func TestHigatjobVACHold(t *testing.T) {
	t.Run("happy: entering P64 parks HIGATJOB on a VAC for several seconds", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AcquireLandingRadar()
		e.AdvanceAGC(100)
		e.EnterP64()
		e.AdvanceAGC(50)
		_, vac := ownerHeld(e, "HIGATJOB")
		if !vac {
			t.Fatal("HIGATJOB must hold a VAC area while awaiting the position-2 discrete")
		}
		if got := e.RunningJob(); got == "HIGATJOB" {
			t.Fatal("HIGATJOB sleeps on the discrete; it must not own the CPU")
		}
		e.AdvanceAGC(10000)
		if _, vac := ownerHeld(e, "HIGATJOB"); vac {
			t.Fatal("HIGATJOB must complete and free its VAC once the antenna reaches position 2")
		}
	})
	t.Run("unhappy: a software restart does not resurrect HIGATJOB", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AcquireLandingRadar()
		e.SetRadarBug(true)
		e.EnterP64()
		runUntilAlarm(t, e, 60000)
		e.AdvanceAGC(200)
		if _, vac := ownerHeld(e, "HIGATJOB"); vac {
			t.Fatal("HIGATJOB is a one-shot; the restart must not rebuild it")
		}
	})
}

// ---------------------------------------------------------------------------
// t38 — flight alarm codes: P64's first alarm is 1201 (no VAC areas)
// ---------------------------------------------------------------------------

func TestP64FirstAlarmIs1201(t *testing.T) {
	t.Run("happy: P64 with the bug alarms 1201 first, like 102:42:17", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.SetRadarBug(true)
		e.AcquireLandingRadar()
		e.AdvanceAGC(4000)
		e.EnterP64()
		runUntilAlarm(t, e, 60000)
		if code := e.Alarms()[0].Code; code != "1201" {
			t.Fatalf("first P64 alarm should be 1201 (no VAC areas), got %s", code)
		}
	})
	t.Run("unhappy: P64 without the bug never alarms", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AcquireLandingRadar()
		e.AdvanceAGC(4000)
		e.EnterP64()
		e.AdvanceAGC(60000)
		if n := len(e.Alarms()); n != 0 {
			t.Fatalf("P64 without TLOSS must stay clean, got %d alarms", n)
		}
	})
}

// ---------------------------------------------------------------------------
// t39 — flight alarm codes: DSKY typing during the P63 overload gives 1202
// ---------------------------------------------------------------------------

// typeDuringOverload replays the flight's DSKY pattern: the monitor keyed
// up, then continued crew keying (V57E, V16N68E re-entries) while the theft
// runs. Keys land mid-cycle, where the LRV radar gate and the monitor's
// display wait already hold core sets.
func typeDuringOverload(t *testing.T, e *Engine) {
	t.Helper()
	for _, base := range []float64{8000, 10000, 12000, 14000, 16000, 18000} {
		for _, off := range []float64{1240, 1360} {
			if dt := base + off - e.AGCTimeMs(); dt > 0 {
				e.AdvanceAGC(dt)
			}
			if len(e.Alarms()) > 0 {
				return
			}
			e.PressKey('V')
		}
	}
}

func TestP63TypingGives1202(t *testing.T) {
	t.Run("happy: typing while overloaded exhausts core sets first — 1202", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.SetRadarBug(true)
		e.AcquireLandingRadar()
		e.AdvanceAGC(4000)
		for _, k := range []byte("V16N68E") { // Aldrin keys the monitor
			e.PressKey(k)
			e.AdvanceAGC(200)
		}
		typeDuringOverload(t, e)
		if len(e.Alarms()) == 0 {
			t.Fatal("overload with typing must alarm")
		}
		if code := e.Alarms()[0].Code; code != "1202" {
			t.Fatalf("P63 overload with DSKY activity should give the historical 1202, got %s", code)
		}
	})
	t.Run("unhappy: same typing without the bug stays clean", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AcquireLandingRadar()
		e.AdvanceAGC(4000)
		for _, k := range []byte("V16N68E") {
			e.PressKey(k)
			e.AdvanceAGC(200)
		}
		typeDuringOverload(t, e)
		if n := len(e.Alarms()); n != 0 {
			t.Fatalf("typing without TLOSS must not alarm, got %d: %+v", n, e.Alarms())
		}
	})
}

// ---------------------------------------------------------------------------
// t40 — no alignment artifact: descent started exactly on a 1s boundary at
//        the knife edge must behave like the desynced case (flight was quiet
//        for ~5 minutes before the monitor went up)
// ---------------------------------------------------------------------------

// t41b — UsedMs: the per-consumer trailing-2s accounting the UI shows next
// to each timeline row.
func TestUsedMs(t *testing.T) {
	t.Run("happy: an idle machine charges the window to CIdle", func(t *testing.T) {
		e := New()
		e.AdvanceAGC(4100)
		if got := e.UsedMs(CIdle); got < 1900 {
			t.Fatalf("idle should hold ~2000ms of the window, got %v", got)
		}
	})
	t.Run("happy: during descent SERVICER dominates the window", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AdvanceAGC(4100)
		if got := e.UsedMs(CServicer); got < 1000 {
			t.Fatalf("SERVICER should exceed 1000ms per window, got %v", got)
		}
	})
	t.Run("unhappy: a fresh engine reports zero everywhere", func(t *testing.T) {
		e := New()
		for c := Consumer(0); c < numConsumers; c++ {
			if e.UsedMs(c) != 0 {
				t.Fatalf("consumer %v must start at zero", c)
			}
		}
	})
}

func TestBoundaryAlignmentNoFalseAlarm(t *testing.T) {
	t.Run("happy: aligned knife edge stays alarm-free for 120s", func(t *testing.T) {
		e := New()
		e.StartDescent() // t=0: READACCS boundary == the 1Hz timer marks
		e.SetRadarBug(true)
		e.AcquireLandingRadar()
		e.AdvanceAGC(120000)
		if n := len(e.Alarms()); n != 0 {
			t.Fatalf("aligned knife edge must not alarm (flight was quiet pre-monitor), got %d: %+v", n, e.Alarms())
		}
	})
	t.Run("unhappy: adding the monitor still overloads the aligned case", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.SetRadarBug(true)
		e.AcquireLandingRadar()
		e.AdvanceAGC(4000)
		for _, k := range []byte("V16N68E") {
			e.PressKey(k)
		}
		runUntilAlarm(t, e, 60000) // fails the test if it never alarms
	})
}
