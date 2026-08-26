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

func TestStarfieldDock(t *testing.T) {
	const (
		dockMin = 25
		seconds = 0.5
	)
	columnLit := func(sp sprite.Sprite, col int) bool {
		for r := 0; r < sp.Height; r++ {
			if !sp.At(r, col).Transparent() {
				return true
			}
		}
		return false
	}
	t.Run("happy: at t=0 the dock has not yet eaten any column", func(t *testing.T) {
		plain := NewStarfield(Still)
		plain.Start(stageW, stageH)
		docked := NewStarfield(Still).Dock(dockMin, seconds)
		docked.Start(stageW, stageH)
		if !snapshotsEqual(spriteSnapshot(plain.Render()), spriteSnapshot(docked.Render())) {
			t.Fatal("the opening frame of a dock wipe must still be the full sky")
		}
	})
	t.Run("happy: after the wipe the right third is dark and the left still shines", func(t *testing.T) {
		plain := NewStarfield(Still)
		plain.Start(stageW, stageH)
		full := plain.Render()
		f := NewStarfield(Still).Dock(dockMin, seconds)
		f.Start(stageW, stageH)
		f.Update(seconds)
		sp := f.Render()
		want := DockCols(stageW, dockMin)
		cut := stageW - want
		for c := cut; c < stageW; c++ {
			if columnLit(sp, c) {
				t.Fatalf("column %d should be dark after the wipe (dock %d)", c, want)
			}
		}
		left := 0
		for r := 0; r < stageH; r++ {
			for c := 0; c < cut; c++ {
				got := sp.At(r, c)
				wantc := full.At(r, c)
				if got != wantc {
					t.Fatalf("left sky at (%d,%d) changed: %+v -> %+v", r, c, wantc, got)
				}
				if !got.Transparent() {
					left++
				}
			}
		}
		if left == 0 {
			t.Fatal("the left two-thirds must keep their stars")
		}
	})
	t.Run("happy: columns go dark from the right, one at a time", func(t *testing.T) {
		plain := NewStarfield(Still)
		plain.Start(stageW, stageH)
		full := plain.Render()
		f := NewStarfield(Still).Dock(dockMin, seconds)
		f.Start(stageW, stageH)
		want := DockCols(stageW, dockMin)
		prev := 0
		for i := 1; i <= 10; i++ {
			f.Update(seconds / 10)
			sp := f.Render()
			expect := want * i / 10
			for c := 0; c < stageW; c++ {
				fromRight := stageW - 1 - c
				shouldWipe := fromRight < expect
				if shouldWipe {
					if columnLit(sp, c) {
						t.Fatalf("step %d: column %d should already be wiped", i, c)
					}
					continue
				}
				for r := 0; r < stageH; r++ {
					if sp.At(r, c) != full.At(r, c) {
						t.Fatalf("step %d: column %d changed before its wipe", i, c)
					}
				}
			}
			if expect < prev {
				t.Fatalf("wiped columns shrank from %d to %d", prev, expect)
			}
			prev = expect
		}
		if prev != want {
			t.Fatalf("after the full wipe %d columns are dark, want %d", prev, want)
		}
	})
	t.Run("unhappy: a sky that was never asked to dock stays full", func(t *testing.T) {
		a := NewStarfield(Still)
		a.Start(stageW, stageH)
		before := spriteSnapshot(a.Render())
		a.Update(seconds)
		if !snapshotsEqual(before, spriteSnapshot(a.Render())) {
			t.Fatal("without Dock a still sky must not lose columns")
		}
	})
	t.Run("unhappy: Dock on a nil sky is still nil", func(t *testing.T) {
		var ghost *Starfield
		if ghost.Dock(dockMin, seconds) != nil {
			t.Fatal("Dock must return the nil receiver")
		}
	})
}

