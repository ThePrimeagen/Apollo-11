package interpreter

// Tests written FIRST: the Interpreter scene is now a composition —
// the code component displays the cards, the scrollcode component
// moves them, and the scene only chooses what to show: the REAL
// SERVICER interpreter code. The scroll is MUNRVG, the average-G
// integration that ran during the powered descent, verbatim from
// Luminary099/SERVICER.agc and consecutive from TC INTPRET to RVQ:
// a prologue (the routine's own header comments and the TC INTPRET
// hand-off), five spotlit chunks — the ΔV load, the guidance push,
// the position out, the velocity out, and the VXV cross product the
// scene is named for — one unfocused chunk between the last two
// stops (the DOT altitude-rate block the camera scrolls through),
// and three trailing chunks (through MUNGRAV to RVQ) so the fade
// below the last stop still has code to sink through. Each spotlit
// chunk ends in an annotated check to DANZIG in its own dress —
// the real assembly (verbatim INTERPRETER.agc), pseudocode, a fork,
// a weighing, a stamp — with every NEWJOB marked in love ink and
// the VXV op itself marked gold. The vignette fades four levels on
// both sides of the spotlight. A resize keeps the clock; Stop then
// Start replays; a nil screen never panics.

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
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 100
	stageH = 40
)

// stock is the scene's default clock — the marks every test that
// plays the stock show steers by.
var stock = DefaultConfig()

// anchor is where the spotlit chunk's first row parks: the scroll
// plays on the stage minus the caption's two rows.
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

// fileLines is a set of every line of an AGC source file.
func fileLines(t *testing.T, path string) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the flight source must be readable: %v", err)
	}
	set := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		set[line] = true
	}
	return set
}

func TestLegitimacy(t *testing.T) {
	t.Run("happy: every source line on the scroll is verbatim SERVICER.agc", func(t *testing.T) {
		servicer := fileLines(t, "../../../Luminary099/SERVICER.agc")
		check := func(kind string, lines []string) {
			t.Helper()
			for _, line := range lines {
				if !servicer[line] {
					t.Fatalf("%s line %q is not in SERVICER.agc — the code must be the real one", kind, line)
				}
			}
		}
		check("prologue", PrologueLines())
		for _, ch := range Chunks() {
			check(ch.Name, ch.Source)
		}
		check("mid", MidLines())
		for i, ep := range EpilogueBlocks() {
			check(fmt.Sprintf("epilogue %d", i), ep)
		}
	})
	t.Run("happy: the run is one consecutive stretch of SERVICER — TC INTPRET through RVQ", func(t *testing.T) {
		raw, err := os.ReadFile("../../../Luminary099/SERVICER.agc")
		if err != nil {
			t.Fatal(err)
		}
		file := strings.Split(string(raw), "\n")
		var run []string
		run = append(run, PrologueLines()[len(PrologueLines())-1]) // TC INTPRET
		chunks := Chunks()
		for _, ch := range chunks[:4] {
			run = append(run, ch.Source...)
		}
		run = append(run, MidLines()...)
		run = append(run, chunks[4].Source...)
		for _, ep := range EpilogueBlocks() {
			run = append(run, ep...)
		}
		start := -1
		for i := range file {
			if file[i] == run[0] {
				start = i
				break
			}
		}
		if start < 0 {
			t.Fatalf("the run's first line %q is not in SERVICER.agc", run[0])
		}
		for j, want := range run {
			if got := file[start+j]; got != want {
				t.Fatalf("the run breaks at offset %d:\nfile %q\nrun  %q — the scroll must be one real consecutive stretch", j, got, want)
			}
		}
		if last := run[len(run)-1]; !strings.Contains(last, "RVQ") {
			t.Fatalf("the run must close on the interpreter's own return, got %q", last)
		}
	})
	t.Run("happy: the assembly check is verbatim INTERPRETER.agc — the real DANZIG lines", func(t *testing.T) {
		interp := fileLines(t, "../../../Luminary099/INTERPRETER.agc")
		var assembly *Chunk
		for i := range Chunks() {
			if Chunks()[i].Style == "assembly" {
				c := Chunks()[i]
				assembly = &c
			}
		}
		if assembly == nil {
			t.Fatal("one check must be the real assembly")
		}
		for _, line := range assembly.Check {
			if !interp[line] {
				t.Fatalf("assembly check line %q is not in INTERPRETER.agc", line)
			}
		}
	})
	t.Run("unhappy: the dressed-up checks never pose as flight code", func(t *testing.T) {
		servicer := fileLines(t, "../../../Luminary099/SERVICER.agc")
		for _, ch := range Chunks() {
			if !strings.HasPrefix(ch.Intro, "#") {
				t.Fatalf("%s intro %q must be a comment — annotations declare themselves", ch.Name, ch.Intro)
			}
			if ch.Style == "assembly" {
				continue
			}
			for _, line := range ch.Check {
				if servicer[line] {
					t.Fatalf("%s check line %q collides with real SERVICER code", ch.Name, line)
				}
			}
		}
	})
}

