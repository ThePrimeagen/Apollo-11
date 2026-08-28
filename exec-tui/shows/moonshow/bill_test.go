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
