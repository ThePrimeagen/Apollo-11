package lander

// Tests written FIRST: DropBeatPath is the north-facing fall with
// pauses. Each beat drops for its Drop seconds, then holds for its
// Hold seconds. Drop distances share the top-to-bottom span in
// proportion to each Drop duration, so equal drops travel equal
// rows. DropBeatHold names the beat currently parked (or -1 while
// falling). A DropBeats ship rides the path. This is how the 1202 /
// 1202 / 1201 talk stops the craft on the way down.

import (
	"testing"
)

func stockBeats() []DropBeat {
	return []DropBeat{
		{Drop: 1.5, Hold: 0.8},
		{Drop: 1.5, Hold: 0.8},
		{Drop: 1.5, Hold: 0.8},
		{Drop: 1.5, Hold: 0},
	}
}

func TestDropBeatPath(t *testing.T) {
	beats := stockBeats()
	t.Run("happy: the beats start off the top and finish off the bottom", func(t *testing.T) {
		row, col := DropBeatPath(screenW, screenH, 0, beats)
		if row != -BodyRows {
			t.Fatalf("t=0 row %d, want %d (fully off the top)", row, -BodyRows)
		}
		if col != (screenW-BodyCols)/2 {
			t.Fatalf("t=0 col %d, want centered %d", col, (screenW-BodyCols)/2)
		}
		end, _ := DropBeatPath(screenW, screenH, 100, beats)
		if end != screenH {
			t.Fatalf("past the last beat row %d, want %d (fully off the bottom)", end, screenH)
		}
	})
	t.Run("happy: each hold parks the hull, and the next drop moves it again", func(t *testing.T) {
		// First drop is 1.5s of the 6s of motion → one quarter of the span.
		start, finish := -BodyRows, screenH
		span := finish - start
		quarter := start + int(float64(span)*0.25+0.5)
		held, _ := DropBeatPath(screenW, screenH, 1.5, beats)
		if held != quarter {
			t.Fatalf("at the first hold row %d, want the ¼ mark %d", held, quarter)
		}
		still, _ := DropBeatPath(screenW, screenH, 1.5+0.7, beats)
		if still != held {
			t.Fatalf("mid-hold row %d drifted, want parked at %d", still, held)
		}
		moved, _ := DropBeatPath(screenW, screenH, 1.5+0.8+0.2, beats)
		if moved <= held {
			t.Fatalf("after the hold the hull must fall again, row %d still at %d", moved, held)
		}
		if DropBeatHold(1.5+0.4, beats) != 0 {
			t.Fatalf("mid first hold, Hold index %d, want 0", DropBeatHold(1.5+0.4, beats))
		}
		if DropBeatHold(1.5+0.8+1.5+0.1, beats) != 1 {
			t.Fatalf("second hold, Hold index %d, want 1", DropBeatHold(1.5+0.8+1.5+0.1, beats))
		}
		if DropBeatHold(1.5+0.8+1.5+0.8+1.5+0.1, beats) != 2 {
			t.Fatalf("third hold, Hold index %d, want 2 — 1201 is the last pause", DropBeatHold(1.5+0.8+1.5+0.8+1.5+0.1, beats))
		}
	})
	t.Run("happy: a DropBeats ship rides the pauses", func(t *testing.T) {
		s := NewShip(82).North().DropBeats(stockBeats())
		s.Start(screenW, screenH)
		if opaqueCells(s.Render()) != 0 {
			t.Fatal("at t=0 the craft must still be off the top")
		}
		warmShip(s, 1.6)
		if opaqueCells(s.Render()) == 0 {
			t.Fatal("into the first hold the hull must be on stage")
		}
		row, _ := s.position()
		want, _ := DropBeatPath(screenW, screenH, s.Clock(), stockBeats())
		if row != want {
			t.Fatalf("the ship sat at row %d, want the beat path %d", row, want)
		}
		parked := row
		warmShip(s, 0.5)
		row, _ = s.position()
		if row != parked {
			t.Fatalf("during the hold the hull moved %d → %d", parked, row)
		}
	})
	t.Run("unhappy: negative time is the opening, empty beats snap off the bottom, Hold is -1 while falling, and DropBeats on nil stays nil", func(t *testing.T) {
		r0, c0 := DropBeatPath(screenW, screenH, 0, beats)
		row, col := DropBeatPath(screenW, screenH, -2, beats)
		if row != r0 || col != c0 {
			t.Fatalf("t<0 at (%d,%d), want the t=0 mark (%d,%d)", row, col, r0, c0)
		}
		gone, _ := DropBeatPath(screenW, screenH, 1, nil)
		if gone != screenH {
			t.Fatalf("no beats row %d, want off the bottom %d", gone, screenH)
		}
		if DropBeatHold(0.2, beats) != -1 {
			t.Fatal("the opening drop is not a hold")
		}
		if DropBeatHold(1.5+0.8+0.1, beats) != -1 {
			t.Fatal("the second drop is not a hold")
		}
		if DropBeatHold(100, beats) != -1 {
			t.Fatal("after the last beat there is no hold left")
		}
		zero := []DropBeat{{Drop: 0, Hold: 1}, {Drop: 0, Hold: 0}}
		snap, _ := DropBeatPath(screenW, screenH, 0.5, zero)
		if snap != screenH {
			t.Fatalf("all-zero drops must snap off the bottom, got %d", snap)
		}
		var ghost *Ship
		if ghost.DropBeats(beats) != nil {
			t.Fatal("DropBeats must return the nil receiver")
		}
	})
}
