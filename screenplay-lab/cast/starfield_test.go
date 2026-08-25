package cast

// Tests written FIRST: the starfield actor wraps stars.Field so a scene
// can hang the sky like any other performer. It sizes itself to the
// stage it paints, drifts with its own clock, and holds under a still
// strategy, a held clock, or no stage at all.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/stars-lab/stars"

	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

func skySnapshot(f *Starfield, w, h int) map[[2]int]rune {
	st := screenplay.NewStage(w, h)
	f.Paint(st)
	snap := map[[2]int]rune{}
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			if cell := st.Board.At(r, c); !cell.Transparent() {
				snap[[2]int{r, c}] = cell.Ch
			}
		}
	}
	return snap
}

func equalSky(a, b map[[2]int]rune) bool {
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
	t.Run("happy: the sky fills the stage with all four stars", func(t *testing.T) {
		f := NewStarfield(stars.Drift)
		st := screenplay.NewStage(stageW, 26)
		f.Paint(st)
		found := map[rune]bool{}
		for r := 0; r < 26; r++ {
			for c := 0; c < stageW; c++ {
				cell := st.Board.At(r, c)
				if cell.Transparent() {
					continue
				}
				found[cell.Ch] = true
				if cell.FG < 0 {
					t.Fatalf("star at (%d,%d) has no tint", r, c)
				}
			}
		}
		for _, g := range stars.Glyphs {
			if !found[g] {
				t.Fatalf("sky missing star %q", string(g))
			}
		}
	})
	t.Run("happy: a drifting sky moves as its clock runs", func(t *testing.T) {
		f := NewStarfield(stars.Drift)
		before := skySnapshot(f, stageW, 26)
		f.Advance(2.0)
		after := skySnapshot(f, stageW, 26)
		if equalSky(before, after) {
			t.Fatal("two seconds of drift left every star in place")
		}
	})
	t.Run("unhappy: a still sky holds no matter how long it runs", func(t *testing.T) {
		f := NewStarfield(stars.Still)
		before := skySnapshot(f, stageW, 26)
		f.Advance(3.0)
		if !equalSky(before, skySnapshot(f, stageW, 26)) {
			t.Fatal("a still sky must never move")
		}
	})
	t.Run("unhappy: dt<=0 holds the drift", func(t *testing.T) {
		f := NewStarfield(stars.Drift)
		before := skySnapshot(f, stageW, 26)
		f.Advance(0)
		f.Advance(-1)
		if !equalSky(before, skySnapshot(f, stageW, 26)) {
			t.Fatal("a held clock must hold the sky")
		}
	})
	t.Run("unhappy: a zero stage takes no stars and no panic", func(t *testing.T) {
		f := NewStarfield(stars.Drift)
		st := screenplay.NewStage(0, 0)
		f.Paint(st)
		if n := litCells(st); n != 0 {
			t.Fatalf("zero stage lit %d cells", n)
		}
	})
}
