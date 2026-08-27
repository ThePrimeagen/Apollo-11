package gunfire

// Tests written FIRST. BlastConfig is the JSON that tunes the Doom
// muzzle flame around the eight compass directions: the muzzle, the
// heading the next squeeze fires (one of N, NE, E, SE, S, SW, W, NW —
// the same eight the lander atlas uses), the two-frame pulse, the
// core brightness ladder, the shared white-hot core, and one Shot per
// direction — the full engine-knob Layer plus the five-stop color
// ramp that direction's flame cools through. Every direction is its
// own tune: the North shot can burn Doom red while the East shot
// burns plasma blue. UseBlast puts a config in effect for every blast
// reading it, so the tuner and the demo stay on the same values.

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestDefaultBlast(t *testing.T) {
	t.Run("happy: the stock blast validates, aims north, and every direction wears the Doom red ramp", func(t *testing.T) {
		c := DefaultBlast()
		if err := c.Validate(); err != nil {
			t.Fatalf("the stock blast must validate: %v", err)
		}
		if c.Heading != sprite.N {
			t.Fatalf("the stock flame leaps straight up — heading %q, want N", c.Heading)
		}
		red := [5]int{226, 208, 196, 160, 124}
		for _, h := range sprite.Headings {
			shot := c.ShotAt(h)
			if shot.Colors != red {
				t.Fatalf("the stock %s shot wears %v, want the Doom ramp %v", h, shot.Colors, red)
			}
			if shot.Lift <= 0 || shot.Drag <= 0 {
				t.Fatalf("the %s flame must rise and die down — lift %v drag %v", h, shot.Lift, shot.Drag)
			}
			if shot != c.ShotAt(sprite.N) {
				t.Fatalf("the stock shots must start uniform, %s differs: %+v", h, shot)
			}
		}
		if c.PulseDelay <= 0 || c.PulseFrac <= 0 {
			t.Fatalf("Doom's flash is two frames — pulse delay %v frac %v", c.PulseDelay, c.PulseFrac)
		}
	})
	t.Run("happy: the stock blast is the active blast at boot", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		if ActiveBlast() != DefaultBlast() {
			t.Fatalf("active blast %+v, want the stock %+v", ActiveBlast(), DefaultBlast())
		}
	})
	t.Run("happy: SetShot retunes one direction and leaves the other seven alone", func(t *testing.T) {
		c := DefaultBlast()
		shot := c.ShotAt(sprite.E)
		shot.Count = 999
		shot.Colors = [5]int{21, 27, 33, 39, 45}
		c.SetShot(sprite.E, shot)
		if got := c.ShotAt(sprite.E); got != shot {
			t.Fatalf("the E shot must carry the retune, got %+v", got)
		}
		if c.ShotAt(sprite.N) != DefaultBlast().ShotAt(sprite.N) {
			t.Fatal("retuning E must not touch N")
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("a blue E shot must be legal: %v", err)
		}
	})
	t.Run("unhappy: every engine is a one-shot — no period anywhere", func(t *testing.T) {
		core, flames := DefaultBlast().Engines(120, 60)
		if core.Period != 0 {
			t.Fatalf("the core auto-emits every %vs — a gunshot is a trigger, not a clock", core.Period)
		}
		for i, cfg := range flames {
			if cfg.Period != 0 {
				t.Fatalf("the %s flame auto-emits every %vs", sprite.Headings[i], cfg.Period)
			}
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("happy: every compass heading is a legal aim", func(t *testing.T) {
		for _, h := range sprite.Headings {
			c := DefaultBlast()
			c.Heading = h
			if err := c.Validate(); err != nil {
				t.Fatalf("heading %s must be legal: %v", h, err)
			}
		}
	})
	t.Run("unhappy: a heading off the compass is rejected", func(t *testing.T) {
		c := DefaultBlast()
		c.Heading = "NNE"
		if err := c.Validate(); !errors.Is(err, ErrHeading) {
			t.Fatalf("got %v, want ErrHeading", err)
		}
		c.Heading = ""
		if err := c.Validate(); !errors.Is(err, ErrHeading) {
			t.Fatalf("an empty heading: got %v, want ErrHeading", err)
		}
	})
	t.Run("unhappy: a muzzle outside the stage is rejected", func(t *testing.T) {
		c := DefaultBlast()
		c.MuzzleX = -0.1
		if err := c.Validate(); !errors.Is(err, ErrMuzzle) {
			t.Fatalf("got %v, want ErrMuzzle", err)
		}
		c = DefaultBlast()
		c.MuzzleY = 1.2
		if err := c.Validate(); !errors.Is(err, ErrMuzzle) {
			t.Fatalf("got %v, want ErrMuzzle", err)
		}
	})
	t.Run("unhappy: a negative pulse delay or an out-of-range pulse fraction is rejected", func(t *testing.T) {
		c := DefaultBlast()
		c.PulseDelay = -0.1
		if err := c.Validate(); !errors.Is(err, ErrDelay) {
			t.Fatalf("got %v, want ErrDelay", err)
		}
		c = DefaultBlast()
		c.PulseFrac = 1.1
		if err := c.Validate(); !errors.Is(err, ErrPulse) {
			t.Fatalf("got %v, want ErrPulse", err)
		}
	})
	t.Run("unhappy: a folded core ladder is rejected", func(t *testing.T) {
		c := DefaultBlast()
		c.MidAt = c.EdgeAt
		if err := c.Validate(); !errors.Is(err, ErrLadder) {
			t.Fatalf("mid<=edge: got %v, want ErrLadder", err)
		}
		c = DefaultBlast()
		c.EdgeAt = 0
		if err := c.Validate(); !errors.Is(err, ErrLadder) {
			t.Fatalf("edge<1: got %v, want ErrLadder", err)
		}
	})
	t.Run("unhappy: a color stop off the xterm cube is rejected and names its direction", func(t *testing.T) {
		c := DefaultBlast()
		shot := c.ShotAt(sprite.SW)
		shot.Colors[2] = 0
		c.SetShot(sprite.SW, shot)
		err := c.Validate()
		if !errors.Is(err, ErrColor) {
			t.Fatalf("got %v, want ErrColor", err)
		}
		if !strings.Contains(err.Error(), "SW") {
			t.Fatalf("the error must name the SW shot, got %q", err)
		}
		c = DefaultBlast()
		shot = c.ShotAt(sprite.N)
		shot.Colors[4] = 256
		c.SetShot(sprite.N, shot)
		if err := c.Validate(); !errors.Is(err, ErrColor) {
			t.Fatalf("got %v, want ErrColor", err)
		}
	})
	t.Run("unhappy: a broken layer is rejected and named — the core and any direction", func(t *testing.T) {
		c := DefaultBlast()
		shot := c.ShotAt(sprite.W)
		shot.Count = -1
		c.SetShot(sprite.W, shot)
		err := c.Validate()
		if !errors.Is(err, particle.ErrCount) {
			t.Fatalf("got %v, want the engine's ErrCount", err)
		}
		if !strings.Contains(err.Error(), "W") {
			t.Fatalf("the error must name the W shot, got %q", err)
		}
		c = DefaultBlast()
		c.Core.MinLife, c.Core.MaxLife = 2, 1
		err = c.Validate()
		if !errors.Is(err, particle.ErrLife) {
			t.Fatalf("got %v, want the engine's ErrLife", err)
		}
		if !strings.Contains(err.Error(), "core") {
			t.Fatalf("the error must name the core, got %q", err)
		}
	})
}

func TestUseActive(t *testing.T) {
	t.Run("happy: UseBlast puts the settings in effect and ResetBlast restores stock", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		c := DefaultBlast()
		c.Heading = sprite.SE
		shot := c.ShotAt(sprite.SE)
		shot.Count = 200
		c.SetShot(sprite.SE, shot)
		if err := UseBlast(c); err != nil {
			t.Fatalf("UseBlast: %v", err)
		}
		if ActiveBlast() != c {
			t.Fatalf("active %+v, want the used %+v", ActiveBlast(), c)
		}
		ResetBlast()
		if ActiveBlast() != DefaultBlast() {
			t.Fatal("ResetBlast must restore the stock blast")
		}
	})
	t.Run("unhappy: an invalid blast is rejected and the active one holds", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		bad := DefaultBlast()
		bad.Heading = "UP"
		if err := UseBlast(bad); !errors.Is(err, ErrHeading) {
			t.Fatalf("got %v, want ErrHeading", err)
		}
		if ActiveBlast() != DefaultBlast() {
			t.Fatal("a rejected blast must not clobber the active one")
		}
	})
}

