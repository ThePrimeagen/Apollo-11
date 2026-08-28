package cpugraph

// Tests written FIRST: the CPU portrait pulled out of the graphs screen
// to be its own thing — the graph and nothing else. No legend text, no
// switch row, no surrounding information: just the grouped lanes (VAC
// JOBS / CORESET JOBS / NO-PRIORITY OPS / COUNTER THEFT), the 20-column
// name gutter, the gridlines, the hard white 2.00 s boundary, and the
// millisecond axis. The component OWNS the 2.5 s msim snapshot and hands
// the whole switch API to whoever composes it: SetDescent, SetMonitor,
// SetRadar, SetApproach re-simulate a fresh still (the 1668 monitor and
// the P64 approach cannot share the DSKY — keying either drops the
// other), and Running/Stolen/Engine expose everything a legend needs.
// Two projections of the one portrait: Rows(width) is the styled-string
// render the graphs screen embeds, Art(width)/Render() is the sprite a
// scene blits — a still screenplay.Component whose Update never moves.

import (
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	msim "github.com/theprimeagen/apollo-11/msim"
)

// The graph is a scene performer: the whole point of the extraction.
var _ screenplay.Component = (*Graph)(nil)

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripAnsi(s string) string { return ansiRe.ReplaceAllString(s, "") }

// blocks are the bar glyphs a lane cell may carry.
const blocks = "▁▂▃▄▅▆▇█"

func hasBlock(s string) bool { return strings.ContainsAny(s, blocks) }

// gutterOf and plotOf split one row at the 20-column gutter.
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

// rowIdx finds the row whose gutter carries exactly the given name.
func rowIdx(lines []string, name string) int {
	for i, l := range lines {
		if gutterOf(l) == name {
			return i
		}
	}
	return -1
}

// plainRows strips the styling off Rows for content checks.
func plainRows(g *Graph, width int) []string {
	rows := g.Rows(width)
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = stripAnsi(r)
	}
	return out
}

// spriteRow is one sprite row's glyphs as a string.
func spriteRow(sp sprite.Sprite, r int) string {
	rs := make([]rune, sp.Width)
	for c := 0; c < sp.Width; c++ {
		ch := sp.At(r, c).Ch
		if ch == 0 {
			ch = ' '
		}
		rs[c] = ch
	}
	return string(rs)
}

// spriteText reports whether any sprite row carries the text.
func spriteText(sp sprite.Sprite, text string) bool {
	for r := 0; r < sp.Height; r++ {
		if strings.Contains(spriteRow(sp, r), text) {
			return true
		}
	}
	return false
}

// spriteGutterRow finds the sprite row whose first 20 glyph columns
// carry exactly the given name.
func spriteGutterRow(sp sprite.Sprite, name string) int {
	for r := 0; r < sp.Height; r++ {
		row := []rune(spriteRow(sp, r))
		if len(row) < 20 {
			continue
		}
		if strings.TrimSpace(string(row[:20])) == name {
			return r
		}
	}
	return -1
}

// wearsInk reports whether any cell anywhere wears the ink.
func wearsInk(sp sprite.Sprite, ink int) bool {
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if sp.At(r, c).FG == ink {
				return true
			}
		}
	}
	return false
}

