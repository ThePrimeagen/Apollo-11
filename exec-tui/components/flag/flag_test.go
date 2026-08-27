package flag

// Tests written FIRST: the flag component is the full-screened American
// flag — thirteen stripes, the blue canton, all fifty stars — painted
// across every cell of the stage, fading in from pure black over
// FadeSeconds. The stripes are mathematically even: the field is laid
// out on half rows — a boundary that falls inside a cell paints an
// upper-half block over the lower stripe's background — so no stripe
// is ever more than half a row taller than another, at any stage
// height. The canton bottom lands exactly on the seventh stripe
// boundary and only fully blue rows carry stars, spread evenly to the
// cell. The fade owns only the colors: the layout is fixed at Start,
// every cell walks its own ramp from black to its finished ink, and
// dt <= 0 never moves the clock. Before Start and after Stop the
// stage is empty; a resize keeps the fade clock so the flag never
// restarts from black mid-scene.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	stageW = 72
	stageH = 26
)

// tick advances the fade the way a 30fps runner would.
func tick(f *Flag, seconds float64) {
	const dt = 1.0 / 30
	for t := 0.0; t < seconds-dt/2; t += dt {
		f.Update(dt)
	}
}

// starCount tallies the star glyphs on a rendered stage.
func starCount(sp sprite.Sprite) int {
	n := 0
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if sp.At(r, c).Ch == StarGlyph {
				n++
			}
		}
	}
	return n
}

// halves splits one rendered cell into its upper and lower half-row
// colors: a plain field cell is one color top to bottom, a boundary
// cell paints its upper half with the half-block ink over the lower
// half's background.
func halves(c sprite.Cell) (up, low int) {
	if c.Ch == '▀' {
		return c.FG, c.BG
	}
	return c.BG, c.BG
}

// column is the 2h half-row colors down one column of the stage.
func column(sp sprite.Sprite, c int) []int {
	out := make([]int, 0, 2*sp.Height)
	for r := 0; r < sp.Height; r++ {
		up, low := halves(sp.At(r, c))
		out = append(out, up, low)
	}
	return out
}

// run is one vertical band of a single ink.
type run struct {
	ink int
	n   int
}

// runLengths collapses a color column into its bands.
func runLengths(colors []int) []run {
	var out []run
	for _, ink := range colors {
		if len(out) > 0 && out[len(out)-1].ink == ink {
			out[len(out)-1].n++
			continue
		}
		out = append(out, run{ink: ink, n: 1})
	}
	return out
}

// finished renders a flag at full color on a stageW×h stage.
func finished(h int) sprite.Sprite {
	f := New(0)
	f.Start(stageW, h)
	return f.Render()
}

// starGrid maps each star row to its sorted columns.
func starGrid(sp sprite.Sprite) map[int][]int {
	rows := map[int][]int{}
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if sp.At(r, c).Ch == StarGlyph {
				rows[r] = append(rows[r], c)
			}
		}
	}
	return rows
}

