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
	// guidance before FINDCDUW, FINDCDUW before throttle, DISPEXIT last
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
	order := []string{"MUNRVG", "GUIDANCE", "FINDCDUW", "THROTTLE", "DISPEXIT"}
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
