package mainshow

// Tests written FIRST: MAIN is the one that puts everything together
// — a composable thirteen-scene bill that is every numbered show's
// bill added together, in shelf order: 01. Moon Orbit (the bare moon,
// the fly-in to orbit), 02. Walkthrough (pause, close-up, fire, fall,
// landing), 03. Mario (run, flagpole, board), then 04. Inverse
// Walkthrough (liftoff, engines on, engines off). Every entry carries
// the same performer its home show casts — the knobbed Shows keep
// their types so the editor can reach their knobs, and the bobble
// entries keep the bill's word on the engine. Each call builds fresh
// instances, and none of the old premiere's scenes ride along.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/bobble"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/fall"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/landing"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/liftoff"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/moonwalk"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

var mainNames = []string{
	"the moon", "orbit",
	"pause", "Lunar Lander Close-Up", "fire", "fall", "landing",
	"run", "flagpole", "board",
	"liftoff", "engines on", "engines off",
}

func TestMainBill(t *testing.T) {
	t.Run("happy: the bill is every show's scenes in shelf order", func(t *testing.T) {
		b := Bill()
		if len(b) != len(mainNames) {
			t.Fatalf("MAIN holds %d scenes, want %d", len(b), len(mainNames))
		}
		for i, want := range mainNames {
			if b[i].Name != want {
				t.Fatalf("scene %d is %q, want %q", i+1, b[i].Name, want)
			}
			if b[i].Scene == nil {
				t.Fatalf("scene %q has no performer", want)
			}
		}
	})
	t.Run("happy: the show is called MAIN and its holds live beside the bill", func(t *testing.T) {
		if Title != "MAIN" {
			t.Fatalf("the show is called %q, want MAIN", Title)
		}
		if HoldsPath != "shows/mainshow/config.json" {
			t.Fatalf("the holds live at %q, want shows/mainshow/config.json", HoldsPath)
		}
	})
	t.Run("happy: the knobbed scenes keep their own types for the editor", func(t *testing.T) {
		b := Bill()
		byName := map[string]screenplay.Scene{}
		for _, e := range b {
			byName[e.Name] = e.Scene
		}
		if _, ok := byName["fall"].(*fall.Show); !ok {
			t.Fatalf("fall is %T, want the fall show", byName["fall"])
		}
		if _, ok := byName["landing"].(*landing.Show); !ok {
			t.Fatalf("landing is %T, want the landing show", byName["landing"])
		}
		if _, ok := byName["liftoff"].(*liftoff.Show); !ok {
			t.Fatalf("liftoff is %T, want the liftoff show", byName["liftoff"])
		}
		beats := map[string]moonwalk.Beat{
			"run":      moonwalk.BeatRun,
			"flagpole": moonwalk.BeatPole,
			"board":    moonwalk.BeatBoard,
		}
		for name, beat := range beats {
			sc, ok := byName[name].(*moonwalk.Show)
			if !ok {
				t.Fatalf("%s is %T, want the moonwalk show", name, byName[name])
			}
			if sc.Beat() != beat {
				t.Fatalf("%s plays beat %v, want %v", name, sc.Beat(), beat)
			}
		}
	})
	t.Run("happy: the bobble entries keep the bill's word on the engine", func(t *testing.T) {
		b := Bill()
		lit, ok := b[11].Scene.(*bobble.Show)
		if !ok || b[11].Name != "engines on" {
			t.Fatalf("scene 12 is %q %T, want the lit bobble", b[11].Name, b[11].Scene)
		}
		if !lit.Cfg.Engine {
			t.Fatal("engines on must burn")
		}
		dark, ok := b[12].Scene.(*bobble.Show)
		if !ok || b[12].Name != "engines off" {
			t.Fatalf("scene 13 is %q %T, want the dark bobble", b[12].Name, b[12].Scene)
		}
		if dark.Cfg.Engine {
			t.Fatal("engines off must fly cold")
		}
	})
	t.Run("happy: every call builds a fresh cast", func(t *testing.T) {
		one, two := Bill(), Bill()
		for i := range one {
			if one[i].Scene == two[i].Scene {
				t.Fatalf("scene %q is shared between calls — the bills must be independent", one[i].Name)
			}
		}
	})
	t.Run("happy: the composed show walks all thirteen scenes and then has nothing left", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		defer p.Stop()
		if p.Len() != len(mainNames) || p.CurrentName() != "the moon" {
			t.Fatalf("the show opens on %d %q, want thirteen starting on the moon", p.Len(), p.CurrentName())
		}
		for i, want := range mainNames[1:] {
			if !p.Next() || p.CurrentName() != want {
				t.Fatalf("cut %d must land on %q, got %q", i+1, want, p.CurrentName())
			}
		}
		if p.Next() {
			t.Fatal("after engines off there is nothing left — the show ends")
		}
	})
	t.Run("unhappy: none of the old premiere rides along", func(t *testing.T) {
		for _, e := range Bill() {
			switch e.Name {
			case "arrival", "dsky", "descent orbit", "the end":
				t.Fatalf("MAIN must not carry old premiere scene %q", e.Name)
			}
		}
	})
	t.Run("unhappy: a scene stopped before its first render never panics", func(t *testing.T) {
		for _, e := range Bill() {
			e.Scene.Start()
			e.Scene.Update(1)
			e.Scene.Stop()
		}
	})
}
