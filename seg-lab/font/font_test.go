package font

// Tests written FIRST.
// Height 1 is the terminal default font (1 row).
// Height 2 is not possible — 14-seg needs three rows.
// Heights 3, 4, 5 are constructed 14-seg of that many rows.

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

func TestHeight2Skipped(t *testing.T) {
	t.Run("unhappy: height 2 is not supported", func(t *testing.T) {
		out, err := Render("HELLO WORLD", 2)
		if err == nil {
			t.Fatal("height 2 must error; two rows cannot hold 14-seg")
		}
		if !errors.Is(err, ErrHeight) {
			t.Fatalf("want ErrHeight, got %v", err)
		}
		if out != "" {
			t.Fatalf("height 2 must not draw, got %q", out)
		}
		if _, _, gerr := GlyphSize(2); !errors.Is(gerr, ErrHeight) {
			t.Fatalf("GlyphSize(2) must be ErrHeight, got %v", gerr)
		}
	})
}

func TestConstructedHeights(t *testing.T) {
	t.Run("happy: 3, 4, 5 are that many rows of filled 14-seg", func(t *testing.T) {
		var prev int
		for _, h := range []int{3, 4, 5} {
			w, rows, err := GlyphSize(h)
			if err != nil {
				t.Fatalf("height %d: %v", h, err)
			}
			if rows != h {
				t.Fatalf("height %d must be %d rows, got %d", h, h, rows)
			}
			if w <= rows {
				t.Fatalf("height %d %dx%d is too skinny", h, w, rows)
			}
			out := mustRender(t, "HELLO WORLD", h)
			if len(lines(out)) != h {
				t.Fatalf("height %d rendered %d rows:\n%s", h, len(lines(out)), out)
			}
			if !strings.ContainsAny(out, "█▀▄") {
				t.Fatalf("height %d must be filled bars:\n%s", h, out)
			}
			if strings.ContainsAny(out, "─│┌┐└┘├┤") {
				t.Fatalf("height %d used thin box-drawing:\n%s", h, out)
			}
			if prev != 0 && rows <= prev {
				t.Fatalf("height %d must be taller than %d", h, prev)
			}
			prev = rows
		}
		if mustRender(t, "W", 5) == mustRender(t, "U", 5) {
			t.Fatalf("W must not collapse to U\n%s", mustRender(t, "W", 5))
		}
	})
	t.Run("unhappy: empty text is empty; 0, 2, and 6 error", func(t *testing.T) {
		out, err := Render("", 3)
		if err != nil || out != "" {
			t.Fatalf("empty text: out=%q err=%v", out, err)
		}
		for _, h := range []int{0, 2, 6, -1, 99} {
			out, err := Render("HELLO", h)
			if !errors.Is(err, ErrHeight) {
				t.Fatalf("height %d: want ErrHeight, got %v", h, err)
			}
			if out != "" {
				t.Fatalf("height %d must not draw, got %q", h, out)
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
		eight := lines(mustRender(t, "8", 5))
		if len(eight) != 5 {
			t.Fatalf("8 at height 5 is %d rows", len(eight))
		}
		last := len([]rune(eight[0])) - 1
		if !hasInk(eight[0]) || !hasInk(eight[len(eight)-1]) {
			t.Fatalf("8 must light A and D:\n%s", mustRender(t, "8", 5))
		}
		if !colHasInk(eight, 0) && !colHasInk(eight, 1) {
			t.Fatalf("8 must light the left verticals:\n%s", mustRender(t, "8", 5))
		}
		if !colHasInk(eight, last) && !colHasInk(eight, last-1) {
			t.Fatalf("8 must light the right verticals:\n%s", mustRender(t, "8", 5))
		}
		if !hasInk(eight[len(eight)/2]) {
			t.Fatalf("8 must light the G bar:\n%s", mustRender(t, "8", 5))
		}
		one := mustRender(t, "1", 5)
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
		out := mustRender(t, "?", 5)
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
		_, _ = Render(in, 3)
		if in != "hello" {
			t.Fatal("Render must not mutate its input")
		}
	})
}
