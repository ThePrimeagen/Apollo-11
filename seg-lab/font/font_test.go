package font

// Tests written FIRST. font is a Go package that draws a string in
// 14-segment strokes — small or large. No TTF, no Python. Pass a string
// and a size. Happy + unhappy throughout.

import (
	"strings"
	"testing"
)

func lines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func glyphW(size Size, n int) int {
	w, _ := GlyphSize(size)
	if n == 0 {
		return 0
	}
	return n*w + (n - 1)
}

func TestGlyphSize(t *testing.T) {
	t.Run("happy: large is a bigger cell than small", func(t *testing.T) {
		sw, sh := GlyphSize(Small)
		lw, lh := GlyphSize(Large)
		if sw < 5 || sh < 5 {
			t.Fatalf("small %dx%d is too tight for a letter", sw, sh)
		}
		if lw <= sw || lh <= sh {
			t.Fatalf("large %dx%d must outsize small %dx%d", lw, lh, sw, sh)
		}
	})
	t.Run("happy: more columns than rows so a terminal cell does not squash the letter", func(t *testing.T) {
		// A terminal cell is about twice as tall as it is wide. A square
		// grid of █ (7×7, 11×13) renders as a skinny stick — that is why
		// the Go port looked worse than the TTF, whose advance/height is ~0.59.
		for _, size := range []Size{Small, Large} {
			w, h := GlyphSize(size)
			if w <= h {
				t.Fatalf("%s %dx%d is too skinny; need width > height", size, w, h)
			}
		}
	})
	t.Run("unhappy: an unknown size reports 0x0", func(t *testing.T) {
		w, h := GlyphSize(Size(99))
		if w != 0 || h != 0 {
			t.Fatalf("unknown size must be 0x0, got %dx%d", w, h)
		}
	})
}

func TestRenderSmall(t *testing.T) {
	t.Run("happy: HELLO is five-row 14-seg and H/E light bars", func(t *testing.T) {
		out := Render("HELLO", Small)
		ls := lines(out)
		_, h := GlyphSize(Small)
		if len(ls) != h {
			t.Fatalf("small height %d, want %d\n%s", len(ls), h, out)
		}
		if len([]rune(ls[0])) != glyphW(Small, 5) {
			t.Fatalf("small width %d, want %d (%q)", len([]rune(ls[0])), glyphW(Small, 5), ls[0])
		}
		body := strings.Join(ls, "\n")
		if !strings.ContainsRune(body, '█') {
			t.Fatalf("small HELLO must be filled LED bars, not a wireframe:\n%s", out)
		}
		if strings.ContainsAny(body, "─│┌┐└┘├┤") {
			t.Fatalf("thin box-drawing is the look we left behind:\n%s", out)
		}
	})
	t.Run("unhappy: junk runes occupy a blank cell, no panic", func(t *testing.T) {
		out := Render("H@", Small)
		ls := lines(out)
		if len(ls) == 0 {
			t.Fatal("H@ must still be a grid")
		}
		if strings.Contains(out, "@") {
			t.Fatalf("junk must not leak through, got %q", out)
		}
	})
}

func TestRenderLarge(t *testing.T) {
	t.Run("happy: HELLO WORLD is large writing with a full W", func(t *testing.T) {
		out := Render("HELLO WORLD", Large)
		ls := lines(out)
		_, h := GlyphSize(Large)
		if len(ls) != h {
			t.Fatalf("large height %d, want %d\n%s", len(ls), h, out)
		}
		if len([]rune(ls[0])) != glyphW(Large, 11) {
			t.Fatalf("large width %d, want %d", len([]rune(ls[0])), glyphW(Large, 11))
		}
		wBody := strings.Join(ls, "")
		if strings.Count(wBody, "█") < 20 {
			t.Fatalf("large HELLO WORLD must be filled bars:\n%s", out)
		}
		u := Render("U", Large)
		w := Render("W", Large)
		if w == u {
			t.Fatalf("W must not collapse to U\n%s", w)
		}
		if !strings.Contains(w, "█") {
			t.Fatalf("W must be filled bars:\n%s", w)
		}
		small := Render("HELLO WORLD", Small)
		if len(ls) <= len(lines(small)) {
			t.Fatal("large must be taller than small")
		}
	})
	t.Run("unhappy: empty text and unknown size stay blank", func(t *testing.T) {
		if got := Render("", Large); got != "" {
			t.Fatalf("empty must be empty, got %q", got)
		}
		if got := Render("HELLO", Size(99)); got != "" {
			t.Fatalf("unknown size must render empty, got %q", got)
		}
	})
}

func hasInk(s string) bool {
	return strings.ContainsAny(s, "█▀▄")
}

func colHasInk(rows []string, col int) bool {
	for _, row := range rows {
		rs := []rune(row)
		if col >= 0 && col < len(rs) && rs[col] != ' ' {
			return true
		}
	}
	return false
}

func TestLEDGeometry(t *testing.T) {
	t.Run("happy: 8 is a closed frame with a mid bar, 1 sits on the right", func(t *testing.T) {
		eight := lines(Render("8", Large))
		if len(eight) < 3 {
			t.Fatal("8 is empty")
		}
		last := len([]rune(eight[0])) - 1
		if !hasInk(eight[0]) || !hasInk(eight[len(eight)-1]) {
			t.Fatalf("8 must light A and D:\n%s", Render("8", Large))
		}
		if !colHasInk(eight, 0) && !colHasInk(eight, 1) {
			t.Fatalf("8 must light the left verticals:\n%s", Render("8", Large))
		}
		if !colHasInk(eight, last) && !colHasInk(eight, last-1) {
			t.Fatalf("8 must light the right verticals:\n%s", Render("8", Large))
		}
		if !hasInk(eight[len(eight)/2]) {
			t.Fatalf("8 must light the G bar:\n%s", Render("8", Large))
		}
		one := Render("1", Large)
		// 1 is B|C — ink belongs on the right half, not a centered I-beam.
		for _, row := range lines(one) {
			rs := []rune(row)
			leftInk := 0
			for i := 0; i < len(rs)/2; i++ {
				if rs[i] != ' ' {
					leftInk++
				}
			}
			if leftInk > 1 {
				t.Fatalf("1 leaked onto the left side:\n%s", one)
			}
		}
	})
	t.Run("unhappy: a letter the map does not have stays a blank cell", func(t *testing.T) {
		out := Render("?", Large)
		if strings.ContainsAny(out, "█▀▄") {
			t.Fatalf("unknown rune must not invent segments:\n%s", out)
		}
	})
}

func TestRenderPurity(t *testing.T) {
	t.Run("happy: Render is deterministic", func(t *testing.T) {
		if Render("APOLLO", Large) != Render("APOLLO", Large) {
			t.Fatal("Render must be pure")
		}
	})
	t.Run("unhappy: lowercase input is not mutated", func(t *testing.T) {
		in := "hello"
		_ = Render(in, Small)
		if in != "hello" {
			t.Fatal("Render must not mutate its input")
		}
	})
}

func TestSizeName(t *testing.T) {
	t.Run("happy: sizes are named small and large", func(t *testing.T) {
		if Small.String() != "small" || Large.String() != "large" {
			t.Fatalf("names %q %q", Small, Large)
		}
	})
	t.Run("unhappy: an unknown size has an empty name", func(t *testing.T) {
		if Size(99).String() != "" {
			t.Fatalf("unknown size named %q", Size(99))
		}
	})
}
