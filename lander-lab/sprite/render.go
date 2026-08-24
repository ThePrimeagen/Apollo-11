package sprite

import "fmt"

const reset = "\x1b[0m"

func fgSeq(n int) string { return fmt.Sprintf("\x1b[38;5;%dm", n) }
func bgSeq(n int) string { return fmt.Sprintf("\x1b[48;5;%dm", n) }

// Render is raw ANSI 256-color, foreground and background. Transparent
// cells reset SGR so a previous color cannot leak into a blank.
func Render(s Sprite) string {
	if s.Width == 0 || s.Height == 0 {
		return ""
	}
	var b []byte
	for r := 0; r < s.Height; r++ {
		if r > 0 {
			b = append(b, '\n')
		}
		curFG, curBG := -2, -2
		for c := 0; c < s.Width; c++ {
			cell := s.At(r, c)
			ch := cell.Ch
			if ch == 0 {
				ch = ' '
			}
			fg, bg := cell.FG, cell.BG
			if cell.Transparent() {
				fg, bg = -1, -1
			}
			if fg != curFG || bg != curBG {
				b = append(b, reset...)
				if fg >= 0 {
					b = append(b, fgSeq(fg)...)
				}
				if bg >= 0 {
					b = append(b, bgSeq(bg)...)
				}
				curFG, curBG = fg, bg
			}
			b = append(b, string(ch)...)
		}
		b = append(b, reset...)
	}
	return string(b)
}
