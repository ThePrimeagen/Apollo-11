package interpreter

// Tests written FIRST: the Interpreter walkthrough slims down. The
// five spotlit blocks keep the real MUNRVG ops — verbatim SERVICER
// opcodes and operands, comments stripped — but every block now
// reads the same simple way: ONE plain comment on top that just
// says what the block does (no "THIS BLOCK" narration), the bare
// instructions, a blank row, and the whole DANZIG construction as
// one pseudo call:
//
//	check_for_higher_priority_jobs()    # DANZIG
//
// No more five dresses — no assembly CCS, no fork, no weighing, no
// stamp. The call is the same line in every block, wears the love
// mark, and names exactly the function the Check Priority scene
// walks. The prologue is one plain comment over the real TC INTPRET;
// the DOT block and the three-chunk tail still scroll by, stripped
// of their inline comments, so the run stays consecutive and the
// fade below the last stop still has code to sink through. A resize
// keeps the clock; Stop then Start replays; a nil screen never
// panics.

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/code"
	"github.com/theprimeagen/apollo-11/exec-tui/components/danzig"
	"github.com/theprimeagen/apollo-11/exec-tui/components/scrollcode"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/checkprio"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 100
	stageH = 40
)

// stock is the scene's default clock — the marks every test that
// plays the stock show steers by.
var stock = DefaultConfig()

// anchor is where the spotlit block's first row (its comment) parks:
// the scroll plays on the stage minus the caption's two rows.
func anchor() int { return scrollcode.AnchorY(stageH - 2) }

func paint(sc screenplay.Scene) *screenplay.Screen {
	scr := screenplay.NewScreen(stageW, stageH)
	sc.Render(scr)
	return scr
}

func tick(sc screenplay.Scene, seconds float64) {
	const dt = 1.0 / 30
	for t := 0.0; t < seconds-dt/2; t += dt {
		sc.Update(dt)
	}
}

func opened(t *testing.T) *Show {
	t.Helper()
	s := New()
	s.Start()
	_ = paint(s)
	return s
}

func seek(t *testing.T, s *Show, by float64) *screenplay.Screen {
	t.Helper()
	tick(s, by)
	return paint(s)
}

func rowText(scr *screenplay.Screen, y int) string {
	w, _ := scr.Size()
	rs := make([]rune, 0, w)
	for x := 0; x < w; x++ {
		c := scr.Cell(x, y)
		if c == nil || c.Content == "" {
			rs = append(rs, ' ')
			continue
		}
		rs = append(rs, []rune(c.Content)[0])
	}
	return string(rs)
}

func findOn(scr *screenplay.Screen, text string) (x, y int, ok bool) {
	_, h := scr.Size()
	for row := 0; row < h; row++ {
		line := rowText(scr, row)
		if i := strings.Index(line, text); i >= 0 {
			return len([]rune(line[:i])), row, true
		}
	}
	return 0, 0, false
}

func mustSee(t *testing.T, scr *screenplay.Screen, text string) (x, y int) {
	t.Helper()
	x, y, ok := findOn(scr, text)
	if !ok {
		t.Fatalf("the stage must show %q", text)
	}
	return x, y
}

func mustNotSee(t *testing.T, scr *screenplay.Screen, text string) {
	t.Helper()
	if _, _, ok := findOn(scr, text); ok {
		t.Fatalf("the stage must not show %q", text)
	}
}

func fgAt(scr *screenplay.Screen, x, y int) int {
	c := scr.Cell(x, y)
	if c == nil {
		return -1
	}
	if ic, ok := c.Style.Fg.(ansi.IndexedColor); ok {
		return int(ic)
	}
	return -1
}

func bgAt(scr *screenplay.Screen, x, y int) int {
	c := scr.Cell(x, y)
	if c == nil {
		return -1
	}
	if ic, ok := c.Style.Bg.(ansi.IndexedColor); ok {
		return int(ic)
	}
	return -1
}

func fgOf(t *testing.T, scr *screenplay.Screen, text string) int {
	t.Helper()
	x, y := mustSee(t, scr, text)
	return fgAt(scr, x, y)
}

