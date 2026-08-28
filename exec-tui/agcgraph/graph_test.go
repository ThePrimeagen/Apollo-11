package agcgraph

// The graphs screen, single-cycle edition: 2.5 seconds of "here is what the
// CPU operates with" under the current switch states — never animated. One
// row per process that consumed CPU, grouped under VAC JOBS / CORESET JOBS /
// NO-PRIORITY OPS headers, a ≤20-character name gutter, gridlines, a HARD
// WHITE line at the 2.00 s guidance boundary, a plain-text legend, and four
// switches. The SERVICER is entered exactly once — the only process that
// does not repeat — so its bar shows the single pass stretching toward (and
// past) the boundary as load is switched on. Every toggle re-simulates a
// fresh 2.5 s snapshot.

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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

// gutterOf and plotOf split one rendered row at the 20-column gutter.
func gutterOf(line string) string {
	r := []rune(line)
	if len(r) < 20 {
		return strings.TrimSpace(line)
	}
	return strings.TrimSpace(string(r[:20]))
}

func plotOf(line string) string {
	r := []rune(line)
	if len(r) <= 20 {
		return ""
	}
	return string(r[20:])
}

// rowIdx finds the graph row whose gutter carries exactly the given name
// (a group header at column 0 or a process name indented inside the
// gutter). Returns -1 when the row is absent.
func rowIdx(lines []string, name string) int {
	for i, l := range lines {
		if gutterOf(l) == name {
			return i
		}
	}
	return -1
}

// servicerLastBusyMs is the last millisecond the portrait's SERVICER
// consumed CPU — the end of the single pass.
func servicerLastBusyMs(m Model) int {
	last := -1
	for _, s := range m.live.Engine().Samples() {
		if s.ByName["SERVICER"] > 0 {
			last = s.AtMs
		}
	}
	return last
}

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
	// no monitor, no radar steal, no approach — with the single SERVICER
	// pass on its own row and the cadences on theirs
	m := sized(New(), 200, 45)
	if !m.descent || m.monitor || m.radar || m.approach {
		t.Fatalf("open state = descent %v, monitor %v, radar %v, approach %v; want on/off/off/off (healthy)",
			m.descent, m.monitor, m.radar, m.approach)
	}
	v := stripAnsi(view(m))
	lines := strings.Split(v, "\n")
	si := rowIdx(lines, "SERVICER")
	if si < 0 || !hasBlock(plotOf(lines[si])) {
		t.Fatalf("SERVICER row missing or empty on the healthy portrait (row %d)", si)
	}
	di := rowIdx(lines, "DAP")
	if di < 0 || !hasBlock(plotOf(lines[di])) {
		t.Fatalf("DAP row missing or empty — the cadences always run (row %d)", di)
	}
	if strings.Count(v, "│") < 10 {
		t.Fatalf("gridlines missing from the portrait")
	}
}

func TestSingleServicerCycle(t *testing.T) {
	// happy: the portrait runs a SINGLE 2 s SERVICER cycle — the first
	// READACCS enters the only copy, while the lattice itself keeps firing
	// all the way through the 2.5 s window
	m := sized(New(), 200, 45)
	e := m.live.Engine()
	if got := e.SpawnCount("SERVICER"); got != 1 {
		t.Fatalf("the portrait entered %d SERVICERs, want exactly 1 — only the servicer does not repeat", got)
	}
	if got := e.TaskFires("READACCS"); got != 2 {
		t.Fatalf("READACCS fired %d times in 2.5 s, want 2 — every other timer keeps firing", got)
	}
	if got := e.TaskFires("R10,R11"); got < 9 {
		t.Fatalf("R10,R11 fired %d times in 2.5 s, want >= 9 — the 0.25 s chain keeps firing", got)
	}
	// unhappy: descent off — the chain never starts, no SERVICER anywhere
	m2, _ := keyed(m, 'd')
	if got := m2.live.Engine().SpawnCount("SERVICER"); got != 0 {
		t.Fatalf("descent off but %d SERVICERs entered", got)
	}
	if v := stripAnsi(view(m2)); strings.Contains(v, "SERVICER") {
		t.Fatalf("SERVICER row must vanish when it never ran:\n%s", v)
	}
}

