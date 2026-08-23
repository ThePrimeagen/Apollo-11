package main

import (
	"math"
	"strings"
	"testing"
)

// ---------- barCells: fractional fill for the CPU bars ----------

func TestBarCellsTinyNonzeroShowsShade(t *testing.T) {
	// happy: compute smaller than one column must still be visible
	full, partial := barCells(0.01, 30) // 0.3 of a cell
	if full != 0 {
		t.Fatalf("full = %d, want 0", full)
	}
	if partial != '░' {
		t.Fatalf("partial = %q, want '░' — sub-cell compute must render", partial)
	}
}

func TestBarCellsZeroShowsNothing(t *testing.T) {
	// unhappy: zero compute must not draw anything
	full, partial := barCells(0, 30)
	if full != 0 || partial != 0 {
		t.Fatalf("got full=%d partial=%q, want 0 and no rune", full, partial)
	}
}

func TestBarCellsPartialShadeSteps(t *testing.T) {
	cells := 10
	cases := []struct {
		frac float64
		full int
		part rune
	}{
		{0.51, 5, '░'}, // 5.1 cells → remainder 0.1
		{0.55, 5, '▒'}, // 5.5 cells → remainder 0.5
		{0.59, 5, '▓'}, // 5.9 cells → remainder 0.9
		{0.50, 5, 0},   // exact boundary → no partial cell
	}
	for _, c := range cases {
		full, part := barCells(c.frac, cells)
		if full != c.full || part != c.part {
			t.Errorf("barCells(%v, %d) = (%d, %q), want (%d, %q)",
				c.frac, cells, full, part, c.full, c.part)
		}
	}
}

func TestBarCellsClampsOverflow(t *testing.T) {
	// unhappy: frac > 1 must not overflow the lane
	full, partial := barCells(1.7, 20)
	if full != 20 || partial != 0 {
		t.Fatalf("got full=%d partial=%q, want full=20 and no partial", full, partial)
	}
	// unhappy: negative frac must not underflow
	full, partial = barCells(-0.3, 20)
	if full != 0 || partial != 0 {
		t.Fatalf("got full=%d partial=%q, want 0 and no rune", full, partial)
	}
	// unhappy: degenerate width
	full, partial = barCells(0.5, 0)
	if full != 0 || partial != 0 {
		t.Fatalf("got full=%d partial=%q, want 0 and no rune for 0-width bar", full, partial)
	}
}

func TestBarLineTinyValueVisible(t *testing.T) {
	// happy: integration — a 0.01s job in a 2.00s period must show a shade
	line := barLine("HIGATJOB", 0.01, periodS, 60, rpGold, 0)
	if !strings.ContainsAny(line, "░▒▓█") {
		t.Fatalf("bar for tiny value rendered nothing visible: %q", line)
	}
	// unhappy: a 0.00s job must show no fill at all (ghost disabled)
	line = barLine("RR ECDU", 0, periodS, 60, rpGold, 0)
	if strings.ContainsAny(line, "░▒▓█") {
		t.Fatalf("bar for zero value rendered fill: %q", line)
	}
}

func TestBarLineGhostShowsShortfall(t *testing.T) {
	// happy: SERVICER got 1.2s of a 1.8s need — the ░ shortfall region must
	// render. The old condition (ghost > total) was never true, so it never did.
	line := barLine("SERVICER", 1.2, periodS, 60, rpFoam, servicerNeedS)
	if strings.Count(line, "░") < 2 {
		t.Fatalf("ghost shortfall region not rendered: %q", line)
	}
	// unhappy: need already met → no ghost region beyond the fill
	line = barLine("SERVICER", 1.8, periodS, 60, rpFoam, 1.8)
	if strings.Count(line, "░") > 1 {
		t.Fatalf("ghost rendered although need is met: %q", line)
	}
}

// ---------- laneCoverage: per-cell burst coverage for the gantt ----------

func TestLaneCoverageFullPeriodBurst(t *testing.T) {
	// happy: a burst spanning the whole period fills every cell
	bursts := []burst{{JobIndex: 0, Start: 0, End: periodS}}
	cover := laneCoverage(bursts, 0, 20)
	for i, c := range cover {
		if c != 1.0 {
			t.Fatalf("cell %d coverage = %v, want 1.0", i, c)
		}
	}
}

func TestLaneCoverageNoBursts(t *testing.T) {
	// unhappy: no bursts for this job → all zero
	bursts := []burst{{JobIndex: 1, Start: 0, End: periodS}}
	cover := laneCoverage(bursts, 0, 20)
	for i, c := range cover {
		if c != 0 {
			t.Fatalf("cell %d coverage = %v, want 0", i, c)
		}
	}
}

