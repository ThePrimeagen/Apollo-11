package ui

// t49 — the top strip reduces to ONE number:
//   - the header is a single FREE COMPUTE line; effective margin = idle −
//     deficit, so under overload it goes NEGATIVE — below zero means broken
//   - the 2s CYCLE counter line is gone; instead a semi-filled white
//     gridline runs down the whole graph at every 2 seconds of AGC time
//   - FAILREG codes and the MON/TYPING/PAUSED badges move to the stats line

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTopIsJustFree(t *testing.T) {
	t.Run("happy: the first line is the free percentage and nothing else", func(t *testing.T) {
		_, m := newTestModel()
		top := stripAnsi(strings.Split(m.View().Content, "\n")[0])
		if !strings.Contains(top, "FREE COMPUTE") || !strings.Contains(top, "%") {
			t.Fatalf("the top line must carry the free percentage, got %q", top)
		}
		for _, gone := range []string{"AGC EXECUTIVE", "duty", "steal", "1s wall", "T+0", "BROKEN"} {
			if strings.Contains(top, gone) {
				t.Fatalf("the top line must drop %q", gone)
			}
		}
	})
	t.Run("happy: the explainer text line is gone — rows start immediately", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(1000)
		v := m.View().Content
		for _, gone := range []string{"2s CYCLE", "of AGC time", "ms/cell", "ruler marks", "[z] zoom", "(ms per 2s cycle)"} {
			if strings.Contains(v, gone) {
				t.Fatalf("the explainer text must be gone, found %q", gone)
			}
		}
		second := stripAnsi(strings.Split(v, "\n")[1])
		if !strings.HasPrefix(second, "SERVICER") {
			t.Fatalf("the rows must start right under the free line, got %q", second)
		}
	})
	t.Run("unhappy: overload drives the free number below zero", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AcquireLandingRadar()
		e.SetRadarBug(true)
		for _, k := range []byte("V16N68E") {
			e.PressKey(k)
		}
		e.AdvanceAGC(8000)
		top := stripAnsi(strings.Split(m.View().Content, "\n")[0])
		if !strings.Contains(top, "-") {
			t.Fatalf("an overloaded machine must show a negative free percentage, got %q", top)
		}
	})
}

func TestCycleGridlines(t *testing.T) {
	t.Run("happy: a 2-second ruler tints every row at the same column", func(t *testing.T) {
		e, m := newTestModel()
		e.AdvanceAGC(4100) // two absolute 2s marks inside the window
		v := m.View().Content
		idleCols := markerCols(t, v, "IDLE")
		if len(idleCols) == 0 {
			t.Fatal("the idle row must show at least one 2s ruler tint")
		}
		servCols := markerCols(t, v, "SERVICER")
		if fmt.Sprint(idleCols) != fmt.Sprint(servCols) {
			t.Fatalf("ruler tints must align across rows: IDLE %v vs SERVICER %v", idleCols, servCols)
		}
	})
	t.Run("happy: ruler tints sit 40 cells (2.0s at 50ms) apart when two are visible", func(t *testing.T) {
		e, m := newTestModel()
		e.AdvanceAGC(6100)
		cols := markerCols(t, m.View().Content, "IDLE")
		if len(cols) >= 2 {
			if d := cols[1] - cols[0]; d != 40 {
				t.Fatalf("ruler spacing must be 40 cells, got %d", d)
			}
		}
	})
	t.Run("unhappy: the ruler is background only — bars stay full blocks in front", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(4100)
		// SERVICER dominates nearly every cell; the ruler must never replace
		// its blocks with a marker glyph — it lives behind them.
		row := rowCells(t, m.View().Content, "SERVICER")
		blocks := 0
		for _, r := range row {
			if r == '█' {
				blocks++
			}
		}
		if blocks < 30 {
			t.Fatalf("dominant activity must stay solid blocks in front of the ruler, got %d blocks", blocks)
		}
		for _, r := range row {
			if r == '▒' {
				t.Fatal("no marker glyph may replace content — the ruler is a background tint")
			}
		}
	})
}

func TestStatsCarriesTheRest(t *testing.T) {
	t.Run("happy: PAUSED and TYPING chips live on the free line, only when active", func(t *testing.T) {
		_, m := newTestModel()
		m = key(m, '.')
		if !strings.Contains(strings.Split(m.View().Content, "\n")[0], "PAUSED") {
			t.Fatal("pause must show on the free line")
		}
		m = key(m, '.')
		m = key(m, 't')
		if !strings.Contains(strings.Split(m.View().Content, "\n")[0], "TYPING") {
			t.Fatal("typing mode must show on the free line")
		}
	})
	t.Run("unhappy: no chips render when nothing is active", func(t *testing.T) {
		_, m := newTestModel()
		top := m.View().Content
		for _, gone := range []string{"PAUSED", "TYPING", "FAILREG", "knife edge", "stubs leaked"} {
			if strings.Contains(top, gone) {
				t.Fatalf("idle view must carry no %q text", gone)
			}
		}
	})
}

// markerCols returns the track columns carrying the ruler's background
// tint, by walking the raw ANSI line and tracking the active background.
func markerCols(t *testing.T, v, label string) []int {
	t.Helper()
	const labelW, trackW = 16, 48
	var line string
	for _, l := range strings.Split(v, "\n") {
		if strings.HasPrefix(l, "\x1b") {
			if strings.Contains(stripAnsi(l)[:min(len(stripAnsi(l)), labelW)], label) {
				line = l
				break
			}
		} else if strings.HasPrefix(l, label) {
			line = l
			break
		}
	}
	if line == "" {
		t.Fatalf("row %q not found", label)
	}
	var cols []int
	col, i, bgOn := 0, 0, false
	for i < len(line) {
		if m := ansiPat.FindStringIndex(line[i:]); m != nil && m[0] == 0 {
			bgOn = strings.Contains(line[i:i+m[1]], "48;5;"+gridBGColor)
			i += m[1]
			continue
		}
		r, size := utf8.DecodeRuneInString(line[i:])
		_ = r
		if bgOn && col >= labelW && col < labelW+trackW {
			cols = append(cols, col-labelW)
		}
		col++
		i += size
	}
	return cols
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
