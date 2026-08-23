package sim

// Tests are written FIRST, against the researched behavior of the real AGC
// Executive (Luminary 099). Every test covers a happy AND an unhappy path.
// Sources for every expected number: ../RESEARCH.md

import (
	"testing"
)

// ---------------------------------------------------------------------------
// t1 — wall->AGC time scale: 1000ms wall = 50ms AGC (20x slow motion)
// ---------------------------------------------------------------------------

func TestTimeScaleWallToAGC(t *testing.T) {
	t.Run("happy: default scale is 50ms AGC per 1000ms wall", func(t *testing.T) {
		e := New()
		e.AdvanceWall(1000)
		if got := e.AGCTimeMs(); got < 49.99 || got > 50.01 {
			t.Fatalf("1000ms wall should advance 50ms AGC, got %v", got)
		}
	})
	t.Run("happy: custom scale honored", func(t *testing.T) {
		e := New()
		e.SetWallToAGC(0.1)
		e.AdvanceWall(1000)
		if got := e.AGCTimeMs(); got < 99.9 || got > 100.1 {
			t.Fatalf("want ~100ms AGC, got %v", got)
		}
	})
	t.Run("unhappy: zero and negative wall deltas are no-ops", func(t *testing.T) {
		e := New()
		e.AdvanceWall(500)
		before := e.AGCTimeMs()
		e.AdvanceWall(0)
		e.AdvanceWall(-250)
		e.AdvanceAGC(-10)
		if got := e.AGCTimeMs(); got != before {
			t.Fatalf("time moved on non-positive advance: %v -> %v", before, got)
		}
	})
}

// ---------------------------------------------------------------------------
// t2 — idle baseline: with just the background running, free compute is high
// ---------------------------------------------------------------------------

func TestIdleBaselineFreeCompute(t *testing.T) {
	t.Run("happy: idle leaves >=90% free (T4RUPT+downlink only)", func(t *testing.T) {
		e := New()
		e.AdvanceAGC(5000)
		free := e.FreeComputePercent()
		if free < 90 || free > 100 {
			t.Fatalf("idle free compute should be in [90,100], got %v", free)
		}
	})
	t.Run("unhappy: under maximum load free compute stays in [0,100]", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AcquireLandingRadar()
		e.SetRadarBug(true)
		for _, k := range []byte("V16N68E") {
			e.PressKey(k)
		}
		for i := 0; i < 40; i++ {
			e.AdvanceAGC(500)
			free := e.FreeComputePercent()
			if free < 0 || free > 100 {
				t.Fatalf("free compute out of bounds at step %d: %v", i, free)
			}
		}
		if free := e.FreeComputePercent(); free > 10 {
			t.Fatalf("under overload free compute should be near zero, got %v", free)
		}
	})
}

// ---------------------------------------------------------------------------
// t3 — READACCS punctuality: T3RUPT keeps perfect time regardless of load
// ---------------------------------------------------------------------------

func cycleStarts(e *Engine) []float64 {
	var out []float64
	for _, ev := range e.Events() {
		if ev.Kind == EvCycleStart {
			out = append(out, ev.AGCTimeMs)
		}
	}
	return out
}

