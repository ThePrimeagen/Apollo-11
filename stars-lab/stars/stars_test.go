package stars

// Tests written FIRST. The component contract: a reusable one-cell starfield
// that any TUI can paint first and overlay on. Four glyphs, any width×height,
// strategy-driven right-to-left fly (far-fast / near-fast / uniform / …).
// Pure Paint / Render. Happy + unhappy throughout.

import (
	"sort"
	"strings"
	"testing"
)

func plain(s string) string {
	var b strings.Builder
	esc := false
	for _, r := range s {
		if r == '\x1b' {
			esc = true
			continue
		}
		if esc {
			if r == 'm' {
				esc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func field(s Strategy, tick int) Field {
	return Field{Width: 40, Height: 24, Tick: tick, Strategy: s}
}

func lines(f Field) []string {
	return strings.Split(plain(f.Render()), "\n")
}

func glyphCols(f Field, row int, g rune) []int {
	ls := lines(f)
	if row < 0 || row >= len(ls) {
		return nil
	}
	var cols []int
	for i, r := range []rune(ls[row]) {
		if r == g {
			cols = append(cols, i)
		}
	}
	return cols
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	as, bs := append([]int(nil), a...), append([]int(nil), b...)
	sort.Ints(as)
	sort.Ints(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

func shiftLeft(cols []int, w, steps int) []int {
	out := append([]int(nil), cols...)
	for n := 0; n < steps; n++ {
		for i := range out {
			out[i]--
			if out[i] < 0 {
				out[i] = w - 1
			}
		}
	}
	return out
}

func TestGlyphs(t *testing.T) {
	t.Run("happy: four distinct one-cell background stars", func(t *testing.T) {
		if Glyphs != [4]rune{'·', '˚', '*', '✦'} {
			t.Fatalf("glyph set %q, want · ˚ * ✦", Glyphs)
		}
		seen := map[rune]bool{}
		for _, g := range Glyphs {
			if seen[g] {
				t.Fatalf("duplicate glyph %q", string(g))
			}
			seen[g] = true
			if n := len([]rune(string(g))); n != 1 {
				t.Fatalf("%q must be one cell, got %d runes", string(g), n)
			}
		}
	})
	t.Run("happy: every glyph appears on a 40×24 field", func(t *testing.T) {
		v := plain(field(FarFast, 0).Render())
		for _, g := range Glyphs {
			if !strings.ContainsRune(v, g) {
				t.Fatalf("missing background star %q", string(g))
			}
		}
	})
	t.Run("unhappy: two fields do not share mutable tick state", func(t *testing.T) {
		a := field(FarFast, 0)
		b := field(FarFast, 0)
		a.Tick = 12
		if plain(b.Render()) != plain(field(FarFast, 0).Render()) {
			t.Fatal("advancing one field must not move another")
		}
	})
}

func TestGeometry(t *testing.T) {
	t.Run("happy: Render is exactly Width × Height", func(t *testing.T) {
		for _, sz := range [][2]int{{40, 24}, {80, 30}, {10, 8}} {
			f := Field{Width: sz[0], Height: sz[1], Strategy: FarFast}
			ls := lines(f)
			if len(ls) != sz[1] {
				t.Fatalf("%dx%d: %d lines", sz[0], sz[1], len(ls))
			}
			for i, l := range ls {
				if got := len([]rune(l)); got != sz[0] {
					t.Fatalf("%dx%d line %d: width %d", sz[0], sz[1], i, got)
				}
			}
		}
	})
	t.Run("unhappy: zero or negative size does not panic and yields empty", func(t *testing.T) {
		for _, f := range []Field{{}, {Width: -3, Height: 10}, {Width: 10, Height: 0}} {
			got := f.Render()
			if got != "" {
				t.Fatalf("empty field must render empty, got %q", got)
			}
		}
	})
}

func TestPaintFirst(t *testing.T) {
	t.Run("happy: Paint writes stars the caller can overlay", func(t *testing.T) {
		f := field(FarFast, 0)
		grid := make([][]Cell, f.Height)
		for i := range grid {
			grid[i] = make([]Cell, f.Width)
			for j := range grid[i] {
				grid[i][j] = Cell{' ', -1}
			}
		}
		n := 0
		f.Paint(func(row, col int, ch rune, fg int) {
			n++
			if row < 0 || row >= f.Height || col < 0 || col >= f.Width {
				t.Fatalf("paint out of bounds %d,%d", row, col)
			}
			grid[row][col] = Cell{ch, fg}
		})
		if n < 8 {
			t.Fatalf("expected a field of stars, painted %d cells", n)
		}
		// overlay a "craft" on top of wherever a star landed
		overwritten := false
		for r := range grid {
			for c := range grid[r] {
				if grid[r][c].Ch != ' ' && grid[r][c].Ch != '█' {
					grid[r][c] = Cell{'█', 252}
					overwritten = true
					break
				}
			}
			if overwritten {
				break
			}
		}
		if !overwritten {
			t.Fatal("need a star cell to prove overlay")
		}
	})
	t.Run("unhappy: a nil put is a no-op, not a panic", func(t *testing.T) {
		field(FarFast, 3).Paint(nil)
	})
}

func TestStrategies(t *testing.T) {
	t.Run("happy: catalog names the fly styles", func(t *testing.T) {
		got := map[string]bool{}
		for _, s := range Strategies() {
			got[s.Name] = true
		}
		for _, want := range []string{"far-fast", "near-fast", "uniform", "uniform-slow", "hyperspace", "dust-rush", "drift"} {
			if !got[want] {
				t.Fatalf("missing strategy %q", want)
			}
		}
	})
	t.Run("happy: far-fast — far dust flies every tick, near stars hold", func(t *testing.T) {
		a, b := field(FarFast, 0), field(FarFast, 1)
		mid := a.Height / 2
		if equalInts(glyphCols(a, mid, '·'), glyphCols(b, mid, '·')) {
			t.Fatal("far dust (·) must streak on the first tick")
		}
		if !equalInts(glyphCols(a, mid, '✦'), glyphCols(b, mid, '✦')) {
			t.Fatal("near stars (✦) must crawl — still on tick 1")
		}
		if !equalInts(glyphCols(b, mid, '·'), shiftLeft(glyphCols(a, mid, '·'), a.Width, 1)) {
			t.Fatalf("· must shift left one cell: %v -> %v", glyphCols(a, mid, '·'), glyphCols(b, mid, '·'))
		}
	})
	t.Run("happy: near-fast is the reverse — near streaks, dust holds", func(t *testing.T) {
		a, b := field(NearFast, 0), field(NearFast, 1)
		mid := a.Height / 2
		if equalInts(glyphCols(a, mid, '✦'), glyphCols(b, mid, '✦')) {
			t.Fatal("near stars must streak on the first tick")
		}
		if !equalInts(glyphCols(a, mid, '·'), glyphCols(b, mid, '·')) {
			t.Fatal("far dust must hold still on tick 1")
		}
	})
	t.Run("happy: uniform — every glyph waits the same, then all shift together", func(t *testing.T) {
		a := field(Uniform, 0)
		mid := a.Height / 2
		still := field(Uniform, Uniform.Delay[0]-1)
		moved := field(Uniform, Uniform.Delay[0])
		for _, g := range Glyphs {
			if !equalInts(glyphCols(a, mid, g), glyphCols(still, mid, g)) {
				t.Fatalf("%q must hold until the shared delay", string(g))
			}
			if !equalInts(glyphCols(moved, mid, g), shiftLeft(glyphCols(a, mid, g), a.Width, 1)) {
				t.Fatalf("%q must shift with everyone else", string(g))
			}
		}
	})
	t.Run("happy: hyperspace — every glyph flies one cell per tick", func(t *testing.T) {
		a, b := field(Hyperspace, 0), field(Hyperspace, 1)
		mid := a.Height / 2
		for _, g := range Glyphs {
			if !equalInts(glyphCols(b, mid, g), shiftLeft(glyphCols(a, mid, g), a.Width, 1)) {
				t.Fatalf("hyperspace %q must move every tick", string(g))
			}
		}
	})
	t.Run("happy: a star wrapping off the left re-enters from the right", func(t *testing.T) {
		a := field(FarFast, 0)
		mid := a.Height / 2
		start := glyphCols(a, mid, '·')
		if len(start) == 0 {
			t.Fatal("need far dust to wrap")
		}
		left := start[0]
		for _, c := range start {
			if c < left {
				left = c
			}
		}
		got := glyphCols(field(FarFast, left+1), mid, '·')
		want := shiftLeft(start, a.Width, left+1)
		if !equalInts(got, want) {
			t.Fatalf("wrap: want %v, got %v", want, got)
		}
		found := false
		for _, c := range got {
			if c == a.Width-1 {
				found = true
			}
		}
		if !found {
			t.Fatal("leaving the left edge must reappear at the right")
		}
	})
	t.Run("unhappy: a frozen field ignores tick", func(t *testing.T) {
		a := Field{Width: 40, Height: 24, Tick: 0, Strategy: FarFast, Frozen: true}
		b := Field{Width: 40, Height: 24, Tick: 40, Strategy: FarFast, Frozen: true}
		if a.Render() != b.Render() {
			t.Fatal("frozen starfields must not fly")
		}
		if !strings.ContainsRune(plain(a.Render()), '·') {
			t.Fatal("frozen still means a night sky, not an empty one")
		}
	})
	t.Run("unhappy: a zero strategy falls back to dust-rush", func(t *testing.T) {
		z := field(Strategy{}, 1)
		ff := field(DustRush, 1)
		if plain(z.Render()) != plain(ff.Render()) {
			t.Fatal("the zero strategy must fly like dust-rush")
		}
	})
	t.Run("unhappy: a zero delay on one layer is clamped to 1", func(t *testing.T) {
		s := Strategy{Name: "mixed", Delay: [4]int{0, 8, 8, 8}}
		a, b := field(s, 0), field(s, 1)
		mid := a.Height / 2
		if equalInts(glyphCols(a, mid, '·'), glyphCols(b, mid, '·')) {
			t.Fatal("clamped delay 0 must fly every tick")
		}
		if !equalInts(glyphCols(a, mid, '✦'), glyphCols(b, mid, '✦')) {
			t.Fatal("near with delay 8 must hold")
		}
	})
}

func TestLookup(t *testing.T) {
	t.Run("happy: Lookup finds a named strategy", func(t *testing.T) {
		s, ok := Lookup("dust-rush")
		if !ok || s.Name != "dust-rush" {
			t.Fatalf("Lookup dust-rush -> %v %v", s, ok)
		}
	})
	t.Run("unhappy: unknown names are rejected", func(t *testing.T) {
		if _, ok := Lookup("nope"); ok {
			t.Fatal("unknown strategy must not succeed")
		}
		if _, ok := Lookup(""); ok {
			t.Fatal("empty name must not succeed")
		}
	})
}

func countKinds(f Field) [4]int {
	var n [4]int
	f.Paint(func(row, col int, ch rune, fg int) {
		for i, g := range Glyphs {
			if ch == g {
				n[i]++
				return
			}
		}
	})
	return n
}

func TestPopulation(t *testing.T) {
	t.Run("happy: dust and sparks stay thick; * and ✦ drop ~75%", func(t *testing.T) {
		n := countKinds(Field{Width: 80, Height: 30, Strategy: DustRush})
		dust, spark, mid, near := n[0], n[1], n[2], n[3]
		if dust < 80 || spark < 40 {
			t.Fatalf("dust/spark must stay thick, got ·%d ˚%d", dust, spark)
		}
		large, small := mid+near, dust+spark
		if large*4 > small {
			t.Fatalf("large stars (*%d ✦%d) should be ~75%% fewer than the dust layers (·%d ˚%d)", mid, near, dust, spark)
		}
	})
	t.Run("unhappy: the sky is not only dust — at least one * and one ✦ remain", func(t *testing.T) {
		n := countKinds(Field{Width: 40, Height: 24, Strategy: DustRush})
		if n[2] < 1 || n[3] < 1 {
			t.Fatalf("need at least one mid and one near star, got *%d ✦%d", n[2], n[3])
		}
	})
}

func TestStarColors(t *testing.T) {
	t.Run("happy: real-star tints — white-blue and white-red, not a flat gold field", func(t *testing.T) {
		out := Field{Width: 80, Height: 30, Strategy: DustRush}.Render()
		blue := false
		for _, code := range []string{"38;5;153", "38;5;189", "38;5;195", "38;5;111", "38;5;109", "38;5;103", "38;5;60"} {
			if strings.Contains(out, code) {
				blue = true
				break
			}
		}
		red := false
		for _, code := range []string{"38;5;224", "38;5;217", "38;5;181", "38;5;174", "38;5;138", "38;5;95"} {
			if strings.Contains(out, code) {
				red = true
				break
			}
		}
		if !blue {
			t.Fatal("need a lightish blue-white like real hot stars")
		}
		if !red {
			t.Fatal("need a slight red-white like real cool stars")
		}
	})
	t.Run("unhappy: the near stars are not gold/yellow", func(t *testing.T) {
		out := Field{Width: 80, Height: 30, Strategy: DustRush}.Render()
		for _, gold := range []string{"38;5;229", "38;5;226", "38;5;220", "38;5;178"} {
			if strings.Contains(out, gold) {
				t.Fatalf("too yellow: found %s", gold)
			}
		}
	})
}