func TestSlideOffset(t *testing.T) {
	const body = 26
	const sec = 4.0
	dist := stageW - (stageW-body)/2
	t.Run("happy: t=0 is still, t>=duration is the full fly-in distance", func(t *testing.T) {
		if got := SlideOffset(stageW, body, 0, sec); got != 0 {
			t.Fatalf("t=0 offset %d, want 0", got)
		}
		if got := SlideOffset(stageW, body, sec, sec); got != dist {
			t.Fatalf("t=duration offset %d, want %d", got, dist)
		}
		if got := SlideOffset(stageW, body, sec+10, sec); got != dist {
			t.Fatalf("past the slide offset %d, want the held distance %d", got, dist)
		}
	})
	t.Run("happy: the slide is ease-out — most of the distance in the first half", func(t *testing.T) {
		half := SlideOffset(stageW, body, sec/2, sec)
		if half <= dist/2 {
			t.Fatalf("ease-out at t=half moved %d of %d; want more than halfway", half, dist)
		}
		prev := 0
		for i := 0; i <= 40; i++ {
			got := SlideOffset(stageW, body, float64(i)/10, sec)
			if got < prev {
				t.Fatalf("t=%.1f offset %d shrank from %d — the ship only flies left", float64(i)/10, got, prev)
			}
			prev = got
		}
	})
	t.Run("unhappy: a zero stage, zero duration, or negative time does not slide", func(t *testing.T) {
		if SlideOffset(0, body, 1, sec) != 0 {
			t.Fatal("zero width must not slide")
		}
		if SlideOffset(stageW, body, 1, 0) != 0 {
			t.Fatal("zero duration must not slide")
		}
		if SlideOffset(stageW, body, -1, sec) != 0 {
			t.Fatal("negative time is the opening mark")
		}
	})
}

func TestStarfieldSlideIn(t *testing.T) {
	const body = 26
	const sec = 4.0
	shifted := func(src map[[2]int]rune, offset int) map[[2]int]rune {
		out := map[[2]int]rune{}
		for k, ch := range src {
			out[[2]int{k[0], wrap(k[1]-offset, stageW)}] = ch
		}
		return out
	}
	t.Run("happy: at t=0 a sliding sky is still the full unshifted sky", func(t *testing.T) {
		plain := NewStarfield(Still)
		plain.Start(stageW, stageH)
		f := NewStarfield(Still).SlideIn(sec, body)
		f.Start(stageW, stageH)
		if !snapshotsEqual(spriteSnapshot(plain.Render()), spriteSnapshot(f.Render())) {
			t.Fatal("the opening frame of a slide must still be the unshifted sky")
		}
	})
	t.Run("happy: every star translates the same amount — the whole sky slides with the ship", func(t *testing.T) {
		f := NewStarfield(Still).SlideIn(sec, body)
		f.Start(stageW, stageH)
		before := spriteSnapshot(f.Render())
		f.Update(1.0)
		offset := SlideOffset(stageW, body, 1.0, sec)
		if offset < 1 {
			t.Fatal("test premise: one second of fly-in must move the sky")
		}
		if !snapshotsEqual(shifted(before, offset), spriteSnapshot(f.Render())) {
			t.Fatal("every star must shift left by the fly-in offset — no parallax during the slide")
		}
	})
	t.Run("happy: after the fly-in the offset holds and the sky's own fly takes over", func(t *testing.T) {
		still := NewStarfield(Still).SlideIn(sec, body)
		still.Start(stageW, stageH)
		still.Update(sec)
		held := spriteSnapshot(still.Render())
		still.Update(2.0)
		if !snapshotsEqual(held, spriteSnapshot(still.Render())) {
			t.Fatal("a still sky must hold the landed offset after the slide")
		}
		drift := NewStarfield(Drift).SlideIn(sec, body)
		drift.Start(stageW, stageH)
		drift.Update(sec)
		parked := spriteSnapshot(drift.Render())
		drift.Update(2.0)
		if snapshotsEqual(parked, spriteSnapshot(drift.Render())) {
			t.Fatal("after the slide a drifting sky must keep flying")
		}
	})
	t.Run("happy: a drifting slide outruns the same sky without a slide", func(t *testing.T) {
		plain := NewStarfield(Drift)
		plain.Start(stageW, stageH)
		plain.Update(1.0)
		rushed := NewStarfield(Drift).SlideIn(sec, body)
		rushed.Start(stageW, stageH)
		rushed.Update(1.0)
		if snapshotsEqual(spriteSnapshot(plain.Render()), spriteSnapshot(rushed.Render())) {
			t.Fatal("the arrival slide must add translation on top of the sky's own fly")
		}
	})
	t.Run("unhappy: a sky that was never asked to slide stays on its own fly", func(t *testing.T) {
		a := NewStarfield(Still)
		a.Start(stageW, stageH)
		before := spriteSnapshot(a.Render())
		a.Update(sec)
		if !snapshotsEqual(before, spriteSnapshot(a.Render())) {
			t.Fatal("without SlideIn a still sky must not translate")
		}
	})
	t.Run("unhappy: SlideIn on a nil sky is still nil", func(t *testing.T) {
		var ghost *Starfield
		if ghost.SlideIn(sec, body) != nil {
			t.Fatal("SlideIn must return the nil receiver")
		}
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
