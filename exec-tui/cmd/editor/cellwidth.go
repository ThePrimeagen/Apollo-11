package editor

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

func init() {
	// Ambiguous-width block/geometry runes stay one cell so a CJK
	// locale cannot double the canvas and wrap the footer off the left.
	runewidth.DefaultCondition.EastAsianWidth = false
}

func runeCols(r rune) int {
	w := runewidth.RuneWidth(r)
	if w < 1 {
		return 1
	}
	return w
}

func displayLen(s string) int {
	return runewidth.StringWidth(strip(s))
}

func clipPad(s string, w int) string {
	if w < 1 {
		return ""
	}
	n := displayLen(s)
	if n == w {
		return s
	}
	if n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return runewidth.Truncate(strip(s), w, "")
}

func clipTo(s string, w int) string {
	if w < 1 {
		return ""
	}
	if displayLen(s) <= w {
		return s
	}
	return runewidth.Truncate(strip(s), w, "")
}

func clipLines(s string, w int) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = clipTo(lines[i], w)
	}
	return strings.Join(lines, "\n")
}
