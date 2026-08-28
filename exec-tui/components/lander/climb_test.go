package lander

// Tests written FIRST: ClimbPath is DropPath run the other way — the
// north-facing hull starts fully off the bottom and finishes fully
// off the top, linear, centered. A Climb ship rides it with the
// booster firing down the whole way. This is the spacelander going
// up, not a pad liftoff: no moon, no ease, no ignition schedule.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestClimbPath(t *testing.T) {
	t.Run("happy: the climb starts fully below the stage and ends fully above it", func(t *testing.T) {
		row, col := ClimbPath(screenW, screenH, 0, DropSeconds)
		if row != screenH {
			t.Fatalf("t=0 row %d, want %d (fully off the bottom)", row, screenH)
		}
		if col != (screenW-BodyCols)/2 {
			t.Fatalf("t=0 col %d, want centered %d", col, (screenW-BodyCols)/2)
		}
		end, _ := ClimbPath(screenW, screenH, DropSeconds, DropSeconds)
		if end != -BodyRows {
			t.Fatalf("t=DropSeconds row %d, want %d (fully off the top)", end, -BodyRows)
		}
	})
	t.Run("happy: the climb is monotonic upward and a Climb ship rides it", func(t *testing.T) {
		prev := screenH
		for ti := 0; ti <= 100; ti++ {
			tt := DropSeconds * float64(ti) / 100
			row, _ := ClimbPath(screenW, screenH, tt, DropSeconds)
			if row > prev {
				t.Fatalf("t=%.2f row %d moved down (was %d)", tt, row, prev)
			}
			prev = row
		}
		s := NewShip(80).North().Climb(DropSeconds)
		s.Start(screenW, screenH)
		if opaqueCells(s.Render()) != 0 {
			t.Fatal("at t=0 the climbing craft must still be off the bottom")
		}
		warmShip(s, DropSeconds/2)
		mid := s.Render()
		if opaqueCells(mid) == 0 {
			t.Fatal("mid-climb the hull must be on stage")
		}
		for r := 0; r < mid.Height; r++ {
			for c := 0; c < mid.Width; c++ {
				if mid.At(r, c).Ch == '▌' {
					t.Fatal("a climbing north craft must not wear the west-facing hull")
				}
			}
		}
		row, col := ClimbPath(screenW, screenH, s.Clock(), DropSeconds)
		fire := 0
		for r := row + BodyRows; r < mid.Height; r++ {
			for c := col; c < col+BodyCols && c < mid.Width; c++ {
				if c < 0 || r < 0 {
					continue
				}
				if flameGlyph(mid.At(r, c).Ch) {
					fire++
				}
			}
		}
		if fire == 0 {
			t.Fatal("no fire under the hull — the plume must fire down while the craft climbs")
		}
	})
	t.Run("unhappy: negative time is the opening mark, seconds<=0 snaps off the top, and Climb on nil stays nil", func(t *testing.T) {
		r0, c0 := ClimbPath(screenW, screenH, 0, DropSeconds)
		row, col := ClimbPath(screenW, screenH, -3, DropSeconds)
		if row != r0 || col != c0 {
			t.Fatalf("t<0 at (%d,%d), want the t=0 mark (%d,%d)", row, col, r0, c0)
		}
		snap, _ := ClimbPath(screenW, screenH, 1, 0)
		if snap != -BodyRows {
			t.Fatalf("seconds<=0 must snap off the top, got %d want %d", snap, -BodyRows)
		}
		var ghost *Ship
		if ghost.Climb(DropSeconds) != nil {
			t.Fatal("Climb must return the nil receiver")
		}
	})
	t.Run("unhappy: ClimbPath is not DropPath and not the eased pad liftoff", func(t *testing.T) {
		// Linear inverses meet at the halfway mark; a quarter later
		// the climber is above the dropper, and neither is the pad
		// liftoff's ease.
		tLate := DropSeconds * 3 / 4
		drop, _ := DropPath(screenW, screenH, tLate, DropSeconds)
		climb, _ := ClimbPath(screenW, screenH, tLate, DropSeconds)
		if drop == climb {
			t.Fatalf("late-climb row %d matches the drop — the two paths must run opposite ways", climb)
		}
		if climb >= drop {
			t.Fatalf("late-climb row %d is not above the drop %d", climb, drop)
		}
		lift, _ := LiftPath(screenW, screenH, DropSeconds/2, 0, DropSeconds)
		if climb == lift {
			t.Fatal("a space climb must stay linear — it is not the pad liftoff's ease")
		}
		s := NewShip(81).North().Climb(DropSeconds)
		if s.heading != sprite.N {
			t.Fatal("Climb must keep the north-facing hull")
		}
	})
}
