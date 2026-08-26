package sprite

// Tests written FIRST. The LM sprite contract: four sizes (twice-as-big
// 26×10 down through two intermediate steps to the current 13×5), eight
// headings (cardinals + 45° offsets), JSON with independent foreground and
// background colors so the art can be hand-edited, a shade ramp for
// partially filled cells, and ANSI render that emits both 38;5 and 48;5.

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func plain(s string) string { return ansiRE.ReplaceAllString(s, "") }

func TestJSONRoundTrip(t *testing.T) {
	t.Run("happy: glyphs, fg, and bg survive marshal/unmarshal", func(t *testing.T) {
		orig := testAtlas(t)
		raw, err := orig.Marshal()
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		got, err := Unmarshal(raw)
		if err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		want, ok := orig.Frame(Size4, N)
		if !ok {
			t.Fatal("size-4 N missing from original")
		}
		have, ok := got.Frame(Size4, N)
		if !ok {
			t.Fatal("size-4 N missing after round-trip")
		}
		if have.Width != want.Width || have.Height != want.Height {
			t.Fatalf("dims %dx%d, want %dx%d", have.Width, have.Height, want.Width, want.Height)
		}
		for r := 0; r < want.Height; r++ {
			for c := 0; c < want.Width; c++ {
				a, b := want.At(r, c), have.At(r, c)
				if a != b {
					t.Fatalf("cell (%d,%d): got %+v want %+v", r, c, b, a)
				}
			}
		}
		if len(got.Palette) != len(orig.Palette) {
			t.Fatalf("palette len %d, want %d", len(got.Palette), len(orig.Palette))
		}
	})
	t.Run("unhappy: mismatched row length is an error, not a panic", func(t *testing.T) {
		raw := []byte(`{
			"palette": [{"id":"S","name":"silver","fg":252,"bg":-1}],
			"frames": {"4": {"N": {
				"glyphs": ["abc", "ab"],
				"fg":     ["SSS", "SS"],
				"bg":     ["...", ".."]
			}}}
		}`)
		if _, err := Unmarshal(raw); err == nil {
			t.Fatal("short glyph row must fail")
		}
	})
	t.Run("unhappy: truncated JSON is an error", func(t *testing.T) {
		if _, err := Unmarshal([]byte(`{"palette":`)); err == nil {
			t.Fatal("truncated JSON must fail")
		}
	})
	t.Run("unhappy: empty glyphs are rejected", func(t *testing.T) {
		raw := []byte(`{"palette":[],"frames":{"1":{"N":{"glyphs":[],"fg":[],"bg":[]}}}}`)
		if _, err := Unmarshal(raw); err == nil {
			t.Fatal("an empty sprite must fail")
		}
	})
}

func TestTransparentAndColors(t *testing.T) {
	t.Run("happy: a space with no colors is transparent", func(t *testing.T) {
		c := Cell{Ch: ' ', FG: -1, BG: -1}
		if !c.Transparent() {
			t.Fatal("empty cell must be transparent")
		}
	})
	t.Run("happy: a glyph can carry independent fg and bg", func(t *testing.T) {
		c := Cell{Ch: '░', FG: 24, BG: 232}
		if c.Transparent() {
			t.Fatal("a window cell is not transparent")
		}
		if c.FG == c.BG {
			t.Fatal("fg and bg must be independently settable")
		}
	})
	t.Run("unhappy: a space still counts as transparent even if someone stuffed a color", func(t *testing.T) {
		// painting a color onto a blank cell without a glyph should not
		// leave an invisible-but-colored hole; Transparent follows the glyph.
		c := Cell{Ch: ' ', FG: 178, BG: -1}
		if !c.Transparent() {
			t.Fatal("a blank glyph is transparent regardless of leftover color")
		}
	})
}

func TestShadeRamp(t *testing.T) {
	t.Run("happy: Ctrl-A walks an empty cell up the shade ramp to a full block", func(t *testing.T) {
		c := Cell{Ch: ' ', FG: -1, BG: -1}
		seen := map[rune]bool{}
		var last Cell
		for i := 0; i < 8; i++ {
			c = IncrementShade(c)
			seen[c.Ch] = true
			last = c
		}
		if last.Ch != '█' {
			t.Fatalf("top of ramp must be █, got %q", string(last.Ch))
		}
		if last.Transparent() {
			t.Fatal("a fully shaded cell is not transparent")
		}
		if !seen['░'] || !seen['▒'] || !seen['▓'] {
			t.Fatalf("ramp must visit ░▒▓, got %v", seen)
		}
	})
	t.Run("happy: Ctrl-B walks back down to transparent", func(t *testing.T) {
		c := Cell{Ch: '█', FG: 252, BG: -1}
		for i := 0; i < 8; i++ {
			c = DecrementShade(c)
		}
		if !c.Transparent() || c.Ch != ' ' {
			t.Fatalf("bottom of ramp must be a transparent space, got %+v", c)
		}
	})
	t.Run("happy: incrementing a geometric cell keeps it in the ramp without panicking", func(t *testing.T) {
		c := IncrementShade(Cell{Ch: '▟', FG: 178, BG: -1})
		if c.Ch == 0 {
			t.Fatal("increment must produce a rune")
		}
	})
	t.Run("unhappy: decrementing empty stays empty", func(t *testing.T) {
		c := DecrementShade(Cell{Ch: ' ', FG: -1, BG: -1})
		if c.Ch != ' ' || !c.Transparent() {
			t.Fatalf("empty decrement must stay empty, got %+v", c)
		}
	})
}

