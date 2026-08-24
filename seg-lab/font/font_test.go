package font

// Tests written FIRST. Render(text, height) is 1–5.
// Height 1 is the terminal's default font. 2–5 are constructed 14-seg.
// Height above 5 returns an error.

import (
	"errors"
	"strings"
	"testing"
)

func mustRender(t *testing.T, text string, height int) string {
	t.Helper()
	out, err := Render(text, height)
	if err != nil {
		t.Fatalf("Render(%q, %d) unexpected error: %v", text, height, err)
	}
	return out
}

func lines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func TestHeight1DefaultFont(t *testing.T) {
	t.Run("happy: height 1 is the string in the terminal's own font", func(t *testing.T) {
		out := mustRender(t, "HELLO WORLD", 1)
		if out != "HELLO WORLD" {
			t.Fatalf("height 1 must be plain text, got %q", out)
		}
		if strings.ContainsAny(out, "█▀▄") {
			t.Fatalf("height 1 must not construct bars:\n%s", out)
		}
		w, rows, err := GlyphSize(1)
		if err != nil || w != 1 || rows != 1 {
			t.Fatalf("height 1 cell must be 1×1, got %dx%d err=%v", w, rows, err)
		}
	})
	t.Run("unhappy: height 1 does not invent letters for junk", func(t *testing.T) {
		out := mustRender(t, "H@", 1)
		if out != "H@" {
			t.Fatalf("height 1 must pass the string through, got %q", out)
		}
	})
}

func TestHeight2Constructed(t *testing.T) {
	t.Run("happy: height 2 is a constructed 14-seg, taller than 1, shorter than 3", func(t *testing.T) {
		out := mustRender(t, "HELLO WORLD", 2)
		ls := lines(out)
		w, rows, err := GlyphSize(2)
		if err != nil {
			t.Fatal(err)
		}
		if rows < 5 {
			t.Fatalf("height 2 is %d rows; a constructed font needs room for bars", rows)
		}
		if w <= rows {
			t.Fatalf("height 2 %dx%d is too skinny", w, rows)
		}
		if len(ls) != rows {
			t.Fatalf("height 2: got %d rows, want %d\n%s", len(ls), rows, out)
		}
		if !strings.ContainsAny(out, "█▀▄") {
			t.Fatalf("height 2 must be filled LED bars:\n%s", out)
		}
		if strings.ContainsAny(out, "─│┌┐└┘├┤") {
			t.Fatalf("height 2 used thin box-drawing:\n%s", out)
		}
		one := mustRender(t, "HELLO WORLD", 1)
		three := mustRender(t, "HELLO WORLD", 3)
		if len(ls) <= len(lines(one)) {
			t.Fatal("height 2 must be taller than the default font")
		}
		if len(ls) >= len(lines(three)) {
			t.Fatal("height 2 must be shorter than height 3")
		}
	})
	t.Run("unhappy: junk runes occupy a blank cell, no panic", func(t *testing.T) {
		out := mustRender(t, "H@", 2)
		if strings.Contains(out, "@") {
			t.Fatalf("junk must not leak through, got %q", out)
		}
	})
}

func TestHeight3(t *testing.T) {
	t.Run("happy: height 3 stays the 10×7 look", func(t *testing.T) {
		w, rows, err := GlyphSize(3)
		if err != nil {
			t.Fatal(err)
		}
		if w != 10 || rows != 7 {
			t.Fatalf("height 3 must stay 10×7, got %dx%d", w, rows)
		}
		out := mustRender(t, "HELLO WORLD", 3)
		if !strings.ContainsAny(out, "█▀▄") {
			t.Fatalf("height 3 must be filled bars:\n%s", out)
		}
		if len(lines(out)) != 7 {
			t.Fatalf("height 3 rows %d", len(lines(out)))
		}
	})
	t.Run("unhappy: empty text at height 3 is empty, not an error", func(t *testing.T) {
		out, err := Render("", 3)
		if err != nil {
			t.Fatalf("empty text is not a height error: %v", err)
		}
		if out != "" {
			t.Fatalf("empty must be empty, got %q", out)
		}
	})
}

func TestHeight4And5(t *testing.T) {
	t.Run("happy: 4 and 5 grow past 3 and stay filled 14-seg", func(t *testing.T) {
		_, r3, err := GlyphSize(3)
		if err != nil {
			t.Fatal(err)
		}
		prev := r3
		for _, h := range []int{4, 5} {
			w, rows, err := GlyphSize(h)
			if err != nil {
				t.Fatalf("height %d: %v", h, err)
			}
			if w <= rows {
				t.Fatalf("height %d %dx%d is too skinny", h, w, rows)
			}
			if rows <= prev {
				t.Fatalf("height %d (%d rows) must outsize the one below (%d)", h, rows, prev)
			}
			out := mustRender(t, "HELLO WORLD", h)
			if len(lines(out)) != rows {
				t.Fatalf("height %d rows %d, want %d", h, len(lines(out)), rows)
			}
			if !strings.ContainsAny(out, "█▀▄") {
				t.Fatalf("height %d must be filled bars:\n%s", h, out)
			}
			prev = rows
		}
		if mustRender(t, "W", 4) == mustRender(t, "U", 4) {
			t.Fatalf("W must not collapse to U\n%s", mustRender(t, "W", 4))
		}
	})
	t.Run("unhappy: height 6 and 0 return an error", func(t *testing.T) {
		for _, h := range []int{0, 6, -1, 99} {
			out, err := Render("HELLO", h)
			if err == nil {
				t.Fatalf("height %d must error", h)
			}
			if !errors.Is(err, ErrHeight) {
				t.Fatalf("height %d: want ErrHeight, got %v", h, err)
			}
			if out != "" {
				t.Fatalf("height %d must not draw, got %q", h, out)
			}
			if _, _, gerr := GlyphSize(h); !errors.Is(gerr, ErrHeight) {
				t.Fatalf("GlyphSize(%d) must be ErrHeight, got %v", h, gerr)
			}
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
		eight := lines(mustRender(t, "8", 4))
		if len(eight) < 3 {
			t.Fatal("8 is empty")
		}
		last := len([]rune(eight[0])) - 1
		if !hasInk(eight[0]) || !hasInk(eight[len(eight)-1]) {
			t.Fatalf("8 must light A and D:\n%s", mustRender(t, "8", 4))
		}
		if !colHasInk(eight, 0) && !colHasInk(eight, 1) {
			t.Fatalf("8 must light the left verticals:\n%s", mustRender(t, "8", 4))
		}
		if !colHasInk(eight, last) && !colHasInk(eight, last-1) {
			t.Fatalf("8 must light the right verticals:\n%s", mustRender(t, "8", 4))
		}
		if !hasInk(eight[len(eight)/2]) {
			t.Fatalf("8 must light the G bar:\n%s", mustRender(t, "8", 4))
		}
		one := mustRender(t, "1", 4)
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
		out := mustRender(t, "?", 4)
		if strings.ContainsAny(out, "█▀▄") {
			t.Fatalf("unknown rune must not invent segments:\n%s", out)
		}
	})
}

func TestRenderPurity(t *testing.T) {
	t.Run("happy: Render is deterministic", func(t *testing.T) {
		a, _ := Render("APOLLO", 3)
		b, _ := Render("APOLLO", 3)
		if a != b {
			t.Fatal("Render must be pure")
		}
	})
	t.Run("unhappy: lowercase input is not mutated", func(t *testing.T) {
		in := "hello"
		_, _ = Render(in, 2)
		if in != "hello" {
			t.Fatal("Render must not mutate its input")
		}
	})
}
