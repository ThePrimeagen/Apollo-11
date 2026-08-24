package termfont

// Tests written FIRST. The component contract: seven-segment numbers at
// variable height. RenderSeven mirrors the Render buffer contract but
// draws true seven-segment digits — straight "_" and "|" segments only,
// no diagonals, no slashed zero. The charset is what a real display can
// show: 0-9, space, and the clock/calculator marks "." ":" "-". Height 1
// is regular terminal printing of those characters; heights 2-5 scale
// the segments; anything else errors. Happy + unhappy throughout.

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// height validation — 1..5 render, everything else is an error
// ---------------------------------------------------------------------------

func TestRenderSevenHeightValidation(t *testing.T) {
	t.Run("happy: every height from 1 through 5 renders a digit", func(t *testing.T) {
		for h := MinHeight; h <= MaxHeight; h++ {
			buf, width, err := RenderSeven(h, "8")
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
			buf, width, err := RenderSeven(h, "8")
			if err == nil {
				t.Fatalf("height %d must be rejected", h)
			}
			if !errors.Is(err, ErrInvalidHeight) {
				t.Fatalf("height %d: error must wrap ErrInvalidHeight, got %v", h, err)
			}
			if buf != nil || width != 0 {
				t.Fatalf("height %d: failed render must return nil buffer and zero width", h)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// height 1 — regular terminal printing, but still a numeric display
// ---------------------------------------------------------------------------

func TestRenderSevenHeightOne(t *testing.T) {
	t.Run("happy: display characters pass through byte-for-byte", func(t *testing.T) {
		for _, in := range []string{"1234567890", "12:34 -5.6"} {
			buf, width, err := RenderSeven(1, in)
			if err != nil {
				t.Fatalf("%q must pass through at height 1: %v", in, err)
			}
			if width != len(in) || string(buf) != in {
				t.Fatalf("height 1 must be the characters themselves: %q width %d, want %q width %d", buf, width, in, len(in))
			}
		}
	})
	t.Run("unhappy: a seven-segment display shows no letters, even at height 1", func(t *testing.T) {
		for _, in := range []string{"ABC", "12caf\u00e9", "1\t2"} {
			buf, width, err := RenderSeven(1, in)
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
// buffer geometry — same row-major contract as Render, fixed digit cells
// ---------------------------------------------------------------------------

func TestRenderSevenGeometry(t *testing.T) {
	t.Run("happy: len(buf) == height*width with one-column gaps", func(t *testing.T) {
		for h := 2; h <= MaxHeight; h++ {
			for _, text := range []string{"10", "12:34", "-9.8 76"} {
				buf, width, err := RenderSeven(h, text)
				if err != nil {
					t.Fatalf("height %d %q: %v", h, text, err)
				}
				rows := rowsOf(t, buf, h, width)
				for i, row := range rows {
					if len(row) != width {
						t.Fatalf("height %d %q row %d: %d bytes, want %d", h, text, i, len(row), width)
					}
				}
				want := 0
				for i, r := range []rune(text) {
					_, w, err := RenderSeven(h, string(r))
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
	t.Run("happy: all ten digits share one display cell width", func(t *testing.T) {
		for h := 2; h <= MaxHeight; h++ {
			_, w0, err := RenderSeven(h, "0")
			if err != nil {
				t.Fatalf("height %d '0': %v", h, err)
			}
			for _, r := range "123456789" {
				_, w, err := RenderSeven(h, string(r))
				if err != nil {
					t.Fatalf("height %d %q: %v", h, r, err)
				}
				if w != w0 {
					t.Fatalf("height %d: digit %q is %d wide, want the shared cell width %d", h, r, w, w0)
				}
			}
		}
	})
	t.Run("happy: the height 5 display cell is one column wider than height 4", func(t *testing.T) {
		_, w4, err := RenderSeven(4, "0")
		if err != nil {
			t.Fatalf("height 4 '0': %v", err)
		}
		_, w5, err := RenderSeven(5, "0")
		if err != nil {
			t.Fatalf("height 5 '0': %v", err)
		}
		if w5 != w4+1 {
			t.Fatalf("height 5 cell must be one wider than height 4 (%d), got %d", w4, w5)
		}
	})
	t.Run("happy: RenderSeven is pure — identical calls yield identical buffers", func(t *testing.T) {
		a, wa, _ := RenderSeven(5, "1969")
		b, wb, _ := RenderSeven(5, "1969")
		if wa != wb || !bytes.Equal(a, b) {
			t.Fatal("RenderSeven must be deterministic")
		}
	})
	t.Run("unhappy: a letter mid-string fails at every height, 1 included", func(t *testing.T) {
		for h := MinHeight; h <= MaxHeight; h++ {
			buf, width, err := RenderSeven(h, "1A2")
			if err == nil {
				t.Fatalf("height %d: letters must be unsupported on seven segments", h)
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
// charset — 0-9, space, and the display marks . : -
// ---------------------------------------------------------------------------

func TestRenderSevenCharset(t *testing.T) {
	t.Run("happy: the published seven-segment charset renders at 1-5", func(t *testing.T) {
		for _, must := range []string{"0123456789", " ", ".:-"} {
			for _, r := range must {
				if !strings.ContainsRune(SevenCharset, r) {
					t.Fatalf("SevenCharset must include %q", r)
				}
			}
		}
		for h := MinHeight; h <= MaxHeight; h++ {
			for _, r := range SevenCharset {
				buf, width, err := RenderSeven(h, string(r))
				if err != nil {
					t.Fatalf("height %d: charset rune %q must render: %v", h, r, err)
				}
				if width <= 0 || len(buf) != h*width {
					t.Fatalf("height %d: rune %q bad geometry (width %d, %d bytes)", h, r, width, len(buf))
				}
			}
		}
	})
	t.Run("unhappy: banner-only and junk runes are rejected at 1-5", func(t *testing.T) {
		for h := MinHeight; h <= MaxHeight; h++ {
			for _, r := range []rune{'A', 'a', '+', '~', '\u00e9', '\U0001F680'} {
				if _, _, err := RenderSeven(h, string(r)); !errors.Is(err, ErrUnsupportedRune) {
					t.Fatalf("height %d: %q must fail with ErrUnsupportedRune, got %v", h, r, err)
				}
			}
		}
	})
}

// ---------------------------------------------------------------------------
// known glyphs — true seven-segment shapes, pinned
// ---------------------------------------------------------------------------

func TestRenderSevenKnownGlyphs(t *testing.T) {
	t.Run("happy: the digit 8 lights every segment at every art height", func(t *testing.T) {
		want := map[int][]string{
			2: {"|_|", "|_|"},
			3: {" _ ", "|_|", "|_|"},
			4: {" _ ", "| |", "|_|", "|_|"},
			5: {" __ ", "|  |", "|__|", "|  |", "|__|"},
		}
		for h, rows := range want {
			buf, width, err := RenderSeven(h, "8")
			if err != nil {
				t.Fatalf("height %d 8: %v", h, err)
			}
			got := rowsOf(t, buf, h, width)
			for i := range rows {
				if got[i] != rows[i] {
					t.Fatalf("height %d 8 row %d: %q, want %q\nfull: %q", h, i, got[i], rows[i], got)
				}
			}
		}
	})
	t.Run("happy: 10 at height 3 is the canonical seven-segment pair", func(t *testing.T) {
		buf, width, err := RenderSeven(3, "10")
		if err != nil {
			t.Fatalf("10: %v", err)
		}
		want := []string{
			"     _ ",
			"  | | |",
			"  | |_|",
		}
		got := rowsOf(t, buf, 3, width)
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("10 row %d: %q, want %q", i, got[i], want[i])
			}
		}
	})
	t.Run("happy: segments stay straight — no diagonals anywhere", func(t *testing.T) {
		for h := 2; h <= MaxHeight; h++ {
			buf, _, err := RenderSeven(h, "1234567890")
			if err != nil {
				t.Fatalf("height %d: %v", h, err)
			}
			if bytes.ContainsAny(buf, "/\\") {
				t.Fatalf("height %d: seven-segment art must not use diagonals: %q", h, buf)
			}
		}
	})
	t.Run("happy: seven-segment 7 and 0 differ from the slashed banner cuts", func(t *testing.T) {
		for h := 2; h <= MaxHeight; h++ {
			banner, _, err := Render(h, "7")
			if err != nil {
				t.Fatalf("height %d banner 7: %v", h, err)
			}
			if !bytes.ContainsAny(banner, "/") {
				t.Fatalf("height %d: the banner 7 is the slanted cut: %q", h, banner)
			}
			seven, _, err := RenderSeven(h, "7")
			if err != nil {
				t.Fatalf("height %d seven-segment 7: %v", h, err)
			}
			if bytes.ContainsAny(seven, "/") {
				t.Fatalf("height %d: the seven-segment 7 must stay straight: %q", h, seven)
			}
		}
		for h := 3; h <= MaxHeight; h++ {
			banner, wb, _ := Render(h, "0")
			seven, ws, _ := RenderSeven(h, "0")
			if wb == ws && bytes.Equal(banner, seven) {
				t.Fatalf("height %d: banner 0 is slashed, seven-segment 0 is not — they must differ", h)
			}
		}
	})
	t.Run("happy: empty text renders an empty zero-width buffer, no error", func(t *testing.T) {
		for h := MinHeight; h <= MaxHeight; h++ {
			buf, width, err := RenderSeven(h, "")
			if err != nil {
				t.Fatalf("height %d: empty text must not error: %v", h, err)
			}
			if width != 0 || len(buf) != 0 {
				t.Fatalf("height %d: empty text must be zero-width, got width %d, %d bytes", h, width, len(buf))
			}
		}
	})
	t.Run("unhappy: the error names the offending rune and its position", func(t *testing.T) {
		_, _, err := RenderSeven(4, "12X")
		if !errors.Is(err, ErrUnsupportedRune) {
			t.Fatalf("want ErrUnsupportedRune, got %v", err)
		}
		msg := err.Error()
		if !strings.Contains(msg, "X") || !strings.Contains(msg, "2") {
			t.Fatalf("error must name the rune and its index: %q", msg)
		}
	})
}

// ---------------------------------------------------------------------------
// LinesSeven — the convenience view over the same buffer contract
// ---------------------------------------------------------------------------

func TestLinesSeven(t *testing.T) {
	t.Run("happy: LinesSeven mirrors the buffer rows exactly", func(t *testing.T) {
		lines, err := LinesSeven(3, "8")
		if err != nil {
			t.Fatalf("LinesSeven: %v", err)
		}
		want := []string{" _ ", "|_|", "|_|"}
		if len(lines) != len(want) {
			t.Fatalf("got %d lines, want %d", len(lines), len(want))
		}
		for i := range want {
			if lines[i] != want[i] {
				t.Fatalf("line %d: %q, want %q", i, lines[i], want[i])
			}
		}
		one, err := LinesSeven(1, "42")
		if err != nil || len(one) != 1 || one[0] != "42" {
			t.Fatalf("height 1 LinesSeven must be the text itself: %q, %v", one, err)
		}
	})
	t.Run("unhappy: invalid height and unsupported runes propagate", func(t *testing.T) {
		if _, err := LinesSeven(0, "8"); !errors.Is(err, ErrInvalidHeight) {
			t.Fatalf("height 0 must fail with ErrInvalidHeight, got %v", err)
		}
		if _, err := LinesSeven(6, "8"); !errors.Is(err, ErrInvalidHeight) {
			t.Fatalf("height 6 must fail with ErrInvalidHeight, got %v", err)
		}
		if _, err := LinesSeven(3, "A"); !errors.Is(err, ErrUnsupportedRune) {
			t.Fatalf("letters must fail with ErrUnsupportedRune, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// glyph table integrity — complete numeric tables, and nothing more
// ---------------------------------------------------------------------------

func TestSevenGlyphTableIntegrity(t *testing.T) {
	t.Run("happy: heights 2-5 ship complete well-formed digit tables", func(t *testing.T) {
		for h := 2; h <= MaxHeight; h++ {
			set, ok := sevenSets[h]
			if !ok {
				t.Fatalf("height %d must have a seven-segment table", h)
			}
			if err := validateGlyphSet(h, set); err != nil {
				t.Fatalf("height %d table invalid: %v", h, err)
			}
			for _, r := range SevenCharset {
				if _, ok := set[r]; !ok {
					t.Fatalf("height %d table missing charset rune %q", h, r)
				}
			}
		}
	})
	t.Run("unhappy: letters never sneak in, and malformed tables are caught", func(t *testing.T) {
		for h := 2; h <= MaxHeight; h++ {
			for r := 'A'; r <= 'Z'; r++ {
				if _, ok := sevenSets[h][r]; ok {
					t.Fatalf("height %d: %q must not exist on a seven-segment display", h, r)
				}
			}
		}
		if err := validateGlyphSet(3, map[rune][]string{'8': {" _ ", "|_|"}}); err == nil {
			t.Fatal("a two-row glyph in a height-3 table must be rejected")
		}
	})
}