func TestRoster(t *testing.T) {
	t.Run("happy: five spotlit chunks, five dresses for the same check, VXV under the last spotlight", func(t *testing.T) {
		chunks := Chunks()
		if len(chunks) != 5 {
			t.Fatalf("the walkthrough covers %d chunks, want 5", len(chunks))
		}
		styles := map[string]bool{}
		checks := map[string]bool{}
		for i, ch := range chunks {
			if ch.Name == "" || len(ch.Source) == 0 || len(ch.Check) == 0 {
				t.Fatalf("chunk %d must carry a name, real source, and a check", i)
			}
			if styles[ch.Style] {
				t.Fatalf("style %q repeats — five versions were asked for", ch.Style)
			}
			styles[ch.Style] = true
			joined := strings.Join(ch.Check, "\n")
			if checks[joined] {
				t.Fatalf("%s repeats another chunk's check verbatim", ch.Name)
			}
			checks[joined] = true
			if !strings.Contains(joined, "NEWJOB") && !strings.Contains(joined, "DANZIG") {
				t.Fatalf("%s check never names the priority machinery", ch.Name)
			}
			if !strings.Contains(ch.Caption, fmt.Sprintf("%d/5", i+1)) || !strings.Contains(ch.Caption, ch.Name) {
				t.Fatalf("%s caption %q must number the stop and name the ops", ch.Name, ch.Caption)
			}
		}
		for _, want := range []string{"assembly", "pseudocode", "fork", "weighing", "stamp"} {
			if !styles[want] {
				t.Fatalf("the promised %q look is missing", want)
			}
		}
		if chunks[4].Name != "VXV VSL2" {
			t.Fatalf("the last spotlight belongs to the cross product, got %q", chunks[4].Name)
		}
	})
	t.Run("happy: the tail is more than an EXIT — three real chunks fade out below the last stop", func(t *testing.T) {
		eps := EpilogueBlocks()
		if len(eps) < 3 {
			t.Fatalf("the tail holds %d chunks, want at least 3 so the fade has code to sink through", len(eps))
		}
		for i, ep := range eps {
			if len(ep) == 0 {
				t.Fatalf("tail chunk %d is empty", i)
			}
		}
		if len(MidLines()) == 0 {
			t.Fatal("the DOT block between the last two stops must still scroll by")
		}
	})
	t.Run("unhappy: no expanded line outgrows the card and the stamp's borders align", func(t *testing.T) {
		all := [][]string{PrologueLines(), MidLines()}
		for _, ch := range Chunks() {
			all = append(all, ch.Source, []string{ch.Intro}, ch.Check)
		}
		all = append(all, EpilogueBlocks()...)
		for _, group := range all {
			for _, line := range code.New(code.LangAGC, group).Lines() {
				if n := len([]rune(line)); n > 88 {
					t.Fatalf("line %q runs %d runes expanded — past 88 the column no longer fits the stage", line, n)
				}
			}
		}
		var stamp []string
		for _, ch := range Chunks() {
			if ch.Style == "stamp" {
				stamp = ch.Check
			}
		}
		expanded := code.New(code.LangAGC, stamp).Lines()
		if len(expanded) < 3 {
			t.Fatalf("the stamp needs a top, a body, and a bottom, got %d lines", len(expanded))
		}
		w := len([]rune(expanded[0]))
		for _, line := range expanded {
			if len([]rune(line)) != w {
				t.Fatalf("the stamp's borders tear: %q is not %d runes wide", line, w)
			}
		}
	})
}