func TestPerJobRowsGrouped(t *testing.T) {
	// happy: every process that consumed CPU gets its own row under its
	// group header — jobs holding a VAC, jobs holding a core set only,
	// then the no-priority operations
	m := sized(New(), 200, 45)
	v := stripAnsi(view(m))
	lines := strings.Split(v, "\n")
	iVac := rowIdx(lines, "VAC JOBS")
	iCore := rowIdx(lines, "CORESET JOBS")
	iOps := rowIdx(lines, "NO-PRIORITY OPS")
	if iVac < 0 || iCore < 0 || iOps < 0 || !(iVac < iCore && iCore < iOps) {
		t.Fatalf("group headers missing or out of order: VAC %d, CORESET %d, OPS %d", iVac, iCore, iOps)
	}
	if i := rowIdx(lines, "SERVICER"); i <= iVac || i >= iCore {
		t.Fatalf("SERVICER row at %d, want inside VAC JOBS (%d..%d)", i, iVac, iCore)
	}
	for _, name := range []string{"MAKEPLAY", "LRHJOB", "LRVJOB", "1/GYRO"} {
		if i := rowIdx(lines, name); i <= iCore || i >= iOps {
			t.Fatalf("%s row at %d, want inside CORESET JOBS (%d..%d)", name, i, iCore, iOps)
		}
	}
	for _, name := range []string{"READACCS", "R10,R11", "LRHTASK", "LRVTASK", "DAP", "T4RUPT", "DOWNRUPT"} {
		if i := rowIdx(lines, name); i <= iOps {
			t.Fatalf("%s row at %d, want under NO-PRIORITY OPS (%d)", name, i, iOps)
		}
	}
	for _, name := range []string{"SERVICER", "R10,R11", "DAP", "DOWNRUPT"} {
		if i := rowIdx(lines, name); !hasBlock(plotOf(lines[i])) {
			t.Fatalf("%s row carries no bars:\n%s", name, lines[i])
		}
	}
	// unhappy: uninvited processes stay off the portrait
	for _, banned := range []string{"MONDO", "CHARIN", "HIGATJOB", "MONREQ"} {
		if strings.Contains(v, banned) {
			t.Fatalf("%s on the healthy portrait — monitor and approach are off", banned)
		}
	}
}

