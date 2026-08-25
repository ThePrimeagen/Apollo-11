package cast

// Tests written FIRST: the starfield actor wraps stars.Field so a scene
// can hang the sky like any other performer. It sizes itself to the
// screen it renders to, drifts with its own clock, and holds under a
// still strategy, a held clock, or no screen at all.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"

	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

func skySnapshot(f *Starfield, w, h int) map[[2]int]string {
	scr := screenplay.NewScreen(w, h)
	f.Render(scr)
	snap := map[[2]int]string{}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if s := contentAt(scr, x, y); s != "" && s != " " {
				snap[[2]int{x, y}] = s
			}
		}
	}
	return snap
}

func equalSky(a, b map[[2]int]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestStarfield(t *testing.T) {
	t.Run("happy: the sky fills the screen with all four stars, tinted", func(t *testing.T) {
		f := NewStarfield(stars.Drift)
		scr := screenplay.NewScreen(screenW, 26)
		f.Render(scr)
		found := map[string]bool{}
		for y := 0; y < 26; y++ {
			for x := 0; x < screenW; x++ {
				cell := scr.Cell(x, y)
				if cell == nil || cell.Content == " " || cell.Content == "" {
					continue
				}
				found[cell.Content] = true
				if cell.Style.Fg == nil {
					t.Fatalf("star at (%d,%d) has no tint", x, y)
				}
			}
		}
		for _, g := range stars.Glyphs {
			if !found[string(g)] {
				t.Fatalf("sky missing star %q", string(g))
			}
		}
	})
	t.Run("happy: a drifting sky moves as its clock runs", func(t *testing.T) {
		f := NewStarfield(stars.Drift)
		before := skySnapshot(f, screenW, 26)
		f.Update(2.0)
		after := skySnapshot(f, screenW, 26)
		if equalSky(before, after) {
			t.Fatal("two seconds of drift left every star in place")
		}
	})
	t.Run("unhappy: a still sky holds no matter how long it runs", func(t *testing.T) {
		f := NewStarfield(stars.Still)
		before := skySnapshot(f, screenW, 26)
		f.Update(3.0)
		if !equalSky(before, skySnapshot(f, screenW, 26)) {
			t.Fatal("a still sky must never move")
		}
	})
	t.Run("unhappy: dt<=0 holds the drift", func(t *testing.T) {
		f := NewStarfield(stars.Drift)
		before := skySnapshot(f, screenW, 26)
		f.Update(0)
		f.Update(-1)
		if !equalSky(before, skySnapshot(f, screenW, 26)) {
			t.Fatal("a held clock must hold the sky")
		}
	})
	t.Run("unhappy: zero screens and nil screens take no stars", func(t *testing.T) {
		f := NewStarfield(stars.Drift)
		scr := screenplay.NewScreen(0, 0)
		f.Render(scr)
		if n := litCount(scr); n != 0 {
			t.Fatalf("zero screen lit %d cells", n)
		}
		f.Render(nil)
	})
}

func TestTunedStarfield(t *testing.T) {
	t.Run("happy: with the stock sky active it renders exactly the drift sky", func(t *testing.T) {
		tuned := skySnapshot(NewTunedStarfield(), screenW, 26)
		drift := skySnapshot(NewStarfield(stars.Drift), screenW, 26)
		if !equalSky(tuned, drift) {
			t.Fatal("an untouched tuned sky must be the stock drift sky, star for star")
		}
	})
	t.Run("happy: a used sky config shows on the very next render", func(t *testing.T) {
		t.Cleanup(stars.ResetSky)
		f := NewTunedStarfield()
		before := countSnapshot(skySnapshot(f, screenW, 26), string(stars.Glyphs[3]))
		if err := stars.UseSky(stars.SkyConfig{
			Delay:   []int{4, 6, 8, 12},
			Density: []int{56, 33, 6, 120},
		}); err != nil {
			t.Fatalf("UseSky: %v", err)
		}
		after := countSnapshot(skySnapshot(f, screenW, 26), string(stars.Glyphs[3]))
		if after <= before*3 {
			t.Fatalf("near density 120 painted ✦%d -> ✦%d; a tuned scene must follow the config", before, after)
		}
	})
	t.Run("unhappy: resetting the sky takes the tuning back out", func(t *testing.T) {
		t.Cleanup(stars.ResetSky)
		f := NewTunedStarfield()
		stock := skySnapshot(f, screenW, 26)
		if err := stars.UseSky(stars.SkyConfig{
			Delay:   []int{1, 1, 1, 1},
			Density: []int{200, 200, 200, 200},
		}); err != nil {
			t.Fatalf("UseSky: %v", err)
		}
		stars.ResetSky()
		if !equalSky(stock, skySnapshot(f, screenW, 26)) {
			t.Fatal("after ResetSky the tuned scene must fly the stock sky again")
		}
	})
}

func countSnapshot(snap map[[2]int]string, glyph string) int {
	n := 0
	for _, s := range snap {
		if s == glyph {
			n++
		}
	}
	return n
}