// stripComment cuts a listing line at its # and trims the tail — the
// shape every displayed op line must already be in.
func stripComment(line string) string {
	if i := strings.Index(line, "#"); i >= 0 {
		line = line[:i]
	}
	return strings.TrimRight(line, " \t")
}

// servicerOps is every op line of the flight source, comments
// stripped, in file order.
func servicerOps(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile("../../../Luminary099/SERVICER.agc")
	if err != nil {
		t.Fatalf("the flight source must be readable: %v", err)
	}
	var ops []string
	for _, line := range strings.Split(string(raw), "\n") {
		if s := stripComment(line); s != "" {
			ops = append(ops, s)
		}
	}
	return ops
}

// opSet is the stripped file as a set.
func opSet(t *testing.T) map[string]bool {
	t.Helper()
	set := map[string]bool{}
	for _, op := range servicerOps(t) {
		set[op] = true
	}
	return set
}

// run is the whole displayed op sequence in scroll order: the
// prologue's hand-off, four blocks, the DOT block, the last block,
// then the tail.
func run() []string {
	var out []string
	out = append(out, PrologueLines()[len(PrologueLines())-1])
	chunks := Chunks()
	for _, ch := range chunks[:4] {
		out = append(out, ch.Source...)
	}
	out = append(out, MidLines()...)
	out = append(out, chunks[4].Source...)
	for _, ep := range EpilogueBlocks() {
		out = append(out, ep...)
	}
	return out
}