func TestReadaccsPunctuality(t *testing.T) {
	t.Run("happy: 2.000s intervals when unloaded", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AdvanceAGC(10001)
		starts := cycleStarts(e)
		if len(starts) < 5 {
			t.Fatalf("want >=5 cycle starts in 10s, got %d", len(starts))
		}
		for i := 1; i < len(starts); i++ {
			d := starts[i] - starts[i-1]
			if d < 1998 || d > 2002 {
				t.Fatalf("interval %d = %vms, want 2000±2", i, d)
			}
		}
	})
	t.Run("unhappy: intervals stay 2.000s even with >100%% demand", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AcquireLandingRadar()
		e.SetRadarBug(true)
		for _, k := range []byte("V16N68E") {
			e.PressKey(k)
		}
		e.AdvanceAGC(6001)
		starts := cycleStarts(e)
		if len(starts) < 3 {
			t.Fatalf("want >=3 cycle starts, got %d", len(starts))
		}
		for i := 1; i < len(starts); i++ {
			d := starts[i] - starts[i-1]
			if d < 1998 || d > 2002 {
				t.Fatalf("overloaded interval %d = %vms, want 2000±2 (T3RUPT is hardware)", i, d)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// t4 — SERVICER allocates one core set + one VAC, frees both at ENDOFJOB
// ---------------------------------------------------------------------------

func countBusy(e *Engine) (cores, vacs int) {
	for _, c := range e.CoreSets() {
		if c.Busy {
			cores++
		}
	}
	for _, v := range e.VACs() {
		if v.Busy {
			vacs++
		}
	}
	return
}

func ownerHeld(e *Engine, name string) (core, vac bool) {
	for _, c := range e.CoreSets() {
		if c.Busy && c.Owner == name {
			core = true
		}
	}
	for _, v := range e.VACs() {
		if v.Busy && v.Owner == name {
			vac = true
		}
	}
	return
}

func TestServicerAllocation(t *testing.T) {
	t.Run("happy: claims exactly one core set and one VAC at priority 20", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AdvanceAGC(5)
		core, vac := ownerHeld(e, "SERVICER")
		if !core || !vac {
			t.Fatalf("SERVICER should hold a core set and a VAC (core=%v vac=%v)", core, vac)
		}
		for _, c := range e.CoreSets() {
			if c.Busy && c.Owner == "SERVICER" && c.Prio != 20 {
				t.Fatalf("SERVICER core set priority = %d, want 20", c.Prio)
			}
		}
	})
	t.Run("happy: ENDOFJOB releases both before the cycle ends", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AdvanceAGC(1985)
		cores, vacs := countBusy(e)
		if cores != 0 || vacs != 0 {
			t.Fatalf("healthy cycle should end with pools free, got cores=%d vacs=%d", cores, vacs)
		}
		if n := e.ServicerCopies(); n != 0 {
			t.Fatalf("no SERVICER copies should be live at cycle end, got %d", n)
		}
	})
}

// ---------------------------------------------------------------------------
// t5 — all 5 VACs busy -> FINDVAC raises 1201
// ---------------------------------------------------------------------------

func TestNoVacBailout1201(t *testing.T) {
	t.Run("unhappy: sixth VAC request bails out with 1201", func(t *testing.T) {
		e := New()
		for i := 0; i < 5; i++ {
			if ok := e.ScheduleJob("HOG", 25, 1e9, true); !ok {
				t.Fatalf("VAC job %d should schedule fine", i)
			}
		}
		if ok := e.ScheduleJob("STRAW", 25, 10, true); ok {
			t.Fatal("sixth FINDVAC should fail")
		}
		alarms := e.Alarms()
		if len(alarms) != 1 || alarms[0].Code != "1201" {
			t.Fatalf("want one 1201 alarm, got %+v", alarms)
		}
		if !e.ProgLamp() {
			t.Fatal("PROG lamp should be lit")
		}
	})
	t.Run("happy: after the restart wipes the pools, scheduling works again", func(t *testing.T) {
		e := New()
		for i := 0; i < 5; i++ {
			e.ScheduleJob("HOG", 25, 1e9, true)
		}
		e.ScheduleJob("STRAW", 25, 10, true) // 1201 + restart
		if ok := e.ScheduleJob("AFTER", 25, 10, true); !ok {
			t.Fatal("restart should have freed all VAC areas")
		}
	})
}

// ---------------------------------------------------------------------------
// t6 — all 8 core sets busy -> 1202
// ---------------------------------------------------------------------------

func TestNoCoreSetBailout1202(t *testing.T) {
	t.Run("unhappy: ninth core-set request bails out with 1202", func(t *testing.T) {
		e := New()
		for i := 0; i < 8; i++ {
			if ok := e.ScheduleJob("HOG", 25, 1e9, false); !ok {
				t.Fatalf("core-set job %d should schedule fine", i)
			}
		}
		if ok := e.ScheduleJob("STRAW", 25, 10, false); ok {
			t.Fatal("ninth core-set request should fail")
		}
		alarms := e.Alarms()
		if len(alarms) != 1 || alarms[0].Code != "1202" {
			t.Fatalf("want one 1202 alarm, got %+v", alarms)
		}
		if e.RestartCount() != 1 {
			t.Fatalf("BAILOUT should trigger one software restart, got %d", e.RestartCount())
		}
	})
	t.Run("happy: pools are usable again after the restart", func(t *testing.T) {
		e := New()
		for i := 0; i < 8; i++ {
			e.ScheduleJob("HOG", 25, 1e9, false)
		}
		e.ScheduleJob("STRAW", 25, 10, false)
		if ok := e.ScheduleJob("AFTER", 25, 10, false); !ok {
			t.Fatal("restart should have freed all core sets")
		}
	})
}

// ---------------------------------------------------------------------------
// t7 — the rendezvous radar bug steals ~15% (2 x 6400 x 11.72us)
// ---------------------------------------------------------------------------

func TestRadarBugTLOSS(t *testing.T) {
	t.Run("happy: bug on steals ~15% of AGC time", func(t *testing.T) {
		e := New()
		e.SetRadarBug(true)
		e.AdvanceAGC(2000)
		steal := e.Accounting().StealPct
		if steal < 14 || steal > 16 {
			t.Fatalf("radar bug should steal ~15%%, got %v", steal)
		}
	})
	t.Run("happy: bug off steals ~0%", func(t *testing.T) {
		e := New()
		e.AdvanceAGC(2000)
		if steal := e.Accounting().StealPct; steal > 1 {
			t.Fatalf("no bug: steal should be ~0, got %v", steal)
		}
	})
	t.Run("unhappy: stealing does not delay the hardware timer", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.SetRadarBug(true)
		e.AdvanceAGC(6001)
		starts := cycleStarts(e)
		for i := 1; i < len(starts); i++ {
			if d := starts[i] - starts[i-1]; d < 1998 || d > 2002 {
				t.Fatalf("steal delayed T3RUPT: interval %v", d)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// t8 — priority preemption: CHARIN(30) preempts SERVICER(20), never reverse
// ---------------------------------------------------------------------------

func TestPriorityPreemption(t *testing.T) {
	t.Run("happy: keystroke job preempts SERVICER", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AdvanceAGC(150)
		if got := e.RunningJob(); got != "SERVICER" {
			t.Fatalf("SERVICER should be running at t=150ms, got %q", got)
		}
		e.PressKey('V')
		e.AdvanceAGC(1)
		if got := e.RunningJob(); got != "CHARIN" {
			t.Fatalf("CHARIN (prio 30) should preempt SERVICER (prio 20), got %q", got)
		}
		e.AdvanceAGC(10)
		if got := e.RunningJob(); got != "SERVICER" {
			t.Fatalf("SERVICER should resume after CHARIN ends, got %q", got)
		}
	})
	t.Run("unhappy: lower priority never preempts a running higher one", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AdvanceAGC(150)
		e.ScheduleJob("LOWLY", 5, 50, false)
		e.AdvanceAGC(1)
		if got := e.RunningJob(); got != "SERVICER" {
			t.Fatalf("prio 5 should not preempt prio 20, got %q", got)
		}
		e.PressKey('V')
		e.AdvanceAGC(0.5)
		e.ScheduleJob("MID", 25, 50, false)
		e.AdvanceAGC(1)
		if got := e.RunningJob(); got != "CHARIN" {
			t.Fatalf("prio 25 should not preempt prio 30 CHARIN, got %q", got)
		}
	})
}

// ---------------------------------------------------------------------------
// t9 — duty >100% accumulates SERVICER stubs; <100% reaches steady state
// ---------------------------------------------------------------------------

func TestServicerOverrunLeak(t *testing.T) {
	t.Run("happy: healthy descent keeps at most one live SERVICER", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AdvanceAGC(20000)
		if n := e.ServicerCopies(); n > 1 {
			t.Fatalf("healthy: want <=1 live SERVICER, got %d", n)
		}
		if len(e.Alarms()) != 0 {
			t.Fatalf("healthy: want no alarms, got %+v", e.Alarms())
		}
	})
	t.Run("unhappy: overload leaks roughly one stub per 2s cycle", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AcquireLandingRadar()
		e.SetRadarBug(true)
		for _, k := range []byte("V16N68E") {
			e.PressKey(k)
		}
		e.AdvanceAGC(6500)
		if n := e.ServicerCopies(); n < 3 {
			t.Fatalf("overload: want >=3 live SERVICER copies after 3+ cycles, got %d", n)
		}
	})
}

// ---------------------------------------------------------------------------
// t10 — BAILOUT restart: FAILREG, PROG lamp, pools freed, one SERVICER
//        rebuilt, monitor dropped in P63; P64 cannot shed -> alarms recur
// ---------------------------------------------------------------------------

func runUntilAlarm(t *testing.T, e *Engine, capMs float64) float64 {
	t.Helper()
	start := len(e.Alarms())
	for spent := 0.0; spent < capMs; spent += 500 {
		e.AdvanceAGC(500)
		if len(e.Alarms()) > start {
			return e.Alarms()[len(e.Alarms())-1].AGCTimeMs
		}
	}
	t.Fatalf("no alarm within %vms", capMs)
	return 0
}

func overloadedP63(t *testing.T) *Engine {
	t.Helper()
	e := New()
	e.StartDescent()
	e.AcquireLandingRadar()
	e.SetRadarBug(true)
	for _, k := range []byte("V16N68E") {
		e.PressKey(k)
	}
	return e
}

func TestBailoutRestartRecovery(t *testing.T) {
	t.Run("happy: P63 restart sheds the monitor and rebuilds one SERVICER", func(t *testing.T) {
		e := overloadedP63(t)
		if !e.MonitorActive() {
			t.Fatal("monitor should be up before the alarm")
		}
		runUntilAlarm(t, e, 60000)
		if got := len(e.FailReg()); got < 1 {
			t.Fatalf("FAILREG should hold the code, got %d entries", got)
		}
		if !e.ProgLamp() {
			t.Fatal("PROG lamp should be lit")
		}
		if e.RestartCount() < 1 {
			t.Fatal("software restart should have run")
		}
		if e.MonitorActive() {
			t.Fatal("restart must drop the unprotected V16N68 monitor")
		}
		e.AdvanceAGC(300)
		if n := e.ServicerCopies(); n != 1 {
			t.Fatalf("phase tables rebuild exactly one SERVICER, got %d", n)
		}
		cores, vacs := countBusy(e)
		if cores > 3 || vacs > 1 {
			t.Fatalf("pools should be nearly clean after restart, cores=%d vacs=%d", cores, vacs)
		}
	})
	t.Run("unhappy: P64 load is protected, so alarms keep recurring", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AcquireLandingRadar()
		e.SetRadarBug(true)
		e.EnterP64()
		before := len(e.Alarms())
		e.AdvanceAGC(60000)
		got := len(e.Alarms()) - before
		if got < 2 {
			t.Fatalf("P64 with the bug should alarm repeatedly (flight: 3 in 40s), got %d in 60s", got)
		}
		if len(e.FailReg()) > 3 {
			t.Fatalf("FAILREG has only 3 slots, got %d", len(e.FailReg()))
		}
	})
}

