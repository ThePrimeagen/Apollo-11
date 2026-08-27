package agcgraph

// The graphs screen, static edition: 2.5 seconds of "here is what the CPU
// operates with" under the current switch states — never animated. Three
// lanes (each three rows tall) over a 180-column window, a ≤20-character
// label gutter, light-gray gridlines marking time, a plain-text legend at
// the bottom describing every job that ran in the interval, and the same
// three switches. Every toggle re-simulates a fresh 2.5 s snapshot.

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

func sized(m Model, w, h int) Model {
	mm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return mm.(Model)
}

func view(m Model) string { return m.View().Content }

// blocks are the bar glyphs a lane cell may carry.
const blocks = "▁▂▃▄▅▆▇█"

func hasBlock(s string) bool { return strings.ContainsAny(s, blocks) }

func TestStaticSnapshotNeverAnimates(t *testing.T) {
	// happy: the screen is a still — New pre-simulates exactly 2.5 s and
	// Init schedules NO ticker; the view is identical frame to frame
	m := New()
	if cmd := m.Init(); cmd != nil {
		t.Fatalf("Init scheduled a command — the graphs screen must never animate")
	}
	if got := m.live.Engine().Now(); got != 2500*msim.Millisecond {
		t.Fatalf("snapshot simulated %d ns, want exactly 2.5 s", got)
	}
	m = sized(m, 200, 45)
	v1 := view(m)
	v2 := view(m)
	if v1 != v2 {
		t.Fatalf("two renders of the same snapshot differ — animation leaked in")
	}
}

func TestHealthyOpenDefaults(t *testing.T) {
	// happy: the opening portrait is the healthy CPU — descent running,
	// no monitor, no radar steal — with SERVICER's work in the VAC lane
	m := sized(New(), 200, 45)
	if !m.descent || m.monitor || m.radar {
		t.Fatalf("open state = descent %v, monitor %v, radar %v; want on/off/off (healthy)",
			m.descent, m.monitor, m.radar)
	}
	v := stripAnsi(view(m))
	lines := strings.Split(v, "\n")
	if !hasBlock(strings.Join(lines[0:3], "")) {
		t.Fatalf("VAC lane empty on the healthy portrait")
	}
	if !hasBlock(strings.Join(lines[8:11], "")) {
		t.Fatalf("ops lane empty — the cadences always run")
	}
	if strings.Count(v, "│") < 10 {
		t.Fatalf("gridlines missing from the portrait")
	}
}

func TestTogglesResimulate(t *testing.T) {
	// happy: each switch key rebuilds a fresh 2.5 s snapshot under the new
	// configuration; unhappy: inert keys change nothing, q quits
	m := sized(New(), 200, 45)
	before := view(m)

	m, _ = keyed(m, 'd')
	if m.descent {
		t.Fatalf("'d' must switch descent off")
	}
	if got := m.live.Engine().Now(); got != 2500*msim.Millisecond {
		t.Fatalf("toggle rebuilt a %d ns snapshot, want a fresh 2.5 s", got)
	}
	offView := stripAnsi(view(m))
	vacLane := strings.Join(strings.Split(offView, "\n")[0:3], "")
	if hasBlock(vacLane) {
		t.Fatalf("descent off but the VAC lane still carries bars:\n%s", vacLane)
	}

	m, _ = keyed(m, '1')
	if !m.monitor {
		t.Fatalf("'1' must key the monitor")
	}
	m, _ = keyed(m, 'r')
	if !m.radar {
		t.Fatalf("'r' must turn the radar steal on")
	}

	m, cmd := keyed(m, 'x')
	if cmd != nil {
		t.Fatalf("'x' must be inert")
	}
	xView := view(m)
	m2, _ := keyed(m, 'x')
	if view(m2) != xView {
		t.Fatalf("an inert key changed the picture")
	}
	_, cmd = keyed(m, 'q')
	if cmd == nil {
		t.Fatalf("'q' must quit")
	}
	_ = before
}

