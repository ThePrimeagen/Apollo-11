package astro

// Tests written FIRST. Clips are the named animations the scenes play:
// Run is the three running frames in order, Pole is the two slide
// grips alternating, Jump is its single airborne pose. Each clip is a
// sprite.Animation — a list of sprites played in order — built from
// the atlas, and each refuses an atlas that lost a pose.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// fingerprint flattens a sprite so two frames can be compared.
func fingerprint(sp sprite.Sprite) string {
	out := ""
	for _, row := range sp.GlyphRows() {
		out += row + "\n"
	}
	return out
}

func mustAtlas(t *testing.T) *sprite.Atlas {
	t.Helper()
	a, err := BuildAtlas()
	if err != nil {
		t.Fatalf("BuildAtlas: %v", err)
	}
	return a
}

func TestClips(t *testing.T) {
	t.Run("happy: Run plays run1, run2, run3 in order and wraps", func(t *testing.T) {
		a := mustAtlas(t)
		run, err := Run(a)
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if run.Len() != 3 {
			t.Fatalf("Run has %d frames, want 3", run.Len())
		}
		if run.FPS != RunFPS {
			t.Fatalf("Run FPS = %v, want %v", run.FPS, RunFPS)
		}
		for i, pose := range []sprite.Heading{PoseRun1, PoseRun2, PoseRun3} {
			want, _ := a.Frame(Size, pose)
			if fingerprint(run.Frame(i)) != fingerprint(want) {
				t.Fatalf("Run frame %d is not %q", i, pose)
			}
		}
		if fingerprint(run.Frame(3)) != fingerprint(run.Frame(0)) {
			t.Fatal("the run cycle must wrap back to its first stride")
		}
	})
	t.Run("happy: Pole alternates the two grips; Jump holds its one pose", func(t *testing.T) {
		a := mustAtlas(t)
		pole, err := Pole(a)
		if err != nil {
			t.Fatalf("Pole: %v", err)
		}
		if pole.Len() != 2 || pole.FPS != PoleFPS {
			t.Fatalf("Pole has %d frames at %v fps, want 2 at %v", pole.Len(), pole.FPS, PoleFPS)
		}
		if fingerprint(pole.Frame(0)) == fingerprint(pole.Frame(1)) {
			t.Fatal("the two pole grips must differ or the slide freezes")
		}
		jump, err := Jump(a)
		if err != nil {
			t.Fatalf("Jump: %v", err)
		}
		if jump.Len() != 1 {
			t.Fatalf("Jump has %d frames, want 1", jump.Len())
		}
		if fingerprint(jump.Frame(0)) != fingerprint(jump.Frame(7)) {
			t.Fatal("a one-frame clip must hold that frame forever")
		}
	})
	t.Run("unhappy: an atlas missing a pose fails loudly for every clip", func(t *testing.T) {
		empty := &sprite.Atlas{Palette: Palette}
		if _, err := Run(empty); err == nil {
			t.Fatal("Run on an empty atlas must error")
		}
		if _, err := Jump(empty); err == nil {
			t.Fatal("Jump on an empty atlas must error")
		}
		if _, err := Pole(empty); err == nil {
			t.Fatal("Pole on an empty atlas must error")
		}
	})
	t.Run("unhappy: a nil atlas errors, never panics", func(t *testing.T) {
		if _, err := Run(nil); err == nil {
			t.Fatal("Run(nil) must error")
		}
	})
}
