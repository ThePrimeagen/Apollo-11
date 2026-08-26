package sprite

// FlipH mirrors a sprite left/right, remapping block/box glyphs so legs and
// windows still read as themselves. Used to derive W from E, NW from NE, SW
// from SE.
func FlipH(s Sprite) Sprite {
	out := New(s.Width, s.Height)
	for r := 0; r < s.Height; r++ {
		for c := 0; c < s.Width; c++ {
			cell := s.At(r, c)
			cell.Ch = flipHRune(cell.Ch)
			out.Set(r, s.Width-1-c, cell)
		}
	}
	return out
}

// FlipV mirrors a sprite top/bottom. Used to derive S from N, SE from NE.
func FlipV(s Sprite) Sprite {
	out := New(s.Width, s.Height)
	for r := 0; r < s.Height; r++ {
		for c := 0; c < s.Width; c++ {
			cell := s.At(r, c)
			cell.Ch = flipVRune(cell.Ch)
			out.Set(s.Height-1-r, c, cell)
		}
	}
	return out
}

func flipHRune(ch rune) rune {
	if m, ok := flipH[ch]; ok {
		return m
	}
	return ch
}

func flipVRune(ch rune) rune {
	if m, ok := flipV[ch]; ok {
		return m
	}
	return ch
}

var flipH = map[rune]rune{
	'▌': '▐', '▐': '▌',
	'▖': '▗', '▗': '▖',
	'▘': '▝', '▝': '▘',
	'▙': '▟', '▟': '▙',
	'▛': '▜', '▜': '▛',
	'▚': '▞', '▞': '▚',
	'◣': '◢', '◢': '◣',
	'◤': '◥', '◥': '◤',
	'╱': '╲', '╲': '╱',
	'╮': '╭', '╭': '╮',
	'╯': '╰', '╰': '╯',
	'└': '┘', '┘': '└',
	'┌': '┐', '┐': '┌',
	'╴': '╶', '╶': '╴',
	'◀': '▶', '▶': '◀',
	'◁': '▷', '▷': '◁',
	'◃': '▹', '▹': '◃',
}

var flipV = map[rune]rune{
	'▀': '▄', '▄': '▀',
	'▖': '▘', '▘': '▖',
	'▗': '▝', '▝': '▗',
	'▙': '▛', '▛': '▙',
	'▟': '▜', '▜': '▟',
	'▁': '▔', '▔': '▁',
	'▂': '▆', '▆': '▂',
	'▃': '▅', '▅': '▃',
	'◣': '◤', '◤': '◣',
	'◢': '◥', '◥': '◢',
	'╱': '╲', '╲': '╱',
	'╭': '╰', '╰': '╭',
	'╮': '╯', '╯': '╮',
	'┳': '┻', '┻': '┳',
	'┴': '┬', '┬': '┴',
}