func TestEverythingOffShowsIdleOps(t *testing.T) {
	// the user's spec: when everything is off, just the jobs that run when
	// nothing is happening — the hardware cadences
	m := sized(New(), 200, 45)
	m, _ = keyed(m, 'd') // descent off (monitor and radar already off)
	v := stripAnsi(view(m))
	lines := strings.Split(v, "\n")
	if hasBlock(strings.Join(lines[0:3], "")) || hasBlock(strings.Join(lines[4:7], "")) {
		t.Fatalf("VAC/coreset lanes must be empty with everything off")
	}
	if !hasBlock(strings.Join(lines[8:11], "")) {
		t.Fatalf("ops lane must still show the cadences")
	}
	for _, want := range []string{"DOWNRUPT:", "T4RUPT:", "DAP:"} {
		if !strings.Contains(v, want) {
			t.Fatalf("idle legend missing %q:\n%s", want, v)
		}
	}
	for _, banned := range []string{"SERVICER:", "MONDO:", "LRHJOB:"} {
		if strings.Contains(v, banned) {
			t.Fatalf("idle legend lists %q — only running jobs belong", banned)
		}
	}
}

func TestLegendDescribesRunningJobs(t *testing.T) {
	// happy: every running job gets its brief text line —
	// "NAME: Xms total :: wakes up every P and runs for Yms"
	m := sized(New(), 200, 45)
	v := stripAnsi(view(m))
	downrupt := regexp.MustCompile(`DOWNRUPT: 25\.0ms total :: wakes up every 20ms and runs for 0\.2ms`)
	if !downrupt.MatchString(v) {
		t.Fatalf("DOWNRUPT legend line wrong or missing (125 fires x 0.2 ms in 2.5 s):\n%s", v)
	}
	dap := regexp.MustCompile(`DAP: 300\.0ms total :: wakes up every 100ms and runs for 12\.0ms`)
	if !dap.MatchString(v) {
		t.Fatalf("DAP legend line wrong or missing (25 fires x 12 ms in 2.5 s):\n%s", v)
	}
	if !regexp.MustCompile(`SERVICER: [0-9.]+ms total :: wakes up every 2s and runs for [0-9.]+ms`).MatchString(v) {
		t.Fatalf("SERVICER legend line missing:\n%s", v)
	}
	if strings.Contains(v, "MONDO:") {
		t.Fatalf("MONDO in the legend with the monitor off")
	}
	// unhappy→happy: keying 1668 brings MONDO into the lanes and legend
	m, _ = keyed(m, '1')
	v = stripAnsi(view(m))
	if !regexp.MustCompile(`MONDO: [0-9.]+ms total :: wakes up every 1s and runs for 30\.0ms`).MatchString(v) {
		t.Fatalf("MONDO legend line missing with 1668 on:\n%s", v)
	}
}

func TestNoHeaderChromeAndLaneGeometry(t *testing.T) {
	// happy: at 200 columns — no title line, a 20-column gutter, a
	// 180-column plot, three lanes of three rows each
	m := sized(New(), 200, 45)
	v := stripAnsi(view(m))
	for _, banned := range []string{"COMMAND SCREEN", "AGC EXECUTIVE", "GET 102"} {
		if strings.Contains(v, banned) {
			t.Fatalf("the graphs screen must carry no header chrome, found %q", banned)
		}
	}
	lines := strings.Split(v, "\n")
	if !hasBlock(lines[0]) && !strings.Contains(lines[0], "│") {
		t.Fatalf("top line must be lane graphics, got %q", lines[0])
	}
	for _, i := range []int{0, 1, 2, 4, 5, 6, 8, 9, 10} {
		if w := len([]rune(lines[i])); w != 200 {
			t.Fatalf("row %d is %d cells wide, want 200 (20 gutter + 180 plot)", i, w)
		}
	}
	for _, want := range []string{"VAC JOBS", "CORESET JOBS", "NO-PRIORITY OPS"} {
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
	for _, want := range []string{"DESCENT", "1668", "RADAR STEAL"} {
		if !strings.Contains(v, want) {
			t.Fatalf("switch row missing %q", want)
		}
	}
}

func TestNarrowTerminalShrinksGracefully(t *testing.T) {
	// unhappy: 120 columns — the plot shrinks, nothing panics, rows fit
	m := sized(New(), 120, 30)
	v := stripAnsi(view(m))
	for i, l := range strings.Split(v, "\n") {
		if w := len([]rune(l)); w > 120 {
			t.Fatalf("row %d is %d cells wide at a 120-column terminal", i, w)
		}
	}
	if v == "" {
		t.Fatalf("narrow terminal rendered nothing")
	}
	tiny := sized(New(), 6, 2)
	_ = view(tiny)
}