func TestStillComponentContract(t *testing.T) {
	// happy: the graph is a still scene performer — Start pins the stage,
	// Render paints a stage-sized sprite, and Update moves NOTHING: two
	// renders around any dt are identical and the snapshot clock stays at
	// exactly 2.5 s
	g := New()
	g.Start(220, 40)
	sp := g.Render()
	if sp.Width != 220 || sp.Height != 40 {
		t.Fatalf("staged render is %dx%d, want the 220x40 stage", sp.Width, sp.Height)
	}
	if !spriteText(sp, "VAC JOBS") || !spriteText(sp, "SERVICER") {
		t.Fatalf("the staged portrait must carry its lanes:\n%s", sprite.Render(sp))
	}
	r1 := sprite.Render(sp)
	if r2 := sprite.Render(g.Render()); r2 != r1 {
		t.Fatalf("two renders of the same still differ — animation leaked in")
	}
	g.Update(1.5)
	if r3 := sprite.Render(g.Render()); r3 != r1 {
		t.Fatalf("Update moved a STILL — the graph must never animate")
	}
	if got := g.Engine().Now(); got != 2500*msim.Millisecond {
		t.Fatalf("the snapshot clock reads %d ns after Update, want exactly 2.5 s", got)
	}

	// unhappy: no stage, no pixels — before Start and after Stop the
	// sprite is empty, while the switches are the graph's identity and
	// survive the restage
	g2 := New()
	if sp := g2.Render(); sp.Width != 0 || sp.Height != 0 {
		t.Fatalf("before Start the graph renders %dx%d, want nothing", sp.Width, sp.Height)
	}
	g2.SetRadar(true)
	g2.Start(200, 30)
	if !spriteText(g2.Render(), "RR CDU") {
		t.Fatalf("radar on but the staged portrait carries no theft lane")
	}
	g2.Stop()
	if sp := g2.Render(); sp.Width != 0 || sp.Height != 0 {
		t.Fatalf("after Stop the graph renders %dx%d, want nothing", sp.Width, sp.Height)
	}
	if !g2.Radar() {
		t.Fatalf("Stop dropped the radar switch — state is identity, the stage is not")
	}
	g2.Start(200, 30)
	if !spriteText(g2.Render(), "RR CDU") {
		t.Fatalf("the restaged portrait lost the theft lane the switches still call for")
	}
}

func TestOpensHealthyAndSwitchAPIResimulates(t *testing.T) {
	// happy: New opens the healthy portrait — descent on, monitor off,
	// radar off, approach off — pre-simulated to exactly 2.5 s with the
	// single SERVICER pass entered once; every switch change re-simulates
	// a FRESH snapshot
	g := New()
	if !g.Descent() || g.Monitor() || g.Radar() || g.Approach() {
		t.Fatalf("open state = descent %v, monitor %v, radar %v, approach %v; want on/off/off/off",
			g.Descent(), g.Monitor(), g.Radar(), g.Approach())
	}
	if got := g.Engine().Now(); got != 2500*msim.Millisecond {
		t.Fatalf("New simulated %d ns, want exactly 2.5 s", got)
	}
	if got := g.Engine().SpawnCount("SERVICER"); got != 1 {
		t.Fatalf("the healthy portrait entered %d SERVICERs, want exactly 1", got)
	}
	e0 := g.Engine()
	g.SetRadar(true)
	if !g.Radar() {
		t.Fatalf("SetRadar(true) must turn the steal on")
	}
	if g.Engine() == e0 {
		t.Fatalf("the radar switch must re-simulate a fresh snapshot")
	}
	if got := g.Engine().Now(); got != 2500*msim.Millisecond {
		t.Fatalf("the rebuilt snapshot ran %d ns, want a fresh 2.5 s", got)
	}
	g.SetDescent(false)
	if got := g.Engine().SpawnCount("SERVICER"); got != 0 {
		t.Fatalf("descent off but %d SERVICERs entered", got)
	}
	g.SetDescent(true)

	// happy: the 1668 monitor and the P64 approach cannot share the DSKY
	// — keying either drops the other, both ways, through the API alone
	g.SetMonitor(true)
	if !g.Monitor() || g.Approach() {
		t.Fatalf("after SetMonitor(true): monitor %v, approach %v; want on/off", g.Monitor(), g.Approach())
	}
	if got := g.Engine().SpawnCount("MONDO"); got == 0 {
		t.Fatalf("monitor on but MONDO never ran")
	}
	g.SetApproach(true)
	if !g.Approach() || g.Monitor() {
		t.Fatalf("keying P64 must drop the 1668 monitor: monitor %v, approach %v", g.Monitor(), g.Approach())
	}
	g.SetMonitor(true)
	if !g.Monitor() || g.Approach() {
		t.Fatalf("keying 1668 must drop P64: monitor %v, approach %v", g.Monitor(), g.Approach())
	}

	// unhappy: a set that changes nothing is refused — same switches,
	// same snapshot, no wasted re-simulation — and dropping a live
	// switch resurrects nothing
	g2 := New()
	e := g2.Engine()
	g2.SetRadar(false)
	g2.SetDescent(true)
	g2.SetMonitor(false)
	g2.SetApproach(false)
	if g2.Engine() != e {
		t.Fatalf("no-change sets re-simulated the identical portrait")
	}
	g3 := New()
	g3.SetMonitor(true)
	g3.SetMonitor(false)
	if g3.Monitor() || g3.Approach() {
		t.Fatalf("after dropping 1668: monitor %v, approach %v; want both off", g3.Monitor(), g3.Approach())
	}
	if got := g3.Engine().SpawnCount("MONDO"); got != 0 {
		t.Fatalf("monitor dropped but MONDO still ran %d times", got)
	}
}

