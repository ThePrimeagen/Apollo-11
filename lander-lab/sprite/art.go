package sprite

import (
	"fmt"
	"unicode/utf8"
)

func pad(w int, s string) string {
	r := []rune(s)
	if len(r) > w {
		panic(fmt.Sprintf("row too long: want %d got %d %q", w, len(r), s))
	}
	for len(r) < w {
		r = append(r, ' ')
	}
	return string(r)
}

func rows(w int, ss ...string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = pad(w, s)
		if utf8.RuneCountInString(out[i]) != w {
			panic(fmt.Sprintf("row %d: %d runes want %d", i, utf8.RuneCountInString(out[i]), w))
		}
	}
	return out
}

func spriteFrom(w, h int, heading Heading, glyphs []string, fgMask []string) Sprite {
	if len(glyphs) != h {
		panic(fmt.Sprintf("%s: %d glyph rows, want %d", heading, len(glyphs), h))
	}
	sp := New(w, h)
	useMask := len(fgMask) == h
	for r, row := range glyphs {
		gr := []rune(row)
		var mr []rune
		if useMask {
			mr = []rune(fgMask[r])
			if len(mr) != w {
				panic(fmt.Sprintf("%s mask row %d: %d want %d", heading, r, len(mr), w))
			}
		}
		if len(gr) != w {
			panic(fmt.Sprintf("%s glyph row %d: %d want %d", heading, r, len(gr), w))
		}
		for c, ch := range gr {
			if ch == ' ' {
				continue
			}
			var mat byte = 'S'
			if useMask {
				mat = byte(mr[c])
				if mat == '.' || mat == ' ' {
					mat = 'S'
				}
			} else {
				mat = inferMat(ch, r, c, w, h, heading)
			}
			fg, bg := materialColor(mat)
			// size-1 stays fg-only so it matches the descent-view sprite
			if h == 5 && (mat != 'W' && mat != 'P') {
				bg = -1
			}
			if h == 5 {
				bg = -1
			}
			sp.Set(r, c, Cell{Ch: ch, FG: fg, BG: bg})
		}
	}
	return sp
}

func materialColor(mat byte) (fg, bg int) {
	for _, p := range DefaultPalette {
		if len(p.ID) == 1 && p.ID[0] == mat {
			return p.FG, p.BG
		}
	}
	return 252, -1
}

func inferMat(ch rune, r, c, w, h int, heading Heading) byte {
	switch ch {
	case '~', '≈':
		return 'P'
	case '░':
		return 'W'
	}
	switch heading {
	case N, S:
		return inferNS(ch, r, c, w, h, heading)
	case E, W:
		return inferEW(ch, r, c, w, h, heading)
	default:
		return inferDiag(ch, r, c, w, h, heading)
	}
}

func inferNS(ch rune, r, c, w, h int, heading Heading) byte {
	rr := r
	if heading == S {
		rr = h - 1 - r
	}
	cx := w / 2
	inBell := abs(c-cx) <= w/6 && rr >= (h*6)/10 && rr <= (h*8)/10
	if ch == '▄' || (inBell && (ch == '█' || ch == '▜' || ch == '▛' || ch == '▟' || ch == '▙')) {
		return 'E'
	}
	// cabin in the top ~20%, gold descent through the mid body, silver
	// legs/footpads around the plume.
	switch {
	case rr <= h/5:
		return 'S'
	case rr < (h*7)/10:
		if ch == '▓' || ch == '█' || ch == '▟' || ch == '▙' || ch == '▗' || ch == '▖' {
			return 'G'
		}
		return 'S'
	default:
		return 'S'
	}
}

func inferEW(ch rune, r, c, w, h int, heading Heading) byte {
	cc := c
	if heading == W {
		cc = w - 1 - c
	}
	// E: plume left, engine, gold descent, silver cabin on the right.
	switch {
	case cc < w/5:
		if ch == '▓' || ch == '█' || ch == '▄' || ch == '▜' || ch == '▛' || ch == '◢' || ch == '◣' {
			return 'E'
		}
		return 'S'
	case cc < (w*6)/10:
		if ch == '▓' || ch == '█' || ch == '▟' || ch == '▙' || ch == '▄' {
			return 'G'
		}
		if ch == '░' {
			return 'W'
		}
		return 'S'
	default:
		return 'S'
	}
}

