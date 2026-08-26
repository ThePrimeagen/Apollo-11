package sprite

// Blit lays src onto dst with src's top-left cell at column x, row y.
// Transparent source cells spare whatever dst already holds, so layers
// compose the way a scene expects; anything past an edge clips (Set
// ignores out-of-bounds writes). A glyph that does not carry its own
// background keeps the destination floor, so fire can sit on the moon.
func Blit(dst Sprite, x, y int, src Sprite) {
	for r := 0; r < src.Height; r++ {
		for c := 0; c < src.Width; c++ {
			cell := src.At(r, c)
			if cell.Transparent() {
				continue
			}
			if cell.BG < 0 {
				under := dst.At(y+r, x+c)
				if under.BG >= 0 && (under.Ch == 0 || under.Ch == ' ') {
					cell.BG = under.BG
				}
			}
			dst.Set(y+r, x+c, cell)
		}
	}
}
