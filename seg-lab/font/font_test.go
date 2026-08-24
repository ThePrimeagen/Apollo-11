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
		if !strings.Contains(body, "│") || !strings.Contains(body, "─") {
			t.Fatalf("small HELLO must light verticals and bars:\n%s", out)
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
		body := strings.Join(ls, "")
		if !strings.ContainsRune(body, '╱') || !strings.ContainsRune(body, '╲') {
			t.Fatalf("W must use both diagonals:\n%s", out)
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
