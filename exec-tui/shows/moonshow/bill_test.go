package moonshow

// Tests written FIRST: the moon screenplay is a composable two-scene
// bill. Scene one, "the moon": the bare disc alone under a parked sky
// — nothing on stage moves at all. The cut, not a clock, brings scene
// two. Scene two, "orbit": the lander opens off the left wing, streaks
// in at orbit height, brakes onto the ring, and circles indefinitely
// until the next cut — no line drawn around the moon, the craft alone
// traces the path. It must never already be on the ring at the cut.
// Waiting out the arrival delay on scene one must never conjure the
// lander. The bill is the composable unit — screenplay.Compose adds
// bills together into one big show — and after this bill's last scene
// there is nothing left.

import (
	"regexp"
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 72
	stageH = 27
)

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// render paints the scene onto a fresh stage-sized screen and hands
// back the styled string.
func render(sc screenplay.Scene) string {
	scr := screenplay.NewScreen(stageW, stageH)
	sc.Render(scr)
	return scr.Render()
}

func hasStar(v string) bool {
	for _, g := range stars.Glyphs {
		if strings.ContainsRune(v, g) {
			return true
		}
	}
	return false
}

// markerCell locates the gold ship in a rendered frame, ANSI stripped.
func markerCell(v string) (row, col int, ok bool) {
	for r, line := range strings.Split(ansiPat.ReplaceAllString(v, ""), "\n") {
		for c, ch := range []rune(line) {
			if ch == moon.MarkerGlyph {
				return r, c, true
			}
		}
	}
	return 0, 0, false
}

// skyColumns is the leftmost n columns of every row, ANSI stripped —
// west of the ring that strip is pure sky.
func skyColumns(v string, n int) string {
	var b strings.Builder
	for _, line := range strings.Split(ansiPat.ReplaceAllString(v, ""), "\n") {
		rs := []rune(line)
		if len(rs) > n {
			rs = rs[:n]
		}
		b.WriteString(string(rs))
		b.WriteString("\n")
	}
	return b.String()
}

func TestMoonShowBill(t *testing.T) {
	t.Run("happy: the bill is two scenes in playing order", func(t *testing.T) {
		b := Bill()
		if len(b) != 2 {
			t.Fatalf("the moon screenplay holds %d scenes, want 2", len(b))
		}
		for i, want := range []string{"the moon", "orbit"} {
			if b[i].Name != want {
				t.Fatalf("scene %d is %q, want %q", i+1, b[i].Name, want)
			}
			if b[i].Scene == nil {
				t.Fatalf("scene %q has no performer", want)
			}
		}
	})
	t.Run("happy: scene one is just the moon — and nothing on stage moves", func(t *testing.T) {
		sc := Bill()[0].Scene
		sc.Start()
		defer sc.Stop()
		v := render(sc)
		if !strings.ContainsRune(v, '▓') {
			t.Fatal("the opening scene must show the moon")
		}
		if !hasStar(v) {
			t.Fatal("the opening scene plays under stars")
		}
		if strings.ContainsRune(v, moon.MarkerGlyph) {
			t.Fatal("no ship yet — just the moon")
		}
		sc.Update(3)
		if render(sc) != v {
			t.Fatal("scene one must hold perfectly still, stars and all")
		}
	})
	t.Run("happy: scene two streaks the lander in and orbits it indefinitely", func(t *testing.T) {
		sc := Bill()[1].Scene
		sc.Start()
		defer sc.Stop()
		opening := render(sc)
		if !strings.ContainsRune(opening, '▓') {
			t.Fatal("the orbit scene still shows the moon")
		}
		if strings.ContainsRune(opening, moon.MarkerGlyph) {
			t.Fatal("the lander must open off stage — it flies in, it does not appear")
		}
		sc.Update(moon.ArriveSeconds / 2)
		streak := render(sc)
		r0, c0, ok := markerCell(streak)
		if !ok {
			t.Fatal("mid-streak the lander must be on stage, flying in")
		}
		_, mergeCol := moon.ArrivalAt(stageW, stageH, moon.ArriveSeconds)
		if c0 >= mergeCol {
			t.Fatalf("mid-streak col %d — the lander must still be west of the merge (%d)", c0, mergeCol)
		}
		sc.Update(moon.ArriveSeconds/2 + 0.5)
		onRing := render(sc)
		r1, c1, ok := markerCell(onRing)
		if !ok {
			t.Fatal("past the streak the lander must merge onto the ring")
		}
		if r0 == r1 && c0 == c1 {
			t.Fatal("the streak must carry the lander onto the orbit")
		}
		sc.Update(3)
		late := render(sc)
		r2, c2, ok := markerCell(late)
		if !ok {
			t.Fatal("the lander must keep orbiting")
		}
		if r1 == r2 && c1 == c2 {
			t.Fatal("the orbit must carry the lander on — it loops until the cut")
		}
		if skyColumns(onRing, 12) != skyColumns(late, 12) {
			t.Fatal("the sky behind the orbit holds still")
		}
	})
	t.Run("unhappy: the first frame of scene two never parks the lander on the ring", func(t *testing.T) {
		sc := Bill()[1].Scene
		sc.Start()
		defer sc.Stop()
		v := render(sc)
		row, col := moon.MarkerAt(stageW, stageH, 0)
		lines := strings.Split(ansiPat.ReplaceAllString(v, ""), "\n")
		if row < 0 || row >= len(lines) {
			t.Fatalf("MarkerAt row %d is off the %d-row stage", row, len(lines))
		}
		rs := []rune(lines[row])
		if col < 0 || col >= len(rs) {
			t.Fatalf("MarkerAt col %d is off the %d-col stage", col, len(rs))
		}
		if rs[col] == moon.MarkerGlyph {
			t.Fatal("the stock orbit start still holds the lander — it must fly in, not appear")
		}
		if _, _, ok := markerCell(v); ok {
			t.Fatal("no lander on the first frame — appearing anywhere is the same cheat")
		}
	})
	t.Run("unhappy: waiting on scene one never brings the lander — the cut is the cue", func(t *testing.T) {
		sc := Bill()[0].Scene
		sc.Start()
		defer sc.Stop()
		_ = render(sc)
		sc.Update(moon.ArriveSeconds + 1)
		v := render(sc)
		if strings.ContainsRune(v, moon.MarkerGlyph) {
			t.Fatal("the lander must not appear until the cut — waiting is not a scene")
		}
		if !strings.ContainsRune(v, '▓') {
			t.Fatal("scene one must still be just the moon")
		}
	})
	t.Run("happy: the composed show walks the bill and then has nothing left", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		if p.CurrentName() != "the moon" {
			t.Fatalf("the show opens on %q, want the moon", p.CurrentName())
		}
		if !p.Next() || p.CurrentName() != "orbit" {
			t.Fatalf("the cut must land on orbit, got %q", p.CurrentName())
		}
		if p.Next() {
			t.Fatal("after orbit there is nothing left — the show ends")
		}
		p.Stop()
	})
	t.Run("unhappy: a scene stopped before its first render never panics", func(t *testing.T) {
		for _, e := range Bill() {
			e.Scene.Start()
			e.Scene.Update(1)
			e.Scene.Stop()
		}
	})
}

