package moonwalk

// Tests written FIRST. One cycle of the moonwalk show, staged tight:
// he runs in, climbs three crate stacks — one, two, three high, parked
// close to the pole so the last leap is a hop, not a superjump — lands
// with his fists on the gold ball, holds the top for a beat, slides
// down while the flag APPEARS at the base and flies up past him, bows,
// and then the camera pans right to the real lunar module (the house
// LM art, north-facing). He runs to it, jumps at its center hatch, and
// disappears inside. Every beat is a pure function of the knobs, the
// stage, and the clock.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/astro"
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	tw = 120
	th = 34
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
	t.Run("happy: three jumps land him on three ever-higher stacks", func(t *testing.T) {
		samples := sweep(cfg, tw, th)
		levels := []int{grounded - blockRows, grounded - 2*blockRows, grounded - 3*blockRows}
		landed := make([]bool, len(levels))
		next := 0
		for _, s := range samples {
			if isPole(s.pose) {
				break
			}
			if s.pose == astro.PoseStand && next < len(levels) && s.y == levels[next] {
				landed[next] = true
				next++
			}
		}
		for i, ok := range landed {
			if !ok {
				t.Fatalf("he never landed on stack %d (y %d) in order — one, two, three high", i+1, levels[i])
			}
		}
	})
	t.Run("happy: the last leap is a hop off the top stack, not a superjump", func(t *testing.T) {
		r := routeFor(cfg, tw, th)
		if dx := r.grabX - r.xC; dx < 2 || dx > 14 {
			t.Fatalf("the stacks must sit close to the pole: leap covers %d cols, want a hop (2..14)", dx)
		}
	})
	t.Run("happy: he grabs the ball, holds the top, then rides down", func(t *testing.T) {
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
		grab := poleTopRow(cfg, th) - 1
		if pole[0].y != grab {
			t.Fatalf("the grab starts at y %d, want on the ball at %d", pole[0].y, grab)
		}
		var held float64
		for _, s := range pole {
			if s.y != grab {
				break
			}
			held = s.t - pole[0].t
		}
		if held < cfg.TopSeconds*0.7 {
			t.Fatalf("he must hold the top about %.2fs, held %.2fs", cfg.TopSeconds, held)
		}
		wantX := poleCol(tw) - astro.GripCol
		lastY := pole[0].y
		for _, s := range pole {
			if s.x != wantX {
				t.Fatalf("t=%.2f: on the pole at x %d, want %d", s.t, s.x, wantX)
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
	t.Run("happy: the flag appears at the slide and flies up while he goes down", func(t *testing.T) {
		r := routeFor(cfg, tw, th)
		if _, visible := flagAt(cfg, tw, th, r.slideAt-0.05); visible {
			t.Fatal("the flag must not exist before he starts sliding")
		}
		early, visible := flagAt(cfg, tw, th, r.slideAt+0.05)
		if !visible {
			t.Fatal("the flag must appear the moment the slide starts")
		}
		if early < groundRow(th)-flagRows-2 {
			t.Fatalf("the flag appears near the base, got top %d", early)
		}
		late, _ := flagAt(cfg, tw, th, r.slideAt+cfg.SlideSeconds)
		if late >= early {
			t.Fatalf("the flag must rise: top went %d -> %d", early, late)
		}
		cyc := CycleSeconds(cfg, tw, th)
		end, visible := flagAt(cfg, tw, th, cyc-0.01)
		if !visible || end != poleTopRow(cfg, th)+1 {
			t.Fatalf("by the end the flag flies at the top: %d (visible %v), want %d", end, visible, poleTopRow(cfg, th)+1)
		}
	})
	t.Run("happy: after the pan he runs to the lunar module, jumps at its middle, and vanishes", func(t *testing.T) {
		samples := sweep(cfg, tw, th)
		r := routeFor(cfg, tw, th)
		var exitRun, boardJump, gone []sample
		for _, s := range samples {
			if s.t < r.exitAt {
				continue
			}
			switch {
			case isRun(s.pose):
				exitRun = append(exitRun, s)
			case s.pose == astro.PoseJump:
				boardJump = append(boardJump, s)
			case s.pose == PoseGone:
				gone = append(gone, s)
			}
		}
		if len(exitRun) == 0 {
			t.Fatal("he must run toward the module after the pan")
		}
		lastX := exitRun[0].x
		for _, s := range exitRun {
			if s.x < lastX {
				t.Fatalf("t=%.2f: the exit run went backward", s.t)
			}
			lastX = s.x
			if s.y != grounded {
				t.Fatalf("t=%.2f: the exit run left the ground", s.t)
			}
		}
		if len(boardJump) == 0 {
			t.Fatal("he must jump to board")
		}
		wantCenter := landerX(cfg, tw) + lander.BodyCols/2
		gotCenter := boardJump[0].x + astro.Cols/2
		if gotCenter < wantCenter-1 || gotCenter > wantCenter+1 {
			t.Fatalf("the boarding jump happens at x-center %d, want the module's middle %d", gotCenter, wantCenter)
		}
		up := false
		for _, s := range boardJump {
			if s.y < grounded {
				up = true
			}
		}
		if !up {
			t.Fatal("the boarding jump must leave the ground")
		}
		if len(gone) == 0 {
			t.Fatal("he must disappear into the module")
		}
		if gone[len(gone)-1].t < samples[len(samples)-1].t-0.1 {
			t.Fatal("once aboard he stays gone until the loop restarts")
		}
	})
	t.Run("happy: the camera holds through the action, then pans to the module", func(t *testing.T) {
		r := routeFor(cfg, tw, th)
		for _, tt := range []float64{0.1, r.slideAt - 0.1, r.slideAt + 0.2, r.panAt - 0.05} {
			if cam := cameraAt(cfg, tw, th, tt); cam != 0 {
				t.Fatalf("t=%.2f: the camera moved (%d) before the pan", tt, cam)
			}
		}
		cyc := CycleSeconds(cfg, tw, th)
		if got := cameraAt(cfg, tw, th, cyc-0.01); got != cfg.PanCols {
			t.Fatalf("the pan must finish at %d cols, got %d", cfg.PanCols, got)
		}
		lx := landerX(cfg, tw)
		if lx < tw {
			t.Fatalf("the module (x %d) must start beyond the %d-wide viewport", lx, tw)
		}
		if lx+lander.BodyCols > tw+cfg.PanCols {
			t.Fatalf("the pan (%d) must fully reveal the module (x %d + %d)", cfg.PanCols, lx, lander.BodyCols)
		}
	})
	t.Run("happy: the scene renders the set into the viewport", func(t *testing.T) {
		a := mustSceneAtlas(t)
		early := Frame(cfg, a, tw, th, 0.2)
		if early.Width != tw || early.Height != th {
			t.Fatalf("frame is %dx%d, want %dx%d", early.Width, early.Height, tw, th)
		}
		var poleGlyph bool
		for r := 0; r < early.Height; r++ {
			for c := 0; c < early.Width; c++ {
				if early.At(r, c).Ch == '│' {
					poleGlyph = true
				}
			}
		}
		if !poleGlyph {
			t.Fatal("the flagpole must be on stage")
		}
		stackTop := groundRow(th) - blockRows
		blockCells := false
		for c := stackX(cfg, tw, 0); c < stackX(cfg, tw, 0)+blockCols; c++ {
			if !early.At(stackTop, c).Transparent() {
				blockCells = true
			}
		}
		if !blockCells {
			t.Fatal("the crates must be on stage")
		}
		cyc := CycleSeconds(cfg, tw, th)
		end := Frame(cfg, a, tw, th, cyc-0.01)
		moduleInk := 0
		viewLX := landerX(cfg, tw) - cfg.PanCols
		for r := 0; r < end.Height; r++ {
			for c := viewLX; c < viewLX+lander.BodyCols; c++ {
				if !end.At(r, c).Transparent() {
					moduleInk++
				}
			}
		}
		if moduleInk < 40 {
			t.Fatalf("the module must be visible after the pan, found %d painted cells", moduleInk)
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

func TestKnobsShapeTheScene(t *testing.T) {
	t.Run("happy: a faster runner finishes the cycle sooner", func(t *testing.T) {
		slow := DefaultConfig()
		fast := DefaultConfig()
		fast.RunSpeed = slow.RunSpeed * 2
		if CycleSeconds(fast, tw, th) >= CycleSeconds(slow, tw, th) {
			t.Fatal("doubling the ground speed must shorten the cycle")
		}
	})
	t.Run("happy: box start moves the stacks", func(t *testing.T) {
		near := DefaultConfig()
		far := DefaultConfig()
		far.BoxStart = near.BoxStart + 12
		if stackX(far, tw, 0) >= stackX(near, tw, 0) {
			t.Fatal("a bigger box start must push the first stack farther from the pole")
		}
	})
	t.Run("happy: the top hold stretches the cycle", func(t *testing.T) {
		quick := DefaultConfig()
		quick.TopSeconds = 0.2
		lazy := DefaultConfig()
		lazy.TopSeconds = 2.0
		if CycleSeconds(lazy, tw, th) <= CycleSeconds(quick, tw, th) {
			t.Fatal("holding the top longer must lengthen the cycle")
		}
	})
	t.Run("happy: a faster exit reaches the module sooner", func(t *testing.T) {
		amble := DefaultConfig()
		amble.ExitSpeed = 6
		sprint := DefaultConfig()
		sprint.ExitSpeed = 30
		if CycleSeconds(sprint, tw, th) >= CycleSeconds(amble, tw, th) {
			t.Fatal("a faster exit must shorten the cycle")
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
	t.Run("happy: the lm gap knob parks the module nearer or farther from the flag", func(t *testing.T) {
		near := DefaultConfig()
		near.LMGap = MinLMGap
		far := DefaultConfig()
		far.LMGap = 60
		if got, want := landerX(near, tw), poleCol(tw)+MinLMGap; got != want {
			t.Fatalf("a near module parks at %d, want pole+%d = %d", got, MinLMGap, want)
		}
		if landerX(far, tw) <= landerX(near, tw) {
			t.Fatal("a bigger gap must push the module farther from the flag")
		}
		if got, want := landerX(DefaultConfig(), tw), tw+2; got != want {
			t.Fatalf("the default staging must keep the module just past the viewport: %d, want %d", got, want)
		}
	})
	t.Run("happy: a near module is on stage before any pan, and he still boards it dead center", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.LMGap = 4
		a := mustSceneAtlas(t)
		early := Frame(cfg, a, tw, th, 0.2)
		ink := 0
		lx := landerX(cfg, tw)
		for r := 0; r < early.Height; r++ {
			for c := lx; c < lx+lander.BodyCols && c < early.Width; c++ {
				if !early.At(r, c).Transparent() {
					ink++
				}
			}
		}
		if ink < 40 {
			t.Fatalf("a close module must be visible before the pan, found %d painted cells", ink)
		}
		r := routeFor(cfg, tw, th)
		wantCenter := lx + lander.BodyCols/2
		if got := r.boardX + astro.Cols/2; got < wantCenter-1 || got > wantCenter+1 {
			t.Fatalf("the boarding jump must follow the module: center %d, want %d", got, wantCenter)
		}
	})
	t.Run("happy: a far module still fits the world buffer", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.LMGap = cfg.PanCols + 40
		a := mustSceneAtlas(t)
		cyc := CycleSeconds(cfg, tw, th)
		end := Frame(cfg, a, tw, th, cyc-0.01)
		if end.Width != tw || end.Height != th {
			t.Fatalf("frame is %dx%d, want %dx%d", end.Width, end.Height, tw, th)
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
			t.Fatal("a tiny stage must still name a pose mid-action")
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