func TestFlagEvenStripes(t *testing.T) {
	t.Run("happy: thirteen stripes even to the half row, at any height", func(t *testing.T) {
		for _, h := range []int{26, 27, 29, 33, 45} {
			runs := runLengths(column(finished(h), stageW-1))
			if len(runs) != Stripes {
				t.Fatalf("h=%d: the fly holds %d bands, want %d stripes", h, len(runs), Stripes)
			}
			lo := (2 * h) / Stripes
			hi := (2*h + Stripes - 1) / Stripes
			for i, band := range runs {
				if band.n < lo || band.n > hi {
					t.Fatalf("h=%d: stripe %d spans %d half-rows, want %d or %d — the stripes must be even", h, i, band.n, lo, hi)
				}
				want := RedInk
				if i%2 == 1 {
					want = WhiteInk
				}
				if band.ink != want {
					t.Fatalf("h=%d: stripe %d wears %d, want %d — red first, then alternate", h, i, band.ink, want)
				}
			}
		}
	})
	t.Run("happy: a boundary inside a cell paints a half block, both halves true", func(t *testing.T) {
		sp := finished(27)
		found := false
		for r := 0; r < 27; r++ {
			cell := sp.At(r, stageW-1)
			if cell.Ch != '▀' {
				continue
			}
			found = true
			if cell.FG == cell.BG {
				t.Fatalf("boundary cell at row %d wears %d over %d — the halves must differ", r, cell.FG, cell.BG)
			}
			if cell.FG != RedInk && cell.FG != WhiteInk {
				t.Fatalf("boundary cell at row %d has a %d upper half — stripes are red or white", r, cell.FG)
			}
		}
		if !found {
			t.Fatal("27 rows cannot hold 13 even stripes on whole cells — a half-block boundary must appear")
		}
	})
	t.Run("happy: the canton bottom lands exactly on the seventh stripe boundary", func(t *testing.T) {
		for _, h := range []int{25, 26, 27} {
			colors := column(finished(h), 0)
			blue := 0
			for _, ink := range colors {
				if ink != BlueInk {
					break
				}
				blue++
			}
			want := (2*CantonStripes*h + Stripes - 1) / Stripes
			if blue != want {
				t.Fatalf("h=%d: the canton runs %d half-rows deep, want %d — seven thirteenths exactly", h, blue, want)
			}
		}
	})
	t.Run("unhappy: no stripe is ever a whole row taller than its neighbor", func(t *testing.T) {
		for _, h := range []int{27, 29, 31} {
			runs := runLengths(column(finished(h), stageW-1))
			min, max := 1<<30, 0
			for _, band := range runs {
				if band.n < min {
					min = band.n
				}
				if band.n > max {
					max = band.n
				}
			}
			if max-min > 1 {
				t.Fatalf("h=%d: stripes span %d..%d half-rows — the old full-row layout is showing", h, min, max)
			}
		}
	})
}

func TestFlagEvenStars(t *testing.T) {
	t.Run("happy: every star row spreads its stars evenly across the canton", func(t *testing.T) {
		sp := finished(stageH)
		cw := CantonCols(stageW)
		lo, hi := cw/6, (cw+5)/6
		grid := starGrid(sp)
		if len(grid) != 9 {
			t.Fatalf("the canton holds %d star rows, want 9", len(grid))
		}
		for r, cols := range grid {
			if len(cols) != 6 && len(cols) != 5 {
				t.Fatalf("star row %d holds %d stars, want 6 or 5", r, len(cols))
			}
			for i := 1; i < len(cols); i++ {
				gap := cols[i] - cols[i-1]
				if gap < lo || gap > hi {
					t.Fatalf("star row %d: gap %d between columns %d and %d, want %d or %d", r, gap, cols[i-1], cols[i], lo, hi)
				}
			}
		}
	})
	t.Run("happy: the nine star rows are spread evenly down the canton", func(t *testing.T) {
		sp := finished(stageH)
		cb := CantonRows(stageH)
		grid := starGrid(sp)
		rows := make([]int, 0, len(grid))
		for r := range grid {
			rows = append(rows, r)
		}
		for i := 0; i < len(rows); i++ {
			for j := i + 1; j < len(rows); j++ {
				if rows[j] < rows[i] {
					rows[i], rows[j] = rows[j], rows[i]
				}
			}
		}
		lo, hi := cb/10, (cb+9)/10
		for i := 1; i < len(rows); i++ {
			gap := rows[i] - rows[i-1]
			if gap < lo || gap > hi {
				t.Fatalf("star rows %d and %d sit %d apart, want %d or %d", rows[i-1], rows[i], gap, lo, hi)
			}
		}
	})
	t.Run("unhappy: no star ever sits on a split cell — stars ride fully blue rows only", func(t *testing.T) {
		for _, h := range []int{25, 26, 27, 29} {
			sp := finished(h)
			for r := 0; r < h; r++ {
				for c := 0; c < stageW; c++ {
					cell := sp.At(r, c)
					if cell.Ch != StarGlyph {
						continue
					}
					if cell.BG != BlueInk {
						t.Fatalf("h=%d: the star at (%d,%d) rides bg %d, want fully blue %d", h, r, c, cell.BG, BlueInk)
					}
				}
			}
		}
	})
}

