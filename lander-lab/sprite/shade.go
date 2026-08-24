package sprite

// Shade ramp for partially filled cells. Ctrl-A walks up, Ctrl-B walks down.
// Geometric glyphs (▟, ▛, ╱, …) sit at the top of the ramp: incrementing
// them is a no-op, decrementing drops them onto ▓ and then down to empty.
var shadeRamp = []rune{' ', '░', '▒', '▓', '█'}

func shadeIndex(ch rune) int {
	for i, r := range shadeRamp {
		if r == ch {
			return i
		}
	}
	if ch == 0 {
		return 0
	}
	// a geometric cell is "solid-ish"
	return len(shadeRamp) - 1
}

// IncrementShade moves one step toward a full block. Empty cells become ░.
func IncrementShade(c Cell) Cell {
	i := shadeIndex(c.Ch)
	if i < len(shadeRamp)-1 {
		c.Ch = shadeRamp[i+1]
	} else {
		c.Ch = shadeRamp[len(shadeRamp)-1]
	}
	if c.FG < 0 && !c.Transparent() {
		c.FG = 252
	}
	return c
}

// DecrementShade moves one step toward transparent. Empty stays empty.
func DecrementShade(c Cell) Cell {
	i := shadeIndex(c.Ch)
	if i <= 0 {
		c.Ch = ' '
		c.FG, c.BG = -1, -1
		return c
	}
	c.Ch = shadeRamp[i-1]
	if c.Ch == ' ' {
		c.FG, c.BG = -1, -1
	}
	return c
}