func TestLegitimacy(t *testing.T) {
	t.Run("happy: every op on the scroll is a real SERVICER op, comments already stripped", func(t *testing.T) {
		ops := opSet(t)
		for _, line := range run() {
			if stripComment(line) != line {
				t.Fatalf("op line %q still carries a comment — the blocks lead with one plain comment instead", line)
			}
			if !ops[line] {
				t.Fatalf("op line %q is not in SERVICER.agc — the code must be the real one", line)
			}
		}
	})
	t.Run("happy: the run is one consecutive stretch of SERVICER ops — TC INTPRET through RVQ", func(t *testing.T) {
		file := servicerOps(t)
		want := run()
		matches := func(start int) bool {
			if start+len(want) > len(file) {
				return false
			}
			for j, w := range want {
				if file[start+j] != w {
					return false
				}
			}
			return true
		}
		found := false
		for i := range file {
			if file[i] == want[0] && matches(i) {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("the scroll must be one real consecutive stretch of SERVICER ops")
		}
		if last := want[len(want)-1]; !strings.Contains(last, "RVQ") {
			t.Fatalf("the run must close on the interpreter's own return, got %q", last)
		}
	})
	t.Run("unhappy: the comments declare themselves and never quote the flight file", func(t *testing.T) {
		ops := opSet(t)
		comments := append([]string{PrologueLines()[0]}, nil...)
		for _, ch := range Chunks() {
			comments = append(comments, ch.Comment)
		}
		for _, c := range comments {
			if !strings.HasPrefix(c, "# ") {
				t.Fatalf("comment %q must declare itself with a leading #", c)
			}
			if strings.Contains(c, "\n") {
				t.Fatalf("comment %q must be a single line", c)
			}
			if strings.Contains(c, "THIS BLOCK") {
				t.Fatalf("comment %q narrates — just say what the block does", c)
			}
			if ops[stripComment(c)] && stripComment(c) != "" {
				t.Fatalf("comment %q poses as flight code", c)
			}
		}
	})
	t.Run("unhappy: the check is declared pseudo — never a line of the flight file", func(t *testing.T) {
		if opSet(t)[stripComment(CheckLine)] {
			t.Fatalf("the check %q collides with real SERVICER code", CheckLine)
		}
		if !strings.Contains(CheckLine, "()") {
			t.Fatalf("the check %q must read as a pseudo function call", CheckLine)
		}
		if !strings.Contains(CheckLine, "# DANZIG") {
			t.Fatalf("the check %q must still name the DANZIG construction", CheckLine)
		}
	})
}

func TestRoster(t *testing.T) {
	t.Run("happy: five blocks — one plain comment on top, bare ops, numbered captions", func(t *testing.T) {
		chunks := Chunks()
		if len(chunks) != 5 {
			t.Fatalf("the walkthrough covers %d blocks, want 5", len(chunks))
		}
		seen := map[string]bool{}
		for i, ch := range chunks {
			if ch.Name == "" || len(ch.Source) == 0 {
				t.Fatalf("block %d must carry a name and real source", i)
			}
			if !strings.HasPrefix(ch.Comment, "# ") {
				t.Fatalf("%s comment %q must open plainly with a #", ch.Name, ch.Comment)
			}
			if strings.Contains(ch.Comment, "THIS BLOCK") {
				t.Fatalf("%s comment %q narrates — just say what the block does", ch.Name, ch.Comment)
			}
			if seen[ch.Comment] {
				t.Fatalf("%s repeats another block's comment — each block says what IT does", ch.Name)
			}
			seen[ch.Comment] = true
			for _, line := range ch.Source {
				if strings.Contains(line, "#") {
					t.Fatalf("%s op line %q still carries an inline comment — the block's one comment rides on top", ch.Name, line)
				}
			}
			if !strings.Contains(ch.Caption, fmt.Sprintf("%d/5", i+1)) || !strings.Contains(ch.Caption, ch.Name) {
				t.Fatalf("%s caption %q must number the stop and name the ops", ch.Name, ch.Caption)
			}
		}
		if chunks[4].Name != "VXV VSL2" {
			t.Fatalf("the last spotlight belongs to the cross product, got %q", chunks[4].Name)
		}
	})
	t.Run("happy: the check is one simple call, the same in every block, naming the walked function", func(t *testing.T) {
		if !strings.Contains(CheckLine, "check_for_higher_priority_jobs()") {
			t.Fatalf("the check %q must be the simple pseudo call", CheckLine)
		}
		if !strings.Contains(CheckLine, checkprio.Lines()[0]) {
			t.Fatalf("the check %q must call exactly the function the Check Priority scene walks, %q", CheckLine, checkprio.Lines()[0])
		}
	})
	t.Run("happy: the tail still scrolls — the DOT block and three stripped chunks below the last stop", func(t *testing.T) {
		if len(MidLines()) == 0 {
			t.Fatal("the DOT block between the last two stops must still scroll by")
		}
		eps := EpilogueBlocks()
		if len(eps) < 3 {
			t.Fatalf("the tail holds %d chunks, want at least 3 so the fade has code to sink through", len(eps))
		}
		for i, ep := range eps {
			if len(ep) == 0 {
				t.Fatalf("tail chunk %d is empty", i)
			}
			for _, line := range ep {
				if strings.Contains(line, "#") {
					t.Fatalf("tail line %q still carries a comment — the whole scroll slims down", line)
				}
			}
		}
	})
	t.Run("unhappy: none of the five dresses survive and no line outgrows the card", func(t *testing.T) {
		var all []string
		all = append(all, PrologueLines()...)
		all = append(all, MidLines()...)
		for _, ch := range Chunks() {
			all = append(all, ch.Comment)
			all = append(all, ch.Source...)
			all = append(all, CheckLine)
		}
		for _, ep := range EpilogueBlocks() {
			all = append(all, ep...)
		}
		for _, line := range code.New(code.LangAGC, all).Lines() {
			for _, glyph := range []string{"╭", "╰", "▶", "▓", "CCS", "NEWJOB", "CHANG2"} {
				if strings.Contains(line, glyph) {
					t.Fatalf("line %q still wears a dressed-up check (%q) — the check is one plain call now", line, glyph)
				}
			}
			if n := len([]rune(line)); n > 88 {
				t.Fatalf("line %q runs %d runes expanded — past 88 the column no longer fits the stage", line, n)
			}
		}
	})
}

func TestOpening(t *testing.T) {
	t.Run("happy: the first block's comment at the anchor, ops below, the prologue dim above", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := paint(s)
		_, cy := mustSee(t, scr, Chunks()[0].Comment)
		if cy != anchor() {
			t.Fatalf("the spotlit block's comment sits at %d, want the anchor %d", cy, anchor())
		}
		vx, vy := mustSee(t, scr, "VLOAD")
		if vy != cy+1 {
			t.Fatalf("the ops start right under the comment: row %d, want %d", vy, cy+1)
		}
		ix, iy := mustSee(t, scr, "INTPRET")
		if iy >= cy {
			t.Fatalf("the prologue (row %d) rides above the spotlight (row %d)", iy, cy)
		}
		px, py := mustSee(t, scr, "PGUIDE")
		if py <= vy {
			t.Fatalf("the next block (row %d) waits below the spotlight (row %d)", py, vy)
		}
		if got := fgAt(scr, vx, vy); got != code.Dim(code.Iris, 0) {
			t.Fatalf("the spotlit opcode wears %d, want the lit iris %d", got, code.Dim(code.Iris, 0))
		}
		if got := fgOf(t, scr, "KPIP2"); got != code.Dim(code.Foam, 0) {
			t.Fatalf("the spotlit operand wears %d, want the lit foam %d", got, code.Dim(code.Foam, 0))
		}
		if got := fgAt(scr, ix, iy); got != code.Dim(code.Foam, 1) {
			t.Fatalf("the prologue wears %d, want the vignette's dim foam %d", got, code.Dim(code.Foam, 1))
		}
		if got := fgAt(scr, px, py); got != code.Dim(code.Foam, 1) {
			t.Fatalf("one block below wears %d, want the same dim foam %d — equally shaded both sides", got, code.Dim(code.Foam, 1))
		}
		mustNotSee(t, scr, "VXV")
		mustNotSee(t, scr, "RVQ")
	})
	t.Run("happy: the base floor, the octal gutter, and the first caption", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := paint(s)
		if got := bgAt(scr, 0, 0); got != danzig.Base256 {
			t.Fatalf("the stage floor wears %d, want the Rose Pine base %d", got, danzig.Base256)
		}
		_, vy := mustSee(t, scr, "VLOAD")
		line := rowText(scr, vy)
		addr := regexp.MustCompile(`[0-7]{5}`).FindString(line)
		if addr == "" {
			t.Fatalf("the op row %q must carry a five-digit octal address", line)
		}
		if strings.Index(line, addr) > strings.Index(line, "VLOAD") {
			t.Fatal("the address gutter leads the line")
		}
		mustSee(t, scr, Chunks()[0].Caption)
	})
	t.Run("happy: the spotlit call wears the love mark, its DANZIG tag muted", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := paint(s)
		if got := fgOf(t, scr, "check_for_higher_priority_jobs"); got != code.Love {
			t.Fatalf("the marked call wears %d, want love %d", got, code.Love)
		}
		if got := fgOf(t, scr, "# DANZIG"); got != code.Dim(code.Muted, 0) {
			t.Fatalf("the DANZIG tag wears %d, want the muted comment ink %d", got, code.Dim(code.Muted, 0))
		}
	})
	t.Run("unhappy: a tiny stage still opens without a panic, a nil screen too", func(t *testing.T) {
		s := New()
		s.Start()
		defer s.Stop()
		tiny := screenplay.NewScreen(12, 5)
		s.Render(tiny)
		s.Render(nil)
		s2 := New()
		s2.Start()
		s2.Update(1)
		s2.Stop()
	})
}

