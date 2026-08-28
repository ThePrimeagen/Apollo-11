package alarms

// Tests written FIRST: Alarms is the allocation lesson as two
// C-style pseudo functions, walked to both of the landing's famous
// codes. find_free_core_set() loops the eight core sets, pulls the
// in-use word out of the data array — core_sets[i].data[11], the
// PRIORITY word, -0 when the set is free — continues past busy sets,
// returns the first free index, and at the bottom of the loop throws
// error 1202. find_free_vac_area() is the same walk over the five
// VAC areas — vac_areas[i].data[0] is the use word, 0 when claimed,
// its own address when free — and throws 1201. Each function
// reveals one line per beat, walks a happy pass (a free slot found,
// its index returned), then the unhappy pass (everything busy, the
// throw line burning alarm red under a PROG ALARM chip). The scene
// ends holding both codes in the caption. A resize keeps the clock;
// Stop then Start replays; a nil screen never panics.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

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

// cursorRow is where the walk cursor sits, or -1.
func cursorRow(scr *screenplay.Screen) int {
	_, y, ok := findOn(scr, "▸")
	if !ok {
		return -1
	}
	return y
}

func TestFunctions(t *testing.T) {
	t.Run("happy: two functions, one shape — the loop, the read, the check, continue, return, throw", func(t *testing.T) {
		for name, lines := range map[string][]string{"core": CoreLines(), "vac": VACLines()} {
			if len(lines) != 11 {
				t.Fatalf("%s reads in %d lines, want 11 — the same shape both times", name, len(lines))
			}
			shape := map[int]string{
				LineName:     "()",
				LineFor:      "for (i = 0;",
				LineRead:     "in_use = ",
				LineIf:       "if (in_use",
				LineContinue: "continue",
				LineReturn:   "return i",
				LineThrow:    "throw new error(",
			}
			for idx, want := range shape {
				if !strings.Contains(lines[idx], want) {
					t.Fatalf("%s line %d reads %q, want it to carry %q", name, idx, lines[idx], want)
				}
			}
		}
	})
	t.Run("happy: the reads pull the real words out of the data arrays", func(t *testing.T) {
		if !strings.Contains(CoreLines()[LineRead], "core_sets[i].data[11]") {
			t.Fatalf("the core read %q must pull the twelfth word — the PRIORITY register", CoreLines()[LineRead])
		}
		if !strings.Contains(CoreLines()[LineFor], "i < 8") {
			t.Fatalf("the core loop %q must walk all eight sets", CoreLines()[LineFor])
		}
		if !strings.Contains(CoreLines()[LineIf], "!= -0") {
			t.Fatalf("the core check %q must test against -0 — a free set's PRIORITY", CoreLines()[LineIf])
		}
		if !strings.Contains(VACLines()[LineRead], "vac_areas[i].data[0]") {
			t.Fatalf("the vac read %q must pull the use word — the area's first word", VACLines()[LineRead])
		}
		if !strings.Contains(VACLines()[LineFor], "i < 5") {
			t.Fatalf("the vac loop %q must walk all five areas", VACLines()[LineFor])
		}
		if !strings.Contains(VACLines()[LineIf], "== 0") {
			t.Fatalf("the vac check %q must test against 0 — a claimed area's use word", VACLines()[LineIf])
		}
	})
	t.Run("unhappy: the codes never swap and no line outgrows the card", func(t *testing.T) {
		if !strings.Contains(CoreLines()[LineThrow], "1202") || strings.Contains(strings.Join(CoreLines(), "\n"), "1201") {
			t.Fatalf("find_free_core_set must throw 1202 and only 1202: %q", CoreLines()[LineThrow])
		}
		if !strings.Contains(VACLines()[LineThrow], "1201") || strings.Contains(strings.Join(VACLines(), "\n"), "1202") {
			t.Fatalf("find_free_vac_area must throw 1201 and only 1201: %q", VACLines()[LineThrow])
		}
		for _, lines := range [][]string{CoreLines(), VACLines()} {
			for _, line := range lines {
				if n := len([]rune(line)); n > 40 {
					t.Fatalf("line %q runs %d runes — past 40 the card crowds the stage", line, n)
				}
				if strings.Contains(line, "\t") {
					t.Fatalf("line %q carries a tab — the pseudo card is spaces only", line)
				}
			}
		}
	})
}

