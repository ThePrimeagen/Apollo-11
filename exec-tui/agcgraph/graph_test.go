package agcgraph

// The graphs screen: three CPU lanes, each three rows tall, over a
// 180-column window covering 2.000 s — no header chrome, a ≤20-character
// label gutter, light-gray vertical gridlines marking time, and the same
// three switches on the bottom row. Opens frozen on one complete prerun
// cycle; space runs/freezes.

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	msim "github.com/theprimeagen/apollo-11/msim"
)

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripAnsi(s string) string { return ansiRe.ReplaceAllString(s, "") }

func keyed(m Model, code rune) (Model, tea.Cmd) {
	mm, cmd := m.Update(tea.KeyPressMsg{Code: code})
	return mm.(Model), cmd
}

func spaced(m Model) Model {
	mm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	return mm.(Model)
}

func sized(m Model, w, h int) Model {
	mm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return mm.(Model)
}

func view(m Model) string { return m.View().Content }

// blocks are the bar glyphs a lane cell may carry.
const blocks = "▁▂▃▄▅▆▇█"

func hasBlock(s string) bool { return strings.ContainsAny(s, blocks) }

func TestOpensFrozenOnOnePrerunCycle(t *testing.T) {
	// happy: New pre-runs exactly one 2 s cycle and freezes there
	m := New(msim.NewLive())
	if got := m.live.Engine().Now(); got != 2*msim.Second {
		t.Fatalf("opened at t=%d ns, want exactly one prerun 2 s cycle", got)
	}
	mm, cmd := m.Update(frameMsg{})
	m = mm.(Model)
	if cmd == nil {
		t.Fatalf("frames must keep scheduling even while frozen")
	}
	if got := m.live.Engine().Now(); got != 2*msim.Second {
		t.Fatalf("a frozen frame advanced the sim to %d ns", got)
	}
	// unhappy→happy flip: space runs, space freezes again
	m = spaced(m)
	mm, _ = m.Update(frameMsg{})
	m = mm.(Model)
	want := 2*msim.Second + msim.Nanos(frameMS)*msim.Millisecond
	if got := m.live.Engine().Now(); got != want {
		t.Fatalf("running frame advanced to %d, want %d", got, want)
	}
	m = spaced(m)
	mm, _ = m.Update(frameMsg{})
	m = mm.(Model)
	if got := m.live.Engine().Now(); got != want {
		t.Fatalf("re-frozen frame advanced to %d, want still %d", got, want)
	}
}

