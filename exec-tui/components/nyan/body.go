package nyan

import "github.com/theprimeagen/apollo-11/exec-tui/components/sprite"

const (
	// BodyCols/BodyRows is the pop-tart cat: frosting slab, cat head
	// on the right (flying east), little legs underneath.
	BodyCols = 22
	BodyRows = 4
	// tartCol/tartRow pin the rainbow origin to the frosting's
	// left-center, so the plume leaves the pastry, not the face.
	tartCol = 6
	tartRow = 1
)

const (
	colFrost = 218
	colCrust = 180
	colFur   = 250
	colEye   = 231
	colNose  = 212
	colMouth = 245
	colLeg   = 250
)

// body is the static hull. Colors are xterm-256; spaces stay transparent.
func body() sprite.Sprite {
	sp := sprite.New(BodyCols, BodyRows)
	tart(sp)
	head(sp)
	legs(sp)
	return sp
}

func tart(sp sprite.Sprite) {
	for r := 0; r < 3; r++ {
		for c := 6; c < 14; c++ {
			sp.Set(r, c, sprite.Cell{Ch: '█', FG: colFrost, BG: colCrust})
		}
	}
	sprinkles := []struct {
		r, c int
		ch   rune
		fg   int
	}{
		{1, 7, '+', 196},
		{1, 9, '.', 39},
		{1, 10, '*', 226},
		{1, 11, '+', 46},
		{1, 12, '.', 213},
	}
	for _, s := range sprinkles {
		sp.Set(s.r, s.c, sprite.Cell{Ch: s.ch, FG: s.fg, BG: colFrost})
	}
}

func head(sp sprite.Sprite) {
	put(sp, 0, 16, `/\_/\`, colFur)
	put(sp, 1, 15, `( o.o )`, colFur)
	sp.Set(1, 17, sprite.Cell{Ch: 'o', FG: colEye, BG: -1})
	sp.Set(1, 19, sprite.Cell{Ch: 'o', FG: colEye, BG: -1})
	sp.Set(1, 18, sprite.Cell{Ch: '.', FG: colNose, BG: -1})
	put(sp, 2, 16, `> ^ <`, colFur)
	sp.Set(2, 18, sprite.Cell{Ch: '^', FG: colMouth, BG: -1})
}

func legs(sp sprite.Sprite) {
	put(sp, 3, 7, `▀ ▀  ▀ ▀`, colLeg)
}

func put(sp sprite.Sprite, row, col int, s string, fg int) {
	for i, r := range []rune(s) {
		if r == ' ' {
			continue
		}
		sp.Set(row, col+i, sprite.Cell{Ch: r, FG: fg, BG: -1})
	}
}