// ---------------------------------------------------------------------------
// t11 — every keystroke costs real compute (KEYRUPT + CHARIN + display)
// ---------------------------------------------------------------------------

func TestKeystrokeCost(t *testing.T) {
	t.Run("happy: a keypress runs a prio-30 CHARIN job holding a core set", func(t *testing.T) {
		e := New()
		e.PressKey('9')
		e.AdvanceAGC(1)
		if got := e.RunningJob(); got != "CHARIN" {
			t.Fatalf("CHARIN should be running after keypress, got %q", got)
		}
		core, _ := ownerHeld(e, "CHARIN")
		if !core {
			t.Fatal("CHARIN must hold a core set")
		}
		if e.PendingDSPTAB() < 1 {
			t.Fatal("keystroke should queue DSKY display (DSPTAB) updates")
		}
		e.AdvanceAGC(10)
		if core, _ := ownerHeld(e, "CHARIN"); core {
			t.Fatal("CHARIN should release its core set when done")
		}
		e.AdvanceAGC(600)
		if e.PendingDSPTAB() != 0 {
			t.Fatalf("T4RUPT should drain DSPTAB within ~0.5s, still %d", e.PendingDSPTAB())
		}
	})
	t.Run("unhappy: typing with all core sets busy raises 1202", func(t *testing.T) {
		e := New()
		for i := 0; i < 8; i++ {
			e.ScheduleJob("HOG", 25, 1e9, false)
		}
		e.PressKey('V')
		alarms := e.Alarms()
		if len(alarms) == 0 || alarms[len(alarms)-1].Code != "1202" {
			t.Fatalf("keystroke into a full Executive should 1202, got %+v", alarms)
		}
	})
}