func inferDiag(ch rune, r, c, w, h int, heading Heading) byte {
	if ch == '▓' || ch == '█' {
		return 'G'
	}
	if ch == '▄' || ch == '▜' || ch == '▛' {
		// engine-ish if in the lower-left for NE
		if heading == NE && r >= h/2 && c < w/2 {
			return 'E'
		}
	}
	return 'S'
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Default builds the four-size, eight-heading atlas from the hand-drawn
// N/NE/E art; the other five headings are mirrors.
func Default() *Atlas {
	a := &Atlas{
		Palette: append([]PaletteEntry(nil), DefaultPalette...),
		frames:  map[Size]map[Heading]Sprite{},
	}
	type drawn struct {
		sz       Size
		n, ne, e []string
		nMask    []string
		neMask   []string
		eMask    []string
	}
	w1, _ := Size1.Dim()
	w2, _ := Size2.Dim()
	w3, _ := Size3.Dim()
	w4, _ := Size4.Dim()

	frames := []drawn{
		{
			sz: Size1,
			n: rows(w1,
				"    ▗▛◣▖     ",
				"   ▟░◣╲▜▙    ",
				"  ▟▓████▓▙   ",
				"╱ ◢▔▔▟▄▙▔▔◣ ╲",
				"▁ ▁  ~~~  ▁ ▁",
			),
			ne: rows(w1,
				"        ▗▛◣▖ ",
				"▁ ╲   ▟░╲▜▙  ",
				"  ╲◢▟▓██▓▙   ",
				"   ◥▜▓▓▛  ╲  ",
				" ~~~▜▙    ╲▁ ",
			),
			e: rows(w1,
				"   ▁╲   ╱▁   ",
				"    ╲▟▓▓▙▛◣▖ ",
				"   ◢▟▓▓▙░██▜▙",
				"  ~~◥▜▓▛╲▝◤  ",
				"~~ ▁╱   ╲▁   ",
			),
			nMask: rows(w1,
				"....SSSS.....",
				"...SWSWSS....",
				"..GGGGGGGG...",
				"S.SSSEEESSS.S",
				"S.S..PPP..S.S",
			),
			neMask: rows(w1,
				"........SSSS.",
				"S.S...SWWSS..",
				"..SGGGGGGG...",
				"...EEGGG..S..",
				".PPPEE....SS.",
			),
			eMask: rows(w1,
				"...SS...SS...",
				"....SGGGGSSS.",
				"...EGGGGWSSSS",
				"..PPEGGGWSS..",
				"PP.SS...SS...",
			),
		},
		{
			sz: Size2,
			n: rows(w2,
				"     ▗▀▜▛▀▖      ",
				"    ▟░▞  ▞░▙     ",
				"   ▟▓██████▓▙    ",
				"  ▟▓▓██████▓▓▙   ",
				" ╱ ◢▔▔▟▄▄▙▔▔◣ ╲  ",
				"▁ ▁    ~~~    ▁ ▁",
				"       ~~~       ",
			),
			ne: rows(w2,
				"          ▗▀▜▛▖  ",
				"         ▟░╲▜▙   ",
				"▁  ╲   ▟▓╲██▙    ",
				"  ╲  ◢▟▓██▓▙     ",
				"   ◥▜▓▓▓▓▛  ╲    ",
				"  ~~~▜▓▙     ╲▁  ",
				" ~~~~            ",
			),
			e: rows(w2,
				"    ▁╲     ╱▁    ",
				"     ╲     ╱     ",
				"      ╲▟▓▓▙▛◣▖   ",
				"    ◢▟▓▓▓▓░██▜▙  ",
				"   ~~◥▜▓▓▛╲▝◤    ",
				"  ~~~ ▁╱   ╲▁    ",
				" ~~~~            ",
			),
		},
		{
			sz: Size3,
			n: rows(w3,
				"        ▗▀▜▛▀▖        ",
				"      ▟█░▞  ▞░█▙      ",
				"    ▗▟▓▓████████▓▓▙▖  ",
				"   ▟▓▓████████████▓▓▙ ",
				" ╱ ◢▔▔▔▔▟▄▄▄▄▙▔▔▔◣ ╲  ",
				"╱       ▜████▛       ╲",
				"▁  ▁      ~~~~     ▁ ▁",
				"          ~~~~        ",
			),
			ne: rows(w3,
				"             ▗▀▜▛▖    ",
				"            ▟░╲▜▙     ",
				"▁  ╲      ▟▓░╲██▙     ",
				"  ╲    ◢▟▓▓███▓▙      ",
				"   ╲  ◢▟▓▓███▓▓▛ ╲    ",
				"    ◥▜▓▓▓▓▓▛     ╲    ",
				"  ~~~~▜▓▓▙        ╲▁  ",
				" ~~~~~                ",
			),
			e: rows(w3,
				"     ▁╲        ╱▁     ",
				"      ╲        ╱      ",
				"       ╲  ▟▓▓▙▛◣▖     ",
				"     ◢▟▓▓▓▓▓▓░███▜▙   ",
				"    ~~◥▜▓▓▓▓▛╲▝██▛    ",
				"   ~~~  ▜▓▛  ╲ ▝◤     ",
				"  ~~~~   ▁╱    ╲▁     ",
				" ~~~~~                ",
			),
		},
		{
			sz: Size4,
			n: rows(w4,
				"        ▗▀▀▀▜▛▀▀▀▖        ",
				"       ▟████▀▀████▙       ",
				"      ▟██░▞    ▞░██▙      ",
				"    ▗▟▓▓████████████▓▓▙▖  ",
				"   ▟▓▓████████████████▓▓▙ ",
				"  ▟▓▓██████████████████▓▓▙",
				" ╱ ◢▔▔▔▔▔▔▟▄▄▄▄▄▙▔▔▔▔▔▔◣ ╲",
				"╱     ▁   ▜█████▛  ▁     ╲",
				"▁  ▁        ~~~~      ▁  ▁",
				"            ~~~~          ",
			),
			ne: rows(w4,
				"                ▗▀▀▜▛▖    ",
				"               ▟█░╲▜▙     ",
				"              ▟░▞ ╲██▙    ",
				"▁  ╲        ▟▓▓╲███▓▙     ",
				"  ╲      ◢▟▓▓▓████▓▙      ",
				"   ╲   ◢▟▓▓██████▓▓▛      ",
				"    ◥▜▓▓▓▓████▛     ╲     ",
				"     ▜▓▓▓▓▛          ╲    ",
				"  ~~~~▜▓▓▙            ╲▁  ",
				" ~~~~~                    ",
			),
			e: rows(w4,
				"      ▁╲          ╱▁      ",
				"       ╲          ╱       ",
				"        ╲    ▗▟▓▓▙▛▀◣▖    ",
				"      ◢▟▓▓▓▓▓▓▓▓░████▜▙   ",
				"     ▟▓▓▓▓▓▓▓▓▓░█████▜▙   ",
				"    ~~ ◥▜▓▓▓▓▓▛▝████▛▘    ",
				"   ~~~   ▜▓▓▛  ╲ ▝▀▘      ",
				"  ~~~~    ╲      ╲        ",
				" ~~~~~     ╲      ╲▁      ",
				"            ▁             ",
			),
		},
	}
	for _, f := range frames {
		w, h := f.sz.Dim()
		n := spriteFrom(w, h, N, f.n, f.nMask)
		ne := spriteFrom(w, h, NE, f.ne, f.neMask)
		e := spriteFrom(w, h, E, f.e, f.eMask)
		a.SetFrame(f.sz, N, n)
		a.SetFrame(f.sz, NE, ne)
		a.SetFrame(f.sz, E, e)
		a.SetFrame(f.sz, NW, FlipH(ne))
		a.SetFrame(f.sz, W, FlipH(e))
		a.SetFrame(f.sz, S, FlipV(n))
		a.SetFrame(f.sz, SE, FlipV(ne))
		a.SetFrame(f.sz, SW, FlipH(FlipV(ne)))
	}
	return a
}