func TestScans(t *testing.T) {
	t.Run("happy: the core walk finds the third set free and returns its index", func(t *testing.T) {
		steps := CoreScan(CoreHappy())
		want := []Step{
			{LineRead, "core_sets[0].data[11] = 20400"},
			{LineContinue, "in_use != -0 — a job holds this set · continue"},
			{LineRead, "core_sets[1].data[11] = 32000"},
			{LineContinue, "in_use != -0 — a job holds this set · continue"},
			{LineRead, "core_sets[2].data[11] = -0"},
			{LineReturn, "-0 — free · return 2: the new job moves in"},
		}
		if len(steps) != len(want) {
			t.Fatalf("the happy core walk speaks %d steps, want %d", len(steps), len(want))
		}
		for i, w := range want {
			if steps[i] != w {
				t.Fatalf("core step %d is %+v, want %+v", i, steps[i], w)
			}
		}
	})
	t.Run("happy: the vac walk finds the second area holding its own address and returns 1", func(t *testing.T) {
		steps := VACScan(VACHappy())
		want := []Step{
			{LineRead, "vac_areas[0].data[0] = 0"},
			{LineContinue, "in_use == 0 — a job claimed this area · continue"},
			{LineRead, "vac_areas[1].data[0] = 454"},
			{LineReturn, "454 — its own address: free · return 1"},
		}
		if len(steps) != len(want) {
			t.Fatalf("the happy vac walk speaks %d steps, want %d", len(steps), len(want))
		}
		for i, w := range want {
			if steps[i] != w {
				t.Fatalf("vac step %d is %+v, want %+v", i, steps[i], w)
			}
		}
	})
	t.Run("unhappy: a full pool walks every set and falls off the loop into the throw", func(t *testing.T) {
		full := CoreScan(CoreFull())
		if len(full) != 2*len(CoreFull())+1 {
			t.Fatalf("the full core walk speaks %d steps, want %d — a read and a continue per set, then the throw", len(full), 2*len(CoreFull())+1)
		}
		last := full[len(full)-1]
		if last.Line != LineThrow || !strings.Contains(last.Text, "1202") {
			t.Fatalf("the full core walk must end on the 1202 throw, got %+v", last)
		}
		for i := 0; i < len(full)-1; i += 2 {
			if full[i].Line != LineRead || full[i+1].Line != LineContinue {
				t.Fatalf("full core steps %d,%d must be a read then a continue, got %+v %+v", i, i+1, full[i], full[i+1])
			}
		}
		vfull := VACScan(VACFull())
		if len(vfull) != 2*len(VACFull())+1 {
			t.Fatalf("the full vac walk speaks %d steps, want %d", len(vfull), 2*len(VACFull())+1)
		}
		vlast := vfull[len(vfull)-1]
		if vlast.Line != LineThrow || !strings.Contains(vlast.Text, "1201") {
			t.Fatalf("the full vac walk must end on the 1201 throw, got %+v", vlast)
		}
	})
	t.Run("unhappy: an empty pool throws at once, a free-first pool returns 0 in two steps", func(t *testing.T) {
		if steps := CoreScan(nil); len(steps) != 1 || steps[0].Line != LineThrow {
			t.Fatalf("no sets, no walk — just the throw; got %+v", steps)
		}
		if steps := VACScan(nil); len(steps) != 1 || steps[0].Line != LineThrow {
			t.Fatalf("no areas, no walk — just the throw; got %+v", steps)
		}
		steps := CoreScan([]Probe{{Value: "-0", Free: true}})
		if len(steps) != 2 || steps[0].Line != LineRead || steps[1].Line != LineReturn {
			t.Fatalf("a free first set returns immediately, got %+v", steps)
		}
		if !strings.Contains(steps[1].Text, "return 0") {
			t.Fatalf("the free first set is index 0, step says %q", steps[1].Text)
		}
	})
}

