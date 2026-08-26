package lander

// The LM art contract, moved here with the art itself: the atlas
// geometry, the size-1 silhouettes, the headings, the shrink
// animation, and the shipped JSON. Aliases below keep the tests
// verbatim from their previous home in the sprite package.

import (
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

type (
	Heading = sprite.Heading
	Sprite  = sprite.Sprite
	Atlas   = sprite.Atlas
	Size    = sprite.Size
)

const (
	Size1 = sprite.Size1
	Size2 = sprite.Size2
	Size3 = sprite.Size3
	Size4 = sprite.Size4
	N     = sprite.N
	NE    = sprite.NE
	E     = sprite.E
	SE    = sprite.SE
	S     = sprite.S
	SW    = sprite.SW
	W     = sprite.W
	NW    = sprite.NW
)

var (
	Sizes          = sprite.Sizes
	Headings       = sprite.Headings
	ShrinkSequence = sprite.ShrinkSequence
	LoadFile       = sprite.LoadFile
)

func TestAtlasGeometry(t *testing.T) {
	a := testAtlas(t)

	wantDim := map[Size][2]int{
		Size1: {13, 5},
		Size2: {17, 7},
		Size3: {22, 8},
		Size4: {26, 10},
	}
	t.Run("happy: four sizes, declared footprints, size-4 only N/S/W", func(t *testing.T) {
		if len(Sizes) != 4 || len(Headings) != 8 {
			t.Fatalf("need 4 sizes and 8 headings, got %d / %d", len(Sizes), len(Headings))
		}
		for _, sz := range Sizes {
			d, ok := wantDim[sz]
			if !ok {
				t.Fatalf("missing expected dim for size %d", sz)
			}
			for _, h := range HeadingsFor(sz) {
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
	t.Run("unhappy: size-4 does not keep NE/E/SE/SW/NW", func(t *testing.T) {
		for _, h := range []Heading{NE, E, SE, SW, NW} {
			if _, ok := a.Frame(Size4, h); ok {
				t.Fatalf("size-4 %s must be removed", h)
			}
		}
		for _, h := range []Heading{N, S, W} {
			if _, ok := a.Frame(Size4, h); !ok {
				t.Fatalf("size-4 %s must stay", h)
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

// The descent-view silhouettes, copied as the size-1 contract so the
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
		"     ▗▛▖     ",
		"    ▟░ ▚░▙   ",
		"  ▄▓████▓▄   ",
		"╱ ◢▄▄▛  ╲    ",
		"▁ ▁    ▁ ▁   ",
	}
	s1E = [5]string{
		"      ▗▛▖    ",
		"     ▟░█▙    ",
		"   ▄▓██▓▄    ",
		" ╱ ◢▄▙  ╲    ",
		"▁ ▁   ▁ ▁    ",
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
			if sz != Size4 && strings.Count(g, "~")+strings.Count(g, "≈") < 3 {
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
	t.Run("happy: size-4 N/S/W are distinct silhouettes", func(t *testing.T) {
		seen := map[string]Heading{}
		for _, h := range HeadingsFor(Size4) {
			sp, ok := a.Frame(Size4, h)
			if !ok {
				t.Fatalf("size-4 %s missing", h)
			}
			key := strings.Join(sp.GlyphRows(), "\n")
			if other, ok := seen[key]; ok {
				t.Fatalf("headings %s and %s rendered identically", h, other)
			}
			seen[key] = h
		}
	})
	t.Run("happy: size-4 S is the vertical mirror of the styled N", func(t *testing.T) {
		n, ok := a.Frame(Size4, N)
		if !ok {
			t.Fatal("size-4 N missing")
		}
		s, ok := a.Frame(Size4, S)
		if !ok {
			t.Fatal("size-4 S missing")
		}
		want := sprite.FlipV(n)
		if err := want.Validate(); err != nil {
			t.Fatalf("flipped N: %v", err)
		}
		for r := 0; r < want.Height; r++ {
			for c := 0; c < want.Width; c++ {
				got, exp := s.At(r, c), want.At(r, c)
				if got != exp {
					t.Fatalf("S is not FlipV(N) at (%d,%d): got %+v want %+v\nS:\n%s\nFlipV(N):\n%s",
						r, c, got, exp,
						strings.Join(s.GlyphRows(), "\n"),
						strings.Join(want.GlyphRows(), "\n"))
				}
			}
		}
	})
	t.Run("unhappy: size-4 S is not a copy of N", func(t *testing.T) {
		n, _ := a.Frame(Size4, N)
		s, _ := a.Frame(Size4, S)
		if strings.Join(s.GlyphRows(), "\n") == strings.Join(n.GlyphRows(), "\n") {
			t.Fatal("S must be the mirrored south craft, not the same drawing as N")
		}
	})
	t.Run("happy: S plume sits in the top half; W is the drawn west craft", func(t *testing.T) {
		s, _ := a.Frame(Size4, S)
		w, _ := a.Frame(Size4, W)
		if plumeHalf(s) != "top" && !strings.Contains(strings.Join(s.GlyphRows(), ""), "█") {
			t.Fatalf("S must still be a south-facing lander\n%s", strings.Join(s.GlyphRows(), "\n"))
		}
		if w.Width != 26 || w.Height != 10 {
			t.Fatalf("W must keep the size-4 box, got %dx%d", w.Width, w.Height)
		}
		filled := 0
		for r := 0; r < w.Height; r++ {
			for c := 0; c < w.Width; c++ {
				if !w.At(r, c).Transparent() {
					filled++
				}
			}
		}
		if filled < 20 {
			t.Fatalf("size-4 W is too empty (%d cells)\n%s", filled, strings.Join(w.GlyphRows(), "\n"))
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

func TestShippedJSON(t *testing.T) {
	t.Run("happy: lm.json ships beside the art", func(t *testing.T) {
		a, err := LoadFile("lm.json")
		if err != nil {
			t.Fatal(err)
		}
		for _, sz := range Sizes {
			for _, h := range HeadingsFor(sz) {
				if _, ok := a.Frame(sz, h); !ok {
					t.Fatalf("shipped JSON missing size %d %s", sz, h)
				}
			}
		}
		if _, ok := a.Frame(Size4, E); ok {
			t.Fatal("shipped size-4 must not keep E")
		}
		got := a.MustFrame(Size1, N).GlyphRows()
		for i, want := range s1N {
			if got[i] != want {
				t.Fatalf("shipped size-1 N drifted from the current lander row %d", i)
			}
		}
	})
	t.Run("happy: assets lm-*.json are the atlas source of truth", func(t *testing.T) {
		a, err := LoadJSONDir(FindArtDir())
		if err != nil {
			t.Fatal(err)
		}
		for _, sz := range Sizes {
			for _, h := range HeadingsFor(sz) {
				if _, ok := a.Frame(sz, h); !ok {
					t.Fatalf("shipped JSON missing size %d %s", sz, h)
				}
			}
		}
		if _, ok := a.Frame(Size4, E); ok {
			t.Fatal("shipped size-4 must not keep E")
		}
	})
	t.Run("unhappy: a missing file is an error", func(t *testing.T) {
		if _, err := LoadFile("no-such-atlas.json"); err == nil {
			t.Fatal("missing file must fail")
		}
		if _, err := LoadJSONDir(t.TempDir()); err == nil {
			t.Fatal("an empty art dir must fail")
		}
	})
}

func testAtlas(t *testing.T) *Atlas {
	t.Helper()
	a := DefaultAtlas()
	if a == nil {
		t.Fatal("DefaultAtlas() returned nil")
	}
	return a
}