// ---------------------------------------------------------------------------
// t12 — V16N68 monitor: ~3% duty at 1Hz; killed by restart
// ---------------------------------------------------------------------------

func TestMonitorVerbLoad(t *testing.T) {
	t.Run("happy: keying V16N68 adds roughly 3% duty", func(t *testing.T) {
		e := New()
		e.AdvanceAGC(5000)
		before := e.FreeComputePercent()
		for _, k := range []byte("V16N68E") {
			e.PressKey(k)
		}
		if !e.MonitorActive() {
			t.Fatal("V16N68 ENTR should start the monitor")
		}
		e.AdvanceAGC(10000)
		after := e.FreeComputePercent()
		drop := before - after
		if drop < 2 || drop > 6 {
			t.Fatalf("monitor should cost ~3%% (Eyles: margin 13%%->~10%%), got %.2f%%", drop)
		}
	})
	t.Run("unhappy: a software restart silently kills the monitor", func(t *testing.T) {
		e := New()
		for _, k := range []byte("V16N68E") {
			e.PressKey(k)
		}
		e.AdvanceAGC(100)
		for i := 0; i < 8; i++ {
			e.ScheduleJob("HOG", 25, 1e9, false)
		}
		e.ScheduleJob("STRAW", 25, 10, false) // 1202 + restart
		if e.MonitorActive() {
			t.Fatal("monitor is not restart-protected; it must vanish")
		}
	})
}

