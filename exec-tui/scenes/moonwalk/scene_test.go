package moonwalk

// Tests written FIRST. One cycle of the moonwalk scene: run in, jump
// onto the low crate, jump again onto the tall stack — twice up, nice
// and high — leap off the top and land on the very tip of the
// flagpole, slide down while the American flag rises past him to the
// top, stand at the base, and then the camera pans a little to the
// right to find the lunar rover parked in the dark. Every beat is a
// pure function of the knobs, the stage, and the clock.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/astro"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	tw = 84
	th = 30
)

type sample struct {
	t    float64
	pose sprite.Heading
	x, y int
}

func sweep(cfg Config, w, h int) []sample {
	var out []sample
	cyc := CycleSeconds(cfg, w, h)
	for t := 0.0; t < cyc; t += 1.0 / 60 {
		pose, x, y := timelineAt(cfg, w, h, t)
		out = append(out, sample{t, pose, x, y})
	}
	return out
}

func isRun(p sprite.Heading) bool {
	return p == astro.PoseRun1 || p == astro.PoseRun2 || p == astro.PoseRun3
}

func isPole(p sprite.Heading) bool {
	return p == astro.PosePole1 || p == astro.PosePole2
}

func mustSceneAtlas(t *testing.T) *sprite.Atlas {
	t.Helper()
	a, err := astro.BuildAtlas()
	if err != nil {
		t.Fatalf("BuildAtlas: %v", err)
	}
	return a
}

func TestChoreography(t *testing.T) {
	cfg := DefaultConfig()
	grounded := groundedY(th)
	t.Run("happy: he opens running on the ground, left to right", func(t *testing.T) {
		samples := sweep(cfg, tw, th)
		if !isRun(samples[0].pose) || samples[0].y != grounded {
			t.Fatalf("the cycle opens running on the ground, got %q y %d", samples[0].pose, samples[0].y)
		}
		lastX := samples[0].x
		for _, s := range samples {
			if isPole(s.pose) {
				break
			}
			if s.x < lastX {
				t.Fatalf("t=%.2f: x went backward (%d after %d)", s.t, s.x, lastX)
			}
			lastX = s.x
		}
	})
	t.Run("happy: two jumps land him on two ever-higher blocks", func(t *testing.T) {
		samples := sweep(cfg, tw, th)
		yA := grounded - blockRows
		yB := grounded - 2*blockRows
		onA, onB, aBeforeB := false, false, false
		for _, s := range samples {
			if isPole(s.pose) {
				break
			}
			if s.pose != astro.PoseJump {
				if s.y == yA {
					onA = true
				}
				if s.y == yB {
					onB = true
					if onA {
						aBeforeB = true
					}
				}
			}
		}
		if !onA {
			t.Fatalf("he never lands on the low crate (y %d)", yA)
		}
		if !onB {
			t.Fatalf("he never lands on the tall stack (y %d)", yB)
		}
		if !aBeforeB {
			t.Fatal("the low crate must come before the tall stack — two jumps, ever higher")
		}
	})
	t.Run("happy: the leap off the stack lands on the very top of the pole", func(t *testing.T) {
		samples := sweep(cfg, tw, th)
		var pole []sample
		for _, s := range samples {
			if isPole(s.pose) {
				pole = append(pole, s)
			}
		}
		if len(pole) == 0 {
			t.Fatal("he never reaches the pole")
		}
		if got, want := pole[0].y, poleTopRow(cfg, th); got != want {
			t.Fatalf("the grab starts at y %d, want the pole tip %d", got, want)
		}
		wantX := poleCol(tw) - astro.GripCol
		lastY := pole[0].y
		for _, s := range pole {
			if s.x != wantX {
				t.Fatalf("t=%.2f: sliding at x %d, want %d", s.t, s.x, wantX)
			}
			if s.y < lastY {
				t.Fatalf("t=%.2f: the slide went up", s.t)
			}
			lastY = s.y
		}
		if pole[len(pole)-1].y != grounded {
			t.Fatalf("the slide ends at y %d, want the ground %d", pole[len(pole)-1].y, grounded)
		}
	})
	t.Run("happy: the flag goes up while he goes down", func(t *testing.T) {
		samples := sweep(cfg, tw, th)
		var start, end float64
		for _, s := range samples {
			if isPole(s.pose) {
				if start == 0 {
					start = s.t
				}
				end = s.t
			}
		}
		top0 := flagTopAt(cfg, tw, th, start)
		top1 := flagTopAt(cfg, tw, th, end)
		if top1 >= top0 {
			t.Fatalf("the flag must rise during the slide: top went %d -> %d", top0, top1)
		}
		_, _, y0 := timelineAt(cfg, tw, th, start)
		_, _, y1 := timelineAt(cfg, tw, th, end)
		if y1 <= y0 {
			t.Fatal("he must descend while the flag rises")
		}
		cyc := CycleSeconds(cfg, tw, th)
		if got, want := flagTopAt(cfg, tw, th, cyc-0.01), poleTopRow(cfg, th)+1; got != want {
			t.Fatalf("by the end the flag flies at the top: %d, want %d", got, want)
		}
	})
	t.Run("happy: the camera holds, then pans a little to the rover", func(t *testing.T) {
		cyc := CycleSeconds(cfg, tw, th)
		samples := sweep(cfg, tw, th)
		for _, s := range samples {
			if isPole(s.pose) || isRun(s.pose) || s.pose == astro.PoseJump {
				if cam := cameraAt(cfg, tw, th, s.t); cam != 0 {
					t.Fatalf("t=%.2f: the camera moved (%d) before the action ended", s.t, cam)
				}
			}
		}
		if got := cameraAt(cfg, tw, th, cyc-0.01); got != cfg.PanCols {
			t.Fatalf("the pan must finish at %d cols, got %d", cfg.PanCols, got)
		}
		rx := roverX(cfg, tw)
		if rx < tw {
			t.Fatalf("the rover (x %d) must start beyond the %d-wide viewport", rx, tw)
		}
		if rx+astro.RoverPxW > tw+cfg.PanCols {
			t.Fatalf("the pan (%d) must fully reveal the rover (x %d + %d)", cfg.PanCols, rx, astro.RoverPxW)
		}
	})
	t.Run("happy: the scene renders the props into the viewport", func(t *testing.T) {
		a := mustSceneAtlas(t)
		early := Frame(cfg, a, tw, th, 0.2)
		if early.Width != tw || early.Height != th {
			t.Fatalf("frame is %dx%d, want %dx%d", early.Width, early.Height, tw, th)
		}
		var poleGlyph, blockCells bool
		for r := 0; r < early.Height; r++ {
			for c := 0; c < early.Width; c++ {
				if early.At(r, c).Ch == '│' {
					poleGlyph = true
				}
			}
		}
		aTop := groundRow(th) - blockRows
		for c := blockAX(tw); c < blockAX(tw)+astro.BlockPxW; c++ {
			if !early.At(aTop, c).Transparent() {
				blockCells = true
			}
		}
		if !poleGlyph {
			t.Fatal("the flagpole must be on stage")
		}
		if !blockCells {
			t.Fatal("the crates must be on stage")
		}
		cyc := CycleSeconds(cfg, tw, th)
		before := Frame(cfg, a, tw, th, cyc-cfg.PanSeconds-2.0)
		after := Frame(cfg, a, tw, th, cyc-0.01)
		if fingerprint(before) == fingerprint(after) {
			t.Fatal("the pan must change what the viewport shows")
		}
	})
	t.Run("happy: one cycle later everything repeats exactly", func(t *testing.T) {
		cyc := CycleSeconds(cfg, tw, th)
		p0, x0, y0 := timelineAt(cfg, tw, th, 0.7)
		p1, x1, y1 := timelineAt(cfg, tw, th, 0.7+cyc)
		if p0 != p1 || x0 != x1 || y0 != y1 {
			t.Fatal("the loop must repeat exactly")
		}
		if cameraAt(cfg, tw, th, 0.7) != cameraAt(cfg, tw, th, 0.7+cyc) {
			t.Fatal("the camera must rewind with the loop")
		}
	})
}

