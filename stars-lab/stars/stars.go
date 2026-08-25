// Package stars is a reusable one-cell starfield. Paint it first; everything
// else draws on top. Four glyphs (· ˚ * ✦), any width×height, strategy-driven
// right-to-left fly. The package has no lander/DSKY/TUI dependencies.
package stars

import (
	"fmt"
	"strings"
)

// The four background stars, far/dim → near/bright. Glyphs only — no twinkle.
var Glyphs = [4]rune{'·', '˚', '*', '✦'}

const (
	kindDust  = 0
	kindSpark = 1
	kindMid   = 2
	kindNear  = 3
)

// palettes are xterm-256 tints per kind: mostly white, with a light blue
// and a slight red like real stars. No gold.
var palettes = [4][]int{
	{238, 242, 60, 66, 95},         // dust: dim gray / blue-gray / rose
	{244, 247, 109, 103, 138},      // spark: mid gray / muted blue / muted rose
	{252, 255, 153, 189, 217},      // mid: white / blue-white / rose-white
	{255, 254, 195, 153, 224, 218}, // near: bright white / pale blue / pale rose
}

func tint(kind, row, col int) int {
	p := palettes[kind]
	h := uint32(row*131 + col*17 + kind*97)
	return p[int(h)%len(p)]
}

// Strategy is one fly style. Delay[kind] is ticks per cell of travel;
// smaller means faster. The zero Strategy flies as DustRush.
type Strategy struct {
	Name  string
	Delay [4]int
}

// Named fly-through styles. DustRush is the default: dust streaks past,
// the bigger stars almost hang.
var (
	DustRush    = Strategy{Name: "dust-rush", Delay: [4]int{1, 2, 8, 10}}
	Still       = Strategy{Name: "still"} // tick is ignored; the sky holds
	FarFast     = Strategy{Name: "far-fast", Delay: [4]int{1, 2, 4, 8}}
	NearFast    = Strategy{Name: "near-fast", Delay: [4]int{8, 4, 2, 1}}
	Uniform     = Strategy{Name: "uniform", Delay: [4]int{2, 2, 2, 2}}
	UniformSlow = Strategy{Name: "uniform-slow", Delay: [4]int{5, 5, 5, 5}}
	Hyperspace  = Strategy{Name: "hyperspace", Delay: [4]int{1, 1, 1, 1}}
	Drift       = Strategy{Name: "drift", Delay: [4]int{4, 6, 8, 12}}
)

// Strategies returns every named fly style in demo order.
func Strategies() []Strategy {
	return []Strategy{DustRush, Still, FarFast, NearFast, Uniform, UniformSlow, Hyperspace, Drift}
}

// Lookup finds a named strategy (case-sensitive).
func Lookup(name string) (Strategy, bool) {
	for _, s := range Strategies() {
		if s.Name == name {
			return s, true
		}
	}
	return Strategy{}, false
}

func (s Strategy) delays() [4]int {
	if s.Name == Still.Name {
		return [4]int{1, 1, 1, 1} // unused: Paint freezes tick
	}
	if s.Delay == [4]int{} {
		return DustRush.Delay
	}
	out := s.Delay
	for i, d := range out {
		if d < 1 {
			out[i] = 1
		}
	}
	return out
}

// Cell is one painted star (or empty, when Ch == ' ' / Fg < 0).
type Cell struct {
	Ch rune
	Fg int
}

// DefaultDensity is how thick each layer scatters, in stars per 1000
// cells: dust, spark, mid, near. A Field with a zero Density paints
// with exactly these.
var DefaultDensity = [4]int{56, 33, 6, 4}

// MaxDensity caps a layer so an absurd knob cannot ask for more stars
// than a sky can hold.
const MaxDensity = 400

// Field is one starfield instance. Width/Height are in terminal cells.
// Density[kind] is stars per 1000 cells for that layer; zero or
// negative means DefaultDensity for that layer.
type Field struct {
	Width, Height int
	Tick          int
	Strategy      Strategy
	Frozen        bool
	Density       [4]int
}

type star struct{ row, col, kind, fg int }

// Paint calls put for every star cell this frame. Draw this FIRST — put
// overwrites whatever is there, and the caller then paints craft/UI on top.
// put may be nil (no-op).
func (f Field) Paint(put func(row, col int, ch rune, fg int)) {
	if put == nil || f.Width < 1 || f.Height < 1 {
		return
	}
	tick := f.Tick
	if f.Frozen || f.Strategy.Name == Still.Name || tick < 0 {
		tick = 0
	}
	delays := f.Strategy.delays()
	for _, st := range catalog(f.Width, f.Height, f.Density) {
		d := delays[st.kind]
		col := wrap(st.col-tick/d, f.Width)
		put(st.row, col, Glyphs[st.kind], st.fg)
	}
}

