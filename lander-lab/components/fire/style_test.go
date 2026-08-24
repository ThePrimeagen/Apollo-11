package fire

import (
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/lander-lab/particle"
)

func TestStyle(t *testing.T) {
	t.Run("happy: the ladder runs single-dot, two-dot, half-square, then bright yellow", func(t *testing.T) {
		one := Style(1)
		if one.Ch != '⠁' {
			t.Fatalf("H=1 should be a single braille dot, got %q", string(one.Ch))
		}
		two := Style(2)
		if two.Ch != '⠒' {
			t.Fatalf("H=2 should be two braille dots, got %q", string(two.Ch))
		}
		half := Style(18)
		if half.Ch != '▄' && half.Ch != '▌' {
			t.Fatalf("mid heat should be a half square, got %q", string(half.Ch))
		}
		core := Style(80)
		if core.Ch != '█' {
			t.Fatalf("high heat should be solid, got %q", string(core.Ch))
		}
		if core.FG == 231 || core.BG == 231 {
			t.Fatal("core must stay bright yellow, never white")
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
		for _, want := range []string{"⠁", "⠒", "▄", "█", "H =", "226"} {
			if !strings.Contains(g, want) {
				t.Fatalf("guide missing %q\n%s", want, g)
			}
		}
	})
}
