package checkprio

// Tests written FIRST: Check Priority is the code scene behind the
// Interpreter's one-line call. The whole DANZIG check as one C-style
// function — check_for_higher_priority_jobs() — that walks every
// core set, pulls the twelfth word out of the data array
// (core_sets[i].data[11], the PRIORITY word), compares new against
// old, and whichever holds the highest priority wins the CPU. The
// scene reveals the function one line per beat over the Rose Pine
// floor, then walks it: a gold cursor steps through the lines in
// execution order — old, the loop, the read, the compare, the win,
// the winner, the run — one caption per step, and rests forever on
// the run line. A resize keeps the clock; Stop then Start replays; a
// nil screen never panics.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/code"
	"github.com/theprimeagen/apollo-11/exec-tui/components/danzig"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 100
	stageH = 30
)

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

// lineAt is the stage row the given function line paints on, found by
// its own text.
func lineAt(t *testing.T, scr *screenplay.Screen, line int) int {
	t.Helper()
	_, y := mustSee(t, scr, strings.TrimSpace(Lines()[line]))
	return y
}

// cursorRow is where the gold walk cursor sits, or -1.
func cursorRow(scr *screenplay.Screen) int {
	_, y, ok := findOn(scr, "▸")
	if !ok {
		return -1
	}
	return y
}

func TestFunction(t *testing.T) {
	t.Run("happy: the function is the C-style scan — every core set's twelfth word, new against old", func(t *testing.T) {
		lines := Lines()
		if got := lines[LineName]; got != "check_for_higher_priority_jobs()" {
			t.Fatalf("the function opens %q, want the call the Interpreter makes", got)
		}
		wants := map[int]string{
			LineOld:    "old = -0",
			LineFor:    "i < 8",
			LineRead:   "new = core_sets[i].data[11]",
			LineIf:     "if (new > old)",
			LineWin:    "old = new",
			LineWinner: "winner = i",
			LineRun:    "run(core_sets[winner])",
		}
		for idx, want := range wants {
			if !strings.Contains(lines[idx], want) {
				t.Fatalf("line %d reads %q, want it to carry %q", idx, lines[idx], want)
			}
		}
	})
	t.Run("happy: the walk speaks every beat in execution order", func(t *testing.T) {
		steps := WalkSteps()
		wantOrder := []int{LineOld, LineFor, LineRead, LineIf, LineWin, LineWinner, LineRun}
		if len(steps) != len(wantOrder) {
			t.Fatalf("the walk speaks %d steps, want %d", len(steps), len(wantOrder))
		}
		for i, st := range steps {
			if st.Line != wantOrder[i] {
				t.Fatalf("step %d walks line %d, want %d", i, st.Line, wantOrder[i])
			}
			if strings.TrimSpace(st.Text) == "" {
				t.Fatalf("step %d must speak", i)
			}
		}
		if !strings.Contains(steps[2].Text, "data[11]") {
			t.Fatalf("the read step must name the twelfth word, says %q", steps[2].Text)
		}
		if !strings.Contains(steps[4].Text, "wins") {
			t.Fatalf("the compare's outcome must say who wins, says %q", steps[4].Text)
		}
	})
	t.Run("unhappy: the card seats beside the Core Sets Two boxes and never walks a brace", func(t *testing.T) {
		for _, line := range Lines() {
			if n := len([]rune(line)); n > 32 {
				t.Fatalf("line %q runs %d runes — past 32 it no longer seats beside the scan", line, n)
			}
			if strings.Contains(line, "\t") {
				t.Fatalf("line %q carries a tab — the pseudo card is spaces only", line)
			}
		}
		for i, st := range WalkSteps() {
			if st.Line < 0 || st.Line >= len(Lines()) {
				t.Fatalf("step %d points off the card: %d", i, st.Line)
			}
			if s := strings.TrimSpace(Lines()[st.Line]); s == "{" || s == "}" {
				t.Fatalf("step %d rests on a bare brace %q — nothing to say there", i, s)
			}
		}
	})
}

