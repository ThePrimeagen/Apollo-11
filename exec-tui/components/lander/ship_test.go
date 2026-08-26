package lander

// Tests written FIRST: the ship is the full zoomed-in (size-4, 26×10)
// craft flying west — nose left, tail right — with the atlas's baked
// tilde plume stripped and the live left-to-right booster fire trailing
// off the tail. Dark() skips the booster so a scene can fly the hull
// alone; Parked() opens already at center stage. As a component:
// Start(w, h) builds the hull and arms the fire for that stage, Update
// moves the clock and burns the fire, Render composes fire-then-hull
// into a stage-sized sprite, and Stop drops both so a stopped ship
// holds no allocation. FlightPath is the whole choreography: fully off
// the right wing at t=0, an eased slide that parks at center stage by
// FlyInSeconds, then a ±1-cell sine bobble with a ten-second period.

import (
	"math"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	screenW = 72
	screenH = 28
)

// The compile-time pin: a Ship plays as a screenplay component.
var _ screenplay.Component = (*Ship)(nil)

// centerRow/centerCol is where the hull's top-left parks on the test stage.
var (
	centerRow = (screenH - BodyRows) / 2
	centerCol = (screenW - BodyCols) / 2
)

func warmShip(s *Ship, seconds float64) {
	const dt = 1.0 / 30
	for i := 0; i < int(seconds*30+0.5); i++ {
		s.Update(dt)
	}
}

// flameGlyph is the fire heat ladder's glyph set — no hull rune is in it.
func flameGlyph(ch rune) bool {
	switch ch {
	case '⠁', '⠒', '⠶', '░', '▒', '▄', '▓', '█':
		return true
	}
	return false
}

