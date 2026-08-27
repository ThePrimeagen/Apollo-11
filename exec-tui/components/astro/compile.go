package astro

import (
	"fmt"
	"unicode/utf8"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// pixelColor resolves one grid letter to its xterm color. '.' is empty
// sky (-1); anything the palette does not name is a loud error, never
// a guessed color.
func pixelColor(px rune) (int, error) {
	if px == '.' {
		return -1, nil
	}
	for _, p := range Palette {
		if p.ID == string(px) {
			return p.FG, nil
		}
	}
	return -1, fmt.Errorf("astro: unknown pixel letter %q", string(px))
}

// CompileGrid folds a pixel grid of palette letters into half-block
// terminal cells, two pixel rows per cell row: ▀ carries a lone top
// pixel, ▄ a lone bottom pixel, █ a matched pair, and a two-color pair
// rides one ▀ with the bottom pixel as its background.
func CompileGrid(rows []string) (sprite.Sprite, error) {
	if len(rows) == 0 {
		return sprite.Sprite{}, fmt.Errorf("astro: empty pixel grid")
	}
	if len(rows)%2 != 0 {
		return sprite.Sprite{}, fmt.Errorf("astro: %d pixel rows — half-block cells need an even count", len(rows))
	}
	w := utf8.RuneCountInString(rows[0])
	if w == 0 {
		return sprite.Sprite{}, fmt.Errorf("astro: empty pixel row")
	}
	for i, row := range rows {
		if utf8.RuneCountInString(row) != w {
			return sprite.Sprite{}, fmt.Errorf("astro: pixel row %d has %d pixels, want %d", i, utf8.RuneCountInString(row), w)
		}
	}
	sp := sprite.New(w, len(rows)/2)
	for cr := 0; cr < len(rows)/2; cr++ {
		top := []rune(rows[2*cr])
		bot := []rune(rows[2*cr+1])
		for c := 0; c < w; c++ {
			tc, err := pixelColor(top[c])
			if err != nil {
				return sprite.Sprite{}, fmt.Errorf("%w (row %d col %d)", err, 2*cr, c)
			}
			bc, err := pixelColor(bot[c])
			if err != nil {
				return sprite.Sprite{}, fmt.Errorf("%w (row %d col %d)", err, 2*cr+1, c)
			}
			switch {
			case tc < 0 && bc < 0:
				// empty sky — sprite.New already left it transparent.
			case tc >= 0 && bc < 0:
				sp.Set(cr, c, sprite.Cell{Ch: '▀', FG: tc, BG: -1})
			case tc < 0 && bc >= 0:
				sp.Set(cr, c, sprite.Cell{Ch: '▄', FG: bc, BG: -1})
			case tc == bc:
				sp.Set(cr, c, sprite.Cell{Ch: '█', FG: tc, BG: -1})
			default:
				sp.Set(cr, c, sprite.Cell{Ch: '▀', FG: tc, BG: bc})
			}
		}
	}
	return sp, nil
}