func TestEngines(t *testing.T) {
	t.Run("happy: the core and all eight flames share the muzzle, straight flight, and the stage", func(t *testing.T) {
		c := DefaultBlast()
		c.MuzzleX, c.MuzzleY = 0.25, 0.5
		core, flames := c.Engines(100, 60)
		muzzle := particle.Vec2{X: 25, Y: 30}
		if core.Origin != muzzle {
			t.Fatalf("core spawns at %+v, want the muzzle %+v", core.Origin, muzzle)
		}
		for i, cfg := range flames {
			if cfg.Origin != muzzle {
				t.Fatalf("%s flame spawns at %+v, want the muzzle %+v", sprite.Headings[i], cfg.Origin, muzzle)
			}
			if cfg.Width != 100 || cfg.Height != 60 {
				t.Fatalf("%s flame lives in %vx%v, want the 100x60 stage", sprite.Headings[i], cfg.Width, cfg.Height)
			}
			if cfg.Mode != particle.ModeStraight {
				t.Fatalf("%s flame must fly straight, got mode %v", sprite.Headings[i], cfg.Mode)
			}
		}
	})
	t.Run("happy: each compass point heads its own way — cardinals square, diagonals unit-length", func(t *testing.T) {
		d := math.Sqrt2 / 2
		want := map[sprite.Heading]particle.Vec2{
			sprite.N:  {X: 0, Y: -1},
			sprite.NE: {X: d, Y: -d},
			sprite.E:  {X: 1, Y: 0},
			sprite.SE: {X: d, Y: d},
			sprite.S:  {X: 0, Y: 1},
			sprite.SW: {X: -d, Y: d},
			sprite.W:  {X: -1, Y: 0},
			sprite.NW: {X: -d, Y: -d},
		}
		_, flames := DefaultBlast().Engines(100, 60)
		for i, h := range sprite.Headings {
			got := flames[i].Direction
			if math.Abs(got.X-want[h].X) > 1e-9 || math.Abs(got.Y-want[h].Y) > 1e-9 {
				t.Fatalf("the %s flame heads %+v, want %+v", h, got, want[h])
			}
		}
	})
	t.Run("happy: the core aims along the active heading", func(t *testing.T) {
		c := DefaultBlast()
		c.Heading = sprite.W
		core, _ := c.Engines(100, 60)
		if math.Abs(core.Direction.X+1) > 1e-9 || math.Abs(core.Direction.Y) > 1e-9 {
			t.Fatalf("a W core must head (-1, 0), got %+v", core.Direction)
		}
	})
	t.Run("happy: each direction's engine carries its own knobs, lift and drag included", func(t *testing.T) {
		c := DefaultBlast()
		shot := c.ShotAt(sprite.SE)
		shot.Count = 111
		shot.Lift = 77
		shot.Drag = 4.5
		shot.MaxDistance = 9
		c.SetShot(sprite.SE, shot)
		_, flames := c.Engines(100, 60)
		se := flames[3] // Headings order: N NE E SE …
		if se.Count != 111 || se.Lift != 77 || se.Drag != 4.5 || se.MaxDistance != 9 {
			t.Fatalf("the SE engine %+v must carry the SE shot %+v", se, shot)
		}
		if flames[0].Count == 111 {
			t.Fatal("the N engine must keep its own count")
		}
	})
	t.Run("unhappy: a muzzle on the edge clamps inside any stage", func(t *testing.T) {
		c := DefaultBlast()
		c.MuzzleX, c.MuzzleY = 1, 1
		core, _ := c.Engines(3, 2)
		if core.Origin.X > core.Width || core.Origin.Y > core.Height || core.Origin.X < 0 || core.Origin.Y < 0 {
			t.Fatalf("origin %+v fell off a %vx%v stage", core.Origin, core.Width, core.Height)
		}
		if err := core.Validate(); err != nil {
			t.Fatalf("an edge muzzle must still validate: %v", err)
		}
	})
}

