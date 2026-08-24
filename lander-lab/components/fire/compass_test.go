package fire

import (
	"math"
	"os"
	"testing"

	"github.com/theprimeagen/apollo-11/lander-lab/particle"
)

func TestCourses(t *testing.T) {
	t.Run("happy: eight directions, clockwise from north", func(t *testing.T) {
		cs := Courses()
		if len(cs) != 8 {
			t.Fatalf("courses %d, want 8", len(cs))
		}
		if cs[0].Name != "N" || cs[2].Name != "E" || cs[4].Name != "S" {
			t.Fatalf("order %v, want N … E … S …", names(cs))
		}
	})
	t.Run("unhappy: a zero course is rejected", func(t *testing.T) {
		if err := Toward(1, particle.Vec2{}).Eng.Validate(); err == nil {
			t.Fatal("zero direction must fail Validate")
		}
	})
}

func TestToward(t *testing.T) {
	t.Run("happy: south travels top to bottom", func(t *testing.T) {
		f := Toward(2, course("S").Dir)
		f.Update(0.01)
		_, oy := avgPos(f.Eng.Particles)
		f.Update(0.12)
		_, ay := avgPos(f.Eng.Particles)
		if ay <= oy {
			t.Fatalf("top-to-bottom should increase y, %.2f → %.2f", oy, ay)
		}
	})
	t.Run("happy: east travels left to right", func(t *testing.T) {
		f := Toward(3, course("E").Dir)
		f.Update(0.01)
		ox, _ := avgPos(f.Eng.Particles)
		f.Update(0.12)
		ax, _ := avgPos(f.Eng.Particles)
		if ax <= ox {
			t.Fatalf("left-to-right should increase x, %.2f → %.2f", ox, ax)
		}
	})
	t.Run("happy: northeast is a 45° climb-right", func(t *testing.T) {
		f := Toward(4, course("NE").Dir)
		d := f.Eng.Cfg.Direction
		if math.Abs(math.Abs(d.X)-math.Abs(d.Y)) > 1e-9 {
			t.Fatalf("NE should be 45°, got %+v", d)
		}
		f.Update(0.01)
		ox, oy := avgPos(f.Eng.Particles)
		f.Update(0.12)
		ax, ay := avgPos(f.Eng.Particles)
		if ax <= ox || ay >= oy {
			t.Fatalf("NE should increase x and decrease y, (%.2f,%.2f)→(%.2f,%.2f)", ox, oy, ax, ay)
		}
	})
}

func TestCompass(t *testing.T) {
	t.Run("happy: the rose is a fixed canvas with all eight labels", func(t *testing.T) {
		c := NewCompass(5)
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
	t.Run("unhappy: a zero-frame compass tape is an error", func(t *testing.T) {
		if _, err := WriteCompassTape(t.TempDir(), NewCompass(1), 0, 8); err == nil {
			t.Fatal("n<=0 must fail")
		}
	})
	t.Run("unhappy: a missing directory is an error", func(t *testing.T) {
		c := NewCompass(1)
		c.Update(0.02)
		if err := WritePNG("/no/such/dir/rose.png", c.View(), 8); err == nil {
			t.Fatal("unwritable path must fail")
		}
	})
}

func TestWriteCompassTape(t *testing.T) {
	t.Run("happy: writes n same-size frames", func(t *testing.T) {
		dir := t.TempDir()
		paths, err := WriteCompassTape(dir, NewCompass(6), 2, 8)
		if err != nil {
			t.Fatal(err)
		}
		if len(paths) != 2 {
			t.Fatalf("paths %d, want 2", len(paths))
		}
		for i, p := range paths {
			st, err := os.Stat(p)
			if err != nil || st.Size() == 0 {
				t.Fatalf("frame %d missing: %v", i, err)
			}
		}
	})
}

func course(name string) Course {
	for _, c := range Courses() {
		if c.Name == name {
			return c
		}
	}
	return Course{}
}

func names(cs []Course) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Name
	}
	return out
}
