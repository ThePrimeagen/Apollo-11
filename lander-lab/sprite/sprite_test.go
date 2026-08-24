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

func TestAtlasGeometry(t *testing.T) {
	a := testAtlas(t)

	wantDim := map[Size][2]int{
		Size1: {13, 5},
		Size2: {17, 7},
		Size3: {22, 8},
		Size4: {26, 10},
	}
	t.Run("happy: four sizes, eight headings, declared footprints", func(t *testing.T) {
		if len(Sizes) != 4 || len(Headings) != 8 {
			t.Fatalf("need 4 sizes and 8 headings, got %d / %d", len(Sizes), len(Headings))
		}
		for _, sz := range Sizes {
			d, ok := wantDim[sz]
			if !ok {
				t.Fatalf("missing expected dim for size %d", sz)
			}
			for _, h := range Headings {
				sp, ok := a.Frame(sz, h)
				if !ok {
					t.Fatalf("missing frame size=%d heading=%s", sz, h)
				}
				if sp.Width != d[0] || sp.Height != d[1] {
					t.Fatalf("size %d %s: %dx%d, want %dx%d", sz, h, sp.Width, sp.Height, d[0], d[1])
				}
				if err := sp.Validate(); err != nil {
					t.Fatalf("size %d %s: %v", sz, h, err)
				}
			}
		}
	})
	t.Run("happy: size 4 is twice size 1", func(t *testing.T) {
		s1, _ := a.Frame(Size1, N)
		s4, _ := a.Frame(Size4, N)
		if s4.Width != s1.Width*2 || s4.Height != s1.Height*2 {
			t.Fatalf("size4 %dx%d is not 2× size1 %dx%d", s4.Width, s4.Height, s1.Width, s1.Height)
		}
	})
	t.Run("unhappy: a missing frame is reported, not a zero sprite", func(t *testing.T) {
		empty := &Atlas{}
		if _, ok := empty.Frame(Size4, N); ok {
			t.Fatal("empty atlas must not claim to have frames")
		}
	})
}

// The current lander-lab silhouettes, copied as the size-1 contract so the
// shrink animation lands on the art that already ships in the descent view.
var (
	s1N = [5]string{
		"    ▗▛◣▖     ",
		"   ▟░◣╲▜▙    ",
		"  ▟▓████▓▙   ",
		"╱ ◢▔▔▟▄▙▔▔◣ ╲",
		"▁ ▁  ~~~  ▁ ▁",
	}
	s1NE = [5]string{
		"        ▗▛◣▖ ",
		"▁ ╲   ▟░╲▜▙  ",
		"  ╲◢▟▓██▓▙   ",
		"   ◥▜▓▓▛  ╲  ",
		" ~~~▜▙    ╲▁ ",
	}
	s1E = [5]string{
		"   ▁╲   ╱▁   ",
		"    ╲▟▓▓▙▛◣▖ ",
		"   ◢▟▓▓▙░██▜▙",
		"  ~~◥▜▓▛╲▝◤  ",
		"~~ ▁╱   ╲▁   ",
	}
)

func TestSize1MatchesCurrentLander(t *testing.T) {
	a := testAtlas(t)
	t.Run("happy: size-1 N/NE/E are the current Vertical/Tilted/Horizontal sprites", func(t *testing.T) {
		cases := []struct {
			h    Heading
			want [5]string
		}{{N, s1N}, {NE, s1NE}, {E, s1E}}
		for _, tc := range cases {
			sp, ok := a.Frame(Size1, tc.h)
			if !ok {
				t.Fatalf("%s missing", tc.h)
			}
			got := sp.GlyphRows()
			if len(got) != 5 {
				t.Fatalf("%s: %d rows, want 5", tc.h, len(got))
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("%s row %d:\n got %q\nwant %q", tc.h, i, got[i], tc.want[i])
				}
			}
		}
	})
	t.Run("unhappy: a heading that is only a flip still has to fill the 13×5 box", func(t *testing.T) {
		sp, ok := a.Frame(Size1, W)
		if !ok {
			t.Fatal("size-1 W missing")
		}
		if sp.Width != 13 || sp.Height != 5 {
			t.Fatalf("flipped heading must keep the size-1 box, got %dx%d", sp.Width, sp.Height)
		}
	})
}

func TestLargerSizesLookLikeTheLander(t *testing.T) {
	a := testAtlas(t)
	t.Run("happy: every N size has a cabin, a wide descent stage, legs, and a plume", func(t *testing.T) {
		for _, sz := range Sizes {
			sp, _ := a.Frame(sz, N)
			g := strings.Join(sp.GlyphRows(), "\n")
			if !strings.Contains(g, "█") {
				t.Fatalf("size %d N: descent stage (█) missing\n%s", sz, g)
			}
			if !strings.ContainsAny(g, "▁▂") {
				t.Fatalf("size %d N: footpads missing\n%s", sz, g)
			}
			if strings.Count(g, "~")+strings.Count(g, "≈") < 3 {
				t.Fatalf("size %d N: plume too thin\n%s", sz, g)
			}
			if !strings.ContainsAny(g, "╱╲") {
				t.Fatalf("size %d N: legs missing\n%s", sz, g)
			}
		}
	})
	t.Run("happy: larger N frames actually occupy more cells", func(t *testing.T) {
		prev := 0
		for _, sz := range Sizes {
			sp, _ := a.Frame(sz, N)
			n := 0
			for r := 0; r < sp.Height; r++ {
				for c := 0; c < sp.Width; c++ {
					if !sp.At(r, c).Transparent() {
						n++
					}
				}
			}
			if n <= prev {
				t.Fatalf("size %d N filled %d cells, not more than size below (%d)", sz, n, prev)
			}
			prev = n
		}
	})
	t.Run("unhappy: a fully transparent frame is not a lander", func(t *testing.T) {
		sp, _ := a.Frame(Size4, N)
		allEmpty := true
		for r := 0; r < sp.Height; r++ {
			for c := 0; c < sp.Width; c++ {
				if !sp.At(r, c).Transparent() {
					allEmpty = false
				}
			}
		}
		if allEmpty {
			t.Fatal("size-4 N is blank")
		}
	})
}

