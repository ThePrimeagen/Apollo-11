package lunarcloseup

// Tests written FIRST: the lunar lander close-up is a composable
// one-scene bill — a copy of the premiere's arrival. Scene one,
// "Lunar Lander Close-Up": three seconds of drifting sky, then the
// zoomed-in Apollo craft slides in from the right wing over a
// starfield that translates with it — hull only, cold engine — parks
// and bobbles at center stage. More scenes will join later; for now
// this is the whole show. After the last scene there is nothing left.
// The bill is the composable unit — screenplay.Compose adds bills
// together into one big show.

import (
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 72
	stageH = 27
)

func render(sc screenplay.Scene) string {
	scr := screenplay.NewScreen(stageW, stageH)
	sc.Render(scr)
	return scr.Render()
}

func hasStar(v string) bool {
	for _, g := range stars.Glyphs {
		if strings.ContainsRune(v, g) {
			return true
		}
	}
	return false
}

func TestLunarCloseUpBill(t *testing.T) {
	t.Run("happy: the bill is one scene named Lunar Lander Close-Up", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the close-up screenplay holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "Lunar Lander Close-Up" {
			t.Fatalf("scene 1 is %q, want Lunar Lander Close-Up", b[0].Name)
		}
		if b[0].Scene == nil {
			t.Fatal("Lunar Lander Close-Up has no performer")
		}
	})
	t.Run("happy: the scene opens under stars with the craft off the right wing", func(t *testing.T) {
		sc := Bill()[0].Scene
		sc.Start()
		defer sc.Stop()
		v := render(sc)
		if !hasStar(v) {
			t.Fatal("the close-up plays under the stars")
		}
		if strings.ContainsRune(v, '▌') {
			t.Fatal("the craft is still off the right wing at t=0")
		}
	})
	t.Run("happy: after the hold the hull flies in with a cold engine", func(t *testing.T) {
		sc := Bill()[0].Scene
		sc.Start()
		defer sc.Stop()
		sc.Update(lander.FlyInHoldSeconds)
		if strings.ContainsRune(render(sc), '▌') {
			t.Fatal("the hold is still running — the craft must stay offstage")
		}
		sc.Update(lander.FlyInSeconds)
		v := render(sc)
		if !strings.ContainsRune(v, '▌') {
			t.Fatal("after the hold the hull must be on screen")
		}
		if strings.ContainsAny(v, "⠁⠒⠶▒") {
			t.Fatal("the close-up must fly a dark engine — no booster fire yet")
		}
	})
	t.Run("happy: the sky slides with the craft — same cells, same ease", func(t *testing.T) {
		hold := lander.FlyInHoldSeconds
		for _, w := range []int{40, 72, 120} {
			for _, sceneT := range []float64{0, 2, hold, hold + 1, hold + lander.FlyInSeconds, hold + lander.FlyInSeconds + 3} {
				flyT := sceneT - hold
				_, c0 := lander.FlightPath(w, 28, 0)
				_, c := lander.FlightPath(w, 28, flyT)
				got := stars.SlideOffset(w, lander.BodyCols, flyT, lander.FlyInSeconds)
				if c0-c != got {
					t.Fatalf("w=%d scene t=%.1f (fly t=%.1f) ship traveled %d, sky slide %d", w, sceneT, flyT, c0-c, got)
				}
			}
		}
	})
	t.Run("happy: the composed show opens on the close-up and then has nothing left", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		if p.Len() != 1 || p.CurrentName() != "Lunar Lander Close-Up" {
			t.Fatalf("the show opens on %d %q, want one Lunar Lander Close-Up", p.Len(), p.CurrentName())
		}
		if p.Next() {
			t.Fatal("after the close-up there is nothing left — the show ends")
		}
		p.Stop()
	})
	t.Run("unhappy: a scene stopped before its first render never panics", func(t *testing.T) {
		for _, e := range Bill() {
			e.Scene.Start()
			e.Scene.Update(1)
			e.Scene.Stop()
		}
	})
	t.Run("unhappy: the close-up is not the four-scene premiere", func(t *testing.T) {
		sc := Bill()[0].Scene
		sc.Start()
		defer sc.Stop()
		sc.Update(lander.FlyInHoldSeconds + lander.FlyInSeconds)
		v := render(sc)
		if strings.Contains(v, "VERB") {
			t.Fatal("the DSKY does not appear in the close-up")
		}
		if strings.ContainsRune(v, moon.MarkerGlyph) {
			t.Fatal("the moon's craft does not appear in the close-up")
		}
		if strings.Contains(v, "THE END") || strings.Contains(v, "___") {
			t.Fatal("the end card does not appear in the close-up")
		}
		for _, e := range Bill() {
			switch e.Name {
			case "arrival", "dsky", "descent orbit", "the end":
				t.Fatalf("the close-up bill must not carry premiere scene %q", e.Name)
			}
		}
	})
}
