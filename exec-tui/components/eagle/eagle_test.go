package eagle

// Tests written FIRST: the eagle component is the very large bald
// eagle — white head low on the left, gold hooked beak and reaching
// talons, dark brown wings spread wide, white tail fanned behind —
// that flies across the stage right to left, off one wing and off the
// other, in CrossSeconds. The model is half-block pixel art: a
// BodyCols×(2·BodyRows) pixel grid rendered as plain field cells,
// upper-half and lower-half blocks — nothing else — so every cell
// carries its color in the foreground, the background, or both, and
// silhouette edges keep a transparent half for whatever flies
// beneath. Delay holds the bird off stage first, so a scene can
// finish its fade before the flyover. Path retunes where the flight
// begins and ends as fractions of the full off-right-to-off-left
// span: the stock flight is 0 to 1, and a scene can start the eagle
// already on stage or cut the flight short of the far wing. Before
// the delay, after the crossing, before Start and after Stop the sky
// is clear; dt <= 0 never moves the flight; a resize keeps the clock
// so the crossing never replays mid-scene.

import (
	"math"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	stageW = 72
	stageH = 27
)

// tick advances the flight the way a 30fps runner would.
func tick(e *Eagle, seconds float64) {
	const dt = 1.0 / 30
	for t := 0.0; t < seconds-dt/2; t += dt {
		e.Update(dt)
	}
}

// inkCells collects every painted cell of a stage-sized render — on
// the eagle's own stage every painted cell is eagle.
func inkCells(sp sprite.Sprite) [][2]int {
	var out [][2]int
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if !sp.At(r, c).Transparent() {
				out = append(out, [2]int{r, c})
			}
		}
	}
	return out
}

// span reports the painted bounding box as leftmost, rightmost,
// topmost, bottommost.
func span(cells [][2]int) (l, r, t, b int) {
	l, t = 1<<30, 1<<30
	r, b = -1, -1
	for _, rc := range cells {
		if rc[1] < l {
			l = rc[1]
		}
		if rc[1] > r {
			r = rc[1]
		}
		if rc[0] < t {
			t = rc[0]
		}
		if rc[0] > b {
			b = rc[0]
		}
	}
	return l, r, t, b
}

// wears reports whether a painted cell carries an ink in either half
// — the foreground of a half block, or the background behind it.
func wears(c sprite.Cell, ink int) bool {
	if c.Transparent() {
		return false
	}
	return c.FG == ink || c.BG == ink
}

// artCols reports which art columns hold a given ink.
func artCols(ink int) map[int]bool {
	art := Art()
	cols := map[int]bool{}
	for r := 0; r < art.Height; r++ {
		for c := 0; c < art.Width; c++ {
			if wears(art.At(r, c), ink) {
				cols[c] = true
			}
		}
	}
	return cols
}

