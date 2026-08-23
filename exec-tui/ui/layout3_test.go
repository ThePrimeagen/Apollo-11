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

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestTopIsJustFree(t *testing.T) {
	t.Run("happy: the first line is the free percentage and nothing else", func(t *testing.T) {
		_, m := newTestModel()
		top := stripAnsi(strings.Split(m.View(), "\n")[0])
		if !strings.Contains(top, "FREE COMPUTE") || !strings.Contains(top, "%") {
			t.Fatalf("the top line must carry the free percentage, got %q", top)
		}
		for _, gone := range []string{"AGC EXECUTIVE", "duty", "steal", "1s wall", "T+0"} {
			if strings.Contains(top, gone) {
				t.Fatalf("the top line must drop %q", gone)
			}
		}
	})
	t.Run("happy: the 2s CYCLE counter line is gone", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(1000)
		if strings.Contains(m.View(), "2s CYCLE") {
			t.Fatal("the cycle counter line must be gone")
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
		top := stripAnsi(strings.Split(m.View(), "\n")[0])
		if !strings.Contains(top, "-") {
			t.Fatalf("an overloaded machine must show a negative free percentage, got %q", top)
		}
	})
}

// withColor forces a color profile so background tints reach the test
// output (lipgloss strips color without a TTY), restoring it afterwards.
func withColor(t *testing.T) {
	t.Helper()
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
}

func TestCycleGridlines(t *testing.T) {
	t.Run("happy: a 2-second ruler tints every row at the same column", func(t *testing.T) {
		withColor(t)
		e, m := newTestModel()
		e.AdvanceAGC(4100) // two absolute 2s marks inside the window
		v := m.View()
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
		withColor(t)
		e, m := newTestModel()
		e.AdvanceAGC(6100)
		cols := markerCols(t, m.View(), "IDLE")
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
		row := rowCells(t, m.View(), "SERVICER")
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
	t.Run("happy: FAILREG codes show on the stats line after an alarm", func(t *testing.T) {
		e, m := newTestModel()
		for i := 0; i < 8; i++ {
			e.ScheduleJob("HOG", 25, 1e9, false)
		}
		e.ScheduleJob("STRAW", 25, 10, false)
		if !strings.Contains(m.View(), "FAILREG 1202") {
			t.Fatal("the stats line must carry the FAILREG codes")
		}
	})
	t.Run("happy: PAUSED and TYPING indicators survive the header removal", func(t *testing.T) {
		_, m := newTestModel()
		m = key(m, '.')
		if !strings.Contains(m.View(), "PAUSED") {
			t.Fatal("pause must stay visible")
		}
		m = key(m, '.')
		m = key(m, 't')
		if !strings.Contains(m.View(), "TYPING") {
			t.Fatal("typing mode must stay visible")
		}
	})
}

// markerCols returns the track columns carrying the ruler's background
// tint, by walking the raw ANSI line and tracking the active background.
func markerCols(t *testing.T, v, label string) []int {
	t.Helper()
	const labelW, trackW = 18, 48
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