func TestRowsAreTheGraphAlone(t *testing.T) {
	// happy: Rows(200) is the whole portrait and nothing else — grouped
	// lanes in order, bars on the running rows, the hard white boundary
	// cutting every lane at its proportional column, and the millisecond
	// axis as the last row
	g := New()
	rows := g.Rows(200)
	lines := plainRows(g, 200)
	iVac := rowIdx(lines, "VAC JOBS")
	iCore := rowIdx(lines, "CORESET JOBS")
	iOps := rowIdx(lines, "NO-PRIORITY OPS")
	iTheft := rowIdx(lines, "COUNTER THEFT")
	if iVac != 0 || !(iVac < iCore && iCore < iOps && iOps < iTheft) {
		t.Fatalf("group headers missing or out of order: VAC %d, CORESET %d, OPS %d, THEFT %d", iVac, iCore, iOps, iTheft)
	}
	si := rowIdx(lines, "SERVICER")
	if si <= iVac || si >= iCore || !hasBlock(plotOf(lines[si])) {
		t.Fatalf("SERVICER row missing or empty inside VAC JOBS (row %d)", si)
	}
	axis := lines[len(lines)-1]
	for _, want := range []string{"0ms", "200ms", "1200ms", "2400ms"} {
		if !strings.Contains(axis, want) {
			t.Fatalf("the last row must be the axis, missing %q: %q", want, axis)
		}
	}
	for i, l := range lines {
		if w := len([]rune(l)); w != 200 {
			t.Fatalf("row %d is %d cells wide, want 200 (20 gutter + 180 plot)", i, w)
		}
		if i < len(lines)-1 {
			if r := []rune(l); r[164] != '│' {
				t.Fatalf("lane row %d misses the 2 s boundary at column 164: %q", i, string(r[160:169]))
			}
		}
	}
	bound := lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Bold(true).Render("│")
	if !strings.Contains(strings.Join(rows, "\n"), bound) {
		t.Fatalf("the styled rows carry no hard-white boundary cell %q", bound)
	}

	// unhappy: none of the surrounding information belongs to the graph —
	// no legend text, no switch row, no quit hint; and narrow terminals
	// shrink the plot without panicking or losing the boundary
	joined := strings.Join(lines, "\n")
	for _, banned := range []string{"total ::", "wakes up", "DESCENT", "[d]", "[1]", "[r]", "[p]", "q quit", "RR CDU:"} {
		if strings.Contains(joined, banned) {
			t.Fatalf("the graph alone must carry no surrounding information, found %q", banned)
		}
	}
	narrow := plainRows(g, 120)
	for i, l := range narrow {
		if w := len([]rune(l)); w > 120 {
			t.Fatalf("narrow row %d is %d cells wide at a 120-column budget", i, w)
		}
	}
	if r := []rune(narrow[0]); len(r) < 101 || r[100] != '│' {
		t.Fatalf("the narrow portrait lost the boundary at column 100: %q", narrow[0])
	}
	if tiny := g.Rows(6); len(tiny) == 0 {
		t.Fatalf("a tiny budget rendered nothing")
	}
}

