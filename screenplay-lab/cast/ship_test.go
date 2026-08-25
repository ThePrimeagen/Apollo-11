package cast

// Tests written FIRST: the ship is the full zoomed-in (size-4, 26×10)
// craft flying west — nose left, tail right — with the atlas's baked
// tilde plume stripped and the live left-to-right booster fire trailing
// off the tail. FlightPath is the whole choreography: fully off the
// right wing at t=0, an eased slide that parks at center stage by
// FlyInSeconds, then a ±1-cell sine bobble with a 10-second period.
// Update moves the clock; Render writes styled cells into the screen.

import (
	"math"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/lander-lab/sprite"

	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

const (
	screenW = 72
	screenH = 28
)

// centerRow/centerCol is where the hull's top-left parks on the test screen.
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
func flameGlyph(s string) bool {
	switch s {
	case "⠁", "⠒", "⠶", "░", "▒", "▄", "▓", "█":
		return true
	}
	return false
}

func TestNewShip(t *testing.T) {
	t.Run("happy: the body is the size-4 west frame minus its baked plume", func(t *testing.T) {
		s := NewShip(1)
		if s.Body.Width != BodyCols || s.Body.Height != BodyRows {
			t.Fatalf("body %dx%d, want %dx%d", s.Body.Width, s.Body.Height, BodyCols, BodyRows)
		}
		want := sprite.Default().MustFrame(sprite.Size4, sprite.W)
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
	t.Run("happy: the fire is valid and aimed left-to-right", func(t *testing.T) {
		s := NewShip(2)
		if err := s.Flame.Eng.Validate(); err != nil {
			t.Fatalf("flame config: %v", err)
		}
		d := s.Flame.Eng.Cfg.Direction
		if math.Abs(d.X-1) > 1e-9 || math.Abs(d.Y) > 1e-9 {
			t.Fatalf("direction %+v, want (1, 0) — fire out the right side", d)
		}
	})
	t.Run("unhappy: before any tick the whole screen stays dark", func(t *testing.T) {
		s := NewShip(3)
		scr := screenplay.NewScreen(screenW, screenH)
		s.Render(scr)
		if n := litCount(scr); n != 0 {
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
	t.Run("unhappy: a screen smaller than the craft still answers", func(t *testing.T) {
		if row, col := FlightPath(8, 3, 100); col > 8 {
			t.Fatalf("tiny screen answered (%d,%d) — parked col can never sit right of the screen", row, col)
		}
	})
}

func TestShipOnScreen(t *testing.T) {
	t.Run("happy: a warmed ship parks with styled hull cells and fire off the tail", func(t *testing.T) {
		s := NewShip(4)
		warmShip(s, 5.0)
		if math.Abs(s.Clock()-5.0) > 1e-9 {
			t.Fatalf("clock %f, want 5.0", s.Clock())
		}
		scr := screenplay.NewScreen(screenW, screenH)
		s.Render(scr)
		row, col := FlightPath(screenW, screenH, s.Clock())
		for r := 0; r < s.Body.Height; r++ {
			for c := 0; c < s.Body.Width; c++ {
				b := s.Body.At(r, c)
				if b.Transparent() {
					continue
				}
				got := scr.Cell(col+c, row+r)
				if got == nil || got.Content != string(b.Ch) {
					t.Fatalf("hull cell (%d,%d) burned or moved: %q -> %+v", r, c, string(b.Ch), got)
				}
				if b.FG >= 0 && got.Style.Fg != ansi.IndexedColor(uint8(b.FG)) {
					t.Fatalf("hull ink at (%d,%d): %v, want indexed %d", r, c, got.Style.Fg, b.FG)
				}
				if b.BG >= 0 && got.Style.Bg != ansi.IndexedColor(uint8(b.BG)) {
					t.Fatalf("hull paper at (%d,%d): %v, want indexed %d", r, c, got.Style.Bg, b.BG)
				}
			}
		}
		fire := 0
		for y := 0; y < screenH; y++ {
			for x := col + BodyCols; x < screenW; x++ {
				if flameGlyph(contentAt(scr, x, y)) {
					fire++
				}
			}
		}
		if fire == 0 {
			t.Fatal("no fire right of the hull — the plume must trail the tail")
		}
	})
	t.Run("happy: the baked tilde plume never reaches the screen", func(t *testing.T) {
		s := NewShip(5)
		warmShip(s, 5.0)
		scr := screenplay.NewScreen(screenW, screenH)
		s.Render(scr)
		for y := 0; y < screenH; y++ {
			for x := 0; x < screenW; x++ {
				if got := contentAt(scr, x, y); got == "~" || got == "≈" {
					t.Fatalf("static plume glyph %q at (%d,%d); the live fire is the plume", got, x, y)
				}
			}
		}
	})
	t.Run("unhappy: dt<=0 holds the clock and the mark", func(t *testing.T) {
		s := NewShip(6)
		s.Update(0)
		s.Update(-1)
		if s.Clock() != 0 {
			t.Fatalf("clock %f moved on dt<=0", s.Clock())
		}
		scr := screenplay.NewScreen(screenW, screenH)
		s.Render(scr)
		if n := litCount(scr); n != 0 {
			t.Fatalf("held ship lit %d cells, want 0 (still offstage)", n)
		}
	})
	t.Run("unhappy: a screen smaller than the craft clips instead of panicking", func(t *testing.T) {
		s := NewShip(7)
		warmShip(s, 5.0)
		scr := screenplay.NewScreen(8, 3)
		s.Render(scr)
		if n := litCount(scr); n > 8*3 {
			t.Fatalf("tiny screen lit %d cells, has only %d", n, 8*3)
		}
	})
	t.Run("unhappy: a nil ship and a nil screen both skip the cue", func(t *testing.T) {
		var ghost *Ship
		ghost.Update(0.1)
		ghost.Render(screenplay.NewScreen(4, 2))
		NewShip(8).Render(nil)
	})
}