func TestScroll(t *testing.T) {
	t.Run("happy: the second stop spotlights the guidance push with the fades re-hung around it", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, stock.StopStart(1)+0.05)
		px, py := mustSee(t, scr, "PGUIDE")
		if got := fgAt(scr, px, py); got != code.Dim(code.Foam, 0) {
			t.Fatalf("the spotlight must hand over: PGUIDE wears %d, want %d", got, code.Dim(code.Foam, 0))
		}
		if got := fgOf(t, scr, "KPIP2"); got != code.Dim(code.Foam, 1) {
			t.Fatalf("the ΔV load steps back into the vignette: %d, want %d", got, code.Dim(code.Foam, 1))
		}
		if got := fgOf(t, scr, "ABVEL"); got != code.Dim(code.Foam, 2) {
			t.Fatalf("two stops down is faint: %d, want %d", got, code.Dim(code.Foam, 2))
		}
		mustNotSee(t, scr, "INTPRET")
		mustSee(t, scr, Chunks()[1].Caption)
	})
	t.Run("happy: the glide carries the code upward and lands exactly on the anchor", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, stock.GlideStart(0)+0.3)
		_, mid := mustSee(t, scr, Chunks()[1].Comment)
		if mid <= anchor() {
			t.Fatalf("mid-glide the next block still rides below the anchor: %d", mid)
		}
		scr = seek(t, s, 0.2)
		_, later := mustSee(t, scr, Chunks()[1].Comment)
		if later > mid {
			t.Fatalf("the code must scroll upward: %d then %d", mid, later)
		}
		scr = seek(t, s, stock.StopStart(1)-stock.GlideStart(0)-0.5-0.02)
		_, before := mustSee(t, scr, Chunks()[1].Comment)
		if before != anchor() {
			t.Fatalf("one frame before the hold the block sits at %d, want the anchor %d", before, anchor())
		}
		scr = seek(t, s, 0.1)
		if _, after := mustSee(t, scr, Chunks()[1].Comment); after != before {
			t.Fatalf("the block hopped %d→%d on the very frame its hold began", before, after)
		}
	})
	t.Run("happy: the last stop is the V cross V, its call still lit, the tail fading through three levels", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, stock.StopStart(4)+0.1)
		_, cy := mustSee(t, scr, Chunks()[4].Comment)
		if cy != anchor() {
			t.Fatalf("the last block's comment sits at %d, want the anchor %d", cy, anchor())
		}
		vx, vy := mustSee(t, scr, "VXV")
		if vy != cy+1 {
			t.Fatalf("VXV sits at %d, want right under the comment %d", vy, cy+1)
		}
		if got := fgAt(scr, vx, vy); got != code.Dim(code.Iris, 0) {
			t.Fatalf("the V cross V wears %d, want the plain lit iris %d — the gold dress is gone", got, code.Dim(code.Iris, 0))
		}
		checkRow := cy + 1 + len(Chunks()[4].Source) + 1
		line := rowText(scr, checkRow)
		ci := strings.Index(line, "check_for_higher_priority_jobs")
		if ci < 0 {
			t.Fatalf("the spotlit block must close on its call, row %d reads %q", checkRow, line)
		}
		if got := fgAt(scr, len([]rune(line[:ci])), checkRow); got != code.Love {
			t.Fatalf("the last block's call wears %d, want love %d", got, code.Love)
		}
		if got := fgOf(t, scr, "HDOTDISP"); got != code.Dim(code.Foam, 1) {
			t.Fatalf("the scrolled-past DOT block dims above: %d, want %d", got, code.Dim(code.Foam, 1))
		}
		if got := fgOf(t, scr, "HCALC"); got != code.Dim(code.Foam, 1) {
			t.Fatalf("one chunk below wears %d, want %d", got, code.Dim(code.Foam, 1))
		}
		if got := fgOf(t, scr, "-MUDTMUN"); got != code.Dim(code.Foam, 2) {
			t.Fatalf("two chunks below wear %d, want %d", got, code.Dim(code.Foam, 2))
		}
		if got := fgOf(t, scr, "RVQ"); got != code.Dim(code.Iris, 3) {
			t.Fatalf("three chunks below wear %d, want the barely-there %d", got, code.Dim(code.Iris, 3))
		}
		mustNotSee(t, scr, "KPIP2")
		mustSee(t, scr, Chunks()[4].Caption)
		before := rowText(scr, vy)
		scr = seek(t, s, 30)
		if got := rowText(scr, vy); got != before {
			t.Fatalf("the final hold drifted:\n%q\n%q", before, got)
		}
	})
	t.Run("unhappy: there is no sixth stop — the tail never takes the spotlight", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, stock.StopStart(4)+100)
		hx, hy := mustSee(t, scr, "HCALC")
		if got := fgAt(scr, hx, hy); got == code.Dim(code.Foam, 0) {
			t.Fatal("the tail stole the spotlight — the walkthrough ends on VXV")
		}
		_, cy := mustSee(t, scr, Chunks()[4].Comment)
		if cy != anchor() {
			t.Fatalf("the last block must still hold the anchor %d, got %d", anchor(), cy)
		}
	})
}

