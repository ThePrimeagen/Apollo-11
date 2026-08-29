package director

// Tests written FIRST: the editor's config is MAIN's own set of
// configs — one row per scene in bill order, each carrying how many
// seconds the scene plays in play mode before the cut, plus that
// scene's own knobs as the scene's JSON shape. It is the show's file,
// not the scenes': saving MAIN never touches a scene package's config
// or its Active knobs. A name the file does not carry reads the stock
// hold and no knobs; the operator's numbers are stored verbatim —
// zero and negative holds included — and only NaN and Inf, numbers no
// knob can be turned onto, are rejected. A broken file is an error
// worth stopping for; a missing file is just the stock show.

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestMainConfig(t *testing.T) {
	t.Run("happy: an empty config reads the stock hold and no knobs", func(t *testing.T) {
		var c Config
		if got := c.HoldFor("the moon"); got != DefaultHoldSeconds {
			t.Fatalf("HoldFor on an empty config = %v, want the stock %v", got, DefaultHoldSeconds)
		}
		if raw := c.KnobsFor("the moon"); raw != nil {
			t.Fatalf("an empty config carries knobs %s, want none", raw)
		}
	})
	t.Run("happy: SetHold appends in first-touch order and updates in place", func(t *testing.T) {
		var c Config
		c.SetHold("one", 3.5)
		c.SetHold("two", 1.0)
		c.SetHold("one", 4.0)
		if len(c.Scenes) != 2 {
			t.Fatalf("config holds %d scenes, want 2", len(c.Scenes))
		}
		if c.Scenes[0].Scene != "one" || c.Scenes[1].Scene != "two" {
			t.Fatalf("scene order %v, want one then two", c.Scenes)
		}
		if c.HoldFor("one") != 4.0 || c.HoldFor("two") != 1.0 {
			t.Fatalf("holds read %v/%v, want 4/1", c.HoldFor("one"), c.HoldFor("two"))
		}
	})
	t.Run("happy: SetKnobs rides the same rows and keeps the hold", func(t *testing.T) {
		var c Config
		c.SetHold("fall", 7.5)
		c.SetKnobs("fall", json.RawMessage(`{"dropSeconds":6}`))
		if len(c.Scenes) != 1 {
			t.Fatalf("knobs must ride the scene's row, got %d rows", len(c.Scenes))
		}
		if c.HoldFor("fall") != 7.5 {
			t.Fatal("SetKnobs must not move the hold")
		}
		if string(c.KnobsFor("fall")) != `{"dropSeconds":6}` {
			t.Fatalf("knobs read %s", c.KnobsFor("fall"))
		}
		c.SetHold("fall", 9)
		if string(c.KnobsFor("fall")) != `{"dropSeconds":6}` {
			t.Fatal("SetHold must not drop the knobs")
		}
	})
	t.Run("happy: SetKnobs on a new name opens the row at the stock hold", func(t *testing.T) {
		var c Config
		c.SetKnobs("orbit", json.RawMessage(`{"lapSeconds":4}`))
		if c.HoldFor("orbit") != DefaultHoldSeconds {
			t.Fatalf("a knobs-first row reads hold %v, want the stock %v", c.HoldFor("orbit"), DefaultHoldSeconds)
		}
	})
	t.Run("happy: zero and negative holds are the operator's numbers — stored verbatim", func(t *testing.T) {
		var c Config
		c.SetHold("zero", 0)
		c.SetHold("negative", -2.5)
		path := filepath.Join(t.TempDir(), "main.json")
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
	t.Run("happy: Save then Load round-trips every scene, in order, knobs intact", func(t *testing.T) {
		var c Config
		c.SetHold("the moon", 6)
		c.SetHold("orbit", 9.5)
		c.SetKnobs("orbit", json.RawMessage(`{"arriveSeconds":1.5,"lapSeconds":4}`))
		c.SetHold("fall", 2)
		path := filepath.Join(t.TempDir(), "main.json")
		if err := c.Save(path); err != nil {
			t.Fatalf("save: %v", err)
		}
		back, err := Load(path)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if len(back.Scenes) != 3 {
			t.Fatalf("the file carries %d scenes, want 3", len(back.Scenes))
		}
		for i, want := range []string{"the moon", "orbit", "fall"} {
			if back.Scenes[i].Scene != want {
				t.Fatalf("row %d is %q, want %q", i, back.Scenes[i].Scene, want)
			}
		}
		if back.HoldFor("orbit") != 9.5 {
			t.Fatalf("orbit hold %v, want 9.5", back.HoldFor("orbit"))
		}
		var knobs struct {
			Arrive float64 `json:"arriveSeconds"`
			Lap    float64 `json:"lapSeconds"`
		}
		if err := json.Unmarshal(back.KnobsFor("orbit"), &knobs); err != nil {
			t.Fatalf("orbit knobs: %v", err)
		}
		if knobs.Arrive != 1.5 || knobs.Lap != 4 {
			t.Fatalf("orbit knobs %+v, want 1.5/4", knobs)
		}
		if raw := back.KnobsFor("the moon"); raw != nil {
			t.Fatalf("the moon carries knobs %s, want none", raw)
		}
	})
	t.Run("happy: LoadOrDefault on a missing file is just the stock show", func(t *testing.T) {
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
		path := filepath.Join(t.TempDir(), "main.json")
		if err := os.WriteFile(path, []byte("{scenes: nope"), 0o644); err != nil {
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
		path := filepath.Join(t.TempDir(), "main.json")
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
		if err := c.Save(filepath.Join(t.TempDir(), "missing-dir", "main.json")); err == nil {
			t.Fatal("an unwritable path must error")
		}
	})
	t.Run("unhappy: a nil config is inert", func(t *testing.T) {
		var c *Config
		c.SetHold("one", 1)
		c.SetKnobs("one", json.RawMessage(`{}`))
	})
}
