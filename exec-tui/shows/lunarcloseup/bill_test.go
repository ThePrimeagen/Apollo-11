package lunarcloseup

// Tests written FIRST: the lunar lander close-up is a composable
// four-scene bill. Scene one, "Lunar Lander Close-Up": a copy of the
// premiere's arrival — drifting stars, then the zoomed-in craft slides
// in from the right, hull only, cold engine. Scene two, "fire": the
// parked craft lights the booster and the stars slow by 60% over five
// seconds. Scene three, "fall": the north-facing lander, fire down,
// drops from the top of the stage to the bottom. Scene four,
// "landing": a huge moon horizon (five rows high in the middle, one
// row at the edges) and the north-facing lander coming down onto it.
// After the last scene there is nothing left.

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

func hasFire(v string) bool {
	return strings.ContainsAny(v, "⠁⠒⠶")
}

func TestLunarCloseUpBill(t *testing.T) {
	t.Run("happy: the bill is four scenes in playing order", func(t *testing.T) {
		b := Bill()
		if len(b) != 4 {
			t.Fatalf("the close-up screenplay holds %d scenes, want 4", len(b))
		}
		for i, want := range []string{"Lunar Lander Close-Up", "fire", "fall", "landing"} {
			if b[i].Name != want {
				t.Fatalf("scene %d is %q, want %q", i+1, b[i].Name, want)
			}
			if b[i].Scene == nil {
				t.Fatalf("scene %q has no performer", want)
			}
		}
	})
	t.Run("happy: scene one opens under stars with the craft off the right wing", func(t *testing.T) {
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
		if hasFire(v) {
			t.Fatal("the close-up must fly a dark engine — no booster fire yet")
		}
	})
	t.Run("happy: scene two parks the west craft with the booster lit", func(t *testing.T) {
		sc := Bill()[1].Scene
		sc.Start()
		defer sc.Stop()
		_ = render(sc) // stage the cast
		sc.Update(0.5)
		v := render(sc)
		if !strings.ContainsRune(v, '▌') {
			t.Fatal("the fire scene must open with the west-facing craft already on stage")
		}
		if !hasFire(v) {
			t.Fatal("the fire scene must light the booster")
		}
		if !hasStar(v) {
			t.Fatal("the fire scene still plays under the stars")
		}
	})
	t.Run("happy: scene two's sky slows 60% over five seconds", func(t *testing.T) {
		if stars.BrakeClock(5, 0.6, 5) != 3.5 {
			t.Fatalf("the fire scene's brake must cut 60% of speed over 5s (fly clock %g, want 3.5)", stars.BrakeClock(5, 0.6, 5))
		}
		if stars.BrakeClock(10, 0.6, 5) != 5.5 {
			t.Fatal("past the window the fire scene's sky must crawl at 40% speed")
		}
	})
	t.Run("happy: scene three drops a north-facing lander with fire, top to bottom", func(t *testing.T) {
		sc := Bill()[2].Scene
		sc.Start()
		defer sc.Stop()
		_ = render(sc)
		if strings.ContainsRune(render(sc), '▌') {
			t.Fatal("the falling craft must not wear the west-facing hull")
		}
		if strings.ContainsRune(render(sc), '▟') {
			t.Fatal("at t=0 the falling craft must still be off the top")
		}
		sc.Update(lander.DropSeconds / 2)
		mid := render(sc)
		if !strings.ContainsRune(mid, '▟') {
			t.Fatal("mid-fall the north hull must be on stage")
		}
		if strings.ContainsRune(mid, '▌') {
			t.Fatal("the falling craft must stay north-facing")
		}
		if !hasFire(mid) {
			t.Fatal("the falling craft must keep the booster lit")
		}
		sc.Update(lander.DropSeconds / 2)
		if strings.ContainsRune(render(sc), '▟') {
			t.Fatal("at the end of the drop the craft must have left the bottom")
		}
	})
	t.Run("happy: scene four is a huge moon horizon the lander comes down onto", func(t *testing.T) {
		sc := Bill()[3].Scene
		sc.Start()
		defer sc.Stop()
		_ = render(sc)
		opening := render(sc)
		if !strings.ContainsRune(opening, '▓') {
			t.Fatal("the landing scene must show the moon")
		}
		if strings.ContainsRune(opening, '▟') {
			t.Fatal("at t=0 the lander must still be off the top")
		}
		// The horizon is a shallow curve: 1 row at the edges, 5 at center.
		plain := stripANSI(opening)
		if moonRows(plain, 0) != moon.HorizonEdgeRows {
			t.Fatalf("left edge holds %d moon rows, want %d", moonRows(plain, 0), moon.HorizonEdgeRows)
		}
		if moonRows(plain, stageW/2) != moon.HorizonCenterRows {
			t.Fatalf("center holds %d moon rows, want %d", moonRows(plain, stageW/2), moon.HorizonCenterRows)
		}
		sc.Update(lander.LandSeconds)
		landed := render(sc)
		if !strings.ContainsRune(landed, '▟') {
			t.Fatal("at touchdown the north hull must sit on the surface")
		}
		if strings.ContainsRune(landed, '▌') {
			t.Fatal("the landing craft must stay north-facing")
		}
	})
	t.Run("happy: the composed show walks the four scenes and then has nothing left", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		if p.Len() != 4 || p.CurrentName() != "Lunar Lander Close-Up" {
			t.Fatalf("the show opens on %d %q, want four starting on Lunar Lander Close-Up", p.Len(), p.CurrentName())
		}
		for i, want := range []string{"fire", "fall", "landing"} {
			if !p.Next() || p.CurrentName() != want {
				t.Fatalf("cut %d must land on %q, got %q", i+1, want, p.CurrentName())
			}
		}
		if p.Next() {
			t.Fatal("after landing there is nothing left — the show ends")
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
		for _, e := range Bill() {
			switch e.Name {
			case "arrival", "dsky", "descent orbit", "the end":
				t.Fatalf("the close-up bill must not carry premiere scene %q", e.Name)
			}
		}
		sc := Bill()[0].Scene
		sc.Start()
		defer sc.Stop()
		sc.Update(lander.FlyInHoldSeconds + lander.FlyInSeconds)
		v := render(sc)
		if strings.Contains(v, "VERB") {
			t.Fatal("the DSKY does not appear in the close-up")
		}
		if strings.Contains(v, "THE END") || strings.Contains(v, "___") {
			t.Fatal("the end card does not appear in the close-up")
		}
	})
	t.Run("unhappy: the horizon is not a round disc in the middle of the sky", func(t *testing.T) {
		sc := Bill()[3].Scene
		sc.Start()
		defer sc.Stop()
		plain := stripANSI(render(sc))
		lines := strings.Split(plain, "\n")
		if len(lines) > stageH/2 {
			lines = lines[:stageH/2]
		}
		top := strings.Join(lines, "\n")
		if strings.ContainsRune(top, '▓') {
			t.Fatal("the landing moon must sit on the bottom, not as a disc in the sky")
		}
	})
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			if i < len(s) {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func moonRows(plain string, col int) int {
	n := 0
	for _, line := range strings.Split(plain, "\n") {
		rs := []rune(line)
		if col < 0 || col >= len(rs) {
			continue
		}
		switch rs[col] {
		case '▓', '▒', '░':
			n++
		}
	}
	return n
}