func TestScenePlaysConfig(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: New plays the Active knobs on the first curtain", func(t *testing.T) {
		t.Cleanup(Reset)
		fast := DefaultConfig()
		fast.HoldSeconds = 0.5
		if err := Use(fast); err != nil {
			t.Fatalf("Use: %v", err)
		}
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, fast.StopStart(1)+0.1)
		if got := fgOf(t, scr, "PGUIDE"); got != code.Dim(code.Foam, 0) {
			t.Fatalf("a fast hold must already spotlight the second block: %d", got)
		}
	})
	t.Run("happy: a nudged knob is what the replay plays", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, 0.3)
		if got := fgOf(t, scr, "KPIP2"); got != code.Dim(code.Foam, 0) {
			t.Fatal("test premise: the stock show opens on the ΔV load")
		}
		s.Stop()
		s.Cfg.HoldSeconds = 0.5
		s.Start()
		_ = paint(s)
		scr = seek(t, s, s.Cfg.StopStart(1)+0.1)
		if got := fgOf(t, scr, "PGUIDE"); got != code.Dim(code.Foam, 0) {
			t.Fatalf("the replay must ride the nudged hold: %d", got)
		}
	})
	t.Run("unhappy: changing knobs mid-flight never retimes the running show", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		_ = seek(t, s, 0.3)
		s.Cfg.HoldSeconds = 0.1
		s.Cfg.GlideSeconds = StepSeconds
		scr := seek(t, s, 1.7)
		if got := fgOf(t, scr, "KPIP2"); got != code.Dim(code.Foam, 0) {
			t.Fatalf("a ride in the air keeps the knobs it launched with: %d", got)
		}
	})
}

