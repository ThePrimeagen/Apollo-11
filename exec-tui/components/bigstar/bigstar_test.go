package bigstar

// Tests written FIRST: Star is the larger star component — a sparkle
// that occupies one cell at size 1 and grows into a multi-cell burst
// (span 2*size-1) at sizes 2..5. Size and heading can be set, or
// rolled random at Start. Place pins the center; a parked star (no
// Place) sits at stage center. Render returns a stage-sized sprite.
// The package does not move: motion is the shooting-star scene's.

import (
	"math"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 40
	stageH = 20
)

var _ screenplay.Component = (*Star)(nil)

func glyphCount(sp sprite.Sprite) int {
	n := 0
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if !sp.At(r, c).Transparent() {
				n++
			}
		}
	}
	return n
}

func hasCore(sp sprite.Sprite) bool {
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if sp.At(r, c).Ch == CoreGlyph {
				return true
			}
		}
	}
	return false
}

func TestArt(t *testing.T) {
	t.Run("happy: size 1 is a single core cell", func(t *testing.T) {
		a := Art(1, particle.Vec2{})
		if a.Width != 1 || a.Height != 1 {
			t.Fatalf("size-1 art is %dx%d, want 1x1", a.Width, a.Height)
		}
		if a.At(0, 0).Ch != CoreGlyph {
			t.Fatalf("size-1 core is %q, want %q", string(a.At(0, 0).Ch), string(CoreGlyph))
		}
		if glyphCount(a) != 1 {
			t.Fatalf("size-1 painted %d cells, want 1", glyphCount(a))
		}
	})
	t.Run("happy: size 3 occupies a 5x5 burst with a core in the middle", func(t *testing.T) {
		a := Art(3, particle.Vec2{})
		if a.Width != 5 || a.Height != 5 {
			t.Fatalf("size-3 art is %dx%d, want 5x5 (span 2*size-1)", a.Width, a.Height)
		}
		if a.At(2, 2).Ch != CoreGlyph {
			t.Fatal("the core must sit at the center of the burst")
		}
		if glyphCount(a) < 5 {
			t.Fatalf("a size-3 star must paint multiple cells, got %d", glyphCount(a))
		}
	})
	t.Run("happy: a heading stretches the burst opposite the flight", func(t *testing.T) {
		plain := Art(3, particle.Vec2{})
		headed := Art(3, particle.Vec2{X: 1, Y: 0})
		if glyphCount(headed) < glyphCount(plain) {
			t.Fatal("a heading should add the trailing spark, not shrink the star")
		}
		// rightward flight: the wake is to the left of center
		if headed.At(2, 0).Transparent() && headed.At(2, 1).Transparent() {
			t.Fatal("a rightward star must paint a wake on the left of the core")
		}
	})
	t.Run("unhappy: size 0 and size 6 are refused, and Art hands back an empty sprite", func(t *testing.T) {
		if err := ValidateSize(0); err == nil {
			t.Fatal("size 0 must be rejected")
		}
		if err := ValidateSize(6); err == nil {
			t.Fatal("size 6 is past MaxSize 5")
		}
		if err := ValidateSize(1); err != nil {
			t.Fatalf("size 1 must pass: %v", err)
		}
		a := Art(0, particle.Vec2{})
		if a.Width != 0 || glyphCount(a) != 0 {
			t.Fatal("Art of a rejected size must be empty")
		}
	})
}

func TestStarComponent(t *testing.T) {
	t.Run("happy: a parked star opens at stage center wearing its size", func(t *testing.T) {
		s := NewSized(2)
		s.Start(stageW, stageH)
		defer s.Stop()
		sp := s.Render()
		if sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("stage %dx%d, want %dx%d", sp.Width, sp.Height, stageW, stageH)
		}
		if !hasCore(sp) {
			t.Fatal("a started star must paint its core")
		}
		col, row := s.Center()
		if col != stageW/2 || row != stageH/2 {
			t.Fatalf("parked center (%d,%d), want stage center (%d,%d)", col, row, stageW/2, stageH/2)
		}
		if sp.At(row, col).Ch != CoreGlyph {
			t.Fatal("the core must sit on the parked center")
		}
	})
	t.Run("happy: Place pins the core, and Span grows with size", func(t *testing.T) {
		s := NewSized(4)
		if s.Span() != 7 {
			t.Fatalf("size 4 span %d, want 7", s.Span())
		}
		s.Start(stageW, stageH)
		defer s.Stop()
		s.Place(10, 6)
		col, row := s.Center()
		if col != 10 || row != 6 {
			t.Fatalf("placed center (%d,%d), want (10,6)", col, row)
		}
		if s.Render().At(6, 10).Ch != CoreGlyph {
			t.Fatal("Place must move the core")
		}
	})
	t.Run("happy: RandomSize rolls a size in 1..MaxSize at Start, same seed same size", func(t *testing.T) {
		a := New(11)
		a.RandomSize = true
		a.Start(stageW, stageH)
		defer a.Stop()
		b := New(11)
		b.RandomSize = true
		b.Start(stageW, stageH)
		defer b.Stop()
		c := New(12)
		c.RandomSize = true
		c.Start(stageW, stageH)
		defer c.Stop()
		if a.Size < MinSize || a.Size > MaxSize {
			t.Fatalf("rolled size %d outside [%d,%d]", a.Size, MinSize, MaxSize)
		}
		if a.Size != b.Size {
			t.Fatalf("same seed rolled %d and %d", a.Size, b.Size)
		}
		// a different seed is allowed to match, but the pair must be
		// able to differ — try a few if 12 collides
		if a.Size == c.Size {
			d := New(99)
			d.RandomSize = true
			d.Start(stageW, stageH)
			defer d.Stop()
			if d.Size == a.Size {
				// still ok: unlucky, but size must stay in range
				if d.Size < MinSize || d.Size > MaxSize {
					t.Fatalf("rolled size %d outside range", d.Size)
				}
			}
		}
	})
	t.Run("happy: RandomDir rolls a non-zero heading at Start", func(t *testing.T) {
		s := New(3)
		s.RandomDir = true
		s.Start(stageW, stageH)
		defer s.Stop()
		if s.Heading == (particle.Vec2{}) {
			t.Fatal("RandomDir must pick a heading")
		}
		if math.Abs(s.Heading.Len()-1) > 1e-9 {
			t.Fatalf("heading %+v is not unit", s.Heading)
		}
	})
	t.Run("unhappy: a stopped star and a star that never started paint nothing, and dt<=0 holds", func(t *testing.T) {
		s := NewSized(3)
		if glyphCount(s.Render()) != 0 {
			t.Fatal("before Start the stage is empty")
		}
		s.Start(stageW, stageH)
		s.Stop()
		if glyphCount(s.Render()) != 0 {
			t.Fatal("after Stop the stage is empty")
		}
		held := NewSized(2)
		held.Start(stageW, stageH)
		defer held.Stop()
		held.Place(8, 4)
		held.Update(0)
		held.Update(-1)
		col, row := held.Center()
		if col != 8 || row != 4 {
			t.Fatal("dt<=0 must hold the place")
		}
	})
	t.Run("unhappy: NewSized of a rejected size falls back to MinSize and does not panic", func(t *testing.T) {
		s := NewSized(0)
		if s == nil {
			t.Fatal("NewSized must still return a star")
		}
		if s.Size != MinSize {
			t.Fatalf("rejected size parked at %d, want MinSize %d", s.Size, MinSize)
		}
		s.Start(stageW, stageH)
		s.Stop()
	})
}