func TestFlagGeometry(t *testing.T) {
	t.Run("happy: the canton spans two fifths of the width and seven stripes of the height", func(t *testing.T) {
		if got := CantonCols(72); got != 29 {
			t.Fatalf("CantonCols(72) = %d, want 29", got)
		}
		if got := CantonRows(26); got != 14 {
			t.Fatalf("CantonRows(26) = %d, want 14 — seven of thirteen stripes", got)
		}
		if got := CantonRows(27); got != 15 {
			t.Fatalf("CantonRows(27) = %d, want 15", got)
		}
		if got := CantonRows(25); got != 13 {
			t.Fatalf("CantonRows(25) = %d, want 13 — the canton ends mid-cell, only fully blue rows count", got)
		}
	})
	t.Run("unhappy: an empty stage has no canton", func(t *testing.T) {
		if got := CantonCols(0); got != 0 {
			t.Fatalf("CantonCols(0) = %d, want 0", got)
		}
		if got := CantonRows(0); got != 0 {
			t.Fatalf("CantonRows(0) = %d, want 0", got)
		}
	})
}

func TestFlagFinished(t *testing.T) {
	t.Run("happy: the finished flag fills every cell of the stage", func(t *testing.T) {
		f := New(2)
		f.Start(stageW, stageH)
		tick(f, 2.5)
		sp := f.Render()
		if sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("the flag renders %dx%d, want the %dx%d stage", sp.Width, sp.Height, stageW, stageH)
		}
		for r := 0; r < stageH; r++ {
			for c := 0; c < stageW; c++ {
				if sp.At(r, c).Transparent() {
					t.Fatalf("cell (%d,%d) is transparent — the flag must be full-screen", r, c)
				}
			}
		}
	})
	t.Run("happy: thirteen stripes — red on top, red on the bottom, white between", func(t *testing.T) {
		f := New(2)
		f.Start(stageW, stageH)
		tick(f, 2.5)
		sp := f.Render()
		if got := sp.At(0, stageW-1).BG; got != RedInk {
			t.Fatalf("the top stripe wears %d, want red %d", got, RedInk)
		}
		if got := sp.At(stageH-1, stageW-1).BG; got != RedInk {
			t.Fatalf("the bottom stripe wears %d, want red %d", got, RedInk)
		}
		// stageH 26 is two rows a stripe: rows 2-3 are stripe two, white.
		if got := sp.At(2, stageW-1).BG; got != WhiteInk {
			t.Fatalf("the second stripe wears %d, want white %d", got, WhiteInk)
		}
	})
	t.Run("happy: the canton is blue and carries all fifty stars", func(t *testing.T) {
		f := New(2)
		f.Start(stageW, stageH)
		tick(f, 2.5)
		sp := f.Render()
		if got := sp.At(0, 0).BG; got != BlueInk {
			t.Fatalf("the canton wears %d, want blue %d", got, BlueInk)
		}
		if got := starCount(sp); got != 50 {
			t.Fatalf("the canton carries %d stars, want 50", got)
		}
		cw, ch := CantonCols(stageW), CantonRows(stageH)
		for r := 0; r < stageH; r++ {
			for c := 0; c < stageW; c++ {
				cell := sp.At(r, c)
				if cell.Ch != StarGlyph {
					continue
				}
				if r >= ch || c >= cw {
					t.Fatalf("a star sits outside the canton at (%d,%d)", r, c)
				}
				if cell.FG != StarInk {
					t.Fatalf("the star at (%d,%d) wears %d, want %d", r, c, cell.FG, StarInk)
				}
			}
		}
	})
	t.Run("happy: a zero fade is at full color from the first frame", func(t *testing.T) {
		f := New(0)
		f.Start(stageW, stageH)
		sp := f.Render()
		if got := sp.At(0, stageW-1).BG; got != RedInk {
			t.Fatalf("a zero fade opens on %d, want red %d", got, RedInk)
		}
	})
}