// ---------------------------------------------------------------------------
// t13 — radar ping: one-shot prio-32 burst job
// ---------------------------------------------------------------------------

func TestRadarPing(t *testing.T) {
	t.Run("happy: ping runs RR READ at prio 32 and frees resources", func(t *testing.T) {
		e := New()
		e.PingRadar()
		e.AdvanceAGC(1)
		if got := e.RunningJob(); got != "RR READ" {
			t.Fatalf("RR READ should run after ping, got %q", got)
		}
		for _, c := range e.CoreSets() {
			if c.Busy && c.Owner == "RR READ" && c.Prio != 32 {
				t.Fatalf("RR READ priority = %d, want 32", c.Prio)
			}
		}
		e.AdvanceAGC(200)
		if core, _ := ownerHeld(e, "RR READ"); core {
			t.Fatal("RR READ should complete and free its core set")
		}
		if len(e.Alarms()) != 0 {
			t.Fatal("a lone ping should never alarm")
		}
	})
	t.Run("unhappy: ping with exhausted pools raises an alarm", func(t *testing.T) {
		e := New()
		for i := 0; i < 8; i++ {
			e.ScheduleJob("HOG", 25, 1e9, false)
		}
		e.PingRadar()
		alarms := e.Alarms()
		if len(alarms) == 0 {
			t.Fatal("ping into a full Executive should alarm")
		}
	})
}