func TestArtWearsTheLaneInks(t *testing.T) {
	// happy: the sprite projection carries the portrait in the lane inks —
	// green VAC bars, the hard white boundary through every lane, the red
	// overrun tail past the line on the crossing portrait, the purple
	// theft ribbon, and the axis on the last row
	g := New()
	g.SetRadar(true)
	g.SetMonitor(true) // radar + 1668: the pass crosses the boundary
	a := g.Art(200)
	if a.Width != 200 {
		t.Fatalf("Art(200) is %d wide, want 200", a.Width)
	}
	if want := len(g.Rows(200)); a.Height != want {
		t.Fatalf("Art carries %d rows, want the same %d rows the string render draws", a.Height, want)
	}
	for r := 0; r < a.Height-1; r++ {
		cell := a.At(r, 164)
		if cell.Ch != '│' || cell.FG != BoundaryInk {
			t.Fatalf("lane row %d boundary cell is %q ink %d, want '│' ink %d", r, string(cell.Ch), cell.FG, BoundaryInk)
		}
	}
	si := spriteGutterRow(a, "SERVICER")
	if si < 0 {
		t.Fatalf("SERVICER lane missing from the sprite")
	}
	var beforeGreen, afterRed bool
	for c := 20; c < a.Width; c++ {
		cell := a.At(si, c)
		if !strings.ContainsRune(blocks, cell.Ch) {
			continue
		}
		if c < 164 && cell.FG == VacInk {
			beforeGreen = true
		}
		if c > 164 && cell.FG == OverrunInk {
			afterRed = true
		}
		if c > 164 && cell.FG == VacInk {
			t.Fatalf("the overrun past the line must never stay green (col %d)", c)
		}
	}
	if !beforeGreen || !afterRed {
		t.Fatalf("the crossing pass must be green before the line and red past it (green %v, red %v)", beforeGreen, afterRed)
	}
	ti := spriteGutterRow(a, "RR CDU")
	if ti < 0 {
		t.Fatalf("the theft lane missing with the steal on")
	}
	if !hasBlock(spriteRow(a, ti)) || !wearsInk(a, TheftInk) {
		t.Fatalf("the theft ribbon must carry bars in the theft ink")
	}
	axis := spriteRow(a, a.Height-1)
	if !strings.Contains(axis, "0ms") || !strings.Contains(axis, "200ms") {
		t.Fatalf("the sprite's last row must be the axis: %q", axis)
	}
	hdr := a.At(0, 0)
	if hdr.Ch != 'V' || hdr.FG != LabelInk {
		t.Fatalf("the VAC JOBS header must sit flush left in the label ink, got %q ink %d", string(hdr.Ch), hdr.FG)
	}

	// happy: a wide stage centers the art — the component behaves like
	// every other performer when a scene hands it a bigger stage
	g2 := New()
	g2.Start(240, 41)
	sp := g2.Render()
	if sp.Width != 240 || sp.Height != 41 {
		t.Fatalf("stage render is %dx%d, want 240x41", sp.Width, sp.Height)
	}
	art := g2.Art(240)
	if art.Width != 200 {
		t.Fatalf("the plot caps at 180 columns — Art(240) is %d wide, want 200", art.Width)
	}
	y0 := (41 - art.Height) / 2
	if row := spriteRow(sp, y0); !strings.Contains(row, "VAC JOBS") || strings.Index(row, "VAC JOBS") != 20 {
		t.Fatalf("the art must sit centered on the stage (row %d): %q", y0, row)
	}

	// unhappy: the healthy portrait shows no alarm ink and no theft; with
	// descent off the SERVICER leaves while the cadences stay; a
	// zero-width budget is refused with an empty sprite
	h := New()
	ha := h.Art(200)
	if wearsInk(ha, OverrunInk) {
		t.Fatalf("the healthy pass fits — no red belongs on the portrait")
	}
	if wearsInk(ha, TheftInk) || spriteText(ha, "RR CDU") {
		t.Fatalf("the steal is off — no theft lane belongs on the portrait")
	}
	h.SetDescent(false)
	idle := h.Art(200)
	if spriteText(idle, "SERVICER") {
		t.Fatalf("descent off but the SERVICER lane remains")
	}
	di := spriteGutterRow(idle, "DAP")
	if di < 0 || !hasBlock(spriteRow(idle, di)) {
		t.Fatalf("the cadences always run — DAP lane missing on the idle portrait")
	}
	if sp := h.Art(0); sp.Width != 0 || sp.Height != 0 {
		t.Fatalf("Art(0) rendered %dx%d, want nothing", sp.Width, sp.Height)
	}
}

