package msim

import (
	"strings"
	"testing"
)

// ---------- the SERVICER instruction array ----------
//
// The array is the transcription of one P63 guidance pass out of the real
// Luminary099 listing: average-G (MUNRVG), radar conversion, the guidance
// equations, FINDCDUW, throttle, DISPEXIT. Every entry is one interpretive
// instruction (or one short basic block) with a source reference, and the
// engine checks NEWJOB between entries — the DANZIG boundary.

func TestServicerScriptStructure(t *testing.T) {
	// happy: every entry costed in (0, 5 ms], labeled, and source-referenced
	s, err := ServicerScript(P63Locked)
	if err != nil {
		t.Fatalf("ServicerScript(P63Locked): %v", err)
	}
	if len(s) < 250 {
		t.Fatalf("script has %d instructions, want >= 250 — one entry per interpretive instruction", len(s))
	}
	for i, in := range s {
		if in.Cost <= 0 || in.Cost > 5*Millisecond {
			t.Fatalf("instr %d (%s %s) cost %d ns, want in (0, 5 ms] — the DANZIG grain", i, in.Section, in.Op, in.Cost)
		}
		if in.Section == "" || in.Op == "" {
			t.Fatalf("instr %d missing section/op: %+v", i, in)
		}
		if !strings.Contains(in.Ref, ".agc") {
			t.Fatalf("instr %d (%s) ref %q — every instruction cites its source line", i, in.Op, in.Ref)
		}
	}
}

func TestServicerScriptCalibration(t *testing.T) {
	// happy: the P63 radar-locked pass costs 1.30-1.45 s of pure compute
	// (65-72%% of the 2 s cycle — Cherry's job table / Eyles' margins)
	s, err := ServicerScript(P63Locked)
	if err != nil {
		t.Fatalf("ServicerScript: %v", err)
	}
	total := s.Total()
	if total < 1300*Millisecond || total > 1450*Millisecond {
		t.Fatalf("P63Locked total = %d ms, want 1300-1450 ms", total/Millisecond)
	}
}

func TestServicerPrelockExcludesRadarSections(t *testing.T) {
	// happy: before LR lock the radar-conversion work is absent —
	// Eyles: margin >15%% before lock vs ~13%% after → a 30-80 ms delta
	locked, err := ServicerScript(P63Locked)
	if err != nil {
		t.Fatalf("locked: %v", err)
	}
	prelock, err := ServicerScript(P63Prelock)
	if err != nil {
		t.Fatalf("prelock: %v", err)
	}
	delta := locked.Total() - prelock.Total()
	if delta < 30*Millisecond || delta > 80*Millisecond {
		t.Fatalf("locked-prelock delta = %d ms, want 30-80 ms (the ~2%% radar conversion)", delta/Millisecond)
	}
	for _, in := range prelock {
		if strings.Contains(in.Section, "LR") || strings.Contains(in.Section, "RADAR") {
			t.Fatalf("prelock script contains radar section %q", in.Section)
		}
	}
}

func TestServicerSectionOrder(t *testing.T) {
	// happy: sections appear in execution order — average-G before guidance,
	// then GDUMP1's order: TC THROTTLE, then CALL FINDCDUW -2 (LUNAR_LANDING_
	// GUIDANCE_EQUATIONS.agc L822-L827), DISPEXIT last
	s, err := ServicerScript(P63Locked)
	if err != nil {
		t.Fatalf("ServicerScript: %v", err)
	}
	first := map[string]int{}
	for i, in := range s {
		if _, seen := first[in.Section]; !seen {
			first[in.Section] = i
		}
	}
	order := []string{"MUNRVG", "GUIDANCE", "THROTTLE", "FINDCDUW", "DISPEXIT"}
	prev := -1
	for _, sec := range order {
		at, ok := first[sec]
		if !ok {
			t.Fatalf("script missing section %q (have %v)", sec, first)
		}
		if at <= prev {
			t.Fatalf("section %q starts at %d, must come after previous section (at %d)", sec, at, prev)
		}
		prev = at
	}
}

func TestServicerUnknownPhaseErrors(t *testing.T) {
	// unhappy: an unmapped phase is an explicit error, not a silent guess
	if _, err := ServicerScript(Phase(99)); err == nil {
		t.Fatalf("ServicerScript(99) returned nil error, want explicit unknown-phase error")
	}
}

func TestServicerScriptP64ApproachRedesignation(t *testing.T) {
	// happy: the P64 approach pass is the locked P63 pass plus REDESIG —
	// the landing-site perturbation equations (LUNAR_LANDING_GUIDANCE_
	// EQUATIONS.agc L335-L408), entered from the WCHPHASE dispatch after
	// TTFINCR and falling into RGVGCALC. The delta is the approach phase's
	// unsheddable extra guidance (Eyles: P64 margin < 10%).
	p64, err := ServicerScript(P64Approach)
	if err != nil {
		t.Fatalf("ServicerScript(P64Approach): %v", err)
	}
	p63, err := ServicerScript(P63Locked)
	if err != nil {
		t.Fatalf("ServicerScript(P63Locked): %v", err)
	}
	delta := p64.Total() - p63.Total()
	if delta < 100*Millisecond || delta > 160*Millisecond {
		t.Fatalf("P64Approach-P63Locked delta = %d ms, want 100-160 ms (the REDESIG load)", delta/Millisecond)
	}
	sawRedesig := false
	for i, in := range p64 {
		if in.Cost <= 0 || in.Cost > 5*Millisecond {
			t.Fatalf("instr %d (%s %s) cost %d ns, want in (0, 5 ms] — the DANZIG grain", i, in.Section, in.Op, in.Cost)
		}
		if !strings.Contains(in.Ref, ".agc") {
			t.Fatalf("instr %d (%s) ref %q — every instruction cites its source line", i, in.Op, in.Ref)
		}
		if in.Section == "REDESIG" {
			sawRedesig = true
			if !strings.Contains(in.Ref, "LUNAR_LANDING_GUIDANCE_EQUATIONS.agc") {
				t.Fatalf("REDESIG instr %d cites %q, want the guidance-equations listing", i, in.Ref)
			}
		}
	}
	if !sawRedesig {
		t.Fatalf("P64Approach script carries no REDESIG section")
	}
	first := map[string]int{}
	for i, in := range p64 {
		if _, seen := first[in.Section]; !seen {
			first[in.Section] = i
		}
	}
	if !(first["GUIDANCE"] < first["REDESIG"] && first["REDESIG"] < first["RGVGCALC"]) {
		t.Fatalf("REDESIG must run after the guidance entry and fall into RGVGCALC (GUIDANCE %d, REDESIG %d, RGVGCALC %d)",
			first["GUIDANCE"], first["REDESIG"], first["RGVGCALC"])
	}
	// unhappy: neither P63 phase carries redesignation work — P63 skips
	// the REDESIG branch entirely
	for _, ph := range []Phase{P63Prelock, P63Locked} {
		s, err := ServicerScript(ph)
		if err != nil {
			t.Fatalf("ServicerScript(%d): %v", int(ph), err)
		}
		for _, in := range s {
			if in.Section == "REDESIG" {
				t.Fatalf("phase %d carries a REDESIG section — that is P64's load", int(ph))
			}
		}
	}
}