func TestRenderANSI(t *testing.T) {
	t.Run("happy: render emits both fg and bg 256-color sequences", func(t *testing.T) {
		sp := New(2, 1)
		sp.Set(0, 0, Cell{Ch: '█', FG: 178, BG: 52})
		sp.Set(0, 1, Cell{Ch: '░', FG: 24, BG: 232})
		out := Render(sp)
		for _, code := range []string{"38;5;178", "48;5;52", "38;5;24", "48;5;232"} {
			if !strings.Contains(out, code) {
				t.Fatalf("render missing %s in %q", code, out)
			}
		}
		if plain(out) != "█░" {
			t.Fatalf("glyphs: %q", plain(out))
		}
	})
	t.Run("happy: a transparent cell is a space with reset, not a leftover color", func(t *testing.T) {
		sp := New(2, 1)
		sp.Set(0, 0, Cell{Ch: '█', FG: 252, BG: -1})
		out := Render(sp)
		if !strings.Contains(out, "\x1b[0m") {
			t.Fatal("transparent cells must reset SGR")
		}
		if got := []rune(plain(out)); len(got) != 2 || got[1] != ' ' {
			t.Fatalf("expected a trailing space, got %q", plain(out))
		}
	})
	t.Run("unhappy: rendering an empty sprite still returns a blank line, not panic", func(t *testing.T) {
		sp := New(0, 0)
		if Render(sp) != "" {
			t.Fatalf("empty sprite should render empty, got %q", Render(sp))
		}
	})
}

func TestCustomColorsSurviveWriteFile(t *testing.T) {
	t.Run("happy: an off-palette cell keeps glyph and colors on disk", func(t *testing.T) {
		a := &Atlas{Palette: append([]PaletteEntry(nil), DefaultPalette...)}
		sp := New(13, 5)
		want := Cell{Ch: 'Ω', FG: 123, BG: 45}
		sp.Set(1, 2, want)
		a.SetFrame(Size1, N, sp)
		path := t.TempDir() + "/lm.json"
		if err := a.WriteFile(path); err != nil {
			t.Fatal(err)
		}
		loaded, err := LoadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got := loaded.MustFrame(Size1, N).At(1, 2)
		if got != want {
			t.Fatalf("disk cell %+v, want %+v", got, want)
		}
	})
	t.Run("unhappy: a space does not invent a color on reload", func(t *testing.T) {
		a := &Atlas{Palette: append([]PaletteEntry(nil), DefaultPalette...)}
		sp := New(13, 5)
		sp.Set(0, 0, Cell{Ch: ' ', FG: 123, BG: 45})
		a.SetFrame(Size1, N, sp)
		path := t.TempDir() + "/lm.json"
		if err := a.WriteFile(path); err != nil {
			t.Fatal(err)
		}
		loaded, err := LoadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got := loaded.MustFrame(Size1, N).At(0, 0)
		if !got.Transparent() || got.FG != -1 || got.BG != -1 {
			t.Fatalf("blank cell must stay blank, got %+v", got)
		}
	})
}

func TestPaletteJSONIsEditable(t *testing.T) {
	t.Run("happy: JSON keeps the lander as rows of characters a human can edit", func(t *testing.T) {
		raw, err := testAtlas(t).Marshal()
		if err != nil {
			t.Fatal(err)
		}
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(raw, &probe); err != nil {
			t.Fatal(err)
		}
		if _, ok := probe["palette"]; !ok {
			t.Fatal("JSON must have a palette")
		}
		if _, ok := probe["frames"]; !ok {
			t.Fatal("JSON must have frames")
		}
		if !bytes.Contains(raw, []byte("glyphs")) {
			t.Fatal("JSON must expose glyphs as editable strings")
		}
	})
}

// testAtlas is a small fixture built only from palette materials, so
// the generic JSON/geometry contract never depends on the LM art (that
// contract lives with the lander component now).
func testAtlas(t *testing.T) *Atlas {
	t.Helper()
	a := &Atlas{Palette: append([]PaletteEntry(nil), DefaultPalette...)}
	for _, sz := range Sizes {
		w, h := sz.Dim()
		sp := New(w, h)
		sp.Set(0, 0, Cell{Ch: '█', FG: 178, BG: 94})
		sp.Set(1, 1, Cell{Ch: '░', FG: 24, BG: 232})
		sp.Set(h-1, w-1, Cell{Ch: '▓', FG: 252, BG: -1})
		for _, hd := range Headings {
			a.SetFrame(sz, hd, sp)
		}
	}
	return a
}