func TestRunningAndStolenFeedTheLegend(t *testing.T) {
	// happy: Running hands a composite everything its legend needs — every
	// process that consumed CPU, in lane order, with its period, busy
	// total, and fire count; Stolen is the theft ledger's window total
	g := New()
	list := g.Running()
	if len(list) == 0 || list[0].Name != "SERVICER" {
		t.Fatalf("Running must open with the SERVICER lane, got %+v", list)
	}
	byName := map[string]Process{}
	for _, p := range list {
		if p.Busy <= 0 || p.Fires <= 0 {
			t.Fatalf("%s is listed but never ran: busy %d, fires %d", p.Name, p.Busy, p.Fires)
		}
		byName[p.Name] = p
	}
	dr, ok := byName["DOWNRUPT"]
	if !ok || dr.Busy != 25*msim.Millisecond || dr.Fires != 125 || dr.Period != "20ms" {
		t.Fatalf("DOWNRUPT = %+v, want 25 ms over 125 fires every 20ms", dr)
	}
	dap, ok := byName["DAP"]
	if !ok || dap.Busy != 300*msim.Millisecond || dap.Fires != 25 || dap.Period != "100ms" {
		t.Fatalf("DAP = %+v, want 300 ms over 25 fires every 100ms", dap)
	}
	g.SetRadar(true)
	if stolen := g.Stolen(); stolen < 350*msim.Millisecond || stolen >= 400*msim.Millisecond {
		t.Fatalf("the crest steal took %d ns of 2.5 s, want the ~15%% band (350-400 ms)", stolen)
	}

	// unhappy: with the steal off nothing is stolen; processes that never
	// ran stay off the list — the monitor's MONDO until it is keyed, the
	// whole descent chain when descent is off
	g2 := New()
	if got := g2.Stolen(); got != 0 {
		t.Fatalf("the steal is off but %d ns went missing", got)
	}
	for _, p := range g2.Running() {
		if p.Name == "MONDO" || p.Name == "HIGATJOB" {
			t.Fatalf("%s listed with its switch off", p.Name)
		}
	}
	g2.SetMonitor(true)
	found := false
	for _, p := range g2.Running() {
		if p.Name == "MONDO" && p.Period == "1s" {
			found = true
		}
	}
	if !found {
		t.Fatalf("monitor on but MONDO missing from Running")
	}
	g2.SetMonitor(false)
	g2.SetDescent(false)
	for _, p := range g2.Running() {
		if p.Name == "SERVICER" || p.Name == "READACCS" || p.Name == "R10,R11" {
			t.Fatalf("descent off but %s still listed", p.Name)
		}
	}
	stillThere := false
	for _, p := range g2.Running() {
		if p.Name == "DOWNRUPT" {
			stillThere = true
		}
	}
	if !stillThere {
		t.Fatalf("the hardware cadences always run — DOWNRUPT missing from the idle list")
	}
}