func TestLoadSave(t *testing.T) {
	t.Run("happy: a blast survives the round trip and the JSON names all eight directions", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "blast.json")
		c := DefaultBlast()
		c.Heading = sprite.SW
		shot := c.ShotAt(sprite.E)
		shot.Colors = [5]int{21, 27, 33, 39, 45}
		shot.Lift = 44
		c.SetShot(sprite.E, shot)
		if err := c.Save(path); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := LoadBlast(path)
		if err != nil {
			t.Fatalf("LoadBlast: %v", err)
		}
		if got != c {
			t.Fatalf("round trip %+v, want %+v", got, c)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("saved JSON must parse: %v", err)
		}
		for _, key := range []string{"heading", "muzzleX", "pulseDelay", "core", "n", "ne", "e", "se", "s", "sw", "w", "nw"} {
			if _, ok := doc[key]; !ok {
				t.Fatalf("saved JSON is missing %q", key)
			}
		}
		var eDoc map[string]json.RawMessage
		if err := json.Unmarshal(doc["e"], &eDoc); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"lift", "drag", "colors"} {
			if _, ok := eDoc[key]; !ok {
				t.Fatalf("the e shot JSON is missing %q", key)
			}
		}
	})
	t.Run("happy: the shipped config loads and validates", func(t *testing.T) {
		c, err := LoadBlast("config.json")
		if err != nil {
			t.Fatalf("the component must ship a loadable config.json: %v", err)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("the shipped config must validate: %v", err)
		}
	})
	t.Run("unhappy: missing, broken, and out-of-range files are errors", func(t *testing.T) {
		if _, err := LoadBlast(filepath.Join(t.TempDir(), "nope.json")); err == nil {
			t.Fatal("a missing file must error")
		}
		bad := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(bad, []byte("{nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadBlast(bad); err == nil {
			t.Fatal("broken JSON must error")
		}
		out := filepath.Join(t.TempDir(), "out.json")
		if err := os.WriteFile(out, []byte(`{"muzzleX": 4}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadBlast(out); err == nil {
			t.Fatal("an out-of-range file must error")
		}
	})
	t.Run("unhappy: Save refuses an invalid blast", func(t *testing.T) {
		c := DefaultBlast()
		c.Heading = "sideways"
		if err := c.Save(filepath.Join(t.TempDir(), "bad.json")); !errors.Is(err, ErrHeading) {
			t.Fatalf("got %v, want ErrHeading", err)
		}
	})
}
