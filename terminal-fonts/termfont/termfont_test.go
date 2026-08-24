package termfont

// Tests written FIRST. The component contract: a multi-height terminal
// banner font, segment-display flavored. Render(height, text) returns a
// row-major byte buffer plus the width of each row, so callers can blit
// rows anywhere inside their own frame data. Height 1 is the plain
// terminal font (the characters themselves); heights 2-5 are ASCII art;
// anything outside 1..5 is an error. Happy + unhappy throughout.

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// rowsOf slices a Render buffer into its rows for readable assertions.
func rowsOf(t *testing.T, buf []byte, height, width int) []string {
	t.Helper()
	if len(buf) != height*width {
		t.Fatalf("buffer is %d bytes, want height*width = %d*%d = %d", len(buf), height, width, height*width)
	}
	rows := make([]string, height)
	for r := 0; r < height; r++ {
		rows[r] = string(buf[r*width : (r+1)*width])
	}
	return rows
}

// ---------------------------------------------------------------------------
// height validation — 1..5 render, everything else is an error
// ---------------------------------------------------------------------------

func TestRenderHeightValidation(t *testing.T) {
	t.Run("happy: every height from 1 through 5 renders", func(t *testing.T) {
		for h := MinHeight; h <= MaxHeight; h++ {
			buf, width, err := Render(h, "A")
			if err != nil {
				t.Fatalf("height %d must render, got error: %v", h, err)
			}
			if width <= 0 {
				t.Fatalf("height %d: width must be positive, got %d", h, width)
			}
			if len(buf) != h*width {
				t.Fatalf("height %d: buffer %d bytes, want %d*%d = %d", h, len(buf), h, width, h*width)
			}
		}
	})
	t.Run("unhappy: heights beyond 5 (and below 1) are errors", func(t *testing.T) {
		for _, h := range []int{0, -1, 6, 42} {
			buf, width, err := Render(h, "A")
			if err == nil {
				t.Fatalf("height %d must be rejected", h)
			}
			if !errors.Is(err, ErrInvalidHeight) {
				t.Fatalf("height %d: error must wrap ErrInvalidHeight, got %v", h, err)
			}
			if buf != nil || width != 0 {
				t.Fatalf("height %d: failed render must return nil buffer and zero width, got %d bytes width %d", h, len(buf), width)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// height 1 — just the terminal font: the characters themselves
// ---------------------------------------------------------------------------

func TestRenderHeightOne(t *testing.T) {
	t.Run("happy: the text passes through byte-for-byte, case preserved", func(t *testing.T) {
		in := "Hello, World! 42 (a/b)"
		buf, width, err := Render(1, in)
		if err != nil {
			t.Fatalf("printable ASCII must pass through, got error: %v", err)
		}
		if width != len(in) {
			t.Fatalf("width %d, want %d", width, len(in))
		}
		if string(buf) != in {
			t.Fatalf("height 1 must be the characters themselves: %q, want %q", buf, in)
		}
		lower, _, err := Render(1, "abc")
		if err != nil || string(lower) != "abc" {
			t.Fatalf("height 1 must not fold case: got %q, %v", lower, err)
		}
	})
	t.Run("unhappy: control bytes and non-ASCII runes are rejected", func(t *testing.T) {
		for _, in := range []string{"caf\u00e9", "a\tb", "a\nb", "rocket \U0001F680"} {
			buf, width, err := Render(1, in)
			if err == nil {
				t.Fatalf("%q must be rejected at height 1", in)
			}
			if !errors.Is(err, ErrUnsupportedRune) {
				t.Fatalf("%q: error must wrap ErrUnsupportedRune, got %v", in, err)
			}
			if buf != nil || width != 0 {
				t.Fatalf("%q: failed render must return nil buffer and zero width", in)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// buffer geometry — row-major, fixed width, one blank column between glyphs
// ---------------------------------------------------------------------------

func TestRenderBufferGeometry(t *testing.T) {
	t.Run("happy: len(buf) == height*width and glyphs are gap-separated", func(t *testing.T) {
		for h := 2; h <= MaxHeight; h++ {
			for _, text := range []string{"AB", "APOLLO 11", "GO 1.22!"} {
				buf, width, err := Render(h, text)
				if err != nil {
					t.Fatalf("height %d %q: %v", h, text, err)
				}
				rows := rowsOf(t, buf, h, width)
				for i, row := range rows {
					if len(row) != width {
						t.Fatalf("height %d %q row %d: %d bytes, want %d", h, text, i, len(row), width)
					}
				}
				// Width is the per-rune widths plus a one-column gap
				// between consecutive glyphs.
				want := 0
				for i, r := range []rune(text) {
					_, w, err := Render(h, string(r))
					if err != nil {
						t.Fatalf("height %d rune %q: %v", h, r, err)
					}
					if i > 0 {
						want++
					}
					want += w
				}
				if width != want {
					t.Fatalf("height %d %q: width %d, want %d (glyphs + gaps)", h, text, width, want)
				}
				for i, b := range buf {
					if b < ' ' || b > '~' {
						t.Fatalf("height %d %q byte %d: 0x%02x is not printable ASCII", h, text, i, b)
					}
				}
			}
		}
	})
	t.Run("happy: Render is pure — identical calls yield identical buffers", func(t *testing.T) {
		a, wa, _ := Render(4, "APOLLO")
		b, wb, _ := Render(4, "APOLLO")
		if wa != wb || !bytes.Equal(a, b) {
			t.Fatal("Render must be deterministic")
		}
	})
	t.Run("unhappy: an unsupported rune mid-string fails the whole render", func(t *testing.T) {
		for h := 2; h <= MaxHeight; h++ {
			buf, width, err := Render(h, "A~B")
			if err == nil {
				t.Fatalf("height %d: '~' must be unsupported in art heights", h)
			}
			if !errors.Is(err, ErrUnsupportedRune) {
				t.Fatalf("height %d: error must wrap ErrUnsupportedRune, got %v", h, err)
			}
			if buf != nil || width != 0 {
				t.Fatalf("height %d: failed render must return nil buffer and zero width", h)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// charset — A-Z, 0-9, space, punctuation; lowercase folds in art heights
// ---------------------------------------------------------------------------

func TestRenderCharsetCoverage(t *testing.T) {
	t.Run("happy: the published charset renders at every art height", func(t *testing.T) {
		for _, must := range []string{"ABCDEFGHIJKLMNOPQRSTUVWXYZ", "0123456789", " ", ".,:;!?'\"-+=()/\\_"} {
			for _, r := range must {
				if !strings.ContainsRune(Charset, r) {
					t.Fatalf("Charset must include %q", r)
				}
			}
		}
		for h := 2; h <= MaxHeight; h++ {
			for _, r := range Charset {
				buf, width, err := Render(h, string(r))
				if err != nil {
					t.Fatalf("height %d: charset rune %q must render: %v", h, r, err)
				}
				if width <= 0 || len(buf) != h*width {
					t.Fatalf("height %d: rune %q bad geometry (width %d, %d bytes)", h, r, width, len(buf))
				}
			}
		}
	})
	t.Run("happy: lowercase folds to uppercase in art heights only", func(t *testing.T) {
		for h := 2; h <= MaxHeight; h++ {
			lo, wlo, err1 := Render(h, "apollo")
			up, wup, err2 := Render(h, "APOLLO")
			if err1 != nil || err2 != nil {
				t.Fatalf("height %d: fold render errors: %v %v", h, err1, err2)
			}
			if wlo != wup || !bytes.Equal(lo, up) {
				t.Fatalf("height %d: lowercase must render as uppercase art", h)
			}
		}
	})
	t.Run("unhappy: junk runes are rejected at every art height", func(t *testing.T) {
		for h := 2; h <= MaxHeight; h++ {
			for _, r := range []rune{'~', '@', '#', '\u00e9', '\U0001F680', '\t'} {
				if _, _, err := Render(h, string(r)); !errors.Is(err, ErrUnsupportedRune) {
					t.Fatalf("height %d: %q must fail with ErrUnsupportedRune, got %v", h, r, err)
				}
			}
		}
	})
}

// ---------------------------------------------------------------------------
// known glyphs — pin the segment-style art so it cannot silently drift
// ---------------------------------------------------------------------------

func TestRenderKnownGlyphs(t *testing.T) {
	t.Run("happy: the letter A is segment art at every art height", func(t *testing.T) {
		want := map[int][]string{
			2: {" /\\ ", "/--\\"},
			3: {" _ ", "|_|", "| |"},
			4: {" _ ", "| |", "|_|", "| |"},
			5: {" _ ", "| |", "|_|", "| |", "| |"},
		}
		for h, rows := range want {
			buf, width, err := Render(h, "A")
			if err != nil {
				t.Fatalf("height %d A: %v", h, err)
			}
			got := rowsOf(t, buf, h, width)
			for i := range rows {
				if got[i] != rows[i] {
					t.Fatalf("height %d A row %d: %q, want %q\nfull: %q", h, i, got[i], rows[i], got)
				}
			}
		}
	})
	t.Run("happy: ABC at height 3 composes glyphs with one-column gaps", func(t *testing.T) {
		buf, width, err := Render(3, "ABC")
		if err != nil {
			t.Fatalf("ABC: %v", err)
		}
		want := []string{
			" _   _   _ ",
			"|_| |_) |  ",
			"| | |_) |_ ",
		}
		got := rowsOf(t, buf, 3, width)
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("ABC row %d: %q, want %q", i, got[i], want[i])
			}
		}
	})
	t.Run("happy: zero is slashed so it never reads as the letter O", func(t *testing.T) {
		for h := 2; h <= MaxHeight; h++ {
			zero, wz, _ := Render(h, "0")
			oh, wo, _ := Render(h, "O")
			if wz == wo && bytes.Equal(zero, oh) {
				t.Fatalf("height %d: '0' and 'O' must be distinguishable", h)
			}
		}
	})
	t.Run("happy: empty text renders an empty zero-width buffer, no error", func(t *testing.T) {
		for h := MinHeight; h <= MaxHeight; h++ {
			buf, width, err := Render(h, "")
			if err != nil {
				t.Fatalf("height %d: empty text must not error: %v", h, err)
			}
			if width != 0 || len(buf) != 0 {
				t.Fatalf("height %d: empty text must be zero-width, got width %d, %d bytes", h, width, len(buf))
			}
		}
	})
	t.Run("unhappy: the error names the offending rune and its position", func(t *testing.T) {
		_, _, err := Render(3, "AB~")
		if !errors.Is(err, ErrUnsupportedRune) {
			t.Fatalf("want ErrUnsupportedRune, got %v", err)
		}
		msg := err.Error()
		if !strings.Contains(msg, "~") || !strings.Contains(msg, "2") {
			t.Fatalf("error must name the rune and its index: %q", msg)
		}
	})
}

// ---------------------------------------------------------------------------
// Lines — the convenience view over the same buffer contract
// ---------------------------------------------------------------------------

func TestLines(t *testing.T) {
	t.Run("happy: Lines mirrors the buffer rows exactly", func(t *testing.T) {
		lines, err := Lines(3, "A")
		if err != nil {
			t.Fatalf("Lines: %v", err)
		}
		want := []string{" _ ", "|_|", "| |"}
		if len(lines) != len(want) {
			t.Fatalf("got %d lines, want %d", len(lines), len(want))
		}
		for i := range want {
			if lines[i] != want[i] {
				t.Fatalf("line %d: %q, want %q", i, lines[i], want[i])
			}
		}
		one, err := Lines(1, "HI")
		if err != nil || len(one) != 1 || one[0] != "HI" {
			t.Fatalf("height 1 Lines must be the text itself: %q, %v", one, err)
		}
	})
	t.Run("unhappy: invalid height and unsupported runes propagate", func(t *testing.T) {
		if _, err := Lines(0, "A"); !errors.Is(err, ErrInvalidHeight) {
			t.Fatalf("height 0 must fail with ErrInvalidHeight, got %v", err)
		}
		if _, err := Lines(6, "A"); !errors.Is(err, ErrInvalidHeight) {
			t.Fatalf("height 6 must fail with ErrInvalidHeight, got %v", err)
		}
		if _, err := Lines(4, "~"); !errors.Is(err, ErrUnsupportedRune) {
			t.Fatalf("unsupported rune must fail with ErrUnsupportedRune, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// glyph table integrity — every table complete, rectangular, ASCII-only
// ---------------------------------------------------------------------------

func TestGlyphTableIntegrity(t *testing.T) {
	t.Run("happy: heights 2-5 ship complete well-formed tables", func(t *testing.T) {
		for h := 2; h <= MaxHeight; h++ {
			set, ok := glyphSets[h]
			if !ok {
				t.Fatalf("height %d must have a glyph table", h)
			}
			if err := validateGlyphSet(h, set); err != nil {
				t.Fatalf("height %d table invalid: %v", h, err)
			}
			for _, r := range Charset {
				if _, ok := set[r]; !ok {
					t.Fatalf("height %d table missing charset rune %q", h, r)
				}
			}
		}
	})
	t.Run("unhappy: malformed tables are caught by validation", func(t *testing.T) {
		bad := []struct {
			name string
			set  map[rune][]string
		}{
			{"wrong row count", map[rune][]string{'A': {" _ ", "|_|"}}},
			{"ragged rows", map[rune][]string{'A': {" _ ", "|_|", "| "}}},
			{"empty glyph", map[rune][]string{'A': {"", "", ""}}},
			{"non-ASCII art", map[rune][]string{'A': {" _ ", "\u2502_\u2502", "| |"}}},
			{"empty table", map[rune][]string{}},
		}
		for _, tc := range bad {
			if err := validateGlyphSet(3, tc.set); err == nil {
				t.Fatalf("%s: validateGlyphSet must reject it", tc.name)
			}
		}
	})
}