func TestHardWhiteLineAtTwoSeconds(t *testing.T) {
	// happy: a hard white line marks the 2.00 s guidance boundary in every
	// graph row — headers and process rows alike — and it cuts THROUGH a
	// bar that crosses the boundary
	m := sized(New(), 200, 45)
	v := stripAnsi(view(m))
	lines := strings.Split(v, "\n")
	iAxis := -1
	for i, l := range lines {
		if strings.Contains(l, "0ms") && strings.Contains(l, "200ms") {
			iAxis = i
			break
		}
	}
	if iAxis <= 0 {
		t.Fatalf("no axis row found:\n%s", v)
	}
	// boundary column: 20 gutter + 2000ms * 180 cols / 2500ms = 164
	for i := 0; i < iAxis; i++ {
		r := []rune(lines[i])
		if len(r) < 200 {
			t.Fatalf("graph row %d is %d cells wide, want 200", i, len(r))
		}
		if r[164] != '│' {
			t.Fatalf("graph row %d misses the 2 s boundary at column 164: %q", i, string(r[160:169]))
		}
	}
	// the line is WHITE — the styled view carries the boundary glyph under
	// the hard-white style
	want := lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Bold(true).Render("│")
	if !strings.Contains(view(m), want) {
		t.Fatalf("styled view carries no hard-white boundary cell %q", want)
	}
	// radar + P64: the servicer bar crosses the boundary and the line
	// still cuts through it
	m, _ = keyed(m, 'r')
	m, _ = keyed(m, 'p')
	lines = strings.Split(stripAnsi(view(m)), "\n")
	si := rowIdx(lines, "SERVICER")
	if si < 0 {
		t.Fatalf("SERVICER row missing on the crossing portrait")
	}
	r := []rune(lines[si])
	if !strings.ContainsAny(string(r[163]), blocks) || !strings.ContainsAny(string(r[165]), blocks) {
		t.Fatalf("SERVICER bar must surround the boundary when it crosses: %q", string(r[158:172]))
	}
	if r[164] != '│' {
		t.Fatalf("the boundary line must cut through the crossing bar: %q", string(r[160:169]))
	}
	// unhappy: a narrow terminal keeps the line at its proportional column
	n := sized(New(), 120, 30)
	nl := strings.Split(stripAnsi(view(n)), "\n")
	nr := []rune(nl[0])
	// 20 gutter + 2000ms * 100 cols / 2500ms = 100
	if len(nr) < 101 || nr[100] != '│' {
		t.Fatalf("narrow portrait lost the boundary at column 100: %q", nl[0])
	}
}