func TestLaneCoverageTinyBurstIsPartialNotFull(t *testing.T) {
	// happy: a 2ms burst lands in exactly one cell as PARTIAL coverage.
	// This is the fix for tiny bursts being forced to a full '█' cell.
	bursts := []burst{{JobIndex: 0, Start: 0.30, End: 0.302}}
	cover := laneCoverage(bursts, 0, 20) // cellDur = 0.1s
	for i, c := range cover {
		if i == 3 {
			if c <= 0 || c >= 1 {
				t.Fatalf("cell 3 coverage = %v, want strictly between 0 and 1", c)
			}
			if coverageRune(c) == '█' {
				t.Fatalf("tiny burst rendered as full block — should be a shade")
			}
			if coverageRune(c) == '·' {
				t.Fatalf("tiny burst rendered as empty — partial compute must be visible")
			}
			continue
		}
		if c != 0 {
			t.Fatalf("cell %d coverage = %v, want 0", i, c)
		}
	}
}

func TestLaneCoverageClampsOutOfRangeBursts(t *testing.T) {
	// unhappy: bursts sticking out of [0, periodS] must clamp, not panic
	bursts := []burst{
		{JobIndex: 0, Start: -0.5, End: 0.05},
		{JobIndex: 0, Start: 1.95, End: 2.50},
	}
	cover := laneCoverage(bursts, 0, 20) // cellDur = 0.1s
	if got := cover[0]; math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("cell 0 coverage = %v, want 0.5", got)
	}
	if got := cover[19]; math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("cell 19 coverage = %v, want 0.5", got)
	}
}

func TestLaneCoverageOverlapClampedToOne(t *testing.T) {
	// unhappy: overlapping bursts must not push coverage past 1.0
	bursts := []burst{
		{JobIndex: 0, Start: 0, End: periodS},
		{JobIndex: 0, Start: 0, End: periodS},
	}
	cover := laneCoverage(bursts, 0, 10)
	for i, c := range cover {
		if c > 1.0 {
			t.Fatalf("cell %d coverage = %v, want clamped to 1.0", i, c)
		}
	}
}

// ---------- coverageRune: shade mapping ----------

func TestCoverageRuneMapping(t *testing.T) {
	cases := []struct {
		c    float64
		want rune
	}{
		{0, '·'},    // empty
		{0.10, '░'}, // faint
		{0.30, '▒'}, // light
		{0.60, '▓'}, // heavy
		{0.95, '█'}, // effectively full
		{1.00, '█'}, // full
		{2.00, '█'}, // unhappy: clamp above 1
		{-0.5, '·'}, // unhappy: clamp below 0
	}
	for _, c := range cases {
		if got := coverageRune(c.c); got != c.want {
			t.Errorf("coverageRune(%v) = %q, want %q", c.c, got, c.want)
		}
	}
}

// ---------- playhead: filled block, never a diamond ----------

func TestGanttPlayheadIsFilledBlock(t *testing.T) {
	m := initialModel()
	m.loadScenario(scenarioHealthy)
	m.width = 80

	// happy: mid-cycle playhead exists and is a solid block, not '◆'
	out := m.renderGantt(60, 1.0)
	if strings.ContainsRune(out, '◆') {
		t.Fatalf("gantt still renders the '◆' diamond (the boot artifact)")
	}
	// RR ECDU has no bursts in HEALTHY → its lane shows only the playhead block
	lines := strings.Split(out, "\n")
	rrLane := lines[1]
	if strings.Count(rrLane, "█") != 1 {
		t.Fatalf("empty lane should contain exactly the playhead block, got: %q", rrLane)
	}
}

func TestGanttPlayheadAbsentPastPeriod(t *testing.T) {
	// unhappy: t == periodS puts the playhead past the lane — nothing drawn
	m := initialModel()
	m.loadScenario(scenarioHealthy)
	m.width = 80

	out := m.renderGantt(60, periodS)
	lines := strings.Split(out, "\n")
	rrLane := lines[1] // no bursts in HEALTHY
	if strings.ContainsAny(rrLane, "░▒▓█◆") {
		t.Fatalf("lane with no bursts and playhead out of range should be empty, got: %q", rrLane)
	}
}

func TestGanttLanesDoNotWrapInBarsBox(t *testing.T) {
	// happy: each gantt lane must sit on the SAME line as its job name.
	// Lanes wider than the box content area get word-wrapped by lipgloss onto
	// their own line, detaching every bar from its label.
	m := initialModel()
	m.width = 100
	for _, w := range []int{60, 80, 100} {
		out := m.renderBars(w)
		for _, line := range strings.Split(out, "\n") {
			// The gantt row is the only place "RR ECDU" appears next to its
			// "HW" priority tag; it must also hold the lane itself.
			if !strings.Contains(line, "RR ECDU") || !strings.Contains(line, "HW") {
				continue
			}
			if !strings.ContainsAny(line, "░▒▓█·") {
				t.Fatalf("width %d: lane wrapped away from its job name: %q", w, line)
			}
		}
	}
}

func TestBarsViewHasNoDiamond(t *testing.T) {
	// The scrub line used '◆' too — it must be a filled block now.
	m := initialModel()
	m.width = 100
	out := m.renderBars(80)
	if strings.ContainsRune(out, '◆') {
		t.Fatalf("renderBars still contains a '◆' diamond")
	}
}