func TestCoreAct(t *testing.T) {
	t.Run("happy: the function reveals one line per beat over the Rose Pine floor", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := paint(s)
		if got := bgAt(scr, 0, 0); got != danzig.Base256 {
			t.Fatalf("the stage floor wears %d, want the Rose Pine base %d", got, danzig.Base256)
		}
		scr = seek(t, s, 0.3)
		mustSee(t, scr, "find_free_core_set()")
		mustNotSee(t, scr, "in_use")
		mustSee(t, scr, CaptionCore)
		scr = seek(t, s, CoreHappyStart()-0.5-0.3)
		mustSee(t, scr, "throw new error(1202)")
		mustNotSee(t, scr, "▸")
	})
	t.Run("happy: the happy walk reads, continues past busy sets, and returns the free index", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		steps := CoreScan(CoreHappy())
		scr := seek(t, s, CoreHappyStart()+0.1)
		_, readRow := mustSee(t, scr, "in_use = core_sets[i].data[11]")
		if got := cursorRow(scr); got != readRow {
			t.Fatalf("the first beat reads: cursor on row %d, want the read line's row %d", got, readRow)
		}
		mustSee(t, scr, steps[0].Text)
		scr = seek(t, s, HappyBeat)
		_, contRow := mustSee(t, scr, "continue")
		if got := cursorRow(scr); got != contRow {
			t.Fatalf("the second beat continues: cursor on row %d, want %d", got, contRow)
		}
		mustSee(t, scr, steps[1].Text)
		scr = seek(t, s, float64(len(steps)-2)*HappyBeat)
		_, retRow := mustSee(t, scr, "return i")
		if got := cursorRow(scr); got != retRow {
			t.Fatalf("the walk must end on return i: cursor on row %d, want %d", got, retRow)
		}
		mustSee(t, scr, "return 2")
	})
	t.Run("happy: the full pool ends on the throw — the line burns red under the chip", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, CoreAlarmAt()+0.1)
		tx, ty := mustSee(t, scr, "throw new error(1202)")
		if got := fgAt(scr, tx, ty); got != AlarmInk {
			t.Fatalf("the thrown line wears %d, want the alarm red %d", got, AlarmInk)
		}
		cx, cy := mustSee(t, scr, "PROG ALARM 1202")
		if got := bgAt(scr, cx, cy); got != AlarmBG {
			t.Fatalf("the chip's floor wears %d, want the alarm red %d", got, AlarmBG)
		}
		mustSee(t, scr, "NO CORE SETS AVAILABLE")
	})
	t.Run("unhappy: no chip and no red before the loop actually falls off the end", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, CoreFullStart()+0.1)
		mustNotSee(t, scr, "PROG ALARM")
		tx, ty := mustSee(t, scr, "throw new error(1202)")
		if got := fgAt(scr, tx, ty); got == AlarmInk {
			t.Fatal("the throw line must stay calm until the loop runs out")
		}
	})
}

func TestVACAct(t *testing.T) {
	t.Run("happy: the vac card takes the stage — the core card gone — and walks to return 1", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, VACStart()+0.2)
		mustSee(t, scr, "find_free_vac_area()")
		mustNotSee(t, scr, "find_free_core_set()")
		mustSee(t, scr, CaptionVAC)
		scr = seek(t, s, VACHappyStart()-VACStart()-0.2+float64(len(VACScan(VACHappy()))-1)*HappyBeat+0.1)
		mustSee(t, scr, "return 1")
		_, retRow := mustSee(t, scr, "return i")
		if got := cursorRow(scr); got != retRow {
			t.Fatalf("the vac walk must end on return i: cursor on row %d, want %d", got, retRow)
		}
	})
	t.Run("happy: the 1201 throw, then the scene rests naming both codes", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, VACAlarmAt()+0.1)
		tx, ty := mustSee(t, scr, "throw new error(1201)")
		if got := fgAt(scr, tx, ty); got != AlarmInk {
			t.Fatalf("the thrown line wears %d, want the alarm red %d", got, AlarmInk)
		}
		mustSee(t, scr, "PROG ALARM 1201")
		scr = seek(t, s, VACEnd()-VACAlarmAt()-0.1+0.3)
		mustSee(t, scr, CaptionFinal)
		_, y := mustSee(t, scr, "find_free_vac_area()")
		before := rowText(scr, y)
		scr = seek(t, s, 20)
		if got := rowText(scr, y); got != before {
			t.Fatalf("the final hold drifted:\n%q\n%q", before, got)
		}
		mustSee(t, scr, CaptionFinal)
	})
	t.Run("unhappy: the core chip never leaks into the vac act", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, VACHappyStart()+0.1)
		mustNotSee(t, scr, "PROG ALARM 1202")
		mustNotSee(t, scr, "1202")
	})
}

func TestLifecycle(t *testing.T) {
	t.Run("happy: a resize keeps the clock — no fall back to the core reveal", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		_ = seek(t, s, VACStart()+1)
		big := screenplay.NewScreen(110, 34)
		s.Render(big)
		if _, _, ok := findOn(big, "find_free_vac_area()"); !ok {
			t.Fatal("after a resize the vac act must still be on stage")
		}
		if _, _, ok := findOn(big, "find_free_core_set()"); ok {
			t.Fatal("a resize must not rewind to the core act")
		}
	})
	t.Run("happy: Stop then Start replays from the top", func(t *testing.T) {
		s := opened(t)
		_ = seek(t, s, VACEnd()+1)
		s.Stop()
		s.Start()
		scr := paint(s)
		mustSee(t, scr, "find_free_core_set()")
		mustNotSee(t, scr, "find_free_vac_area()")
		s.Stop()
	})
	t.Run("happy: the bill is one scene named Alarms", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the bill holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "Alarms" || b[0].Scene == nil {
			t.Fatalf("the bill must carry the Alarms scene, got %+v", b[0])
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
		_, before := mustSee(t, scr, "find_free_core_set()")
		s2.Update(0)
		s2.Update(-2)
		scr = paint(s2)
		if _, after := mustSee(t, scr, "find_free_core_set()"); after != before {
			t.Fatal("dt<=0 must hold the card still")
		}
	})
}
