package mario

// Tests written FIRST: 03. Mario is a composable three-scene bill —
// the astronaut's flagpole run. Scene one, "run": he sprints in from
// the left wing and climbs three crate stacks, then holds on the top
// crate. Scene two, "flagpole": the leap onto the gold ball, a beat
// at the top, the slide down while the flag flies up, then the bow.
// Scene three, "board": the camera pans to the lunar module, he runs
// over, jumps the hatch, and vanishes; the empty pad holds. After
// that there is nothing left — the runner ends the show.
//
// The moonwalk scene plays all three beats. The bill is the
// composable unit: append it to other shows' bills and hand the lot
// to screenplay.Compose.

import (
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/moonwalk"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 84
	stageH = 30
)

var sceneNames = []string{"run", "flagpole", "board"}

func openShow() (*screenplay.Screenplay, *screenplay.Screen) {
	p := screenplay.Compose(Bill())
	p.Start()
	scr := screenplay.NewScreen(stageW, stageH)
	p.Render(scr)
	return p, scr
}

func run(p *screenplay.Screenplay, seconds float64) {
	const dt = 1.0 / 30
	for t := 0.0; t < seconds-dt/2; t += dt {
		p.Update(dt)
	}
}

func frame(p *screenplay.Screenplay, scr *screenplay.Screen) string {
	p.Render(scr)
	return scr.Render()
}

func hasPole(v string) bool {
	return strings.ContainsRune(v, '│') && strings.ContainsRune(v, '●')
}

func TestMarioBill(t *testing.T) {
	t.Cleanup(moonwalk.Reset)
	t.Run("happy: the bill is three scenes in playing order", func(t *testing.T) {
		b := Bill()
		if len(b) != 3 {
			t.Fatalf("mario holds %d scenes, want 3", len(b))
		}
		for i, want := range sceneNames {
			if b[i].Name != want {
				t.Fatalf("scene %d is %q, want %q", i+1, b[i].Name, want)
			}
			if b[i].Scene == nil {
				t.Fatalf("scene %q has no performer", want)
			}
		}
	})
	t.Run("happy: the moonwalk scene plays all three beats", func(t *testing.T) {
		b := Bill()
		want := []moonwalk.Beat{moonwalk.BeatRun, moonwalk.BeatPole, moonwalk.BeatBoard}
		for i, beat := range want {
			sc, ok := b[i].Scene.(*moonwalk.Show)
			if !ok {
				t.Fatalf("scene %q must be the moonwalk, got %T", b[i].Name, b[i].Scene)
			}
			if sc.Beat() != beat {
				t.Fatalf("scene %q plays beat %v, want %v", b[i].Name, sc.Beat(), beat)
			}
		}
	})
	t.Run("happy: the show opens on the run — pole up, crates out, no flag, no module", func(t *testing.T) {
		p, scr := openShow()
		defer p.Stop()
		v := frame(p, scr)
		if !hasPole(v) {
			t.Fatal("the run opens on the flagpole set")
		}
		if strings.ContainsRune(v, '▟') {
			t.Fatal("the lunar module waits offstage until the board")
		}
		if p.CurrentName() != "run" || p.Len() != 3 {
			t.Fatalf("the show opens on %d %q, want three starting on run", p.Len(), p.CurrentName())
		}
	})
	t.Run("happy: the composed show walks the three scenes and then has nothing left", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		if p.Len() != 3 || p.CurrentName() != "run" {
			t.Fatalf("the show opens on %d %q, want three starting on run", p.Len(), p.CurrentName())
		}
		for i, want := range sceneNames[1:] {
			if !p.Next() || p.CurrentName() != want {
				t.Fatalf("cut %d must land on %q, got %q", i+1, want, p.CurrentName())
			}
		}
		if p.Next() {
			t.Fatal("after board there is nothing left — the show ends")
		}
		p.Stop()
	})
	t.Run("happy: the flagpole cut lands on the leap with the pole still on stage", func(t *testing.T) {
		p, scr := openShow()
		defer p.Stop()
		if !p.Next() {
			t.Fatal("the run must cut to the flagpole")
		}
		v := frame(p, scr)
		if !hasPole(v) {
			t.Fatal("the flagpole scene must keep the set")
		}
		if p.CurrentName() != "flagpole" {
			t.Fatalf("after the cut the show is %q, want flagpole", p.CurrentName())
		}
	})
	t.Run("happy: the board cut reveals the lunar module", func(t *testing.T) {
		p, scr := openShow()
		defer p.Stop()
		p.Next()
		p.Next()
		run(p, 4)
		v := frame(p, scr)
		if p.CurrentName() != "board" {
			t.Fatalf("two cuts must land on board, got %q", p.CurrentName())
		}
		if !strings.ContainsRune(v, '▟') {
			t.Fatal("the board scene must pan the lunar module on stage")
		}
	})
	t.Run("happy: the bill plays the active knobs on the first curtain", func(t *testing.T) {
		t.Cleanup(moonwalk.Reset)
		c := moonwalk.DefaultConfig()
		c.RunSpeed = 40
		if err := moonwalk.Use(c); err != nil {
			t.Fatal(err)
		}
		b := Bill()
		sc, ok := b[0].Scene.(*moonwalk.Show)
		if !ok {
			t.Fatal("run must be the moonwalk")
		}
		if sc.Cfg.RunSpeed != 40 {
			t.Fatalf("the run entry must carry the active sprint, speed %v want 40", sc.Cfg.RunSpeed)
		}
	})
	t.Run("unhappy: a scene stopped before its first render never panics", func(t *testing.T) {
		for _, e := range Bill() {
			e.Scene.Start()
			e.Scene.Update(1)
			e.Scene.Stop()
		}
	})
	t.Run("unhappy: mario is not the inverse walkthrough and not the premiere", func(t *testing.T) {
		for _, e := range Bill() {
			switch e.Name {
			case "liftoff", "engines on", "engines off":
				t.Fatalf("the mario bill must not carry inverse scene %q", e.Name)
			case "arrival", "dsky", "descent orbit", "the end":
				t.Fatalf("the mario bill must not carry premiere scene %q", e.Name)
			case "pause", "Lunar Lander Close-Up", "fire", "fall", "landing":
				t.Fatalf("the mario bill must not carry walkthrough scene %q", e.Name)
			}
		}
		p, scr := openShow()
		defer p.Stop()
		v := frame(p, scr)
		if strings.Contains(v, "VERB") {
			t.Fatal("the DSKY does not appear in mario")
		}
		if strings.Contains(v, "THE END") {
			t.Fatal("the end card does not appear in mario")
		}
	})
}