func fingerprint(sp sprite.Sprite) string {
	out := ""
	for _, row := range sp.GlyphRows() {
		out += row + "\n"
	}
	return out
}

func TestKnobsShapeTheScene(t *testing.T) {
	t.Run("happy: a faster runner finishes the cycle sooner", func(t *testing.T) {
		slow := DefaultConfig()
		fast := DefaultConfig()
		fast.RunSpeed = slow.RunSpeed * 2
		if CycleSeconds(fast, tw, th) >= CycleSeconds(slow, tw, th) {
			t.Fatal("doubling the ground speed must shorten the cycle")
		}
	})
	t.Run("happy: a taller pole grabs higher", func(t *testing.T) {
		short := DefaultConfig()
		tall := DefaultConfig()
		tall.PoleRows = short.PoleRows + 4
		if poleTopRow(tall, th) >= poleTopRow(short, th) {
			t.Fatal("more pole rows must raise the tip")
		}
	})
	t.Run("happy: the stride knob changes which frame plays", func(t *testing.T) {
		slow := DefaultConfig()
		slow.StrideFPS = 3
		fast := DefaultConfig()
		fast.StrideFPS = 24
		differs := false
		for _, tt := range []float64{0.15, 0.25, 0.35, 0.45} {
			p0, _, _ := timelineAt(slow, tw, th, tt)
			p1, _, _ := timelineAt(fast, tw, th, tt)
			if p0 != p1 {
				differs = true
			}
		}
		if !differs {
			t.Fatal("the stride knob must change the playing frame")
		}
	})
	t.Run("happy: a slower hoist has the flag lower at mid-slide", func(t *testing.T) {
		quick := DefaultConfig()
		quick.FlagSeconds = 0.4
		lazy := DefaultConfig()
		lazy.FlagSeconds = 4.0
		var mid float64
		for _, s := range sweep(quick, tw, th) {
			if isPole(s.pose) {
				mid = s.t + 0.2
				break
			}
		}
		if flagTopAt(quick, tw, th, mid) >= flagTopAt(lazy, tw, th, mid) {
			t.Fatal("a quick hoist must be higher than a lazy one mid-slide")
		}
	})
	t.Run("unhappy: time before the curtain clamps to the opening run", func(t *testing.T) {
		pose, _, y := timelineAt(DefaultConfig(), tw, th, -3)
		if !isRun(pose) || y != groundedY(th) {
			t.Fatalf("t<0 must clamp to the opening stride, got %q y %d", pose, y)
		}
	})
	t.Run("unhappy: a tiny stage still answers and renders", func(t *testing.T) {
		cfg := DefaultConfig()
		pose, _, _ := timelineAt(cfg, 8, 6, 1.0)
		if pose == "" {
			t.Fatal("a tiny stage must still name a pose")
		}
		a := mustSceneAtlas(t)
		sp := Frame(cfg, a, 8, 6, 1.0)
		if sp.Width != 8 || sp.Height != 6 {
			t.Fatalf("tiny frame is %dx%d, want 8x6", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a nil atlas renders the set, never panics", func(t *testing.T) {
		sp := Frame(DefaultConfig(), nil, tw, th, 1.0)
		if sp.Width != tw || sp.Height != th {
			t.Fatalf("nil-atlas frame is %dx%d, want %dx%d", sp.Width, sp.Height, tw, th)
		}
	})
}
