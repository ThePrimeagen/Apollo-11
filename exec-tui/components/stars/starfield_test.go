package stars

// Tests written FIRST: the Starfield is the sky as a scene component.
// Start(w, h) scatters and caches the catalog for that stage — a tuned
// sky samples the active config right there — Update runs the fly
// clock, Render paints the cached catalog into a stage-sized sprite,
// and Stop deletes the catalog so a stopped sky holds no allocation.
// Start may come again: the sky re-scatters and flies on.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 72
	stageH = 26
)

// The compile-time pin: a Starfield plays as a screenplay component.
var _ screenplay.Component = (*Starfield)(nil)

func spriteSnapshot(sp sprite.Sprite) map[[2]int]rune {
	snap := map[[2]int]rune{}
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if cell := sp.At(r, c); !cell.Transparent() {
				snap[[2]int{r, c}] = cell.Ch
			}
		}
	}
	return snap
}

func snapshotsEqual(a, b map[[2]int]rune) bool {
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

func countGlyph(snap map[[2]int]rune, glyph rune) int {
	n := 0
	for _, ch := range snap {
		if ch == glyph {
			n++
		}
	}
	return n
}

func TestStarfieldComponent(t *testing.T) {
	t.Run("happy: a started sky fills its stage with all four stars, tinted", func(t *testing.T) {
		f := NewStarfield(Drift)
		f.Start(stageW, stageH)
		sp := f.Render()
		if sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("stage sprite %dx%d, want %dx%d", sp.Width, sp.Height, stageW, stageH)
		}
		found := map[rune]bool{}
		for r := 0; r < sp.Height; r++ {
			for c := 0; c < sp.Width; c++ {
				cell := sp.At(r, c)
				if cell.Transparent() {
					continue
				}
				found[cell.Ch] = true
				if cell.FG < 0 {
					t.Fatalf("star at (%d,%d) has no tint", r, c)
				}
			}
		}
		for _, g := range Glyphs {
			if !found[g] {
				t.Fatalf("sky missing star %q", string(g))
			}
		}
	})
	t.Run("happy: a drifting sky moves as its clock runs", func(t *testing.T) {
		f := NewStarfield(Drift)
		f.Start(stageW, stageH)
		before := spriteSnapshot(f.Render())
		f.Update(2.0)
		if snapshotsEqual(before, spriteSnapshot(f.Render())) {
			t.Fatal("two seconds of drift left every star in place")
		}
	})
	t.Run("happy: stop deletes the catalog; a fresh start scatters again", func(t *testing.T) {
		f := NewStarfield(Drift)
		f.Start(stageW, stageH)
		lit := spriteSnapshot(f.Render())
		if len(lit) == 0 {
			t.Fatal("test premise: a started sky must hold stars")
		}
		f.Stop()
		if sp := f.Render(); len(spriteSnapshot(sp)) != 0 {
			t.Fatal("a stopped sky must render empty — its catalog is gone")
		}
		f.Start(stageW, stageH)
		if !snapshotsEqual(spriteSnapshot(f.Render()), lit) {
			t.Fatal("a restarted sky must re-scatter the same deterministic homes")
		}
	})
	t.Run("happy: a restart at a new stage size re-scatters to fit", func(t *testing.T) {
		f := NewStarfield(Drift)
		f.Start(stageW, stageH)
		f.Stop()
		f.Start(30, 10)
		sp := f.Render()
		if sp.Width != 30 || sp.Height != 10 {
			t.Fatalf("restaged sprite %dx%d, want 30x10", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a still sky holds no matter how long it runs", func(t *testing.T) {
		f := NewStarfield(Still)
		f.Start(stageW, stageH)
		before := spriteSnapshot(f.Render())
		f.Update(3.0)
		if !snapshotsEqual(before, spriteSnapshot(f.Render())) {
			t.Fatal("a still sky must never move")
		}
	})
	t.Run("unhappy: dt<=0 holds the drift", func(t *testing.T) {
		f := NewStarfield(Drift)
		f.Start(stageW, stageH)
		before := spriteSnapshot(f.Render())
		f.Update(0)
		f.Update(-1)
		if !snapshotsEqual(before, spriteSnapshot(f.Render())) {
			t.Fatal("a held clock must hold the sky")
		}
	})
	t.Run("unhappy: rendering before the first start is an empty stage", func(t *testing.T) {
		f := NewStarfield(Drift)
		if sp := f.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("an unstarted sky rendered %dx%d", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: zero stages and nil skies take no stars", func(t *testing.T) {
		f := NewStarfield(Drift)
		f.Start(0, 0)
		if got := spriteSnapshot(f.Render()); len(got) != 0 {
			t.Fatalf("zero stage painted %d stars", len(got))
		}
		var ghost *Starfield
		ghost.Start(4, 4)
		ghost.Update(1)
		ghost.Render()
		ghost.Stop()
	})
}

func TestTunedStarfieldComponent(t *testing.T) {
	t.Run("happy: with the stock sky active it flies exactly the drift sky", func(t *testing.T) {
		tuned := NewTunedStarfield()
		tuned.Start(stageW, stageH)
		drift := NewStarfield(Drift)
		drift.Start(stageW, stageH)
		if !snapshotsEqual(spriteSnapshot(tuned.Render()), spriteSnapshot(drift.Render())) {
			t.Fatal("an untouched tuned sky must be the stock drift sky, star for star")
		}
	})
	t.Run("happy: start samples the active sky — the config it opens with", func(t *testing.T) {
		t.Cleanup(ResetSky)
		if err := UseSky(SkyConfig{
			Delay:   []int{4, 6, 8, 12},
			Density: []int{56, 33, 6, 120},
		}); err != nil {
			t.Fatalf("UseSky: %v", err)
		}
		f := NewTunedStarfield()
		f.Start(stageW, stageH)
		stock := NewStarfield(Drift)
		stock.Start(stageW, stageH)
		tuned := countGlyph(spriteSnapshot(f.Render()), Glyphs[3])
		plain := countGlyph(spriteSnapshot(stock.Render()), Glyphs[3])
		if tuned <= plain*3 {
			t.Fatalf("near density 120 painted ✦%d vs stock ✦%d; start must sample the config", tuned, plain)
		}
	})
	t.Run("happy: a config change lands on the next start, not mid-flight", func(t *testing.T) {
		t.Cleanup(ResetSky)
		f := NewTunedStarfield()
		f.Start(stageW, stageH)
		before := spriteSnapshot(f.Render())
		if err := UseSky(SkyConfig{
			Delay:   []int{1, 1, 1, 1},
			Density: []int{200, 200, 200, 200},
		}); err != nil {
			t.Fatalf("UseSky: %v", err)
		}
		if !snapshotsEqual(before, spriteSnapshot(f.Render())) {
			t.Fatal("a started sky caches its catalog — mid-flight config changes must wait")
		}
		f.Stop()
		f.Start(stageW, stageH)
		if snapshotsEqual(before, spriteSnapshot(f.Render())) {
			t.Fatal("the next start must pick the new config up")
		}
	})
	t.Run("unhappy: resetting the sky takes the tuning back out on restart", func(t *testing.T) {
		t.Cleanup(ResetSky)
		f := NewTunedStarfield()
		f.Start(stageW, stageH)
		stock := spriteSnapshot(f.Render())
		if err := UseSky(SkyConfig{
			Delay:   []int{1, 1, 1, 1},
			Density: []int{200, 200, 200, 200},
		}); err != nil {
			t.Fatalf("UseSky: %v", err)
		}
		ResetSky()
		f.Stop()
		f.Start(stageW, stageH)
		if !snapshotsEqual(stock, spriteSnapshot(f.Render())) {
			t.Fatal("after ResetSky a restarted tuned sky must fly stock again")
		}
	})
}