func TestReveal(t *testing.T) {
	t.Run("happy: one line per beat over the Rose Pine floor, under the reveal caption", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := paint(s)
		if got := bgAt(scr, 0, 0); got != danzig.Base256 {
			t.Fatalf("the stage floor wears %d, want the Rose Pine base %d", got, danzig.Base256)
		}
		scr = seek(t, s, 0.3)
		mustSee(t, scr, "check_for_higher_priority_jobs()")
		mustNotSee(t, scr, "old = -0")
		mustSee(t, scr, CaptionReveal)
		scr = seek(t, s, float64(LineRead)*RevealBeat)
		mustSee(t, scr, "old = -0")
		mustSee(t, scr, strings.TrimSpace(Lines()[LineRead]))
	})
	t.Run("happy: the reveal ends parked — the whole card up before the walk begins", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, WalkStart()-0.2)
		for _, line := range Lines() {
			if strings.TrimSpace(line) == "" {
				continue
			}
			mustSee(t, scr, strings.TrimSpace(line))
		}
		mustNotSee(t, scr, "▸")
	})
	t.Run("unhappy: none of the walk captions speak before their beats", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, WalkStart()-0.2)
		mustNotSee(t, scr, WalkSteps()[0].Text)
		mustNotSee(t, scr, CaptionHold)
	})
}

func TestWalk(t *testing.T) {
	t.Run("happy: the gold cursor steps the lines in order, the captions speaking each beat", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		steps := WalkSteps()
		scr := seek(t, s, StepAt(0)+0.1)
		cx, cy, ok := findOn(scr, "▸")
		if !ok {
			t.Fatal("the walk must raise its cursor")
		}
		if got := fgAt(scr, cx, cy); got != code.Gold {
			t.Fatalf("the cursor wears %d, want gold %d", got, code.Gold)
		}
		if want := lineAt(t, scr, steps[0].Line); cy != want {
			t.Fatalf("step 0 rests on row %d, want line %d's row %d", cy, steps[0].Line, want)
		}
		mustSee(t, scr, steps[0].Text)
		scr = seek(t, s, StepBeat)
		if got, want := cursorRow(scr), lineAt(t, scr, steps[1].Line); got != want {
			t.Fatalf("step 1 rests on row %d, want %d", got, want)
		}
		mustSee(t, scr, steps[1].Text)
		mustNotSee(t, scr, steps[0].Text)
		scr = seek(t, s, StepAt(2)-StepAt(1))
		if got, want := cursorRow(scr), lineAt(t, scr, steps[2].Line); got != want {
			t.Fatalf("the read step rests on row %d, want %d", got, want)
		}
		mustSee(t, scr, steps[2].Text)
	})
	t.Run("happy: the walk ends resting on the run line, forever", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, HoldStart()+0.5)
		if got, want := cursorRow(scr), lineAt(t, scr, LineRun); got != want {
			t.Fatalf("the hold rests on row %d, want the run line's row %d", got, want)
		}
		mustSee(t, scr, CaptionHold)
		_, ry := mustSee(t, scr, "run(core_sets[winner])")
		before := rowText(scr, ry)
		scr = seek(t, s, 30)
		if got := rowText(scr, ry); got != before {
			t.Fatalf("the hold drifted:\n%q\n%q", before, got)
		}
	})
	t.Run("unhappy: the cursor never jumps the gun", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, 0.6)
		mustNotSee(t, scr, "▸")
		mustNotSee(t, scr, WalkSteps()[0].Text)
	})
}

func TestLifecycle(t *testing.T) {
	t.Run("happy: a resize keeps the clock — the walk stays walked", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		_ = seek(t, s, StepAt(2)+0.2)
		big := screenplay.NewScreen(110, 34)
		s.Render(big)
		if _, _, ok := findOn(big, "▸"); !ok {
			t.Fatal("after a resize the cursor must still be up")
		}
		mustSee(t, big, "run(core_sets[winner])")
	})
	t.Run("happy: Stop then Start replays from the top", func(t *testing.T) {
		s := opened(t)
		_ = seek(t, s, HoldStart()+1)
		s.Stop()
		s.Start()
		scr := paint(s)
		mustSee(t, scr, "check_for_higher_priority_jobs()")
		mustNotSee(t, scr, "run(core_sets[winner])")
		mustNotSee(t, scr, "▸")
		s.Stop()
	})
	t.Run("happy: the bill is one scene named Check Priority", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the bill holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "Check Priority" || b[0].Scene == nil {
			t.Fatalf("the bill must carry the Check Priority scene, got %+v", b[0])
		}
	})
	t.Run("unhappy: a nil screen, a tiny stage, and dt<=0 never break the show", func(t *testing.T) {
		s := New()
		s.Start()
		s.Render(nil)
		tiny := screenplay.NewScreen(8, 3)
		s.Render(tiny)
		s.Stop()

		s2 := opened(t)
		defer s2.Stop()
		scr := paint(s2)
		_, before := mustSee(t, scr, "check_for_higher_priority_jobs()")
		s2.Update(0)
		s2.Update(-2)
		scr = paint(s2)
		if _, after := mustSee(t, scr, "check_for_higher_priority_jobs()"); after != before {
			t.Fatal("dt<=0 must hold the card still")
		}
	})
}