func opaqueCells(sp sprite.Sprite) int {
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

func TestNewShip(t *testing.T) {
	t.Run("happy: start builds the size-4 west frame minus its baked plume", func(t *testing.T) {
		s := NewShip(1)
		s.Start(screenW, screenH)
		if s.Body.Width != BodyCols || s.Body.Height != BodyRows {
			t.Fatalf("body %dx%d, want %dx%d", s.Body.Width, s.Body.Height, BodyCols, BodyRows)
		}
		want := DefaultAtlas().MustFrame(sprite.Size4, sprite.W)
		for r := 0; r < want.Height; r++ {
			for c := 0; c < want.Width; c++ {
				wc := want.At(r, c)
				got := s.Body.At(r, c)
				if wc.Ch == '~' || wc.Ch == '≈' {
					if !got.Transparent() {
						t.Fatalf("baked plume survived at (%d,%d): %+v", r, c, got)
					}
					continue
				}
				if got != wc {
					t.Fatalf("hull cell (%d,%d) changed: %+v -> %+v", r, c, wc, got)
				}
			}
		}
	})
	t.Run("happy: start arms a valid fire aimed left-to-right", func(t *testing.T) {
		s := NewShip(2)
		s.Start(screenW, screenH)
		if s.Flame == nil || s.Flame.Eng == nil {
			t.Fatal("start must arm the booster fire")
		}
		if err := s.Flame.Eng.Validate(); err != nil {
			t.Fatalf("flame config: %v", err)
		}
		d := s.Flame.Eng.Cfg.Direction
		if math.Abs(d.X-1) > 1e-9 || math.Abs(d.Y) > 1e-9 {
			t.Fatalf("direction %+v, want (1, 0) — fire out the right side", d)
		}
	})
	t.Run("unhappy: before start there is nothing to allocate or render", func(t *testing.T) {
		s := NewShip(3)
		if s.Flame != nil {
			t.Fatal("nothing may allocate before Start")
		}
		if sp := s.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("an unstarted ship rendered %dx%d", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: at t=0 the whole stage stays dark — the craft is offstage", func(t *testing.T) {
		s := NewShip(3)
		s.Start(screenW, screenH)
		if n := opaqueCells(s.Render()); n != 0 {
			t.Fatalf("an unticked ship lit %d cells — it is offstage and unlit", n)
		}
	})
}

func TestFlightPath(t *testing.T) {
	t.Run("happy: starts fully off the right wing, level at center height", func(t *testing.T) {
		row, col := FlightPath(screenW, screenH, 0)
		if col != screenW {
			t.Fatalf("t=0 col %d, want %d (fully offstage)", col, screenW)
		}
		if row != centerRow {
			t.Fatalf("t=0 row %d, want %d", row, centerRow)
		}
	})
	t.Run("happy: the slide is monotonic and parks at center stage", func(t *testing.T) {
		prev := screenW
		for ti := 0; ti <= 400; ti++ {
			tt := float64(ti) / 100
			row, col := FlightPath(screenW, screenH, tt)
			if col > prev {
				t.Fatalf("t=%.2f col %d moved right (was %d)", tt, col, prev)
			}
			if row != centerRow {
				t.Fatalf("t=%.2f row %d, want a level slide at %d", tt, row, centerRow)
			}
			prev = col
		}
		if _, col := FlightPath(screenW, screenH, FlyInSeconds); col != centerCol {
			t.Fatalf("parked col %d, want %d", col, centerCol)
		}
		if _, col := FlightPath(screenW, screenH, 1000); col != centerCol {
			t.Fatalf("col %d drifted long after parking, want %d", col, centerCol)
		}
	})
	t.Run("happy: the bobble is a one-cell sine with a ten-second period", func(t *testing.T) {
		quarters := []struct {
			at   float64
			want int
		}{
			{FlyInSeconds, centerRow},
			{FlyInSeconds + 2.5, centerRow - 1}, // crest: one cell up
			{FlyInSeconds + 5.0, centerRow},
			{FlyInSeconds + 7.5, centerRow + 1}, // trough: one cell down
			{FlyInSeconds + 10.0, centerRow},    // full period
		}
		for _, q := range quarters {
			if row, _ := FlightPath(screenW, screenH, q.at); row != q.want {
				t.Fatalf("t=%.1f row %d, want %d", q.at, row, q.want)
			}
		}
	})
	t.Run("happy: the bobble never rides beyond one cell", func(t *testing.T) {
		for ti := 0; ti <= 2000; ti++ {
			tt := FlyInSeconds + float64(ti)/100
			row, col := FlightPath(screenW, screenH, tt)
			if d := row - centerRow; d < -BobAmplitudeCells || d > BobAmplitudeCells {
				t.Fatalf("t=%.2f row %d rode %d cells from center", tt, row, d)
			}
			if col != centerCol {
				t.Fatalf("t=%.2f col %d, want a steady park at %d", tt, col, centerCol)
			}
		}
	})
	t.Run("unhappy: negative time is the opening mark", func(t *testing.T) {
		row, col := FlightPath(screenW, screenH, -3)
		r0, c0 := FlightPath(screenW, screenH, 0)
		if row != r0 || col != c0 {
			t.Fatalf("t<0 at (%d,%d), want the t=0 mark (%d,%d)", row, col, r0, c0)
		}
	})
	t.Run("unhappy: a stage smaller than the craft still answers", func(t *testing.T) {
		if row, col := FlightPath(8, 3, 100); col > 8 {
			t.Fatalf("tiny stage answered (%d,%d) — parked col can never sit right of the stage", row, col)
		}
	})
}

func TestShipOnStage(t *testing.T) {
	t.Run("happy: a warmed ship parks with hull cells intact and fire off the tail", func(t *testing.T) {
		s := NewShip(4)
		s.Start(screenW, screenH)
		warmShip(s, 5.0)
		if math.Abs(s.Clock()-5.0) > 1e-9 {
			t.Fatalf("clock %f, want 5.0", s.Clock())
		}
		stage := s.Render()
		if stage.Width != screenW || stage.Height != screenH {
			t.Fatalf("stage %dx%d, want %dx%d", stage.Width, stage.Height, screenW, screenH)
		}
		row, col := FlightPath(screenW, screenH, s.Clock())
		for r := 0; r < s.Body.Height; r++ {
			for c := 0; c < s.Body.Width; c++ {
				b := s.Body.At(r, c)
				if b.Transparent() {
					continue
				}
				if got := stage.At(row+r, col+c); got != b {
					t.Fatalf("hull cell (%d,%d) burned or moved: %+v -> %+v", r, c, b, got)
				}
			}
		}
		fire := 0
		for r := 0; r < stage.Height; r++ {
			for c := col + BodyCols; c < stage.Width; c++ {
				if flameGlyph(stage.At(r, c).Ch) {
					fire++
				}
			}
		}
		if fire == 0 {
			t.Fatal("no fire right of the hull — the plume must trail the tail")
		}
	})
	t.Run("happy: the baked tilde plume never reaches the stage", func(t *testing.T) {
		s := NewShip(5)
		s.Start(screenW, screenH)
		warmShip(s, 5.0)
		stage := s.Render()
		for r := 0; r < stage.Height; r++ {
			for c := 0; c < stage.Width; c++ {
				if ch := stage.At(r, c).Ch; ch == '~' || ch == '≈' {
					t.Fatalf("static plume glyph %q at (%d,%d); the live fire is the plume", string(ch), r, c)
				}
			}
		}
	})
	t.Run("happy: stop drops the hull and the fire; a fresh start re-arms", func(t *testing.T) {
		s := NewShip(6)
		s.Start(screenW, screenH)
		warmShip(s, 5.0)
		s.Stop()
		if s.Flame != nil {
			t.Fatal("stop must drop the fire for the collector")
		}
		if sp := s.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a stopped ship rendered %dx%d", sp.Width, sp.Height)
		}
		s.Start(screenW, screenH)
		if s.Flame == nil {
			t.Fatal("a fresh start must re-arm the fire")
		}
		if math.Abs(s.Clock()-5.0) > 1e-9 {
			t.Fatalf("the clock must survive a restage, got %f", s.Clock())
		}
	})
	t.Run("unhappy: dt<=0 holds the clock and the mark", func(t *testing.T) {
		s := NewShip(6)
		s.Start(screenW, screenH)
		s.Update(0)
		s.Update(-1)
		if s.Clock() != 0 {
			t.Fatalf("clock %f moved on dt<=0", s.Clock())
		}
		if n := opaqueCells(s.Render()); n != 0 {
			t.Fatalf("held ship lit %d cells, want 0 (still offstage)", n)
		}
	})
	t.Run("unhappy: a stage smaller than the craft clips instead of panicking", func(t *testing.T) {
		s := NewShip(7)
		s.Start(8, 3)
		warmShip(s, 5.0)
		if n := opaqueCells(s.Render()); n > 8*3 {
			t.Fatalf("tiny stage lit %d cells, has only %d", n, 8*3)
		}
	})
	t.Run("unhappy: a nil ship skips every cue", func(t *testing.T) {
		var ghost *Ship
		ghost.Start(4, 2)
		ghost.Update(0.1)
		ghost.Render()
		ghost.Stop()
	})
}

func TestDarkShip(t *testing.T) {
	t.Run("happy: a dark start builds the hull and leaves the booster cold", func(t *testing.T) {
		s := NewShip(8).Dark()
		s.Start(screenW, screenH)
		if s.Flame != nil {
			t.Fatal("a dark ship must not arm the booster")
		}
		if s.Body.Width != BodyCols || s.Body.Height != BodyRows {
			t.Fatalf("body %dx%d, want %dx%d", s.Body.Width, s.Body.Height, BodyCols, BodyRows)
		}
	})
	t.Run("happy: a warmed dark ship parks with hull cells and no plume", func(t *testing.T) {
		s := NewShip(9).Dark()
		s.Start(screenW, screenH)
		warmShip(s, 5.0)
		stage := s.Render()
		if opaqueCells(stage) == 0 {
			t.Fatal("the hull must still land on stage without fire")
		}
		row, col := FlightPath(screenW, screenH, s.Clock())
		for r := 0; r < s.Body.Height; r++ {
			for c := 0; c < s.Body.Width; c++ {
				b := s.Body.At(r, c)
				if b.Transparent() {
					continue
				}
				if got := stage.At(row+r, col+c); got != b {
					t.Fatalf("hull cell (%d,%d) moved: %+v -> %+v", r, c, b, got)
				}
			}
		}
		for r := 0; r < stage.Height; r++ {
			for c := col + BodyCols; c < stage.Width; c++ {
				ch := stage.At(r, c).Ch
				if ch == '⠁' || ch == '⠒' || ch == '⠶' || ch == '░' || ch == '▒' || ch == '▄' {
					t.Fatalf("plume glyph %q right of the hull at (%d,%d)", string(ch), r, c)
				}
			}
		}
	})
	t.Run("unhappy: Dark on a nil ship is still nil", func(t *testing.T) {
		var ghost *Ship
		if ghost.Dark() != nil {
			t.Fatal("Dark must return the nil receiver")
		}
	})
}

func TestParkedShip(t *testing.T) {
	t.Run("happy: a parked ship opens at center stage with no fly-in", func(t *testing.T) {
		s := NewShip(10).Dark().Parked()
		s.Start(screenW, screenH)
		if s.Clock() != FlyInSeconds {
			t.Fatalf("clock %f, want FlyInSeconds so the bobble starts from the park", s.Clock())
		}
		stage := s.Render()
		if opaqueCells(stage) == 0 {
			t.Fatal("a parked ship must already be on stage at t=0 of the scene")
		}
		wantRow, wantCol := FlightPath(screenW, screenH, FlyInSeconds)
		if wantCol != centerCol || wantRow != centerRow {
			t.Fatalf("test premise: park is (%d,%d), got FlightPath (%d,%d)", centerRow, centerCol, wantRow, wantCol)
		}
		for r := 0; r < s.Body.Height; r++ {
			for c := 0; c < s.Body.Width; c++ {
				b := s.Body.At(r, c)
				if b.Transparent() {
					continue
				}
				if got := stage.At(wantRow+r, wantCol+c); got != b {
					t.Fatalf("parked hull cell (%d,%d) missing: %+v -> %+v", r, c, b, got)
				}
			}
		}
	})
	t.Run("unhappy: Parked on a nil ship is still nil", func(t *testing.T) {
		var ghost *Ship
		if ghost.Parked() != nil {
			t.Fatal("Parked must return the nil receiver")
		}
	})
}

func TestHoldShip(t *testing.T) {
	const hold = FlyInHoldSeconds
	t.Run("happy: a held ship stays offstage until the wait is over", func(t *testing.T) {
		s := NewShip(12).Dark().Hold(hold)
		s.Start(screenW, screenH)
		warmShip(s, hold-0.1)
		if n := opaqueCells(s.Render()); n != 0 {
			t.Fatalf("during the hold the craft lit %d cells — it is still offstage", n)
		}
		s.Update(0.2)
		if opaqueCells(s.Render()) == 0 {
			t.Fatal("once the hold ends the fly-in must have started")
		}
	})
	t.Run("happy: after the hold the fly-in parks on the same mark", func(t *testing.T) {
		s := NewShip(13).Dark().Hold(hold)
		s.Start(screenW, screenH)
		warmShip(s, hold+FlyInSeconds)
		wantRow, wantCol := FlightPath(screenW, screenH, FlyInSeconds)
		stage := s.Render()
		for r := 0; r < s.Body.Height; r++ {
			for c := 0; c < s.Body.Width; c++ {
				b := s.Body.At(r, c)
				if b.Transparent() {
					continue
				}
				if got := stage.At(wantRow+r, wantCol+c); got != b {
					t.Fatalf("held hull cell (%d,%d) missing at park: %+v -> %+v", r, c, b, got)
				}
			}
		}
	})
	t.Run("happy: Hold then Parked skips the wait and the fly-in", func(t *testing.T) {
		s := NewShip(14).Dark().Hold(hold).Parked()
		s.Start(screenW, screenH)
		if s.Clock() != hold+FlyInSeconds {
			t.Fatalf("clock %f, want hold+FlyInSeconds (%f)", s.Clock(), hold+FlyInSeconds)
		}
		if opaqueCells(s.Render()) == 0 {
			t.Fatal("Hold().Parked() must already be on stage")
		}
	})
	t.Run("unhappy: Hold on a nil ship is still nil", func(t *testing.T) {
		var ghost *Ship
		if ghost.Hold(hold) != nil {
			t.Fatal("Hold must return the nil receiver")
		}
	})
}

func TestNorthShip(t *testing.T) {
	t.Run("happy: North builds the size-4 north frame and aims fire straight down", func(t *testing.T) {
		s := NewShip(20).North()
		s.Start(screenW, screenH)
		want := stripPlume(DefaultAtlas().MustFrame(sprite.Size4, sprite.N))
		if s.Body.Width != want.Width || s.Body.Height != want.Height {
			t.Fatalf("north body %dx%d, want %dx%d", s.Body.Width, s.Body.Height, want.Width, want.Height)
		}
		for r := 0; r < want.Height; r++ {
			for c := 0; c < want.Width; c++ {
				if got, wc := s.Body.At(r, c), want.At(r, c); got != wc {
					t.Fatalf("north hull cell (%d,%d) %+v, want %+v", r, c, got, wc)
				}
			}
		}
		if s.Flame == nil || s.Flame.Eng == nil {
			t.Fatal("a north ship must arm the booster")
		}
		d := s.Flame.Eng.Cfg.Direction
		if math.Abs(d.X) > 1e-9 || math.Abs(d.Y-1) > 1e-9 {
			t.Fatalf("direction %+v, want (0, 1) — fire out the bottom", d)
		}
	})
	t.Run("unhappy: North on a nil ship is still nil", func(t *testing.T) {
		var ghost *Ship
		if ghost.North() != nil {
			t.Fatal("North must return the nil receiver")
		}
	})
}

func TestDropPath(t *testing.T) {
	t.Run("happy: the drop starts fully above the stage and ends fully below it", func(t *testing.T) {
		row, col := DropPath(screenW, screenH, 0, DropSeconds)
		if row != -BodyRows {
			t.Fatalf("t=0 row %d, want %d (fully off the top)", row, -BodyRows)
		}
		if col != (screenW-BodyCols)/2 {
			t.Fatalf("t=0 col %d, want centered %d", col, (screenW-BodyCols)/2)
		}
		end, _ := DropPath(screenW, screenH, DropSeconds, DropSeconds)
		if end != screenH {
			t.Fatalf("t=DropSeconds row %d, want %d (fully off the bottom)", end, screenH)
		}
	})
	t.Run("happy: the drop is monotonic downward and a Drop ship rides it", func(t *testing.T) {
		prev := -BodyRows
		for ti := 0; ti <= 100; ti++ {
			tt := DropSeconds * float64(ti) / 100
			row, _ := DropPath(screenW, screenH, tt, DropSeconds)
			if row < prev {
				t.Fatalf("t=%.2f row %d moved up (was %d)", tt, row, prev)
			}
			prev = row
		}
		s := NewShip(21).North().Drop(DropSeconds)
		s.Start(screenW, screenH)
		if opaqueCells(s.Render()) != 0 {
			t.Fatal("at t=0 the falling craft must still be off the top")
		}
		warmShip(s, DropSeconds/2)
		mid := s.Render()
		if opaqueCells(mid) == 0 {
			t.Fatal("mid-drop the hull must be on stage")
		}
		for r := 0; r < mid.Height; r++ {
			for c := 0; c < mid.Width; c++ {
				if mid.At(r, c).Ch == '▌' {
					t.Fatal("a falling north craft must not wear the west-facing hull")
				}
			}
		}
		row, col := DropPath(screenW, screenH, s.Clock(), DropSeconds)
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
			t.Fatal("no fire under the hull — the plume must fire down")
		}
	})
	t.Run("unhappy: negative time is the opening mark, and Drop on nil stays nil", func(t *testing.T) {
		r0, c0 := DropPath(screenW, screenH, 0, DropSeconds)
		row, col := DropPath(screenW, screenH, -3, DropSeconds)
		if row != r0 || col != c0 {
			t.Fatalf("t<0 at (%d,%d), want the t=0 mark (%d,%d)", row, col, r0, c0)
		}
		var ghost *Ship
		if ghost.Drop(DropSeconds) != nil {
			t.Fatal("Drop must return the nil receiver")
		}
	})
}

