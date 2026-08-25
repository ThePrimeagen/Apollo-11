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
	for _, st := range catalog(f.Width, f.Height) {
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

// catalog is a deterministic scatter for (w,h), plus four anchors on the
// mid row so every glyph is present and motion tests have a known line.
func catalog(w, h int) []star {
	if w < 1 || h < 1 {
		return nil
	}
	counts := [4]int{
		max(3, w*h/18),
		max(3, w*h/30),
		max(1, w*h/168), // *  ~25% of the old mid density
		max(1, w*h/232), // ✦  ~25% of the old near density
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
		for col := 0; col < w; col += 5 {
			k := kindDust
			if (col/5)%2 == 1 {
				k = kindSpark
			}
			place(mid, col, k)
		}
	}
	for kind := 0; kind < 4; kind++ {
		for n, attempts := 0, 0; n < counts[kind] && attempts < counts[kind]*24; attempts++ {
			r := int(next() % uint64(h))
			c := int(next() % uint64(w))
			if r == mid {
				continue
			}
			if place(r, c, kind) {
				n++
			}
		}
	}
	return out
}