// Tests written FIRST: the orbit entry grows its editable face — a
// tunable show whose two knobs, the arriving streak and the lap, feed
// the paced orbit. Stock knobs fly the stock show, cell for cell; the
// numbers are the operator's, verbatim — a nudge below zero stands.
// Each Bill() call casts a fresh show, so no two bills share knobs.
func TestOrbitShow(t *testing.T) {
	t.Run("happy: the orbit entry is the tunable orbit show at stock pace", func(t *testing.T) {
		sc, ok := Bill()[1].Scene.(*OrbitShow)
		if !ok {
			t.Fatalf("the orbit entry is %T, want the orbit show", Bill()[1].Scene)
		}
		if sc.Cfg != DefaultOrbitConfig() {
			t.Fatalf("a fresh show carries %+v, want stock", sc.Cfg)
		}
		want := OrbitConfig{ArriveSeconds: moon.ArriveSeconds, LapSeconds: moon.OrbitSeconds}
		if DefaultOrbitConfig() != want {
			t.Fatalf("stock pace is %+v, want the moon consts %+v", DefaultOrbitConfig(), want)
		}
	})
	t.Run("happy: the knob face reads arrive then lap", func(t *testing.T) {
		c := DefaultOrbitConfig()
		if c.KnobCount() != 2 {
			t.Fatalf("the orbit show carries %d knobs, want 2", c.KnobCount())
		}
		if c.KnobLabel(0) != "arrive" || c.KnobLabel(1) != "lap" {
			t.Fatalf("labels %q/%q, want arrive/lap", c.KnobLabel(0), c.KnobLabel(1))
		}
		if c.Value(0) != c.ArriveSeconds || c.Value(1) != c.LapSeconds {
			t.Fatalf("values %v/%v must read the config", c.Value(0), c.Value(1))
		}
		c.Nudge(0, 2)
		if c.ArriveSeconds != moon.ArriveSeconds+0.5 {
			t.Fatalf("two arrive steps read %v, want %v", c.ArriveSeconds, moon.ArriveSeconds+0.5)
		}
		c.Nudge(1, -1)
		if c.LapSeconds != moon.OrbitSeconds-0.25 {
			t.Fatalf("one lap step down reads %v, want %v", c.LapSeconds, moon.OrbitSeconds-0.25)
		}
	})
	t.Run("happy: the knobs reach the stage — a paced lap flies its own ring", func(t *testing.T) {
		sc, ok := Bill()[1].Scene.(*OrbitShow)
		if !ok {
			t.Fatal("the orbit entry must be the orbit show")
		}
		sc.Cfg = OrbitConfig{ArriveSeconds: 0.5, LapSeconds: 4}
		sc.Start()
		defer sc.Stop()
		_ = render(sc) // stage the cast
		sc.Update(0.5 + 1)
		cx, cy, _, ringR := moon.Geometry(stageW, stageH)
		wantRow, wantCol := cy, cx+ringR // a quarter lap past the top merge
		r, c, ok := markerCell(render(sc))
		if !ok {
			t.Fatal("the paced craft must be on stage")
		}
		if r != wantRow || c != wantCol {
			t.Fatalf("the paced craft sits at (%d,%d), want the ring's east point (%d,%d)", r, c, wantRow, wantCol)
		}
	})
	t.Run("unhappy: a nudge below zero stands — never clamped", func(t *testing.T) {
		c := DefaultOrbitConfig()
		c.Nudge(1, -100)
		if want := moon.OrbitSeconds - 25.0; c.LapSeconds != want {
			t.Fatalf("a hundred steps down reads %v, want %v — the floor is the operator's", c.LapSeconds, want)
		}
		c.Nudge(9, 1) // a bad cursor is a no-op
		if c.ArriveSeconds != moon.ArriveSeconds {
			t.Fatal("a bad cursor must not move any knob")
		}
	})
	t.Run("unhappy: no two bills share knobs", func(t *testing.T) {
		one := Bill()[1].Scene.(*OrbitShow)
		two := Bill()[1].Scene.(*OrbitShow)
		one.Cfg.Nudge(1, 4)
		if two.Cfg.LapSeconds != moon.OrbitSeconds {
			t.Fatal("nudging one bill's orbit must not touch another's")
		}
	})
}