// ---------------------------------------------------------------------------
// t14 — the historical Apollo 11 scenario
// ---------------------------------------------------------------------------

func TestHistoricalScenario(t *testing.T) {
	t.Run("happy: P63+LR+V16N68+bug alarms 8-40s after the bug appears", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AdvanceAGC(2000)
		e.AcquireLandingRadar()
		e.AdvanceAGC(2000)
		// Aldrin types at a human pace: one key every ~200ms.
		for _, k := range []byte("V16N68E") {
			e.PressKey(k)
			e.AdvanceAGC(200)
		}
		e.AdvanceAGC(4000)
		if len(e.Alarms()) != 0 {
			t.Fatalf("no alarms should fire before the bug, got %+v", e.Alarms())
		}
		t0 := e.AGCTimeMs()
		e.SetRadarBug(true)
		at := runUntilAlarm(t, e, 60000)
		dt := at - t0
		if dt < 8000 || dt > 40000 {
			t.Fatalf("first alarm %vms after bug; flight was ~12s after V16N68, want 8-40s", dt)
		}
		code := e.Alarms()[0].Code
		if code != "1201" && code != "1202" {
			t.Fatalf("alarm code %q not an executive overflow code", code)
		}
		// P63 recovery: the restart shed the monitor, so with the marginal
		// budget the computer breathes again (flight: alarms only returned
		// when the crew re-keyed the monitor).
		preAlarms := len(e.Alarms())
		e.AdvanceAGC(6000)
		if got := len(e.Alarms()); got != preAlarms {
			t.Fatalf("P63 after shedding should hold for a while, got %d new alarms", got-preAlarms)
		}
	})
	t.Run("unhappy control: same flight without the bug stays clean for 120s", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.AcquireLandingRadar()
		for _, k := range []byte("V16N68E") {
			e.PressKey(k)
		}
		e.AdvanceAGC(120000)
		if len(e.Alarms()) != 0 {
			t.Fatalf("without the RR bug there must be no alarms, got %+v", e.Alarms())
		}
	})
}

// ---------------------------------------------------------------------------
// t15 — free-compute accounting adds up
// ---------------------------------------------------------------------------

func TestFreeComputeAccounting(t *testing.T) {
	sum := func(a Accounting) float64 {
		return a.JobsPct + a.InterruptsPct + a.StealPct + a.RestartPct + a.IdlePct
	}
	t.Run("happy: idle accounting sums to 100", func(t *testing.T) {
		e := New()
		e.AdvanceAGC(5000)
		if s := sum(e.Accounting()); s < 99 || s > 101 {
			t.Fatalf("accounting sums to %v, want ~100", s)
		}
	})
	t.Run("happy: descent-with-bug accounting sums to 100", func(t *testing.T) {
		e := New()
		e.StartDescent()
		e.SetRadarBug(true)
		e.AdvanceAGC(8000)
		if s := sum(e.Accounting()); s < 99 || s > 101 {
			t.Fatalf("accounting sums to %v, want ~100", s)
		}
	})
	t.Run("unhappy: overload shows ~0 free and a positive deficit", func(t *testing.T) {
		e := overloadedP63(t)
		e.AdvanceAGC(8000)
		a := e.Accounting()
		if a.IdlePct > 1.5 {
			t.Fatalf("overload should leave ~no idle, got %v", a.IdlePct)
		}
		if a.DeficitPct <= 0 {
			t.Fatalf("overload must report a positive deficit, got %v", a.DeficitPct)
		}
		if free := e.FreeComputePercent(); free > 1.5 {
			t.Fatalf("free compute should be ~0 under overload, got %v", free)
		}
	})
}
