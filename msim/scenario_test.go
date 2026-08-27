package msim

import (
	"testing"
)

// ---------- the two timelines ----------
//
// Window: t=0 is PDI+290 s (GET 102:37:55), just after landing-radar lock.
// Baseline: P63 radar-locked, RR bug stealing, nobody touches the DSKY.
// 1668: Aldrin keys V16N68 at the flight offsets —
//   ENTR at t=15.8 s  (PDI+304-306, Cherry p. 13: V16N68 at +304)
//   first 1202 followed ~12 s later on the flight (+316)
//   V57E at t=48 s    (+338) — the restart already killed the monitor
//   re-key ENTR t=57.8 s (+346), second 1202 ~10-12 s later (+356/358)
//   third use keyed t=84 s, KEY REL t=90 s (+374/+380) — too short, no alarm

func TestBaselineP63RunsClean(t *testing.T) {
	// happy: 120 s on the knife edge — demand ~99-100%, zero alarms
	res := RunBaselineP63(120_000)
	if n := len(res.Alarms); n != 0 {
		t.Fatalf("baseline threw %d alarms (%+v), want 0 — the quiet knife edge", n, res.Alarms[0])
	}
	busy := float64(res.SoftwareNs+res.TheftNs) / float64(res.ElapsedNs)
	if busy < 0.97 || busy > 1.0 {
		t.Fatalf("baseline total demand = %.3f, want 0.97-1.00 (knife edge)", busy)
	}
	duty := float64(res.SoftwareNs) / float64(res.ElapsedNs)
	if duty < 0.80 || duty > 0.90 {
		t.Fatalf("baseline software duty = %.3f, want 0.80-0.90 (Eyles: <85%% + LR ~2%%)", duty)
	}
	if res.MaxCores > 6 {
		t.Fatalf("baseline max cores = %d, want <= 6 — no unbounded stub pile", res.MaxCores)
	}
}

func TestBaselineWithoutBugIsComfortable(t *testing.T) {
	// unhappy control: RR bug off — the same software fits with margin
	res := RunBaselineP63NoBug(120_000)
	if n := len(res.Alarms); n != 0 {
		t.Fatalf("no-bug baseline threw %d alarms, want 0", n)
	}
	busy := float64(res.SoftwareNs+res.TheftNs) / float64(res.ElapsedNs)
	if busy >= 0.90 {
		t.Fatalf("no-bug total demand = %.3f, want < 0.90 — the margin the designers had", busy)
	}
	if res.MaxCores > 4 {
		t.Fatalf("no-bug max cores = %d, want <= 4", res.MaxCores)
	}
}

func TestMonitor1668ThrowsTwo1202s(t *testing.T) {
	// happy: the monitor's running load tips the knife edge — two 1202s at
	// the flight-anchored offsets, and no alarm from the short third use
	res := RunMonitor1668(100_000)
	if len(res.Alarms) < 2 {
		t.Fatalf("1668 run threw %d alarms, want >= 2", len(res.Alarms))
	}
	for i, a := range res.Alarms {
		if a.Code != 1202 {
			t.Fatalf("alarm %d = %d, want 1202 — P63 hit the core-set wall, never the VAC wall", i, a.Code)
		}
	}
	a1 := res.Alarms[0].At
	lo, hi := Monitor1EntrMS*Millisecond+8*Second, Monitor1EntrMS*Millisecond+16*Second
	if a1 < lo || a1 > hi {
		t.Fatalf("first 1202 at %.1f s, want %.1f-%.1f s (flight: ~12 s after the monitor started)",
			float64(a1)/1e9, float64(lo)/1e9, float64(hi)/1e9)
	}
	a2 := res.Alarms[1].At
	lo2, hi2 := Monitor2EntrMS*Millisecond+8*Second, Monitor2EntrMS*Millisecond+16*Second
	if a2 < lo2 || a2 > hi2 {
		t.Fatalf("second 1202 at %.1f s, want %.1f-%.1f s (flight: +356, ~10-12 s after re-key)",
			float64(a2)/1e9, float64(lo2)/1e9, float64(hi2)/1e9)
	}
	// the third, short monitor use must stay clean
	for _, a := range res.Alarms {
		if a.At > Monitor3EntrMS*Millisecond {
			t.Fatalf("alarm at %.1f s after the third (short) monitor use — flight stayed clean there", float64(a.At)/1e9)
		}
	}
}

func TestMonitor1668AlarmIsCoreWallWithFreeVAC(t *testing.T) {
	// happy detail (the whole point): at the failing request the core sets
	// are exhausted while a VAC is still free — 1202, not 1201
	res := RunMonitor1668(100_000)
	if len(res.Alarms) == 0 {
		t.Fatalf("no alarms")
	}
	a := res.Alarms[0]
	if a.CoresHeld != 8 {
		t.Fatalf("first alarm cores = %d/8, want 8/8", a.CoresHeld)
	}
	if a.VACsHeld >= 5 {
		t.Fatalf("first alarm VACs = %d/5, want < 5 — the VAC wall was NOT the failure", a.VACsHeld)
	}
	if a.Requester == "" {
		t.Fatalf("alarm requester empty — the timeline must name the failing request")
	}
}

func TestMonitor1668RestartShedsTheBacklog(t *testing.T) {
	// unhappy path of the flight story: the restart flushes the stubs (pools
	// drop back), guidance resumes (SERVICER spawns continue), and the
	// monitor chain dies with it
	res := RunMonitor1668(100_000)
	if res.Restarts < 2 {
		t.Fatalf("restarts = %d, want >= 2 (one per alarm)", res.Restarts)
	}
	if len(res.Alarms) == 0 {
		t.Fatalf("no alarms")
	}
	// within 3 s after the first restart the core count must have dropped to
	// the rebuilt chain's level (SERVICER + transients: <= 3)
	alarmMs := int(res.Alarms[0].At / Millisecond)
	seen := false
	for _, s := range res.Samples {
		if s.AtMs > alarmMs && s.AtMs < alarmMs+3000 && s.Cores <= 3 {
			seen = true
			break
		}
	}
	if !seen {
		t.Fatalf("cores never dropped <= 3 within 3 s of the restart — the flush must shed the stubs")
	}
	// SERVICER keeps cycling after the restart (guidance never stopped)
	spawnsAfter := 0
	for _, ev := range res.Events {
		if ev.Kind == "spawn" && ev.Job == "SERVICER" && ev.At > res.Alarms[0].At {
			spawnsAfter++
		}
	}
	if spawnsAfter < 10 {
		t.Fatalf("only %d SERVICER spawns after the first restart — guidance must keep flying", spawnsAfter)
	}
}