func TestEagleArt(t *testing.T) {
	t.Run("happy: the model is very large and rectangular", func(t *testing.T) {
		art := Art()
		if err := art.Validate(); err != nil {
			t.Fatalf("the eagle art must validate: %v", err)
		}
		if art.Width != BodyCols || art.Height != BodyRows {
			t.Fatalf("the art is %dx%d, want the exported %dx%d", art.Width, art.Height, BodyCols, BodyRows)
		}
		if BodyCols < 50 {
			t.Fatalf("BodyCols = %d — a very large eagle spans at least 50 columns", BodyCols)
		}
		if BodyRows < 12 {
			t.Fatalf("BodyRows = %d — a very large eagle stands at least 12 rows", BodyRows)
		}
	})
	t.Run("happy: the model is pure half-block pixel art", func(t *testing.T) {
		art := Art()
		for r := 0; r < art.Height; r++ {
			for c := 0; c < art.Width; c++ {
				cell := art.At(r, c)
				if cell.Transparent() {
					continue
				}
				if cell.Ch != ' ' && cell.Ch != '▀' && cell.Ch != '▄' {
					t.Fatalf("cell (%d,%d) draws %q — the model is field cells and half blocks only", r, c, cell.Ch)
				}
			}
		}
	})
	t.Run("happy: silhouette edges keep a transparent half for what flies beneath", func(t *testing.T) {
		art := Art()
		soft := 0
		for r := 0; r < art.Height; r++ {
			for c := 0; c < art.Width; c++ {
				cell := art.At(r, c)
				if cell.Transparent() {
					continue
				}
				if (cell.Ch == '▀' || cell.Ch == '▄') && cell.BG < 0 {
					soft++
				}
			}
		}
		if soft < 10 {
			t.Fatalf("only %d edge cells keep a transparent half — the silhouette must blend, not box", soft)
		}
	})
	t.Run("happy: a bald eagle — white head and gold beak lead on the left", func(t *testing.T) {
		art := Art()
		headLeft, beakLeft := false, false
		for r := 0; r < art.Height; r++ {
			for c := 0; c < BodyCols/3; c++ {
				cell := art.At(r, c)
				if wears(cell, HeadInk) {
					headLeft = true
				}
				if wears(cell, BeakInk) {
					beakLeft = true
				}
			}
		}
		if !headLeft {
			t.Fatal("the white head must lead — HeadInk in the left third")
		}
		if !beakLeft {
			t.Fatal("the gold beak must lead — BeakInk in the left third")
		}
	})
	t.Run("happy: gold talons reach below, a white tail fans behind", func(t *testing.T) {
		art := Art()
		talons, tail := false, false
		for r := 0; r < art.Height; r++ {
			for c := 0; c < art.Width; c++ {
				cell := art.At(r, c)
				if wears(cell, BeakInk) && r >= BodyRows/2 {
					talons = true
				}
				if wears(cell, HeadInk) && c >= 2*BodyCols/3 {
					tail = true
				}
			}
		}
		if !talons {
			t.Fatal("the gold talons must reach into the lower half")
		}
		if !tail {
			t.Fatal("the white tail must fan out behind, on the right")
		}
	})
	t.Run("happy: dark wings spread over most of the wingspan and up to the top", func(t *testing.T) {
		art := Art()
		wings := map[int]bool{}
		top := false
		for r := 0; r < art.Height; r++ {
			for c := 0; c < art.Width; c++ {
				cell := art.At(r, c)
				if wears(cell, BodyInk) || wears(cell, ShadowInk) {
					wings[c] = true
					if r < 2 {
						top = true
					}
				}
			}
		}
		if got, want := len(wings), (BodyCols*7)/10; got < want {
			t.Fatalf("the dark wings cover %d columns, want at least %d — spread them", got, want)
		}
		if !top {
			t.Fatal("the raised wings must reach the top rows of the model")
		}
	})
	t.Run("happy: the signature inks are everywhere the detectors need them", func(t *testing.T) {
		art := Art()
		signed := 0
		for r := 0; r < art.Height; r++ {
			for c := 0; c < art.Width; c++ {
				for _, ink := range SignatureInks() {
					if wears(art.At(r, c), ink) {
						signed++
						break
					}
				}
			}
		}
		if signed < 150 {
			t.Fatalf("only %d cells wear a signature ink — the scene detectors key on them", signed)
		}
	})
	t.Run("happy: the signature inks are the eagle's own — never the flag's, never white", func(t *testing.T) {
		inks := SignatureInks()
		has := func(ink int) bool {
			for _, i := range inks {
				if i == ink {
					return true
				}
			}
			return false
		}
		if !has(BodyInk) || !has(BeakInk) {
			t.Fatal("the brown body and the gold beak are the eagle's signature")
		}
		for _, banned := range []int{ShadowInk, HeadInk, 16, 236, 250} {
			if has(banned) {
				t.Fatalf("ink %d can appear on the fading flag or the white head — it must not be a signature", banned)
			}
		}
	})
	t.Run("happy: both wingtips are inked, so the crossing enters and exits visibly", func(t *testing.T) {
		if len(artCols(BodyInk))+len(artCols(ShadowInk)) == 0 {
			t.Fatal("test premise: the wings hold ink")
		}
		art := Art()
		first, last := false, false
		for r := 0; r < art.Height; r++ {
			if !art.At(r, 0).Transparent() {
				first = true
			}
			if !art.At(r, art.Width-1).Transparent() {
				last = true
			}
		}
		if !first || !last {
			t.Fatalf("the outermost columns must carry ink, first=%v last=%v", first, last)
		}
	})
	t.Run("unhappy: the old glyph shading is gone — no ▓▒░ blocks anywhere", func(t *testing.T) {
		art := Art()
		for r := 0; r < art.Height; r++ {
			for c := 0; c < art.Width; c++ {
				switch art.At(r, c).Ch {
				case '▓', '▒', '░', '█', '◣', '◢', '◤', '●':
					t.Fatalf("cell (%d,%d) still draws %q — the pixel model replaced glyph shading", r, c, art.At(r, c).Ch)
				}
			}
		}
	})
}

