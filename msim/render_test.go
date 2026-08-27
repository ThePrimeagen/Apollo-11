package msim

import (
	"strings"
	"testing"
)

// ---------- the timeline report ----------
//
// One rendered report per scenario: a per-second occupancy strip (cores 0-8,
// VACs 0-5, running job), the event log with GET timestamps, and for every
// alarm the pool snapshot naming the failing request.

func TestRenderTimelineShowsOccupancyAndAlarms(t *testing.T) {
	// happy: the 1668 report carries the occupancy strip and the alarm rows
	res := RunMonitor1668(100_000)
	out := RenderTimeline(res, "P63 with V16N68 keyed")
	if !strings.Contains(out, "P63 with V16N68 keyed") {
		t.Fatalf("report missing title")
	}
	// one occupancy row per second: t=0 .. t=99 → at least 100 rows
	rows := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "102:3") && strings.Contains(line, "|") {
			rows++
		}
	}
	if rows < 100 {
		t.Fatalf("occupancy rows = %d, want >= 100 (one per second with GET stamps)", rows)
	}
	if !strings.Contains(out, "1202") {
		t.Fatalf("report missing the 1202 alarm")
	}
	if !strings.Contains(out, "8/8") {
		t.Fatalf("report missing the cores-8/8 alarm snapshot")
	}
	if !strings.Contains(out, "V16N68") {
		t.Fatalf("report missing the monitor keying event")
	}
	if strings.Contains(out, "1201") {
		t.Fatalf("report shows a 1201 — P63 must only ever hit the core wall")
	}
}

func TestRenderTimelineBaselineHasNoAlarms(t *testing.T) {
	// happy: the baseline report renders the same strip, with zero alarm rows
	res := RunBaselineP63(30_000)
	out := RenderTimeline(res, "P63 baseline")
	if strings.Contains(out, "ALARM") || strings.Contains(out, "1202") || strings.Contains(out, "1201") {
		t.Fatalf("baseline report contains an alarm")
	}
	if !strings.Contains(out, "cores") || !strings.Contains(out, "vacs") {
		t.Fatalf("report missing the occupancy column headers")
	}
}

func TestRenderTimelineEmptyRunDoesNotPanic(t *testing.T) {
	// unhappy: a zero-length run still renders its header — no panic, no rows
	res := RunBaselineP63(0)
	out := RenderTimeline(res, "empty")
	if !strings.Contains(out, "empty") {
		t.Fatalf("empty report missing title")
	}
}