func TestLandPath(t *testing.T) {
	t.Run("happy: the lander starts off the top and parks on the horizon pad", func(t *testing.T) {
		pad := LandPadRow(screenH)
		row, col := LandPath(screenW, screenH, 0, LandSeconds)
		if row != -BodyRows {
			t.Fatalf("t=0 row %d, want %d (fully off the top)", row, -BodyRows)
		}
		if col != (screenW-BodyCols)/2 {
			t.Fatalf("t=0 col %d, want centered %d", col, (screenW-BodyCols)/2)
		}
		park, _ := LandPath(screenW, screenH, LandSeconds, LandSeconds)
		if park != pad {
			t.Fatalf("parked row %d, want pad %d", park, pad)
		}
		late, _ := LandPath(screenW, screenH, LandSeconds+8, LandSeconds)
		if late != pad {
			t.Fatalf("after touchdown row %d drifted, want pad %d", late, pad)
		}
	})
	t.Run("happy: a Land ship rides the path and stays put after the pad", func(t *testing.T) {
		s := NewShip(22).North().Land(LandSeconds)
		s.Start(screenW, screenH)
		if opaqueCells(s.Render()) != 0 {
			t.Fatal("at t=0 the landing craft must still be off the top")
		}
		warmShip(s, LandSeconds)
		stage := s.Render()
		pad := LandPadRow(screenH)
		if opaqueCells(stage) == 0 {
			t.Fatal("at touchdown the hull must sit on the pad")
		}
		want := stripPlume(DefaultAtlas().MustFrame(sprite.Size4, sprite.N))
		for r := 0; r < want.Height; r++ {
			for c := 0; c < want.Width; c++ {
				b := want.At(r, c)
				if b.Transparent() {
					continue
				}
				if got := stage.At(pad+r, (screenW-BodyCols)/2+c); got.Ch != b.Ch {
					t.Fatalf("landed hull cell (%d,%d) missing at pad: got %q want %q", r, c, string(got.Ch), string(b.Ch))
				}
			}
		}
	})
	t.Run("unhappy: Land on a nil ship is still nil", func(t *testing.T) {
		var ghost *Ship
		if ghost.Land(LandSeconds) != nil {
			t.Fatal("Land must return the nil receiver")
		}
	})
}