func TestEagleFlight(t *testing.T) {
	t.Run("happy: it flies across, right to left, and is gone by CrossSeconds", func(t *testing.T) {
		e := New().Cross(10)
		e.Start(stageW, stageH)
		if got := len(inkCells(e.Render())); got != 0 {
			t.Fatalf("at t=0 the eagle is still off the right wing, found %d cells", got)
		}
		tick(e, 2.5)
		first := inkCells(e.Render())
		if len(first) == 0 {
			t.Fatal("a quarter in, the eagle must be on stage")
		}
		l1, _, _, _ := span(first)
		tick(e, 2.5)
		second := inkCells(e.Render())
		if len(second) == 0 {
			t.Fatal("halfway in, the eagle must be on stage")
		}
		l2, _, _, _ := span(second)
		if l2 >= l1 {
			t.Fatalf("the eagle must fly leftward: leftmost went %d -> %d", l1, l2)
		}
		tick(e, 6)
		if got := len(inkCells(e.Render())); got != 0 {
			t.Fatalf("past the crossing the sky must be clear, found %d cells", got)
		}
	})
	t.Run("happy: mid-flight the whole huge model is on stage", func(t *testing.T) {
		e := New().Cross(10)
		e.Start(stageW, stageH)
		tick(e, 5)
		cells := inkCells(e.Render())
		if len(cells) < 200 {
			t.Fatalf("mid-flight only %d cells are painted — the model must be huge", len(cells))
		}
		l, r, top, b := span(cells)
		if got, want := r-l+1, BodyCols-6; got < want {
			t.Fatalf("mid-flight the wingspan covers %d columns, want at least %d", got, want)
		}
		if got, want := b-top+1, BodyRows-4; got < want {
			t.Fatalf("mid-flight the eagle stands %d rows, want at least %d", got, want)
		}
		if top < 1 || b > stageH-2 {
			t.Fatalf("the flight must not scrape the stage edges: rows %d..%d on %d", top, b, stageH)
		}
	})
	t.Run("happy: a bare New flies on the default crossing", func(t *testing.T) {
		if DefaultCrossSeconds <= 0 {
			t.Fatal("DefaultCrossSeconds must be a duration")
		}
		e := New()
		e.Start(stageW, stageH)
		tick(e, DefaultCrossSeconds/2)
		if len(inkCells(e.Render())) == 0 {
			t.Fatal("a bare New must fly the default crossing")
		}
	})
	t.Run("happy: a resize keeps the clock — the crossing never replays", func(t *testing.T) {
		e := New().Cross(10)
		e.Start(stageW, stageH)
		tick(e, 5)
		e.Stop()
		e.Start(100, 30)
		if len(inkCells(e.Render())) == 0 {
			t.Fatal("a mid-flight resize must keep the eagle on stage")
		}
	})
	t.Run("unhappy: before the delay the sky is clear", func(t *testing.T) {
		e := New().Delay(5).Cross(10)
		e.Start(stageW, stageH)
		tick(e, 4.5)
		if got := len(inkCells(e.Render())); got != 0 {
			t.Fatalf("before the delay the sky must be clear, found %d cells", got)
		}
		tick(e, 3)
		if len(inkCells(e.Render())) == 0 {
			t.Fatal("past the delay the eagle must fly")
		}
	})
	t.Run("unhappy: dt <= 0 holds the flight still", func(t *testing.T) {
		e := New().Cross(10)
		e.Start(stageW, stageH)
		tick(e, 2.5)
		before := sprite.Render(e.Render())
		e.Update(0)
		e.Update(-3)
		if sprite.Render(e.Render()) != before {
			t.Fatal("dt <= 0 must never move the eagle")
		}
	})
	t.Run("unhappy: before Start and after Stop the stage is empty", func(t *testing.T) {
		e := New().Cross(10)
		if sp := e.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("before Start the eagle renders %dx%d, want nothing", sp.Width, sp.Height)
		}
		e.Start(stageW, stageH)
		tick(e, 5)
		e.Stop()
		if sp := e.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("after Stop the eagle renders %dx%d, want nothing", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a tiny stage clips without panic", func(t *testing.T) {
		e := New().Cross(10)
		e.Start(8, 4)
		tick(e, 5)
		sp := e.Render()
		if sp.Width != 8 || sp.Height != 4 {
			t.Fatalf("the tiny stage renders %dx%d, want 8x4", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a nil eagle never panics", func(t *testing.T) {
		var e *Eagle
		_ = e.Delay(1)
		_ = e.Cross(1)
		_ = e.Path(0.2, 0.8)
		e.Start(10, 5)
		e.Update(1)
		_ = e.Render()
		e.Stop()
	})
}

func TestEaglePath(t *testing.T) {
	t.Run("happy: a late start point opens the flight already deep on stage", func(t *testing.T) {
		e := New().Cross(10).Path(0.5, 1.0)
		e.Start(stageW, stageH)
		e.Update(1.0 / 30)
		cells := inkCells(e.Render())
		if len(cells) < 200 {
			t.Fatalf("a 0.5 start must open mid-stage, found only %d cells", len(cells))
		}
		l, _, _, _ := span(cells)
		if l > stageW/2 {
			t.Fatalf("a 0.5 start opens with the leading edge at %d — the bird starts halfway along the span", l)
		}
		tick(e, 10)
		if got := len(inkCells(e.Render())); got != 0 {
			t.Fatalf("past the crossing the sky must be clear, found %d cells", got)
		}
	})
	t.Run("happy: an early end point cuts the flight short of the far wing", func(t *testing.T) {
		e := New().Cross(10).Path(0, 0.5)
		e.Start(stageW, stageH)
		tick(e, 9.8)
		cells := inkCells(e.Render())
		if len(cells) < 200 {
			t.Fatalf("just before an 0.5 end the bird is still mid-stage, found only %d cells", len(cells))
		}
		l, _, _, _ := span(cells)
		if l < 1 {
			t.Fatalf("an 0.5 end must never reach the left wing, leading edge at %d", l)
		}
		tick(e, 0.5)
		if got := len(inkCells(e.Render())); got != 0 {
			t.Fatalf("past the crossing the flight is over, found %d cells", got)
		}
	})
	t.Run("happy: the flight between two points still moves leftward", func(t *testing.T) {
		e := New().Cross(10).Path(0.25, 0.75)
		e.Start(stageW, stageH)
		tick(e, 2.5)
		first := inkCells(e.Render())
		if len(first) == 0 {
			t.Fatal("a quarter in, the eagle must be on stage")
		}
		l1, _, _, _ := span(first)
		tick(e, 2.5)
		second := inkCells(e.Render())
		if len(second) == 0 {
			t.Fatal("halfway in, the eagle must be on stage")
		}
		l2, _, _, _ := span(second)
		if l2 >= l1 {
			t.Fatalf("the eagle must fly leftward: leftmost went %d -> %d", l1, l2)
		}
	})
	t.Run("unhappy: a backwards, out-of-range or unreal path keeps the stock flight", func(t *testing.T) {
		for _, bad := range [][2]float64{
			{0.7, 0.3},
			{0.5, 0.5},
			{-0.2, 0.5},
			{0, 1.2},
			{math.NaN(), 0.5},
			{0, math.Inf(1)},
		} {
			e := New().Cross(10).Path(bad[0], bad[1])
			e.Start(stageW, stageH)
			if got := len(inkCells(e.Render())); got != 0 {
				t.Fatalf("Path(%v, %v) must keep the stock off-stage start, found %d cells", bad[0], bad[1], got)
			}
			tick(e, 5)
			if len(inkCells(e.Render())) == 0 {
				t.Fatalf("Path(%v, %v) must keep the stock mid-flight, sky is clear", bad[0], bad[1])
			}
		}
	})
}
