package director

// Tests written FIRST: the editor's own config is one number per
// scene — how many seconds the scene plays in play mode before the
// cut. Holds are kept by scene name in bill order, a name the file
// does not carry reads the stock hold, and the operator's number is
// the number: zero and negative holds are stored and played verbatim,
// never rewritten to a "sane" default. Only NaN and Inf — numbers no
// operator can turn a knob onto — are rejected, and JSON cannot even
// carry them. A broken file is an error worth stopping for; a missing
// file is just the stock holds.

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestHoldsConfig(t *testing.T) {
	t.Run("happy: an empty config reads the stock hold for any scene", func(t *testing.T) {
		var c Config
		if got := c.HoldFor("the moon"); got != DefaultHoldSeconds {
			t.Fatalf("HoldFor on an empty config = %v, want the stock %v", got, DefaultHoldSeconds)
		}
	})
	t.Run("happy: SetHold appends in first-touch order and updates in place", func(t *testing.T) {
		var c Config
		c.SetHold("one", 3.5)
		c.SetHold("two", 1.0)
		c.SetHold("one", 4.0)
		if len(c.Holds) != 2 {
			t.Fatalf("config holds %d scenes, want 2", len(c.Holds))
		}
		if c.Holds[0].Scene != "one" || c.Holds[1].Scene != "two" {
			t.Fatalf("holds order %v, want one then two", c.Holds)
		}
		if c.HoldFor("one") != 4.0 || c.HoldFor("two") != 1.0 {
			t.Fatalf("holds read %v/%v, want 4/1", c.HoldFor("one"), c.HoldFor("two"))
		}
	})
	t.Run("happy: zero and negative holds are the operator's numbers — stored verbatim", func(t *testing.T) {
		var c Config
		c.SetHold("zero", 0)
		c.SetHold("negative", -2.5)
		if c.HoldFor("zero") != 0 {
			t.Fatalf("a zero hold reads %v, want 0 — never rewrite it", c.HoldFor("zero"))
		}
		if c.HoldFor("negative") != -2.5 {
			t.Fatalf("a negative hold reads %v, want -2.5 — never clamp it", c.HoldFor("negative"))
		}
		path := filepath.Join(t.TempDir(), "holds.json")
		if err := c.Save(path); err != nil {
			t.Fatalf("save: %v", err)
		}
		back, err := Load(path)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if back.HoldFor("zero") != 0 || back.HoldFor("negative") != -2.5 {
			t.Fatalf("the file round-trips %v/%v, want 0/-2.5 untouched",
				back.HoldFor("zero"), back.HoldFor("negative"))
		}
	})
	t.Run("happy: Save then Load round-trips every hold in order", func(t *testing.T) {
		var c Config
		c.SetHold("the moon", 6)
		c.SetHold("orbit", 9.5)
		c.SetHold("fall", 2)
		path := filepath.Join(t.TempDir(), "holds.json")
		if err := c.Save(path); err != nil {
			t.Fatalf("save: %v", err)
		}
		back, err := Load(path)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if len(back.Holds) != 3 {
			t.Fatalf("the file carries %d holds, want 3", len(back.Holds))
		}
		for i, want := range []SceneHold{
			{Scene: "the moon", Seconds: 6},
			{Scene: "orbit", Seconds: 9.5},
			{Scene: "fall", Seconds: 2},
		} {
			if back.Holds[i] != want {
				t.Fatalf("hold %d = %+v, want %+v", i, back.Holds[i], want)
			}
		}
	})
	t.Run("happy: LoadOrDefault on a missing file is just the stock holds", func(t *testing.T) {
		c, err := LoadOrDefault(filepath.Join(t.TempDir(), "nope.json"))
		if err != nil {
			t.Fatalf("a missing file must not be an error: %v", err)
		}
		if got := c.HoldFor("anything"); got != DefaultHoldSeconds {
			t.Fatalf("stock hold = %v, want %v", got, DefaultHoldSeconds)
		}
	})
	t.Run("unhappy: Load on a missing file reports it", func(t *testing.T) {
		_, err := Load(filepath.Join(t.TempDir(), "nope.json"))
		if !os.IsNotExist(err) {
			t.Fatalf("want a does-not-exist error, got %v", err)
		}
	})
	t.Run("unhappy: a broken file is an error worth stopping for", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "holds.json")
		if err := os.WriteFile(path, []byte("{holds: nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatal("broken JSON must not load")
		}
		if _, err := LoadOrDefault(path); err == nil {
			t.Fatal("LoadOrDefault must still stop on a broken file")
		}
	})
	t.Run("unhappy: NaN and Inf never reach a file", func(t *testing.T) {
		var c Config
		c.SetHold("cursed", math.NaN())
		path := filepath.Join(t.TempDir(), "holds.json")
		if err := c.Save(path); err == nil {
			t.Fatal("a NaN hold must not save")
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("a rejected save must leave no file behind")
		}
		c = Config{}
		c.SetHold("cursed", math.Inf(1))
		if err := c.Save(path); err == nil {
			t.Fatal("an Inf hold must not save")
		}
	})
	t.Run("unhappy: saving somewhere unwritable reports the error", func(t *testing.T) {
		var c Config
		c.SetHold("one", 1)
		if err := c.Save(filepath.Join(t.TempDir(), "missing-dir", "holds.json")); err == nil {
			t.Fatal("an unwritable path must error")
		}
	})
	t.Run("unhappy: a nil config is inert", func(t *testing.T) {
		var c *Config
		c.SetHold("one", 1)
	})
}