func TestNoHeaderChromeAndLaneGeometry(t *testing.T) {
	// happy: at 200 columns — no title line, a 20-column gutter, a
	// 180-column plot, three lanes of three rows each
	m := sized(New(msim.NewLive()), 200, 40)
	v := stripAnsi(view(m))
	for _, banned := range []string{"COMMAND SCREEN", "AGC EXECUTIVE", "GET 102"} {
		if strings.Contains(v, banned) {
			t.Fatalf("the graphs screen must carry no header chrome, found %q", banned)
		}
	}
	lines := strings.Split(v, "\n")
	if len(lines) < 12 {
		t.Fatalf("screen too short: %d lines", len(lines))
	}
	// the very top line belongs to the first lane — graph cells, not prose
	if !hasBlock(lines[0]) && !strings.Contains(lines[0], "│") {
		t.Fatalf("top line must be lane graphics, got %q", lines[0])
	}
	// every lane row is exactly gutter+plot = 200 cells wide (rows 3 and 7
	// are the blank separators between lanes)
	for _, i := range []int{0, 1, 2, 4, 5, 6, 8, 9, 10} {
		if w := len([]rune(lines[i])); w != 200 {
			t.Fatalf("row %d is %d cells wide, want 200 (20 gutter + 180 plot)", i, w)
		}
	}
	// labels sit inside the 20-column gutter on each lane's middle row
	for _, want := range []string{"VAC JOBS", "CORESET JOBS", "NO-PRIORITY OPS"} {
		if len(want) > 20 {
			t.Fatalf("label %q exceeds the 20-character budget", want)
		}
		found := false
		for _, l := range lines {
			if idx := strings.Index(l, want); idx >= 0 && idx+len(want) <= 20 {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("label %q not found inside the 20-column gutter", want)
		}
	}
}

func TestNarrowTerminalShrinksGracefully(t *testing.T) {
	// unhappy: 120 columns — the plot shrinks, nothing panics, rows fit
	m := sized(New(msim.NewLive()), 120, 30)
	v := stripAnsi(view(m))
	for i, l := range strings.Split(v, "\n") {
		if w := len([]rune(l)); w > 120 {
			t.Fatalf("row %d is %d cells wide at a 120-column terminal", i, w)
		}
	}
	if v == "" {
		t.Fatalf("narrow terminal rendered nothing")
	}
	// tiny: no panic
	tiny := sized(New(msim.NewLive()), 6, 2)
	_ = view(tiny)
}

func TestLanesCarryTheirClasses(t *testing.T) {
	// happy: the frozen prerun cycle shows SERVICER's work in the VAC lane,
	// the gates/display in the coreset lane, the cadences in the ops lane,
	// and light-gray gridlines marking time in every lane
	m := sized(New(msim.NewLive()), 200, 40)
	v := stripAnsi(view(m))
	lines := strings.Split(v, "\n")
	lane := func(first int) string { return strings.Join(lines[first:first+3], "\n") }
	vac, core, ops := lane(0), lane(4), lane(8)
	if !hasBlock(vac) {
		t.Fatalf("VAC lane empty over a full SERVICER cycle:\n%s", vac)
	}
	if !hasBlock(core) {
		t.Fatalf("coreset lane empty — the gates and the display ran:\n%s", core)
	}
	if !hasBlock(ops) {
		t.Fatalf("ops lane empty — DAP/T4RUPT/DOWNRUPT fired all cycle:\n%s", ops)
	}
	if strings.Count(v, "│") < 10 {
		t.Fatalf("gridlines missing — want vertical time lines through the lanes:\n%s", v)
	}
}

func TestLanesIdleShowOnlyGrid(t *testing.T) {
	// unhappy: descent off, monitor off — after the machine drains, a
	// streamed window shows no VAC-lane bars, only the time grid
	l := msim.NewLive()
	m := New(l)
	l.SetRadar(false)
	l.SetDescent(false)
	l.StepMS(6_000) // drain, then 2+ quiet seconds
	m = sized(m, 200, 40)
	v := stripAnsi(view(m))
	vacLane := strings.Join(strings.Split(v, "\n")[0:3], "\n")
	if hasBlock(vacLane) {
		t.Fatalf("VAC lane shows bars on a drained machine:\n%s", vacLane)
	}
	if !strings.Contains(vacLane, "│") {
		t.Fatalf("the time grid must still be there when nothing runs:\n%s", vacLane)
	}
}

func TestSwitchesRowAndKeys(t *testing.T) {
	// happy: the same three switches, driving the live engine, and any
	// switch key also unfreezes the stream
	l := msim.NewLive()
	m := sized(New(l), 200, 40)
	v := view(m)
	for _, want := range []string{"DESCENT", "1668", "RADAR STEAL"} {
		if !strings.Contains(v, want) {
			t.Fatalf("switch row missing %q", want)
		}
	}
	m, _ = keyed(m, 'r')
	if l.RadarOn() {
		t.Fatalf("'r' must flip the radar steal off")
	}
	if m.frozen {
		t.Fatalf("a switch key must unfreeze the stream")
	}
	m, _ = keyed(m, '1')
	if !l.MonitorOn() {
		t.Fatalf("'1' must key the monitor up")
	}
	m, _ = keyed(m, 'd')
	if l.DescentOn() {
		t.Fatalf("'d' must switch descent off")
	}
	// unhappy: inert keys change nothing, q quits
	m, cmd := keyed(m, 'x')
	if cmd != nil {
		t.Fatalf("'x' must be inert")
	}
	_, cmd = keyed(m, 'q')
	if cmd == nil {
		t.Fatalf("'q' must quit")
	}
}

func TestStreamingKeepsTheWindowWidth(t *testing.T) {
	// happy: streaming for 3 s keeps a 180-column window ending now
	l := msim.NewLive()
	m := sized(New(l), 200, 40)
	m = spaced(m)
	for i := 0; i < 60; i++ { // 60 frames x 50 ms = 3 s
		mm, _ := m.Update(frameMsg{})
		m = mm.(Model)
	}
	if got := l.Engine().Now(); got != 5*msim.Second {
		t.Fatalf("after prerun+3 s streaming Now = %d, want 5 s", got)
	}
	v := stripAnsi(view(m))
	lines := strings.Split(v, "\n")
	if w := len([]rune(lines[0])); w != 200 {
		t.Fatalf("streamed row width = %d, want 200", w)
	}
	if !hasBlock(lines[0] + lines[1] + lines[2]) {
		t.Fatalf("streamed VAC lane lost its bars")
	}
}