func TestOpening(t *testing.T) {
	t.Run("happy: the ΔV load spotlit at the anchor, the prologue dim above, the fade running down", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := paint(s)
		vx, vy := mustSee(t, scr, "VLOAD")
		if vy != anchor() {
			t.Fatalf("the spotlit chunk's first row sits at %d, want the anchor %d", vy, anchor())
		}
		ix, iy := mustSee(t, scr, "INTPRET")
		if iy >= vy {
			t.Fatalf("the prologue (row %d) rides above the spotlight (row %d)", iy, vy)
		}
		px, py := mustSee(t, scr, "PGUIDE")
		if py <= vy {
			t.Fatalf("the next chunk (row %d) waits below the spotlight (row %d)", py, vy)
		}
		mx, my := mustSee(t, scr, "MUNGRAV")
		if my <= py {
			t.Fatalf("the chunk after (row %d) sits lower still (row %d)", my, py)
		}
		lit := code.Dim(code.Foam, 0)
		dim := code.Dim(code.Foam, 1)
		faint := code.Dim(code.Foam, 2)
		if got := fgAt(scr, vx, vy); got != code.Dim(code.Iris, 0) {
			t.Fatalf("the spotlit opcode wears %d, want the lit iris %d", got, code.Dim(code.Iris, 0))
		}
		if got := fgOf(t, scr, "KPIP2"); got != lit {
			t.Fatalf("the spotlit operand wears %d, want the lit foam %d", got, lit)
		}
		if got := fgAt(scr, ix, iy); got != dim {
			t.Fatalf("the prologue wears %d, want the vignette's dim foam %d", got, dim)
		}
		if got := fgAt(scr, px, py); got != dim {
			t.Fatalf("one chunk below wears %d, want the same dim foam %d — equally shaded both sides", got, dim)
		}
		if got := fgAt(scr, mx, my); got != faint {
			t.Fatalf("two chunks below wear %d, want the faint %d", got, faint)
		}
		mustNotSee(t, scr, "SL1")
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
	t.Run("happy: the NEWJOB in the spotlit check wears the love mark", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := paint(s)
		if got := fgOf(t, scr, "NEWJOB"); got != code.Love {
			t.Fatalf("the marked NEWJOB wears %d, want love %d", got, code.Love)
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
		_, mid := mustSee(t, scr, "2ND PUSH")
		if mid <= anchor() {
			t.Fatalf("mid-glide the next chunk still rides below the anchor: %d", mid)
		}
		scr = seek(t, s, 0.2)
		_, later := mustSee(t, scr, "2ND PUSH")
		if later > mid {
			t.Fatalf("the code must scroll upward: %d then %d", mid, later)
		}
		scr = seek(t, s, stock.StopStart(1)-stock.GlideStart(0)-0.5-0.02)
		_, before := mustSee(t, scr, "2ND PUSH")
		if before != anchor() {
			t.Fatalf("one frame before the hold the chunk sits at %d, want the anchor %d", before, anchor())
		}
		scr = seek(t, s, 0.1)
		if _, after := mustSee(t, scr, "2ND PUSH"); after != before {
			t.Fatalf("the chunk hopped %d→%d on the very frame its hold began", before, after)
		}
	})
	t.Run("happy: the last stop is the VXV itself, marked gold, the tail fading through three levels", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, stock.StopStart(4)+0.1)
		vx, vy := mustSee(t, scr, "VXV")
		if vy != anchor() {
			t.Fatalf("VXV sits at %d, want the anchor %d", vy, anchor())
		}
		if got := fgAt(scr, vx, vy); got != code.Gold {
			t.Fatalf("the V cross V wears %d, want the gold mark %d", got, code.Gold)
		}
		mustSee(t, scr, "CARRY ON")
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
		_, vy := mustSee(t, scr, "VXV")
		if vy != anchor() {
			t.Fatalf("VXV must still hold the anchor %d, got %d", anchor(), vy)
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
			t.Fatalf("a fast hold must already spotlight the second chunk: %d", got)
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