// Render is the standalone ANSI view of the field. Pure.
func (f Field) Render() string {
	if f.Width < 1 || f.Height < 1 {
		return ""
	}
	grid := make([][]Cell, f.Height)
	for i := range grid {
		grid[i] = make([]Cell, f.Width)
		for j := range grid[i] {
			grid[i][j] = Cell{' ', -1}
		}
	}
	f.Paint(func(row, col int, ch rune, fg int) {
		if row < 0 || row >= f.Height || col < 0 || col >= f.Width {
			return
		}
		grid[row][col] = Cell{ch, fg}
	})
	var b strings.Builder
	for i, row := range grid {
		if i > 0 {
			b.WriteString("\n")
		}
		cur := -2
		for _, c := range row {
			if c.Fg != cur {
				if c.Fg < 0 {
					b.WriteString("\x1b[0m")
				} else {
					b.WriteString(fmt.Sprintf("\x1b[38;5;%dm", c.Fg))
				}
				cur = c.Fg
			}
			b.WriteRune(c.Ch)
		}
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

func wrap(col, w int) int {
	if w <= 0 {
		return 0
	}
	return ((col % w) + w) % w
}

// layerCount is one layer's scatter target: density stars per 1000
// cells, the stock density when the knob is unset or negative, capped
// so a flood cannot swamp the scatter loop, floored so a sky is never
// empty of a layer.
func layerCount(w, h, density, kind int) int {
	if density <= 0 {
		density = DefaultDensity[kind]
	}
	if density > MaxDensity {
		density = MaxDensity
	}
	floor := 3
	if kind >= kindMid {
		floor = 1
	}
	return max(floor, w*h*density/1000)
}

// catalog is a deterministic scatter for (w,h) at the given per-layer
// densities, plus four fixed anchors on the mid row so every glyph is
// present and motion tests have a known line. The scatter is evenly
// spread in a random order — never a normal distribution: rows are
// stratified (each layer walks the sky with a stride coprime to the
// height, so no row collects a stripe), and columns are stratified too
// (each star owns an evenly spaced slice of its row and lands at a
// hashed offset inside it, with a per-row phase so rows never align
// into a lattice). Pure hashing would leave holes and clumps — a
// 30-row sky used to double up its middle row and open half-screen
// gaps inside rows.
func catalog(w, h int, density [4]int) []star {
	if w < 1 || h < 1 {
		return nil
	}
	counts := [4]int{
		layerCount(w, h, density[0], kindDust),
		layerCount(w, h, density[1], kindSpark),
		layerCount(w, h, density[2], kindMid),
		layerCount(w, h, density[3], kindNear),
	}
	seen := make(map[int]bool, w*h/8)
	out := make([]star, 0, counts[0]+counts[1]+counts[2]+counts[3]+4)
	seed := uint64(0x9E3779B97F4A7C15)
	next := func() uint64 {
		seed += 0x9E3779B97F4A7C15
		z := seed
		z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
		z = (z ^ (z >> 27)) * 0x94D049BB133111EB
		return z ^ (z >> 31)
	}
	place := func(row, col, kind int) bool {
		if row < 0 || row >= h || col < 0 || col >= w {
			return false
		}
		key := row*w + col
		if seen[key] {
			return false
		}
		seen[key] = true
		out = append(out, star{row, col, kind, tint(kind, row, col)})
		return true
	}
	mid := h / 2
	if w >= 10 {
		place(mid, 4, kindDust)
		place(mid, 10, kindSpark)
		place(mid, 16, kindMid)
		if w > 22 {
			place(mid, 22, kindNear)
		} else {
			place(mid, w-1, kindNear)
		}
	}
	stride := rowStride(h)
	for kind := 0; kind < 4; kind++ {
		start := kind * h / 4
		// span is the layer's fair column spacing within one row; each
		// star owns one span-wide stratum and jitters inside half of
		// it, so a row is evenly covered yet randomly placed. Sparse
		// layers (under one star per row) roam the whole width.
		perRow := counts[kind] / h
		if perRow < 1 {
			perRow = 1
		}
		span := w / perRow
		if span < 1 {
			span = 1
		}
		for n, attempts, stall := 0, 0, 0; n < counts[kind] && attempts < counts[kind]*24; attempts++ {
			r := (start + n*stride) % h
			// the stride visits every row once per h stars, so n/h is
			// which stratum of its row this star fills; the hashed
			// phase de-aligns the strata from row to row. A stalled
			// star widens its jitter window until it escapes a
			// congested stratum.
			stratum := n / h * span
			phase := int(hash2(r, kind) % uint64(span))
			base := span / 2
			if perRow == 1 {
				base = span // sparse layers roam the whole width
			}
			window := max(1, base) * (1 + stall)
			if window > w {
				window = w
			}
			c := (stratum + phase + int(next()%uint64(window))) % w
			if place(r, c, kind) {
				n++
				stall = 0
			} else {
				stall++
			}
		}
	}
	return out
}

// hash2 mixes two small ints into a well-avalanched 64-bit hash, so
// per-row stratum phases never form diagonal patterns.
func hash2(a, b int) uint64 {
	z := uint64(a)*0x9E3779B97F4A7C15 + uint64(b)*0xBF58476D1CE4E5B9
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// rowStride is a row step coprime with h, close to h/φ, so a layer's
// stars visit every row in a low-discrepancy order: a sparse layer
// spreads over the whole sky instead of banding, and no row is ever
// visited twice before all rows are visited once.
func rowStride(h int) int {
	s := h * 13 / 21
	if s < 1 {
		s = 1
	}
	for gcd(s, h) != 1 {
		s--
	}
	return s
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