func TestSceneLifecycle(t *testing.T) {
	t.Run("happy: a resize keeps the clock — no fall back to the first stop", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		_ = seek(t, s, stock.StopStart(1)+0.2)
		big := screenplay.NewScreen(110, 44)
		s.Render(big)
		px, py := mustSee(t, big, "PGUIDE")
		if got := fgAt(big, px, py); got != code.Dim(code.Foam, 0) {
			t.Fatalf("the resize must keep the spotlight: %d", got)
		}
	})
	t.Run("happy: Stop then Start replays from the top", func(t *testing.T) {
		s := opened(t)
		_ = seek(t, s, stock.StopStart(4)+1)
		s.Stop()
		s.Start()
		scr := paint(s)
		if got := fgOf(t, scr, "KPIP2"); got != code.Dim(code.Foam, 0) {
			t.Fatalf("the replay must open back on the ΔV load: %d", got)
		}
		mustSee(t, scr, "INTPRET")
		s.Stop()
	})
	t.Run("happy: the bill is one scene named Interpreter", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the bill holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "Interpreter" || b[0].Scene == nil {
			t.Fatalf("the bill must carry the Interpreter scene, got %+v", b[0])
		}
	})
	t.Run("unhappy: after the one scene there is nothing left, and dt<=0 holds the clock", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		defer p.Stop()
		if p.Len() != 1 || p.CurrentName() != "Interpreter" {
			t.Fatalf("the show opens on %d %q, want one Interpreter", p.Len(), p.CurrentName())
		}
		if p.Next() {
			t.Fatal("after the walkthrough there is nothing left")
		}

		s := opened(t)
		defer s.Stop()
		scr := paint(s)
		_, before := mustSee(t, scr, "VLOAD")
		s.Update(0)
		s.Update(-3)
		scr = paint(s)
		if _, after := mustSee(t, scr, "VLOAD"); after != before {
			t.Fatal("dt<=0 must hold the spotlight still")
		}
	})
}