func TestEightHeadings(t *testing.T) {
	a := testAtlas(t)
	t.Run("happy: eight headings at size 4 are distinct silhouettes", func(t *testing.T) {
		seen := map[string]Heading{}
		for _, h := range Headings {
			sp, _ := a.Frame(Size4, h)
			key := strings.Join(sp.GlyphRows(), "\n")
			if other, ok := seen[key]; ok {
				t.Fatalf("headings %s and %s rendered identically", h, other)
			}
			seen[key] = h
		}
	})
	t.Run("happy: N plume sits in the bottom half; S plume sits in the top half", func(t *testing.T) {
		n, _ := a.Frame(Size4, N)
		s, _ := a.Frame(Size4, S)
		if plumeHalf(n) != "bottom" {
			t.Fatalf("N plume should be in the bottom half, got %s\n%s", plumeHalf(n), strings.Join(n.GlyphRows(), "\n"))
		}
		if plumeHalf(s) != "top" {
			t.Fatalf("S plume should be in the top half, got %s\n%s", plumeHalf(s), strings.Join(s.GlyphRows(), "\n"))
		}
	})
	t.Run("happy: E plume sits on the left (engine-aft), W on the right", func(t *testing.T) {
		e, _ := a.Frame(Size4, E)
		w, _ := a.Frame(Size4, W)
		if plumeSide(e) != "left" {
			t.Fatalf("E plume should be on the left, got %s\n%s", plumeSide(e), strings.Join(e.GlyphRows(), "\n"))
		}
		if plumeSide(w) != "right" {
			t.Fatalf("W plume should be on the right, got %s\n%s", plumeSide(w), strings.Join(w.GlyphRows(), "\n"))
		}
	})
	t.Run("unhappy: unknown heading does not match a real one", func(t *testing.T) {
		if _, ok := a.Frame(Size4, Heading("X")); ok {
			t.Fatal("bogus heading must not resolve")
		}
	})
}

func plumeHalf(sp Sprite) string {
	top, bot := 0, 0
	mid := sp.Height / 2
	for r := 0; r < sp.Height; r++ {
		for _, ch := range sp.GlyphRows()[r] {
			if ch == '~' || ch == '≈' {
				if r < mid {
					top++
				} else {
					bot++
				}
			}
		}
	}
	if top == 0 && bot == 0 {
		return "none"
	}
	if top > bot {
		return "top"
	}
	return "bottom"
}

func plumeSide(sp Sprite) string {
	left, right := 0, 0
	mid := sp.Width / 2
	for _, row := range sp.GlyphRows() {
		for c, ch := range []rune(row) {
			if ch == '~' || ch == '≈' {
				if c < mid {
					left++
				} else {
					right++
				}
			}
		}
	}
	if left == 0 && right == 0 {
		return "none"
	}
	if left > right {
		return "left"
	}
	return "right"
}

func TestShrinkSequence(t *testing.T) {
	a := testAtlas(t)
	t.Run("happy: shrink is four frames, largest to current", func(t *testing.T) {
		frames := ShrinkSequence(a, N)
		if len(frames) != 4 {
			t.Fatalf("shrink must have 4 frames, got %d", len(frames))
		}
		want := []Size{Size4, Size3, Size2, Size1}
		for i, sp := range frames {
			orig, _ := a.Frame(want[i], N)
			if strings.Join(sp.GlyphRows(), "\n") != strings.Join(orig.GlyphRows(), "\n") {
				t.Fatalf("frame %d is not size %d N", i, want[i])
			}
		}
		if frames[0].Width <= frames[3].Width || frames[0].Height <= frames[3].Height {
			t.Fatal("first shrink frame must be bigger than the last")
		}
	})
	t.Run("unhappy: shrink of a heading with no frames is empty, not a panic", func(t *testing.T) {
		frames := ShrinkSequence(&Atlas{}, N)
		if len(frames) != 0 {
			t.Fatalf("empty atlas shrink must be empty, got %d", len(frames))
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

func TestShippedJSON(t *testing.T) {
	t.Run("happy: sprites/lm.json loads as a full 4×8 atlas", func(t *testing.T) {
		a, err := LoadFile("../sprites/lm.json")
		if err != nil {
			t.Fatal(err)
		}
		for _, sz := range Sizes {
			for _, h := range Headings {
				if _, ok := a.Frame(sz, h); !ok {
					t.Fatalf("shipped JSON missing size %d %s", sz, h)
				}
			}
		}
		got := a.MustFrame(Size1, N).GlyphRows()
		for i, want := range s1N {
			if got[i] != want {
				t.Fatalf("shipped size-1 N drifted from the current lander row %d", i)
			}
		}
	})
	t.Run("unhappy: a missing file is an error", func(t *testing.T) {
		if _, err := LoadFile("no-such-atlas.json"); err == nil {
			t.Fatal("missing file must fail")
		}
	})
}

func testAtlas(t *testing.T) *Atlas {
	t.Helper()
	a := Default()
	if a == nil {
		t.Fatal("Default() returned nil")
	}
	return a
}
