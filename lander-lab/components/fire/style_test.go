package fire

import (
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/lander-lab/particle"
)

func TestBands(t *testing.T) {
	ResetHeat()
	t.Run("happy: each rung needs 15% more heat than the old ladder", func(t *testing.T) {
		// old mins 1,6,11,21,41,71,121,200 → round(min*1.15), first stays 1
		want := []struct {
			min, max int
			glyph    rune
		}{
			{1, 6, '⠁'},
			{7, 12, '⠒'},
			{13, 23, '⠶'},
			{24, 46, '░'},
			{47, 81, '▒'},
			{82, 138, '▄'},
			{139, 229, '▓'},
			{230, 1 << 30, '█'},
		}
		got := Bands()
		if len(got) != len(want) {
			t.Fatalf("bands %d, want %d", len(got), len(want))
		}
		for i, w := range want {
			if got[i].Min != w.min || got[i].Max != w.max || got[i].Glyph != w.glyph {
				t.Fatalf("band %d is %d..%d %q, want %d..%d %q",
					i, got[i].Min, got[i].Max, string(got[i].Glyph),
					w.min, w.max, string(w.glyph))
			}
		}
	})
	t.Run("unhappy: the old cutoffs no longer reach the next glyph", func(t *testing.T) {
		if Style(200).Ch == '█' {
			t.Fatal("H=200 used to be solid yellow; after +15% it must not be")
		}
		if Style(121).Ch == '▓' {
			t.Fatal("H=121 used to be heavy shade; after +15% it must not be")
		}
		if Style(71).Ch == '▄' {
			t.Fatal("H=71 used to be a half square; after +15% it must not be")
		}
	})
}

func TestStyle(t *testing.T) {
	t.Run("happy: the ladder runs single-dot, two-dot, half-square, then bright yellow", func(t *testing.T) {
		one := Style(3)
		if one.Ch != '⠁' {
			t.Fatalf("H=3 should be a single braille dot, got %q", string(one.Ch))
		}
		two := Style(10)
		if two.Ch != '⠒' {
			t.Fatalf("H=10 should be two braille dots, got %q", string(two.Ch))
		}
		half := Style(90)
		if half.Ch != '▄' && half.Ch != '▌' {
			t.Fatalf("mid heat should be a half square, got %q", string(half.Ch))
		}
		core := Style(250)
		if core.Ch != '█' {
			t.Fatalf("high heat should be solid, got %q", string(core.Ch))
		}
		if core.FG == 231 || core.BG == 231 {
			t.Fatal("core must stay bright yellow, never white")
		}
	})
	t.Run("happy: the 15% cutoffs map onto the next glyph", func(t *testing.T) {
		if Style(6).Ch != '⠁' {
			t.Fatalf("H=6 should still be a single dot, got %q", string(Style(6).Ch))
		}
		if Style(7).Ch != '⠒' {
			t.Fatalf("H=7 should be two dots, got %q", string(Style(7).Ch))
		}
		if Style(229).Ch != '▓' {
			t.Fatalf("H=229 should be heavy shade, got %q", string(Style(229).Ch))
		}
		if Style(230).Ch != '█' {
			t.Fatalf("H=230 should be solid yellow, got %q", string(Style(230).Ch))
		}
	})
	t.Run("unhappy: zero and negative heat paint nothing", func(t *testing.T) {
		if !Style(0).Transparent() {
			t.Fatalf("H=0 must be empty, got %+v", Style(0))
		}
		if !Style(-4).Transparent() {
			t.Fatal("negative heat must be empty")
		}
	})
}

func TestHeat(t *testing.T) {
	t.Run("happy: heat is self plus every side except the incoming one", func(t *testing.T) {
		occ := map[particle.Cell]int{
			{Col: 2, Row: 2}: 10, // self
			{Col: 1, Row: 2}: 7,  // west — incoming for +X, excluded
			{Col: 3, Row: 2}: 4,  // east
			{Col: 2, Row: 1}: 3,  // north
			{Col: 2, Row: 3}: 2,  // south
		}
		h := Heat(occ, particle.Cell{Col: 2, Row: 2}, particle.Vec2{X: 1, Y: 0})
		if h != 10+4+3+2 {
			t.Fatalf("H=%d, want 19 (self+N+S+E, not west)", h)
		}
	})
	t.Run("unhappy: an empty neighborhood is zero, not a panic", func(t *testing.T) {
		if Heat(nil, particle.Cell{Col: 0, Row: 0}, particle.Vec2{X: 1, Y: 0}) != 0 {
			t.Fatal("nil occupancy must be H=0")
		}
	})
}

func TestGuide(t *testing.T) {
	t.Run("happy: the guide lists every band and its equation", func(t *testing.T) {
		g := Guide()
		for _, want := range []string{"⠁", "⠒", "▄", "█", "H(c)", "226", "H >= 230", "1 <= H <= 6"} {
			if !strings.Contains(g, want) {
				t.Fatalf("guide missing %q\n%s", want, g)
			}
		}
	})
}
