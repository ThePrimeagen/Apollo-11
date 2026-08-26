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

	"github.com/theprimeagen/apollo-11/exec-tui/components/dust"
	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
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

// plumeRowCounts sums flame glyphs per hull-relative row over the next
// frames, scanning only the columns right of the hull art (col+22 on)
// so no hull rune is ever miscounted as fire.
func plumeRowCounts(s *Ship, frames int) map[int]int {
	counts := map[int]int{}
	for i := 0; i < frames; i++ {
		s.Update(1.0 / 30)
		stage := s.Render()
		row, col := FlightPath(screenW, screenH, s.Clock())
		for r := 0; r < stage.Height; r++ {
			for c := col + 22; c < stage.Width; c++ {
				if flameGlyph(stage.At(r, c).Ch) {
					counts[r-row]++
				}
			}
		}
	}
	return counts
}

func TestWestPlumeAlignment(t *testing.T) {
	t.Run("happy: the parked plume centers on the engine bell — beam on the nozzle rows, flare above and below", func(t *testing.T) {
		s := NewShip(4).Parked()
		s.Start(screenW, screenH)
		warmShip(s, 3.0)
		counts := plumeRowCounts(s, 10)
		if counts[3] == 0 {
			t.Fatal("no fire ever flared above the bell (hull row 3) — the plume hangs one cell low")
		}
		if counts[6] == 0 {
			t.Fatal("no fire ever flared below the bell (hull row 6)")
		}
		if beam, flare := counts[4]+counts[5], counts[3]+counts[6]; beam < flare {
			t.Fatalf("the beam must ride the two nozzle rows: rows 4+5 hold %d cells, rows 3+6 hold %d", beam, flare)
		}
	})
	t.Run("unhappy: the plume never spills past the bell's lip rows", func(t *testing.T) {
		s := NewShip(4).Parked()
		s.Start(screenW, screenH)
		warmShip(s, 3.0)
		counts := plumeRowCounts(s, 10)
		for r, n := range counts {
			if r < 3 || r > 6 {
				t.Fatalf("fire painted on hull row %d (%d cells) — the box is rows 3..6 only", r, n)
			}
		}
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
		ridge := moon.HorizonTop(screenW, screenH, screenW/2)
		if pad+BodyRows-1 != ridge {
			t.Fatalf("feet at row %d, moon ridge at %d — the hull must sit one more square down, on the surface", pad+BodyRows-1, ridge)
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

func TestLandEase(t *testing.T) {
	start, end := -BodyRows, LandPadRow(screenH)
	span := float64(end - start)
	rowAt := func(tSec float64) int {
		row, _ := LandPath(screenW, screenH, tSec, LandSeconds)
		return row
	}
	progress := func(tSec float64) float64 {
		return float64(rowAt(tSec)-start) / span
	}

	t.Run("happy: the drop is fast then a long crawl onto the pad", func(t *testing.T) {
		// Ease-out quintic (and stronger): by 40% of the clock the hull
		// has already covered most of the fall, then it clinks on.
		if got := progress(0.4 * LandSeconds); got <= 0.85 {
			t.Fatalf("at 40%% of the landing the hull is only %.2f of the way — want a heavy ease-out past 0.85", got)
		}
		first := rowAt(0.2*LandSeconds) - start
		last := end - rowAt(0.8*LandSeconds)
		if last >= first {
			t.Fatalf("last 20%% of time moved %d rows, first 20%% moved %d — the finish must crawl", last, first)
		}
		s := NewShip(22).North().Land(LandSeconds)
		s.Start(screenW, screenH)
		warmShip(s, 0.4*LandSeconds)
		got, _ := s.position()
		if got != rowAt(s.Clock()) {
			t.Fatalf("a Land ship must ride the eased path, row %d want %d", got, rowAt(s.Clock()))
		}
	})
	t.Run("unhappy: a linear fall is rejected, and DropPath stays linear", func(t *testing.T) {
		mid := rowAt(LandSeconds / 2)
		linear := start + int(math.Round(0.5*span))
		if mid == linear {
			t.Fatalf("halfway row %d is the linear midpoint — the landing must ease out, not fall uniformly", mid)
		}
		if mid <= linear {
			t.Fatalf("halfway row %d is not past the linear midpoint %d — ease-out covers distance early", mid, linear)
		}
		dropStart, dropEnd := -BodyRows, screenH
		dropMid, _ := DropPath(screenW, screenH, DropSeconds/2, DropSeconds)
		dropLinear := dropStart + int(math.Round(0.5*float64(dropEnd-dropStart)))
		if dropMid != dropLinear {
			t.Fatalf("the fall scene must stay linear, mid %d want %d — landing ease must not leak", dropMid, dropLinear)
		}
		if rowAt(-2) != start {
			t.Fatal("t<0 must still be the opening mark")
		}
		snap, _ := LandPath(screenW, screenH, 1, 0)
		if snap != end {
			t.Fatalf("seconds<=0 must snap to the pad, got %d want %d", snap, end)
		}
	})
}

func TestLandThrottle(t *testing.T) {
	t.Run("happy: full until the last three stages, then ¾, ½, ¼, off at the pad", func(t *testing.T) {
		if ThrottleStageSeconds >= 1.0 {
			t.Fatalf("ThrottleStageSeconds is %v — the booster must step down faster than the old 1s stages", ThrottleStageSeconds)
		}
		if LandThrottleLead != 3*ThrottleStageSeconds {
			t.Fatalf("LandThrottleLead is %v, want 3 stages of %v", LandThrottleLead, ThrottleStageSeconds)
		}
		if got := LandThrottle(0, LandSeconds); got != 1 {
			t.Fatalf("opening throttle %v, want full strength", got)
		}
		if got := LandThrottle(LandSeconds-LandThrottleLead-0.01, LandSeconds); got != 1 {
			t.Fatalf("still approaching, throttle %v, want full", got)
		}
		step := LandThrottleLead / 3
		if got := LandThrottle(LandSeconds-LandThrottleLead+step/2, LandSeconds); got != 0.75 {
			t.Fatalf("first interval throttle %v, want 0.75", got)
		}
		if got := LandThrottle(LandSeconds-2*step+step/2, LandSeconds); got != 0.5 {
			t.Fatalf("second interval throttle %v, want 0.5", got)
		}
		if got := LandThrottle(LandSeconds-step/2, LandSeconds); got != 0.25 {
			t.Fatalf("third interval throttle %v, want 0.25", got)
		}
		if got := LandThrottle(LandSeconds, LandSeconds); got != 0 {
			t.Fatalf("touchdown throttle %v, want off", got)
		}
	})
	t.Run("happy: a landing ship scales count and distance, then cuts the fire on the pad", func(t *testing.T) {
		s := NewShip(22).North().Land(LandSeconds)
		s.Start(screenW, screenH)
		base := s.Flame.Config()
		if base.Count < 1 {
			t.Fatal("a landing ship must arm a live booster")
		}
		warmShip(s, 0.5)
		if got := s.Flame.Config().Count; got != base.Count {
			t.Fatalf("still far from the pad, count %d, want full %d", got, base.Count)
		}
		warmShip(s, LandSeconds-LandThrottleLead/6-0.5)
		got := s.Flame.Config()
		if got.Count != int(math.Round(float64(base.Count)*0.25)) {
			t.Fatalf("last interval count %d, want ¼ of %d", got.Count, base.Count)
		}
		if base.MaxDistance > 0 && got.MaxDistance > base.MaxDistance*0.25+1e-9 {
			t.Fatalf("last interval max distance %v, want ¼ of %v", got.MaxDistance, base.MaxDistance)
		}
		warmShip(s, LandThrottleLead/6+0.5)
		if s.Flame == nil {
			t.Fatal("the flame engine stays attached; it just emits nothing")
		}
		if s.Flame.Config().Count != 0 {
			t.Fatalf("on the pad count %d, want 0", s.Flame.Config().Count)
		}
		if n := len(s.Flame.Eng.Particles); n != 0 {
			t.Fatalf("on the pad %d particles still live — the fire must cut off", n)
		}
		stage := s.Render()
		for r := 0; r < stage.Height; r++ {
			for c := 0; c < stage.Width; c++ {
				cell := stage.At(r, c)
				switch cell.Ch {
				case '⠁', '⠒', '⠶':
					if cell.FG >= dust.GrayMin {
						// The touchdown kick's braille wears pad
						// grays; only hot inks are the booster.
						continue
					}
					t.Fatalf("fire still painting at (%d,%d) after touchdown", r, c)
				}
			}
		}
	})
	t.Run("unhappy: a fly-in or a drop never throttles, and time past the pad stays off", func(t *testing.T) {
		if got := LandThrottle(-2, LandSeconds); got != 1 {
			t.Fatalf("t<0 throttle %v, want full, not off", got)
		}
		if got := LandThrottle(LandSeconds+40, LandSeconds); got != 0 {
			t.Fatalf("long after touchdown throttle %v, want off", got)
		}
		if got := LandThrottle(1, 0); got != 0 {
			t.Fatalf("no landing duration throttle %v, want off", got)
		}
		west := NewShip(9).Parked()
		west.Start(screenW, screenH)
		want := west.Flame.Config().Count
		warmShip(west, LandSeconds+1)
		if west.Flame.Config().Count != want {
			t.Fatalf("a westbound parked ship must not throttle, count %d want %d", west.Flame.Config().Count, want)
		}
		drop := NewShip(10).North().Drop(DropSeconds)
		drop.Start(screenW, screenH)
		dropWant := drop.Flame.Config().Count
		warmShip(drop, DropSeconds/2)
		if drop.Flame.Config().Count != dropWant {
			t.Fatalf("a falling ship must not throttle, count %d want %d", drop.Flame.Config().Count, dropWant)
		}
	})
	t.Run("happy: ThrottleStage shortens the lead so the booster steps down faster", func(t *testing.T) {
		s := NewShip(22).North().Land(LandSeconds).ThrottleStage(0.2)
		s.Start(screenW, screenH)
		base := s.Flame.Config().Count
		if base < 1 {
			t.Fatal("test premise: a landing ship must arm a live booster")
		}
		// Default lead is 1.2s; a 0.2s stage is a 0.6s lead. At 0.7s
		// remaining the stock ship is already in ½, the short-stage
		// ship is still full.
		warmShip(s, LandSeconds-0.7)
		if got := s.Flame.Config().Count; got != base {
			t.Fatalf("0.7s from the pad a 0.2s-stage ship must still be full, count %d want %d", got, base)
		}
		stock := NewShip(23).North().Land(LandSeconds)
		stock.Start(screenW, screenH)
		warmShip(stock, LandSeconds-0.7)
		if got := stock.Flame.Config().Count; got == base {
			t.Fatal("0.7s from the pad the stock ship must already have stepped down")
		}
	})
	t.Run("unhappy: ThrottleStage on nil stays nil, and a non-positive stage keeps the stock lead", func(t *testing.T) {
		var ghost *Ship
		if ghost.ThrottleStage(0.2) != nil {
			t.Fatal("ThrottleStage must return the nil receiver")
		}
		s := NewShip(24).North().Land(LandSeconds).ThrottleStage(0)
		s.Start(screenW, screenH)
		base := s.Flame.Config().Count
		warmShip(s, LandSeconds-0.7)
		if got := s.Flame.Config().Count; got == base {
			t.Fatal("ThrottleStage(0) must keep the stock lead — 0.7s from the pad the booster has already stepped down")
		}
	})
	t.Run("happy: ThrottleAt times ¾, ½, ¼, and off from t=0, independent of the pad", func(t *testing.T) {
		s := NewShip(40).North().Land(LandSeconds).ThrottleAt(1.0, 2.0, 3.0, 4.0)
		s.Start(screenW, screenH)
		base := s.Flame.Config().Count
		if base < 1 {
			t.Fatal("test premise: a landing ship must arm a live booster")
		}
		warmShip(s, 0.5)
		if got := s.Flame.Config().Count; got != base {
			t.Fatalf("before ¾ the booster must be full, count %d want %d", got, base)
		}
		warmShip(s, 0.7)
		if got := s.Flame.Config().Count; got != int(math.Round(float64(base)*0.75)) {
			t.Fatalf("at ¾ count %d, want ¾ of %d", got, base)
		}
		warmShip(s, 1.0)
		if got := s.Flame.Config().Count; got != int(math.Round(float64(base)*0.5)) {
			t.Fatalf("at ½ count %d, want ½ of %d", got, base)
		}
		warmShip(s, 1.0)
		if got := s.Flame.Config().Count; got != int(math.Round(float64(base)*0.25)) {
			t.Fatalf("at ¼ count %d, want ¼ of %d", got, base)
		}
		warmShip(s, 1.1)
		if s.Flame.Config().Count != 0 {
			t.Fatalf("past fire-off count %d, want 0 — the pad is still 1s away", s.Flame.Config().Count)
		}
	})
	t.Run("unhappy: ThrottleAt on nil stays nil, and off at t=0 never lights", func(t *testing.T) {
		var ghost *Ship
		if ghost.ThrottleAt(1, 2, 3, 4) != nil {
			t.Fatal("ThrottleAt must return the nil receiver")
		}
		s := NewShip(41).North().Land(LandSeconds).ThrottleAt(1, 2, 3, 0)
		s.Start(screenW, screenH)
		warmShip(s, 0.5)
		if s.Flame.Config().Count != 0 {
			t.Fatalf("fire-off at 0 must keep the booster dark, count %d", s.Flame.Config().Count)
		}
	})
}

// dustRune reports a dust-ladder glyph — braille or a shade block.
// Beyond the hull columns no fire or hull rune can appear (the north
// plume lives in a 12-column box under the bell), so any such glyph
// out there is the landing kick.
func dustRune(ch rune) bool {
	return (ch >= '⠀' && ch <= '⣿') || ch == '░' || ch == '▒'
}

// kickedDust scans the stage beyond the hull columns and reports dust
// left and right of the hull, plus the row band it painted in.
func kickedDust(stage sprite.Sprite, hullCol int) (left, right bool, minRow, maxRow int) {
	minRow, maxRow = stage.Height, -1
	for r := 0; r < stage.Height; r++ {
		for c := 0; c < stage.Width; c++ {
			if c >= hullCol && c < hullCol+BodyCols {
				continue
			}
			if !dustRune(stage.At(r, c).Ch) {
				continue
			}
			if c < hullCol {
				left = true
			} else {
				right = true
			}
			if r < minRow {
				minRow = r
			}
			if r > maxRow {
				maxRow = r
			}
		}
	}
	return left, right, minRow, maxRow
}

// noKick asserts a stage with no dust beyond the hull columns.
func noKick(t *testing.T, s *Ship, when string) {
	t.Helper()
	if l, r, _, _ := kickedDust(s.Render(), centerCol); l || r {
		t.Fatalf("%s: dust on stage, left=%v right=%v", when, l, r)
	}
}

// TestLandDust: a landing kicks one continuous pad cloud — mirrored
// engines blowing out of the surface on both sides of the booster,
// leftward and rightward, climbing away from the bell — the moment
// the booster starts slowing the craft (LandThrottleLead before the
// pad). The cloud keeps emitting through every throttle step and
// through touchdown; only when the booster cuts off does it count
// its particles down to zero over dust.FadeSeconds. There is no
// second kick.
func TestLandDust(t *testing.T) {
	surface := screenH - landSurfaceRows // the ridge row the feet land on
	t.Run("happy: the pad blows dust out both sides the moment the throttle steps", func(t *testing.T) {
		s := NewShip(30).North().Land(LandSeconds)
		s.Start(screenW, screenH)
		warmShip(s, LandSeconds-LandThrottleLead-0.1)
		noKick(t, s, "before the craft starts slowing down")
		warmShip(s, 0.2)
		l, r, minRow, maxRow := kickedDust(s.Render(), centerCol)
		if !l || !r {
			t.Fatalf("the pad must blow dust out both sides, left=%v right=%v", l, r)
		}
		if maxRow < surface-2 || maxRow > surface+2 {
			t.Fatalf("the kick must hug the pad surface (row %d), lowest dust on row %d", surface, maxRow)
		}
		if minRow < screenH/3 {
			t.Fatalf("dust climbed to row %d — a ground kick, not a sky plume", minRow)
		}
		if s.padDust == nil || s.padDust.Left == nil || s.padDust.Right == nil {
			t.Fatal("the pad kick must be a live two-engine cloud")
		}
		ld := s.padDust.Left.Cfg.Direction
		rd := s.padDust.Right.Cfg.Direction
		if ld.X >= 0 || ld.Y >= 0 || rd.X <= 0 || rd.Y >= 0 {
			t.Fatalf("the engines must climb away mirrored — left %+v, right %+v", ld, rd)
		}
		bell := float64(centerCol) + float64(BodyCols)/2
		if lx, rx := s.padDust.Left.Cfg.Origin.X, s.padDust.Right.Cfg.Origin.X; lx >= bell || rx <= bell {
			t.Fatalf("the nozzles must sit on both sides of the booster (col %.0f): left %.1f, right %.1f", bell, lx, rx)
		}
	})
	t.Run("happy: the same cloud keeps blowing through every stage and past the pad until the fade runs out", func(t *testing.T) {
		s := NewShip(31).North().Land(LandSeconds)
		s.Start(screenW, screenH)
		warmShip(s, LandSeconds-LandThrottleLead+0.15)
		opening := livePad(s)
		if opening == 0 {
			t.Fatal("test premise: the pad must already be dusty")
		}
		step := LandThrottleLead / 3
		for _, extra := range []float64{step, step, step} { // ¾, ½, ¼, onto the pad
			warmShip(s, extra)
			if l, r, _, _ := kickedDust(s.Render(), centerCol); !l || !r {
				t.Fatalf("at t=%.2f the pad must still blow both ways, left=%v right=%v", s.Clock(), l, r)
			}
			if n := livePad(s); n == 0 {
				t.Fatalf("at t=%.2f the cloud went silent — it must run continuously through off", s.Clock())
			}
		}
		if s.padDust == nil {
			t.Fatal("touchdown keeps the same cloud; it does not arm a second burst")
		}
		mid := livePad(s)
		warmShip(s, dust.FadeSeconds/2)
		if got := livePad(s); got == 0 || got >= mid {
			t.Fatalf("mid-fade %d specks, opening-on-pad %d — the countdown must be falling, not gone and not holding", got, mid)
		}
		if l, r, _, _ := kickedDust(s.Render(), centerCol); !l || !r {
			t.Fatalf("mid-fade the blown fringe must still drift beyond the hull, left=%v right=%v", l, r)
		}
		warmShip(s, dust.FadeSeconds/2+0.3)
		noKick(t, s, "past the fade after booster-off")
		if n := livePad(s); n != 0 {
			t.Fatalf("%d specks still live after the countdown, want zero", n)
		}
	})
	t.Run("happy: DustAt times the cloud from an offset for a run, independent of the booster", func(t *testing.T) {
		s := NewShip(37).North().Land(LandSeconds).DustAt(1.0, 0.40)
		s.Start(screenW, screenH)
		warmShip(s, 0.95)
		noKick(t, s, "before the dust offset")
		warmShip(s, 0.15)
		if l, r, _, _ := kickedDust(s.Render(), centerCol); !l || !r {
			t.Fatalf("at the offset the pad must blow both ways, left=%v right=%v", l, r)
		}
		warmShip(s, 0.20)
		if l, r, _, _ := kickedDust(s.Render(), centerCol); !l || !r {
			t.Fatalf("mid-run the cloud must still blow, left=%v right=%v", l, r)
		}
	})
	t.Run("happy: DustLoss drains the cloud as the engines start cutting, not in a blink", func(t *testing.T) {
		s := NewShip(42).North().Land(LandSeconds).
			ThrottleAt(1.0, 2.0, 3.0, 4.0).
			DustAt(0.2, 8.0).
			DustLoss(0.05)
		s.Start(screenW, screenH)
		warmShip(s, 0.9)
		held := livePad(s)
		if held == 0 {
			t.Fatal("test premise: the pad must already be dusty before ¾")
		}
		warmShip(s, 0.3)
		got := livePad(s)
		if got == 0 {
			t.Fatal("0.3s after ¾ the cloud must still hold specks — a drain, not a blink")
		}
		if got >= held {
			t.Fatalf("after ¾ %d specks, held %d — the drain must already be falling", got, held)
		}
		if l, r, _, _ := kickedDust(s.Render(), centerCol); !l || !r {
			t.Fatalf("mid-drain the fringe must still blow both ways, left=%v right=%v", l, r)
		}
	})
	t.Run("unhappy: DustAt on nil stays nil, and a non-positive run never kicks", func(t *testing.T) {
		var ghost *Ship
		if ghost.DustAt(1, 1) != nil {
			t.Fatal("DustAt must return the nil receiver")
		}
		s := NewShip(38).North().Land(LandSeconds).DustAt(0.2, 0)
		s.Start(screenW, screenH)
		warmShip(s, LandSeconds)
		noKick(t, s, "DustAt run 0")
		if s.padDust != nil {
			t.Fatal("a zero run must never arm a cloud")
		}
	})
	t.Run("happy: Dust shortens how long the particles last after the booster cuts", func(t *testing.T) {
		s := NewShip(35).North().Land(LandSeconds).Dust(0.4)
		s.Start(screenW, screenH)
		warmShip(s, LandSeconds+0.05)
		if l, r, _, _ := kickedDust(s.Render(), centerCol); !l || !r {
			t.Fatalf("on the pad the cloud must still be blowing, left=%v right=%v", l, r)
		}
		warmShip(s, 0.5)
		noKick(t, s, "a 0.4s linger must be gone half a second after booster-off")
		stock := NewShip(36).North().Land(LandSeconds)
		stock.Start(screenW, screenH)
		warmShip(stock, LandSeconds+0.5)
		if l, r, _, _ := kickedDust(stock.Render(), centerCol); !l || !r {
			t.Fatalf("the stock linger must still be blowing 0.5s after off, left=%v right=%v", l, r)
		}
	})
	t.Run("unhappy: DustLoss on nil stays nil, and a zero rate does not blink the cloud out", func(t *testing.T) {
		var ghost *Ship
		if ghost.DustLoss(0.05) != nil {
			t.Fatal("DustLoss must return the nil receiver")
		}
		s := NewShip(43).North().Land(LandSeconds).
			ThrottleAt(0.4, 0.5, 0.6, 0.7).
			DustAt(0.1, 8.0).
			DustLoss(0)
		s.Start(screenW, screenH)
		warmShip(s, 0.35)
		held := livePad(s)
		if held == 0 {
			t.Fatal("test premise: the pad must already be dusty")
		}
		warmShip(s, 0.4)
		if got := livePad(s); got == 0 {
			t.Fatalf("zero loss must not trim the live specks away, held %d now %d", held, got)
		}
	})
	t.Run("unhappy: a fall and a dark landing never kick dust", func(t *testing.T) {
		drop := NewShip(32).North().Drop(DropSeconds)
		drop.Start(screenW, screenH)
		for i := 0; i < int(DropSeconds*30); i++ {
			drop.Update(1.0 / 30)
			if i%15 == 0 {
				noKick(t, drop, "a falling ship")
			}
		}
		if drop.padDust != nil {
			t.Fatal("a falling ship must never arm a dust kick")
		}
		dark := NewShip(33).Dark().North().Land(LandSeconds)
		dark.Start(screenW, screenH)
		warmShip(dark, LandSeconds+0.5)
		noKick(t, dark, "a dark landing")
		if dark.padDust != nil {
			t.Fatal("a cold engine kicks no dust")
		}
	})
	t.Run("unhappy: Stop drops the dust, Dust on nil stays nil, and a restart long after never replays", func(t *testing.T) {
		var ghost *Ship
		if ghost.Dust(0.4) != nil {
			t.Fatal("Dust must return the nil receiver")
		}
		s := NewShip(34).North().Land(LandSeconds)
		s.Start(screenW, screenH)
		warmShip(s, LandSeconds+0.2)
		if s.padDust == nil {
			t.Fatal("test premise: the pad cloud must be live")
		}
		s.Stop()
		if s.padDust != nil {
			t.Fatal("Stop must drop the pad cloud for the collector")
		}
		warmShip(s, dust.FadeSeconds)
		if s.padDust != nil {
			t.Fatal("a stopped ship must not re-arm a kick")
		}
		s.Start(screenW, screenH)
		warmShip(s, 0.5)
		noKick(t, s, "a restart long after touchdown")
		if s.padDust != nil {
			t.Fatal("a cloud whose fade is over must never replay")
		}
	})
}

// livePad is how many specks the pad cloud holds this instant.
func livePad(s *Ship) int {
	if s == nil || s.padDust == nil || s.padDust.Left == nil || s.padDust.Right == nil {
		return 0
	}
	return len(s.padDust.Left.Particles) + len(s.padDust.Right.Particles)
}
