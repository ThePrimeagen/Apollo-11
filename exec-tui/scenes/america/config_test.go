package america

// Tests written FIRST: Config is the nine live knobs — how long the
// flag takes to fade in from black, when the eagle enters, how long
// its crossing takes (the eagle's speed), where the flight starts and
// ends as fractions of the full off-right-to-off-left span, and the
// talon shotguns: how many times the gun in each talon fires across
// one crossing, and which of the eight compass points each barrel
// aims. The time knobs nudge 50ms at a time, the path knobs 0.05 of
// the span, the shot counts one shell, the aims one compass point
// with wrap. Play rebuilds the scene from the current knobs so
// iteration does not require a restart. Save/Load round-trip the JSON
// next to the scene; Use is what New plays on the first curtain; a
// file missing a key keeps that knob at stock.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestConfig(t *testing.T) {
	t.Run("happy: the stock knobs match the scene defaults", func(t *testing.T) {
		c := DefaultConfig()
		if c.FadeSeconds != FadeSeconds {
			t.Fatalf("fade %v, want %v", c.FadeSeconds, FadeSeconds)
		}
		if c.EagleDelay != FadeSeconds {
			t.Fatalf("eagle delay %v, want %v — the eagle enters when the fade lands", c.EagleDelay, FadeSeconds)
		}
		if c.CrossSeconds != CrossSeconds {
			t.Fatalf("cross %v, want %v", c.CrossSeconds, CrossSeconds)
		}
		if c.EagleStart != StartPoint || c.EagleEnd != EndPoint {
			t.Fatalf("path %v..%v, want the stock %v..%v — off one wing and off the other", c.EagleStart, c.EagleEnd, StartPoint, EndPoint)
		}
		if StepSeconds != 0.050 {
			t.Fatalf("step %v, want 50ms", StepSeconds)
		}
		if StepPoint != 0.050 {
			t.Fatalf("path step %v, want 0.05 of the span", StepPoint)
		}
		if c.LeftShots != StockShots || c.RightShots != StockShots {
			t.Fatalf("shots %d/%d, want the stock %d each", c.LeftShots, c.RightShots, StockShots)
		}
		if StockShots < 1 {
			t.Fatalf("StockShots = %d — the stock birds fires on its way across", StockShots)
		}
		if c.LeftAim != StockLeftAim || c.RightAim != StockRightAim {
			t.Fatalf("aims %s/%s, want the stock %s/%s", c.LeftAim, c.RightAim, StockLeftAim, StockRightAim)
		}
		if StockLeftAim != sprite.W || StockRightAim != sprite.E {
			t.Fatalf("stock aims %s/%s, want W/E — the clean side-on frames, the leading barrel raking ahead, the trailing one behind", StockLeftAim, StockRightAim)
		}
		if KnobCount != 9 {
			t.Fatalf("KnobCount %d, want 9 (fade, delay, cross, start, end, left shots, left aim, right shots, right aim)", KnobCount)
		}
	})
	t.Run("happy: Display reads every knob in its own language", func(t *testing.T) {
		c := DefaultConfig()
		if got := c.Display(KnobFade); got != "  2.000s" {
			t.Fatalf("Display(fade) %q, want %q", got, "  2.000s")
		}
		if got := c.Display(KnobStart); got != "  0.000" {
			t.Fatalf("Display(start) %q, want %q — a fraction, not seconds", got, "  0.000")
		}
		if got := c.Display(KnobLeftShots); got != fmt.Sprintf("%7d", StockShots) {
			t.Fatalf("Display(left shots) %q, want a bare count", got)
		}
		if got := c.Display(KnobRightAim); got != fmt.Sprintf("%7s", string(StockRightAim)) {
			t.Fatalf("Display(right aim) %q, want the compass point", got)
		}
		if got := c.Display(KnobCount); got != "" {
			t.Fatalf("an off-panel knob displays %q, want nothing", got)
		}
	})
	t.Run("happy: every knob carries a label and a unit and reads its own value", func(t *testing.T) {
		c := DefaultConfig()
		seen := map[string]bool{}
		for k := Knob(0); k < KnobCount; k++ {
			label := KnobLabel(k)
			if label == "" {
				t.Fatalf("knob %d has no label", k)
			}
			if seen[label] {
				t.Fatalf("label %q repeats", label)
			}
			seen[label] = true
		}
		if got := c.Value(KnobFade); got != c.FadeSeconds {
			t.Fatalf("Value(fade) %v, want %v", got, c.FadeSeconds)
		}
		if got := c.Value(KnobDelay); got != c.EagleDelay {
			t.Fatalf("Value(delay) %v, want %v", got, c.EagleDelay)
		}
		if got := c.Value(KnobCross); got != c.CrossSeconds {
			t.Fatalf("Value(cross) %v, want %v", got, c.CrossSeconds)
		}
		if got := c.Value(KnobStart); got != c.EagleStart {
			t.Fatalf("Value(start) %v, want %v", got, c.EagleStart)
		}
		if got := c.Value(KnobEnd); got != c.EagleEnd {
			t.Fatalf("Value(end) %v, want %v", got, c.EagleEnd)
		}
		if got := c.Value(KnobLeftShots); got != float64(c.LeftShots) {
			t.Fatalf("Value(left shots) %v, want %v", got, float64(c.LeftShots))
		}
		if got := c.Value(KnobRightShots); got != float64(c.RightShots) {
			t.Fatalf("Value(right shots) %v, want %v", got, float64(c.RightShots))
		}
		for _, k := range []Knob{KnobFade, KnobDelay, KnobCross} {
			if KnobUnit(k) != "s" {
				t.Fatalf("knob %q is seconds, unit %q", KnobLabel(k), KnobUnit(k))
			}
		}
		for _, k := range []Knob{KnobStart, KnobEnd} {
			if KnobUnit(k) != "" {
				t.Fatalf("knob %q is a fraction of the span, unit %q", KnobLabel(k), KnobUnit(k))
			}
		}
		if KnobLabel(KnobCount) != "" || c.Value(KnobCount) != 0 || KnobUnit(KnobCount) != "" {
			t.Fatal("an off-panel knob has no label, no value and no unit")
		}
	})
	t.Run("happy: Nudge walks the selected knob by 50ms and stays on the grid", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobFade, 1)
		if got := c.FadeSeconds; math.Abs(got-(FadeSeconds+StepSeconds)) > 1e-9 {
			t.Fatalf("fade after +50ms is %v, want %v", got, FadeSeconds+StepSeconds)
		}
		c.Nudge(KnobDelay, -1)
		if got := c.EagleDelay; math.Abs(got-(FadeSeconds-StepSeconds)) > 1e-9 {
			t.Fatalf("delay after -50ms is %v, want %v", got, FadeSeconds-StepSeconds)
		}
		c.Nudge(KnobCross, 1)
		if got := c.CrossSeconds; math.Abs(got-(CrossSeconds+StepSeconds)) > 1e-9 {
			t.Fatalf("cross after +50ms is %v, want %v", got, CrossSeconds+StepSeconds)
		}
		for i := 0; i < 23; i++ {
			c.Nudge(KnobCross, 1)
		}
		want := CrossSeconds + 24*StepSeconds
		if math.Abs(c.CrossSeconds-want) > 1e-12 {
			t.Fatalf("cross after 24 steps is %v, want %v on the 50ms grid", c.CrossSeconds, want)
		}
		c.Nudge(KnobStart, 1)
		if math.Abs(c.EagleStart-(StartPoint+StepPoint)) > 1e-9 {
			t.Fatalf("start after +0.05 is %v, want %v", c.EagleStart, StartPoint+StepPoint)
		}
		c.Nudge(KnobEnd, -1)
		if math.Abs(c.EagleEnd-(EndPoint-StepPoint)) > 1e-9 {
			t.Fatalf("end after -0.05 is %v, want %v", c.EagleEnd, EndPoint-StepPoint)
		}
		c.Nudge(KnobLeftShots, 1)
		if c.LeftShots != StockShots+1 {
			t.Fatalf("left shots after +1 is %d, want %d", c.LeftShots, StockShots+1)
		}
		c.Nudge(KnobRightShots, -1)
		if c.RightShots != StockShots-1 {
			t.Fatalf("right shots after -1 is %d, want %d", c.RightShots, StockShots-1)
		}
	})
	t.Run("happy: the aim knobs walk the compass and wrap at the ends", func(t *testing.T) {
		c := DefaultConfig()
		c.LeftAim = sprite.NW
		c.Nudge(KnobLeftAim, 1)
		if c.LeftAim != sprite.N {
			t.Fatalf("aim after NW+1 is %s, want the wrap to N", c.LeftAim)
		}
		c.Nudge(KnobLeftAim, -1)
		if c.LeftAim != sprite.NW {
			t.Fatalf("aim after N-1 is %s, want the wrap back to NW", c.LeftAim)
		}
		c.RightAim = sprite.SE
		c.Nudge(KnobRightAim, 1)
		if c.RightAim != sprite.S {
			t.Fatalf("aim after SE+1 is %s, want S — one compass point clockwise", c.RightAim)
		}
	})
	t.Run("unhappy: Nudge will not walk a knob past its rails, and a bad cursor is a no-op", func(t *testing.T) {
		c := DefaultConfig()
		c.FadeSeconds = 0
		c.Nudge(KnobFade, -1)
		if c.FadeSeconds != 0 {
			t.Fatalf("fade %v, want 0 — an instant flag is allowed", c.FadeSeconds)
		}
		c.EagleDelay = 0
		c.Nudge(KnobDelay, -1)
		if c.EagleDelay != 0 {
			t.Fatalf("delay %v, want 0 — the eagle may enter at t=0", c.EagleDelay)
		}
		c.CrossSeconds = StepSeconds
		c.Nudge(KnobCross, -1)
		if c.CrossSeconds != StepSeconds {
			t.Fatalf("cross %v, want the 50ms floor — the crossing must be a duration", c.CrossSeconds)
		}
		c.EagleStart = 0
		c.Nudge(KnobStart, -1)
		if c.EagleStart != 0 {
			t.Fatalf("start %v, want 0 — the path never starts before the span", c.EagleStart)
		}
		c.EagleEnd = 1
		c.Nudge(KnobEnd, 1)
		if c.EagleEnd != 1 {
			t.Fatalf("end %v, want 1 — the path never ends past the span", c.EagleEnd)
		}
		c.EagleStart = 0.5
		c.EagleEnd = 0.5 + StepPoint
		c.Nudge(KnobStart, 1)
		if math.Abs(c.EagleStart-0.5) > 1e-9 {
			t.Fatalf("start %v, want 0.5 — the start never catches the end", c.EagleStart)
		}
		c.Nudge(KnobEnd, -1)
		if math.Abs(c.EagleEnd-(0.5+StepPoint)) > 1e-9 {
			t.Fatalf("end %v, want %v — the end never falls onto the start", c.EagleEnd, 0.5+StepPoint)
		}
		c.LeftShots = 0
		c.Nudge(KnobLeftShots, -1)
		if c.LeftShots != 0 {
			t.Fatalf("left shots %d, want 0 — a silent gun is allowed, a negative one is not", c.LeftShots)
		}
		c.RightShots = 0
		c.Nudge(KnobRightShots, -1)
		if c.RightShots != 0 {
			t.Fatalf("right shots %d, want 0 — a silent gun is allowed, a negative one is not", c.RightShots)
		}
		before := c
		c.Nudge(-1, 1)
		c.Nudge(99, 1)
		if c != before {
			t.Fatalf("a bad cursor must not change the knobs, got %+v", c)
		}
	})
	t.Run("happy: Save then Load round-trips the nine knobs", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "america.json")
		c := DefaultConfig()
		c.FadeSeconds = 2.5
		c.EagleDelay = 3.0
		c.CrossSeconds = 6.25
		c.EagleStart = 0.25
		c.EagleEnd = 0.75
		c.LeftShots = 5
		c.LeftAim = sprite.NW
		c.RightShots = 0
		c.RightAim = sprite.E
		if err := c.Save(path); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if math.Abs(got.FadeSeconds-c.FadeSeconds) > 1e-9 ||
			math.Abs(got.EagleDelay-c.EagleDelay) > 1e-9 ||
			math.Abs(got.CrossSeconds-c.CrossSeconds) > 1e-9 ||
			math.Abs(got.EagleStart-c.EagleStart) > 1e-9 ||
			math.Abs(got.EagleEnd-c.EagleEnd) > 1e-9 {
			t.Fatalf("round-trip %+v, want %+v", got, c)
		}
		if got.LeftShots != c.LeftShots || got.LeftAim != c.LeftAim ||
			got.RightShots != c.RightShots || got.RightAim != c.RightAim {
			t.Fatalf("gun round-trip %+v, want %+v", got, c)
		}
	})
	t.Run("happy: a file missing a key keeps that knob at stock", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "partial.json")
		if err := os.WriteFile(path, []byte(`{"fadeSeconds":2.0}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.FadeSeconds != 2.0 {
			t.Fatalf("fade %v, want the file's 2.0", got.FadeSeconds)
		}
		if got.EagleDelay != FadeSeconds || got.CrossSeconds != CrossSeconds {
			t.Fatalf("missing keys loaded %+v, want stock delay and cross", got)
		}
		if got.EagleStart != StartPoint || got.EagleEnd != EndPoint {
			t.Fatalf("missing path keys loaded %v..%v, want the stock %v..%v", got.EagleStart, got.EagleEnd, StartPoint, EndPoint)
		}
		if got.LeftShots != StockShots || got.LeftAim != StockLeftAim ||
			got.RightShots != StockShots || got.RightAim != StockRightAim {
			t.Fatalf("missing gun keys loaded %+v, want stock shots and aims", got)
		}
	})
	t.Run("happy: LoadOrDefault is stock when the file is missing, and Use is that config", func(t *testing.T) {
		t.Cleanup(Reset)
		c, err := LoadOrDefault(filepath.Join(t.TempDir(), "nope.json"))
		if err != nil {
			t.Fatalf("a missing file must keep stock: %v", err)
		}
		if c != DefaultConfig() {
			t.Fatalf("LoadOrDefault %+v, want stock", c)
		}
		live := DefaultConfig()
		live.CrossSeconds = 4.0
		if err := Use(live); err != nil {
			t.Fatal(err)
		}
		if Active() != live {
			t.Fatalf("Active %+v, want %+v", Active(), live)
		}
		Reset()
		if Active() != DefaultConfig() {
			t.Fatalf("Reset left %+v, want stock", Active())
		}
	})
	t.Run("unhappy: missing, broken, and out-of-range files error, and Save/Use refuse a bad knob", func(t *testing.T) {
		t.Cleanup(Reset)
		if _, err := Load(filepath.Join(t.TempDir(), "nope.json")); err == nil {
			t.Fatal("a missing file must error")
		}
		bad := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(bad, []byte("{nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(bad); err == nil {
			t.Fatal("broken JSON must error")
		}
		out := filepath.Join(t.TempDir(), "out.json")
		if err := os.WriteFile(out, []byte(`{"crossSeconds":0}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(out); err == nil {
			t.Fatal("a crossing below 50ms must error")
		}
		for name, body := range map[string]string{
			"a start before the span": `{"eagleStart":-0.1}`,
			"an end past the span":    `{"eagleEnd":1.2}`,
			"an end behind the start": `{"eagleStart":0.8,"eagleEnd":0.4}`,
			"an end on the start":     `{"eagleStart":0.5,"eagleEnd":0.5}`,
			"a negative shell count":  `{"leftShots":-1}`,
			"an aim off the compass":  `{"rightAim":"UP"}`,
			"an empty aim":            `{"leftAim":""}`,
		} {
			bad := filepath.Join(t.TempDir(), "path.json")
			if err := os.WriteFile(bad, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(bad); err == nil {
				t.Fatalf("%s must error", name)
			}
		}
		negFade := DefaultConfig()
		negFade.FadeSeconds = -0.1
		if err := negFade.Save(filepath.Join(t.TempDir(), "x.json")); err == nil {
			t.Fatal("Save must refuse a negative fade")
		}
		negDelay := DefaultConfig()
		negDelay.EagleDelay = -0.1
		if err := negDelay.Save(filepath.Join(t.TempDir(), "y.json")); err == nil {
			t.Fatal("Save must refuse a negative delay")
		}
		backPath := DefaultConfig()
		backPath.EagleStart = 0.9
		backPath.EagleEnd = 0.1
		if err := backPath.Save(filepath.Join(t.TempDir(), "z.json")); err == nil {
			t.Fatal("Save must refuse a backwards flight path")
		}
		negShots := DefaultConfig()
		negShots.RightShots = -2
		if err := negShots.Save(filepath.Join(t.TempDir(), "w.json")); err == nil {
			t.Fatal("Save must refuse a negative shell count")
		}
		badAim := DefaultConfig()
		badAim.LeftAim = sprite.Heading("XX")
		if err := Use(badAim); err == nil {
			t.Fatal("Use must reject an aim off the compass")
		}
		before := Active()
		badUse := DefaultConfig()
		badUse.CrossSeconds = 0
		if err := Use(badUse); err == nil {
			t.Fatal("Use must reject a crossing below 50ms")
		}
		badPath := DefaultConfig()
		badPath.EagleEnd = badPath.EagleStart
		if err := Use(badPath); err == nil {
			t.Fatal("Use must reject a flight path with no length")
		}
		if Active() != before {
			t.Fatalf("Active after a rejected Use is %+v, want %+v", Active(), before)
		}
	})
}
