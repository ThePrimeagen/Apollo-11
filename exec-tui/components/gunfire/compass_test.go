package gunfire

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestCourses(t *testing.T) {
	t.Run("happy: eight directions, clockwise from north", func(t *testing.T) {
		cs := Courses()
		if len(cs) != 8 {
			t.Fatalf("courses %d, want 8", len(cs))
		}
		if cs[0].Name != "N" || cs[2].Name != "E" || cs[4].Name != "S" {
			t.Fatalf("order %v, want N … E … S …", []string{cs[0].Name, cs[2].Name, cs[4].Name})
		}
		if cs[0].Heading != sprite.N || cs[1].Heading != sprite.NE {
			t.Fatalf("headings %s %s, want N NE", cs[0].Heading, cs[1].Heading)
		}
	})
	t.Run("unhappy: a zero course is rejected", func(t *testing.T) {
		if err := ConfigToward(particle.Vec2{}, DefaultBlast().Core).Validate(); err == nil {
			t.Fatal("zero direction must fail Validate")
		}
	})
}

func TestConfigToward(t *testing.T) {
	t.Run("happy: south travels top to bottom from the incoming wall", func(t *testing.T) {
		cfg := ConfigToward(dirOf(sprite.S), DefaultBlast().ShotAt(sprite.S).Layer)
		if err := cfg.Validate(); err != nil {
			t.Fatalf("south panel must validate: %v", err)
		}
		if cfg.Origin.Y >= panelBox/2 {
			t.Fatalf("south origin Y %.2f must sit on the incoming (top) wall", cfg.Origin.Y)
		}
	})
	t.Run("happy: east travels left to right from the incoming wall", func(t *testing.T) {
		cfg := ConfigToward(dirOf(sprite.E), DefaultBlast().ShotAt(sprite.E).Layer)
		if cfg.Origin.X >= panelBox/2 {
			t.Fatalf("east origin X %.2f must sit on the incoming (left) wall", cfg.Origin.X)
		}
	})
}

func TestCompass(t *testing.T) {
	t.Run("happy: the rose is a fixed canvas with all eight labels", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		c := NewCompass(5)
		c.Fire()
		c.Update(0.05)
		sp := c.View()
		if sp.Width != CompassCols || sp.Height != CompassRows {
			t.Fatalf("compass %dx%d, want %dx%d", sp.Width, sp.Height, CompassCols, CompassRows)
		}
		got := map[rune]bool{}
		for r := 0; r < sp.Height; r++ {
			for col := 0; col < sp.Width; col++ {
				got[sp.At(r, col).Ch] = true
			}
		}
		for _, name := range []string{"N", "E", "S", "W"} {
			if !got[[]rune(name)[0]] {
				t.Fatalf("compass missing label %s", name)
			}
		}
	})
	t.Run("happy: Fire lights every heading at once", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		c := NewCompass(7)
		if c.Live() != 0 {
			t.Fatal("a fresh rose must hold fire")
		}
		c.Fire()
		if c.Live() == 0 {
			t.Fatal("Fire must light the whole rose")
		}
		for _, s := range c.Slots {
			if len(s.Flame.Particles) == 0 {
				t.Fatalf("the %s flame held fire — every heading fires together", s.Name)
			}
		}
	})
	t.Run("happy: a retune mid-burn wears the next squeeze", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		c := NewCompass(7)
		blast := ActiveBlast()
		east := blast.ShotAt(sprite.E)
		east.Count = 11
		blast.SetShot(sprite.E, east)
		if err := UseBlast(blast); err != nil {
			t.Fatalf("UseBlast: %v", err)
		}
		c.Update(1.0 / 30)
		c.Fire()
		for _, s := range c.Slots {
			if s.Heading != sprite.E {
				continue
			}
			if got := len(s.Flame.Particles); got != 11 {
				t.Fatalf("the E panel burst %d, want the retuned 11", got)
			}
		}
	})
	t.Run("unhappy: a nil compass skips every cue", func(t *testing.T) {
		var ghost *Compass
		ghost.Fire()
		ghost.Update(0.1)
		if sp := ghost.View(); sp.Width != CompassCols || sp.Height != CompassRows {
			t.Fatalf("a nil rose still hands back a %dx%d board, got %dx%d", CompassCols, CompassRows, sp.Width, sp.Height)
		}
		if ghost.Live() != 0 {
			t.Fatal("a nil rose holds no fire")
		}
	})
}
