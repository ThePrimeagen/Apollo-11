package moonshow

// Tests written FIRST: the moon screenplay is a composable two-scene
// bill. Scene one, "the moon": the bare disc alone under a parked sky
// — nothing on stage moves at all. Scene two, "orbit": a spaceship
// streaks in fast off the left wing, brakes smoothly onto its orbit,
// and circles the moon indefinitely until the next cut — no line
// drawn around the moon, the craft alone traces the path. The bill is
// the composable unit — screenplay.Compose adds bills together into
// one big show — and after this bill's last scene there is nothing
// left.

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
	t.Run("happy: scene two opens exactly like scene one — no line rises, only the ship will", func(t *testing.T) {
		one := Bill()[0].Scene
		one.Start()
		defer one.Stop()
		two := Bill()[1].Scene
		two.Start()
		defer two.Stop()
		if render(two) != render(one) {
			t.Fatal("before the ship enters, the orbit scene is the moon scene — nothing else on stage")
		}
	})
	t.Run("happy: scene two streaks the ship in and orbits it indefinitely", func(t *testing.T) {
		sc := Bill()[1].Scene
		sc.Start()
		defer sc.Stop()
		if strings.ContainsRune(render(sc), moon.MarkerGlyph) {
			t.Fatal("the ship must open off stage")
		}
		sc.Update(moon.ArriveSeconds + 0.5)
		mid := render(sc)
		r0, c0, ok := markerCell(mid)
		if !ok {
			t.Fatal("past the streak the ship must be on stage")
		}
		sc.Update(3)
		late := render(sc)
		r1, c1, ok := markerCell(late)
		if !ok {
			t.Fatal("the ship must keep orbiting")
		}
		if r0 == r1 && c0 == c1 {
			t.Fatal("the orbit must carry the ship on — it loops until the cut")
		}
		if skyColumns(mid, 12) != skyColumns(late, 12) {
			t.Fatal("the sky behind the orbit holds still")
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
