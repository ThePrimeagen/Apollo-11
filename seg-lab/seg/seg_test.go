package seg

// Tests written FIRST. The component contract: a terminal segmented-character
// viewer. Unicode ships only ten segmented digits (U+1FBF0–U+1FBF9). Letters
// have no codepoints, so they are composed: 7-segment (limited alphabet) and
// 14-segment (full A–Z). Render is pure, fixed per-glyph footprint. Happy +
// unhappy throughout.

import (
	"strings"
	"testing"
	"unicode"
)

func glyphWidth(rows []string) int {
	if len(rows) == 0 {
		return 0
	}
	w := len([]rune(rows[0]))
	for _, r := range rows {
		if n := len([]rune(r)); n != w {
			return -1
		}
	}
	return w
}

func allSpaces(rows []string) bool {
	for _, row := range rows {
		if strings.TrimSpace(row) != "" {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Unicode segmented digits — the only official codepoints
// ---------------------------------------------------------------------------

func TestUnicodeDigits(t *testing.T) {
	t.Run("happy: 0-9 map onto U+1FBF0 through U+1FBF9", func(t *testing.T) {
		for i, r := range "0123456789" {
			got, ok := UnicodeDigit(r)
			want := rune(0x1FBF0 + i)
			if !ok {
				t.Fatalf("digit %q must be a real Unicode segmented digit", r)
			}
			if got != want {
				t.Fatalf("digit %q: got U+%04X, want U+%04X", r, got, want)
			}
		}
	})
	t.Run("unhappy: letters have no segmented Unicode codepoint", func(t *testing.T) {
		for _, r := range "ABCxyzHELLO" {
			if _, ok := UnicodeDigit(r); ok {
				t.Fatalf("%q must not pretend to be a Unicode segmented digit", r)
			}
		}
	})
}

func TestUnicodeRender(t *testing.T) {
	t.Run("happy: a digit string is the ten official glyphs", func(t *testing.T) {
		out := Render("0123456789", StyleUnicode)
		plain := stripANSI(out)
		if strings.Contains(plain, "\n") {
			t.Fatal("Unicode digits are one cell each — a single row")
		}
		got := []rune(strings.TrimRight(plain, " "))
		if len(got) != 10 {
			t.Fatalf("want 10 segmented digits, got %d %q", len(got), plain)
		}
		for i, r := range got {
			if r != rune(0x1FBF0+i) {
				t.Fatalf("col %d: U+%04X, want U+%04X", i, r, 0x1FBF0+i)
			}
		}
	})
	t.Run("unhappy: letters in Unicode style stay blank, not ASCII lookalikes", func(t *testing.T) {
		out := stripANSI(Render("HELLO", StyleUnicode))
		for _, r := range out {
			if unicode.IsLetter(r) {
				t.Fatalf("Unicode style must not fall back to Latin letters, got %q", out)
			}
			if r >= 0x1FBF0 && r <= 0x1FBF9 {
				t.Fatalf("letters must not render as segmented digits, got %q", out)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// 7-segment composed glyphs (digits + the letters that fit)
// ---------------------------------------------------------------------------

func TestSevenSegment(t *testing.T) {
	t.Run("happy: distinctive digit and letter shapes render", func(t *testing.T) {
		rows8, ok := Seven('8')
		if !ok {
			t.Fatal("digit 8 must be supported")
		}
		if rows8[0] != " _ " || rows8[1] != "|_|" || rows8[2] != "|_|" {
			t.Fatalf("digit 8 wrong: %q", rows8)
		}
		rowsA, ok := Seven('A')
		if !ok {
			t.Fatal("A is a classic 7-segment letter")
		}
		body := rowsA[0] + rowsA[1] + rowsA[2]
		if !strings.Contains(rowsA[0], "_") || !strings.Contains(rowsA[1], "_") {
			t.Fatalf("A must light the top and middle bars: %q", rowsA)
		}
		if strings.Count(body, "|") < 4 {
			t.Fatalf("A must light both verticals on both rows: %q", rowsA)
		}
		rowsE, ok := Seven('E')
		if !ok || !strings.Contains(rowsE[1]+rowsE[2], "|_") {
			t.Fatalf("E must light the left + both horizontals: %q", rowsE)
		}
	})
	t.Run("unhappy: K, M, V, W, X cannot be drawn on seven segments", func(t *testing.T) {
		for _, r := range "KMVWXkmvwx" {
			rows, ok := Seven(r)
			if ok {
				t.Fatalf("%q must be unsupported on 7-segment, got %q", r, rows)
			}
			if !allSpaces(rows[:]) {
				t.Fatalf("unsupported %q must render blank, got %q", r, rows)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// 14-segment composed glyphs — full alphabet
// ---------------------------------------------------------------------------

func TestFourteenSegment(t *testing.T) {
	t.Run("happy: the letters 7-seg cannot draw are distinct here", func(t *testing.T) {
		need := map[rune][]rune{
			'K': {'│', '╱', '╲'},
			'M': {'│', '╲', '╱'},
			'X': {'╱', '╲'},
			'R': {'│', '╲'},
			'V': {'╱', '╲'},
			'W': {'│', '╱', '╲'},
		}
		for r, marks := range need {
			rows, ok := Fourteen(r)
			if !ok {
				t.Fatalf("%q must be supported on 14-segment", r)
			}
			if glyphWidth(rows[:]) != 5 || len(rows) != 5 {
				t.Fatalf("%q must be a 5×5 glyph, got %q", r, rows)
			}
			body := strings.Join(rows[:], "")
			for _, m := range marks {
				if !strings.ContainsRune(body, m) {
					t.Fatalf("%q missing %q: %q", r, string(m), rows)
				}
			}
		}
		rowsA, ok := Fourteen('A')
		if !ok || !strings.Contains(rowsA[0], "─") || !strings.Contains(rowsA[2], "─") {
			t.Fatalf("A must light the top and middle bars: %q", rowsA)
		}
	})
	t.Run("unhappy: unknown runes degrade to a blank 5×5, no panic", func(t *testing.T) {
		for _, r := range []rune{'@', 'é', '🚀', 0} {
			rows, ok := Fourteen(r)
			if ok {
				t.Fatalf("%q must be unsupported, got %q", r, rows)
			}
			if glyphWidth(rows[:]) != 5 || !allSpaces(rows[:]) {
				t.Fatalf("unsupported %q must be a blank 5×5, got %q", r, rows)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// geometry & purity
// ---------------------------------------------------------------------------

func TestRenderGeometry(t *testing.T) {
	t.Run("happy: each style keeps a fixed per-glyph footprint", func(t *testing.T) {
		cases := []struct {
			style  Style
			text   string
			height int
			gWidth int
			gap    int
		}{
			{StyleSeven, "HI", 3, 3, 1},
			{StyleFourteen, "HI", 5, 5, 1},
			// Official segmented digits sit in adjacent cells; no composed gap.
			{StyleUnicode, "12", 1, 1, 0},
		}
		for _, tc := range cases {
			out := stripANSI(Render(tc.text, tc.style))
			ls := strings.Split(out, "\n")
			if len(ls) != tc.height {
				t.Fatalf("%s %q: %d lines, want %d", tc.style, tc.text, len(ls), tc.height)
			}
			n := len([]rune(tc.text))
			wantW := n*tc.gWidth + (n-1)*tc.gap
			for i, l := range ls {
				if got := len([]rune(l)); got != wantW {
					t.Fatalf("%s %q line %d: width %d, want %d (%q)", tc.style, tc.text, i, got, wantW, l)
				}
			}
		}
	})
	t.Run("unhappy: empty text and unknown style stay blank, no panic", func(t *testing.T) {
		if got := Render("", StyleSeven); got != "" {
			t.Fatalf("empty 7-seg must be empty, got %q", got)
		}
		if got := Render("HELLO", Style(99)); got != "" {
			t.Fatalf("unknown style must render empty, got %q", got)
		}
	})
}

func TestRenderPurity(t *testing.T) {
	t.Run("happy: Render is deterministic", func(t *testing.T) {
		a := Render("APOLLO 11", StyleFourteen)
		b := Render("APOLLO 11", StyleFourteen)
		if a != b {
			t.Fatal("Render must be pure")
		}
	})
	t.Run("unhappy: lowercase input does not mutate the caller's string", func(t *testing.T) {
		in := "hello"
		_ = Render(in, StyleSeven)
		if in != "hello" {
			t.Fatal("Render must not mutate its input")
		}
	})
}

func TestStyleNames(t *testing.T) {
	t.Run("happy: the three styles are named and listed", func(t *testing.T) {
		got := Styles()
		if len(got) != 3 {
			t.Fatalf("want 3 styles, got %v", got)
		}
		if StyleUnicode.String() != "unicode" || StyleSeven.String() != "7-seg" || StyleFourteen.String() != "14-seg" {
			t.Fatalf("style names: %q %q %q", StyleUnicode, StyleSeven, StyleFourteen)
		}
	})
	t.Run("unhappy: an unknown style has an empty name", func(t *testing.T) {
		if Style(99).String() != "" {
			t.Fatalf("unknown style must not invent a name, got %q", Style(99))
		}
	})
}

func stripANSI(s string) string {
	var b strings.Builder
	esc := false
	for _, r := range s {
		if r == '\x1b' {
			esc = true
			continue
		}
		if esc {
			if r == 'm' {
				esc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
