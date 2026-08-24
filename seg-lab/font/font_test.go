package font

// Tests written FIRST. font.Render(text, height) takes height units 1–5.
// Happy + unhappy throughout.

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

func glyphW(height, n int) int {
	w, _ := GlyphSize(height)
	if n == 0 {
		return 0
	}
	return n*w + (n - 1)
}

func TestGlyphSize(t *testing.T) {
	t.Run("happy: units 1–5 grow in both axes and stay wider than tall", func(t *testing.T) {
		var prevW, prevH int
		for h := 1; h <= 5; h++ {
			w, rows := GlyphSize(h)
			if rows < 3 {
				t.Fatalf("height %d is %d rows; 14-seg needs a top, mid, and bottom", h, rows)
			}
			if w <= rows {
				t.Fatalf("height %d is %dx%d; terminal cells squash square grids", h, w, rows)
			}
			if h > 1 && (w <= prevW || rows <= prevH) {
				t.Fatalf("height %d (%dx%d) must outsize height %d (%dx%d)", h, w, rows, h-1, prevW, prevH)
			}
			prevW, prevH = w, rows
		}
	})
	t.Run("unhappy: height 0 and 6 report 0x0", func(t *testing.T) {
		for _, h := range []int{0, 6, -1, 99} {
			w, rows := GlyphSize(h)
			if w != 0 || rows != 0 {
				t.Fatalf("height %d must be 0x0, got %dx%d", h, w, rows)
			}
		}
	})
}

func TestRenderHeight(t *testing.T) {
	t.Run("happy: HELLO WORLD at every unit is that many rows of filled bars", func(t *testing.T) {
		var prevRows int
		for h := 1; h <= 5; h++ {
			out := Render("HELLO WORLD", h)
			ls := lines(out)
			_, rows := GlyphSize(h)
			if len(ls) != rows {
				t.Fatalf("height %d: got %d rows, want %d\n%s", h, len(ls), rows, out)
			}
			if len([]rune(ls[0])) != glyphW(h, 11) {
				t.Fatalf("height %d: width %d, want %d", h, len([]rune(ls[0])), glyphW(h, 11))
			}
			body := strings.Join(ls, "\n")
			if !strings.ContainsAny(body, "█▀▄") {
				t.Fatalf("height %d must be filled LED bars:\n%s", h, out)
			}
			if strings.ContainsAny(body, "─│┌┐└┘├┤") {
				t.Fatalf("height %d used thin box-drawing:\n%s", h, out)
			}
			if h > 1 && len(ls) <= prevRows {
				t.Fatalf("height %d must be taller than %d", h, h-1)
			}
			prevRows = len(ls)
		}
		if Render("W", 4) == Render("U", 4) {
			t.Fatalf("W must not collapse to U\n%s", Render("W", 4))
		}
	})
	t.Run("unhappy: empty text, junk runes, and a bad height stay blank", func(t *testing.T) {
		if got := Render("", 3); got != "" {
			t.Fatalf("empty must be empty, got %q", got)
		}
		if got := Render("HELLO", 0); got != "" {
			t.Fatalf("height 0 must render empty, got %q", got)
		}
		if got := Render("HELLO", 6); got != "" {
			t.Fatalf("height 6 must render empty, got %q", got)
		}
		out := Render("H@", 2)
		if len(lines(out)) == 0 {
			t.Fatal("H@ must still be a grid")
		}
		if strings.Contains(out, "@") {
			t.Fatalf("junk must not leak through, got %q", out)
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
		eight := lines(Render("8", 4))
		if len(eight) < 3 {
			t.Fatal("8 is empty")
		}
		last := len([]rune(eight[0])) - 1
		if !hasInk(eight[0]) || !hasInk(eight[len(eight)-1]) {
			t.Fatalf("8 must light A and D:\n%s", Render("8", 4))
		}
		if !colHasInk(eight, 0) && !colHasInk(eight, 1) {
			t.Fatalf("8 must light the left verticals:\n%s", Render("8", 4))
		}
		if !colHasInk(eight, last) && !colHasInk(eight, last-1) {
			t.Fatalf("8 must light the right verticals:\n%s", Render("8", 4))
		}
		if !hasInk(eight[len(eight)/2]) {
			t.Fatalf("8 must light the G bar:\n%s", Render("8", 4))
		}
		one := Render("1", 4)
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
		out := Render("?", 4)
		if strings.ContainsAny(out, "█▀▄") {
			t.Fatalf("unknown rune must not invent segments:\n%s", out)
		}
	})
}

func TestRenderPurity(t *testing.T) {
	t.Run("happy: Render is deterministic", func(t *testing.T) {
		if Render("APOLLO", 3) != Render("APOLLO", 3) {
			t.Fatal("Render must be pure")
		}
	})
	t.Run("unhappy: lowercase input is not mutated", func(t *testing.T) {
		in := "hello"
		_ = Render(in, 2)
		if in != "hello" {
			t.Fatal("Render must not mutate its input")
		}
	})
}