func TestFlagFade(t *testing.T) {
	t.Run("happy: the curtain rises on pure black", func(t *testing.T) {
		f := New(6)
		f.Start(stageW, stageH)
		sp := f.Render()
		for r := 0; r < stageH; r++ {
			for c := 0; c < stageW; c++ {
				cell := sp.At(r, c)
				if cell.BG != Black {
					t.Fatalf("cell (%d,%d) opens on bg %d, want black %d", r, c, cell.BG, Black)
				}
				if cell.Ch != ' ' && cell.FG != Black {
					t.Fatalf("glyph at (%d,%d) opens on fg %d, want black %d", r, c, cell.FG, Black)
				}
			}
		}
	})
	t.Run("happy: mid-fade the stripes are dim — no longer black, not yet red", func(t *testing.T) {
		f := New(6)
		f.Start(stageW, stageH)
		tick(f, 3)
		sp := f.Render()
		got := sp.At(0, stageW-1).BG
		if got == Black || got == RedInk {
			t.Fatalf("mid-fade the top stripe wears %d — it must sit between black %d and red %d", got, Black, RedInk)
		}
		blue := sp.At(0, 0).BG
		if blue == Black || blue == BlueInk {
			t.Fatalf("mid-fade the canton wears %d — it must sit between black %d and blue %d", blue, Black, BlueInk)
		}
	})
	t.Run("happy: past FadeSeconds the flag holds its finished colors", func(t *testing.T) {
		f := New(1)
		f.Start(stageW, stageH)
		tick(f, 1.2)
		before := sprite.Render(f.Render())
		tick(f, 2)
		if sprite.Render(f.Render()) != before {
			t.Fatal("a finished fade must hold — the flag never drifts past full color")
		}
		if got := f.Render().At(0, stageW-1).BG; got != RedInk {
			t.Fatalf("past the fade the top stripe wears %d, want red %d", got, RedInk)
		}
	})
	t.Run("happy: a resize keeps the fade clock — the flag never falls back to black", func(t *testing.T) {
		f := New(6)
		f.Start(40, 13)
		tick(f, 3)
		f.Stop()
		f.Start(stageW, stageH)
		if got := f.Render().At(0, stageW-1).BG; got == Black {
			t.Fatal("a mid-fade resize must keep the clock, not restart from black")
		}
	})
	t.Run("unhappy: dt <= 0 never moves the fade", func(t *testing.T) {
		f := New(6)
		f.Start(stageW, stageH)
		f.Update(-1)
		f.Update(0)
		if got := f.Render().At(0, stageW-1).BG; got != Black {
			t.Fatalf("dt <= 0 moved the fade to %d — time never runs backwards", got)
		}
	})
}

func TestFlagLifecycle(t *testing.T) {
	t.Run("unhappy: before Start and after Stop the stage is empty", func(t *testing.T) {
		f := New(6)
		if sp := f.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("before Start the flag renders %dx%d, want nothing", sp.Width, sp.Height)
		}
		f.Start(stageW, stageH)
		f.Stop()
		if sp := f.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("after Stop the flag renders %dx%d, want nothing", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a tiny stage is still fully painted, without panic", func(t *testing.T) {
		f := New(1)
		f.Start(3, 2)
		tick(f, 1.5)
		sp := f.Render()
		if sp.Width != 3 || sp.Height != 2 {
			t.Fatalf("the tiny flag renders %dx%d, want 3x2", sp.Width, sp.Height)
		}
		for r := 0; r < 2; r++ {
			for c := 0; c < 3; c++ {
				if sp.At(r, c).Transparent() {
					t.Fatalf("tiny cell (%d,%d) is transparent — the flag must cover the stage", r, c)
				}
			}
		}
	})
	t.Run("unhappy: a zero-size stage renders empty, without panic", func(t *testing.T) {
		f := New(1)
		f.Start(0, 0)
		f.Update(1)
		if sp := f.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a 0x0 stage renders %dx%d, want nothing", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a nil flag never panics", func(t *testing.T) {
		var f *Flag
		f.Start(10, 5)
		f.Update(1)
		_ = f.Render()
		f.Stop()
	})
}