func TestTogglesResimulate(t *testing.T) {
	// happy: each switch key rebuilds a fresh 2.5 s snapshot under the new
	// configuration; unhappy: inert keys change nothing, q quits
	m := sized(New(), 200, 45)

	m, _ = keyed(m, 'd')
	if m.descent {
		t.Fatalf("'d' must switch descent off")
	}
	if got := m.live.Engine().Now(); got != 2500*msim.Millisecond {
		t.Fatalf("toggle rebuilt a %d ns snapshot, want a fresh 2.5 s", got)
	}
	if v := stripAnsi(view(m)); strings.Contains(v, "SERVICER") {
		t.Fatalf("descent off but SERVICER still on the portrait")
	}

	m, _ = keyed(m, '1')
	if !m.monitor {
		t.Fatalf("'1' must key the monitor")
	}
	m, _ = keyed(m, 'r')
	if !m.radar {
		t.Fatalf("'r' must turn the radar steal on")
	}
	m, _ = keyed(m, 'p')
	if !m.approach {
		t.Fatalf("'p' must key the approach phase")
	}
	if got := m.live.Engine().Now(); got != 2500*msim.Millisecond {
		t.Fatalf("'p' rebuilt a %d ns snapshot, want a fresh 2.5 s", got)
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
}

func TestApproachSwitchP64(t *testing.T) {
	// happy: 'p' keys the P64 approach — its own set of jobs joins the
	// portrait (HIGATJOB and the flashing V06N64 under VAC JOBS) and the
	// REDESIG load pushes the single pass past the white line with the
	// radar steal on
	m := sized(New(), 200, 45)
	m, _ = keyed(m, 'r')
	m, _ = keyed(m, 'p')
	if !m.approach {
		t.Fatalf("'p' must key the approach")
	}
	v := stripAnsi(view(m))
	lines := strings.Split(v, "\n")
	iVac := rowIdx(lines, "VAC JOBS")
	iCore := rowIdx(lines, "CORESET JOBS")
	for _, name := range []string{"SERVICER", "MAKEPLAY", "HIGATJOB"} {
		if i := rowIdx(lines, name); i <= iVac || i >= iCore {
			t.Fatalf("%s row at %d, want inside VAC JOBS (%d..%d) — P64's own jobs hold VACs", name, i, iVac, iCore)
		}
	}
	si := rowIdx(lines, "SERVICER")
	if p := []rune(plotOf(lines[si])); !hasBlock(string(p[145:])) {
		t.Fatalf("radar+P64 must push the servicer past the boundary column:\n%s", lines[si])
	}
	// unhappy: 'p' again — the P64 jobs leave, MAKEPLAY returns to the
	// coreset group, the pass fits again
	m, _ = keyed(m, 'p')
	if m.approach {
		t.Fatalf("'p' must also key the approach back off")
	}
	v = stripAnsi(view(m))
	lines = strings.Split(v, "\n")
	if strings.Contains(v, "HIGATJOB") {
		t.Fatalf("HIGATJOB still on the portrait with approach off")
	}
	iCore = rowIdx(lines, "CORESET JOBS")
	iOps := rowIdx(lines, "NO-PRIORITY OPS")
	if i := rowIdx(lines, "MAKEPLAY"); i <= iCore || i >= iOps {
		t.Fatalf("MAKEPLAY row at %d, want back inside CORESET JOBS (%d..%d)", i, iCore, iOps)
	}
	si = rowIdx(lines, "SERVICER")
	if p := []rune(plotOf(lines[si])); hasBlock(string(p[146:])) {
		t.Fatalf("radar alone must stay under the boundary:\n%s", lines[si])
	}
}

func TestMonitor1668AndP64MutuallyExclusive(t *testing.T) {
	// happy: the DELTAH monitor and the P64 approach cannot share the
	// DSKY — keying either one drops the other, both ways
	m := sized(New(), 200, 45)
	m, _ = keyed(m, '1')
	if !m.monitor || m.approach {
		t.Fatalf("after '1': monitor %v, approach %v; want on/off", m.monitor, m.approach)
	}
	m, _ = keyed(m, 'p')
	if !m.approach || m.monitor {
		t.Fatalf("keying P64 must drop the 1668 monitor: monitor %v, approach %v", m.monitor, m.approach)
	}
	v := stripAnsi(view(m))
	if strings.Contains(v, "MONDO") || !strings.Contains(v, "HIGATJOB") {
		t.Fatalf("P64 portrait must carry HIGATJOB and no MONDO:\n%s", v)
	}
	if !strings.Contains(v, "[1] 1668 OFF") || !strings.Contains(v, "[p] P64 ON") {
		t.Fatalf("switch row must show 1668 OFF / P64 ON:\n%s", v)
	}
	m, _ = keyed(m, '1')
	if !m.monitor || m.approach {
		t.Fatalf("keying 1668 must drop P64: monitor %v, approach %v", m.monitor, m.approach)
	}
	v = stripAnsi(view(m))
	if !strings.Contains(v, "MONDO") || strings.Contains(v, "HIGATJOB") {
		t.Fatalf("1668 portrait must carry MONDO and no HIGATJOB:\n%s", v)
	}
	// unhappy: dropping the live switch resurrects nothing
	m, _ = keyed(m, '1')
	if m.monitor || m.approach {
		t.Fatalf("after dropping 1668: monitor %v, approach %v; want both off", m.monitor, m.approach)
	}
	v = stripAnsi(view(m))
	if strings.Contains(v, "MONDO") || strings.Contains(v, "HIGATJOB") {
		t.Fatalf("both off but their jobs remain on the portrait:\n%s", v)
	}
}

func TestServicerBoundaryCrossings(t *testing.T) {
	// The story the portrait must tell, one switch at a time:
	//   descent alone          — the pass fits with margin
	//   + radar steal          — the knife edge: still fits
	//   + 1668 (flight config) — the pass EXCEEDS the 2 s boundary
	//   + P64 instead of 1668  — exceeds it harder, monitor off
	m := sized(New(), 200, 45)
	d := servicerLastBusyMs(m)
	if d < 1000 || d >= 1900 {
		t.Fatalf("descent-only pass ended at %d ms, want inside (1000, 1900) — fits with margin", d)
	}
	m, _ = keyed(m, 'r')
	dr := servicerLastBusyMs(m)
	if dr <= d || dr >= 2000 {
		t.Fatalf("descent+radar pass ended at %d ms, want later than %d and still under 2000 — the knife edge", dr, d)
	}
	m, _ = keyed(m, '1')
	dm := servicerLastBusyMs(m)
	if dm < 2000 || dm >= 2400 {
		t.Fatalf("descent+radar+1668 pass ended at %d ms, want past the 2000 ms boundary (and inside the window)", dm)
	}
	m, _ = keyed(m, 'p')
	dp := servicerLastBusyMs(m)
	if dp <= dm || dp >= 2400 {
		t.Fatalf("descent+radar+P64 pass ended at %d ms, want past the 1668 case (%d) — the unsheddable guidance", dp, dm)
	}
	// unhappy: with descent off there is no pass to cross anything
	m, _ = keyed(m, 'd')
	if got := servicerLastBusyMs(m); got != -1 {
		t.Fatalf("descent off but the servicer consumed CPU up to %d ms", got)
	}
}

func TestEverythingOffShowsIdleOps(t *testing.T) {
	// the user's spec: when everything is off, just the jobs that run when
	// nothing is happening — the hardware cadences under NO-PRIORITY OPS,
	// with both job groups standing empty
	m := sized(New(), 200, 45)
	m, _ = keyed(m, 'd') // descent off (monitor, radar, approach already off)
	v := stripAnsi(view(m))
	lines := strings.Split(v, "\n")
	iVac := rowIdx(lines, "VAC JOBS")
	iCore := rowIdx(lines, "CORESET JOBS")
	iOps := rowIdx(lines, "NO-PRIORITY OPS")
	if iCore != iVac+1 || iOps != iCore+1 {
		t.Fatalf("job groups must stand empty with everything off (VAC %d, CORESET %d, OPS %d)", iVac, iCore, iOps)
	}
	for _, name := range []string{"DAP", "T4RUPT", "DOWNRUPT"} {
		i := rowIdx(lines, name)
		if i <= iOps {
			t.Fatalf("%s row missing from the idle ops group (row %d)", name, i)
		}
		if !hasBlock(plotOf(lines[i])) {
			t.Fatalf("%s row carries no bars on the idle portrait", name)
		}
	}
	for _, want := range []string{"DOWNRUPT:", "T4RUPT:", "DAP:"} {
		if !strings.Contains(v, want) {
			t.Fatalf("idle legend missing %q:\n%s", want, v)
		}
	}
	for _, banned := range []string{"SERVICER", "MONDO", "LRHJOB", "READACCS", "R10,R11", "HIGATJOB"} {
		if strings.Contains(v, banned) {
			t.Fatalf("idle portrait lists %q — only running processes belong", banned)
		}
	}
}

func TestLegendDescribesRunningJobs(t *testing.T) {
	// happy: every running process gets its brief text line —
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
	// keying P64 swaps the monitor's legend for the approach's
	m, _ = keyed(m, 'p')
	v = stripAnsi(view(m))
	if strings.Contains(v, "MONDO:") {
		t.Fatalf("MONDO legend must leave with the monitor dropped by P64:\n%s", v)
	}
	if !regexp.MustCompile(`HIGATJOB: [0-9.]+ms total :: wakes up every high gate and runs for [0-9.]+ms`).MatchString(v) {
		t.Fatalf("HIGATJOB legend line missing with P64 on:\n%s", v)
	}
}

func TestNoHeaderChromeAndLaneGeometry(t *testing.T) {
	// happy: at 200 columns — no title line, a 20-column gutter, a
	// 180-column plot; the group headers sit flush left in the gutter and
	// every process name is indented inside it
	m := sized(New(), 200, 45)
	v := stripAnsi(view(m))
	for _, banned := range []string{"COMMAND SCREEN", "AGC EXECUTIVE", "GET 102"} {
		if strings.Contains(v, banned) {
			t.Fatalf("the graphs screen must carry no header chrome, found %q", banned)
		}
	}
	lines := strings.Split(v, "\n")
	if !strings.HasPrefix(lines[0], "VAC JOBS") {
		t.Fatalf("top line must be the VAC JOBS header, got %q", lines[0])
	}
	iAxis := -1
	for i, l := range lines {
		if strings.Contains(l, "0ms") && strings.Contains(l, "200ms") {
			iAxis = i
			break
		}
	}
	if iAxis < 0 {
		t.Fatalf("no axis row found:\n%s", v)
	}
	for i := 0; i <= iAxis; i++ {
		if w := len([]rune(lines[i])); w != 200 {
			t.Fatalf("graph row %d is %d cells wide, want 200 (20 gutter + 180 plot)", i, w)
		}
	}
	for _, want := range []string{"VAC JOBS", "CORESET JOBS", "NO-PRIORITY OPS"} {
		i := rowIdx(lines, want)
		if i < 0 || !strings.HasPrefix(lines[i], want) {
			t.Fatalf("group header %q not flush left inside the gutter (row %d)", want, i)
		}
	}
	si := rowIdx(lines, "SERVICER")
	if si < 0 || !strings.HasPrefix(lines[si], " SERVICER") {
		t.Fatalf("SERVICER name must sit indented inside the 20-column gutter: %q", lines[si])
	}
	for _, want := range []string{"DESCENT", "1668", "RADAR STEAL", "P64"} {
		if !strings.Contains(v, want) {
			t.Fatalf("switch row missing %q", want)
		}
	}
}

func TestTimeAxisLabelsEveryOtherGridline(t *testing.T) {
	// happy: under the lanes sits a millisecond axis — a label on every
	// other gray line (every 200 ms), anchored at its gridline column
	m := sized(New(), 200, 45)
	v := stripAnsi(view(m))
	var axis string
	for _, l := range strings.Split(v, "\n") {
		if strings.Contains(l, "0ms") && strings.Contains(l, "200ms") {
			axis = l
			break
		}
	}
	if axis == "" {
		t.Fatalf("no time axis row found:\n%s", v)
	}
	for _, want := range []string{"0ms", "200ms", "400ms", "1200ms", "2400ms"} {
		if !strings.Contains(axis, want) {
			t.Fatalf("axis missing %q: %q", want, axis)
		}
	}
	// labels anchor at their gridline column: t*180/2500 + the 20 gutter
	if idx := strings.Index(axis, "1200ms"); idx < 104 || idx > 108 {
		t.Fatalf("1200ms label at column %d, want ~106 (20 + 1200*180/2500)", idx)
	}
	if idx := strings.Index(axis, "400ms"); idx < 46 || idx > 50 {
		t.Fatalf("400ms label at column %d, want ~48", idx)
	}
	// the odd gridlines (100, 300, ...) stay unlabeled
	if strings.Contains(axis, "100ms") || strings.Contains(axis, "300ms") {
		t.Fatalf("axis labels every OTHER line only: %q", axis)
	}
	// unhappy: labels never collide into each other — single spaces between
	// tokens would mean overlap
	for _, tok := range strings.Fields(axis) {
		if !strings.HasSuffix(tok, "ms") {
			t.Fatalf("stray axis token %q", tok)
		}
	}
}

func TestTimeAxisSurvivesNarrowAndIdle(t *testing.T) {
	// unhappy: a narrow plot keeps whatever labels fit, without overlap or
	// panic, and the axis persists on the everything-off portrait
	m := sized(New(), 120, 30)
	v := stripAnsi(view(m))
	if !strings.Contains(v, "0ms") {
		t.Fatalf("narrow axis lost its origin label:\n%s", v)
	}
	m2 := sized(New(), 200, 45)
	m2, _ = keyed(m2, 'd')
	v2 := stripAnsi(view(m2))
	if !strings.Contains(v2, "2400ms") {
		t.Fatalf("idle portrait lost the axis:\n%s", v2)
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
