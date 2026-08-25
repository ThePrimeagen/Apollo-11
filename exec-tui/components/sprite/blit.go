package sprite

// Blit lays src onto dst with src's top-left cell at column x, row y.
// Transparent source cells spare whatever dst already holds, so layers
// compose the way a scene expects; anything past an edge clips (Set
// ignores out-of-bounds writes).
func Blit(dst Sprite, x, y int, src Sprite) {
	for r := 0; r < src.Height; r++ {
		for c := 0; c < src.Width; c++ {
			cell := src.At(r, c)
			if cell.Transparent() {
				continue
			}
			dst.Set(y+r, x+c, cell)
		}
	}
}
