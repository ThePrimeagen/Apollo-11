package shotgun

// Tests written FIRST: Shotgun is the Doom pump gun as a scene
// component, aimed on the same eight-point compass the gunfire blast
// already speaks. Start builds every heading for a w×h stage; Aim
// points the barrel; Fire pulls the trigger so the muzzle flame leaps
// from this heading's barrel tip; Update burns the blast; Render
// paints the aimed gun with whatever flame is still in the air.
// W is the horizontal mirror of E, S the vertical mirror of N — the
// four cardinals and four diagonals are all distinct, and an east
// barrel is a long gun while a north barrel is a tall one. Before
// Start and after Stop the stage is empty; a heading off the compass
// is refused; the trigger needs a stage.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 72
	stageH = 24
)

// The compile-time pin: a Gun plays as a screenplay component.
var _ screenplay.Component = (*Gun)(nil)

func opaqueCount(sp sprite.Sprite) int {
	n := 0
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if !sp.At(r, c).Transparent() {
				n++
			}
		}
	}
	return n
}

func bounds(sp sprite.Sprite) (minC, minR, maxC, maxR, n int) {
	minC, minR = sp.Width, sp.Height
	maxC, maxR = -1, -1
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if sp.At(r, c).Transparent() {
				continue
			}
			n++
			if c < minC {
				minC = c
			}
			if c > maxC {
				maxC = c
			}
			if r < minR {
				minR = r
			}
			if r > maxR {
				maxR = r
			}
		}
	}
	return
}

func sameSprite(a, b sprite.Sprite) bool {
	if a.Width != b.Width || a.Height != b.Height {
		return false
	}
	for r := 0; r < a.Height; r++ {
		for c := 0; c < a.Width; c++ {
			if a.At(r, c) != b.At(r, c) {
				return false
			}
		}
	}
	return true
}

func liveBlast(g *Gun) int {
	if g == nil || g.Blast == nil || g.Blast.Core == nil {
		return 0
	}
	n := len(g.Blast.Core.Particles)
	for _, e := range g.Blast.Flames {
		if e != nil {
			n += len(e.Particles)
		}
	}
	return n
}

func TestNewGun(t *testing.T) {
	t.Run("happy: a fresh gun aims east — the Doom side-on stock shot — and has not fired", func(t *testing.T) {
		g := New()
		if g == nil {
			t.Fatal("New must return a gun")
		}
		if g.Heading() != sprite.E {
			t.Fatalf("stock heading %q, want E", g.Heading())
		}
		if liveBlast(g) != 0 {
			t.Fatal("a fresh gun must hold fire")
		}
	})
	t.Run("unhappy: methods on a nil gun never panic", func(t *testing.T) {
		var g *Gun
		g.Start(stageW, stageH)
		g.Update(1)
		g.Aim(sprite.N)
		g.Step(1)
		if g.Fire() {
			t.Fatal("a nil gun must refuse the trigger")
		}
		if n := opaqueCount(g.Render()); n != 0 {
			t.Fatalf("a nil gun rendered %d cells", n)
		}
		g.Stop()
	})
}

func TestStart(t *testing.T) {
	t.Run("happy: Start builds all eight headings, every one a painted gun", func(t *testing.T) {
		g := New()
		g.Start(stageW, stageH)
		for _, h := range sprite.Headings {
			sp := g.Frame(h)
			if n := opaqueCount(sp); n < 12 {
				t.Fatalf("%s frame is nearly empty (%d cells) — a shotgun must read as a gun", h, n)
			}
		}
		stage := g.Render()
		if stage.Width != stageW || stage.Height != stageH {
			t.Fatalf("stage %dx%d, want %dx%d", stage.Width, stage.Height, stageW, stageH)
		}
		if n := opaqueCount(stage); n < 12 {
			t.Fatalf("the aimed gun is missing from the stage (%d cells)", n)
		}
	})
	t.Run("unhappy: before Start and after Stop the stage is empty", func(t *testing.T) {
		g := New()
		if n := opaqueCount(g.Render()); n != 0 {
			t.Fatalf("unstarted gun painted %d cells", n)
		}
		g.Start(stageW, stageH)
		g.Stop()
		if n := opaqueCount(g.Render()); n != 0 {
			t.Fatalf("stopped gun painted %d cells", n)
		}
		if g.Frame(sprite.E).Width != 0 && opaqueCount(g.Frame(sprite.E)) != 0 {
			t.Fatal("Stop must drop the frames so a stopped gun holds nothing")
		}
	})
}

func TestEightDirections(t *testing.T) {
	t.Run("happy: the eight compass frames are all distinct, W mirrors E, S mirrors N", func(t *testing.T) {
		g := New()
		g.Start(stageW, stageH)
		seen := map[string]sprite.Heading{}
		for _, h := range sprite.Headings {
			sp := g.Frame(h)
			key := sprite.Render(sp)
			if other, ok := seen[key]; ok {
				t.Fatalf("%s is identical to %s — every heading needs its own gun", h, other)
			}
			seen[key] = h
		}
		if !sameSprite(g.Frame(sprite.W), sprite.FlipH(g.Frame(sprite.E))) {
			t.Fatal("W must be the horizontal mirror of E")
		}
		if !sameSprite(g.Frame(sprite.S), sprite.FlipV(g.Frame(sprite.N))) {
			t.Fatal("S must be the vertical mirror of N")
		}
		if !sameSprite(g.Frame(sprite.NW), sprite.FlipH(g.Frame(sprite.NE))) {
			t.Fatal("NW must be the horizontal mirror of NE")
		}
		if !sameSprite(g.Frame(sprite.SE), sprite.FlipV(g.Frame(sprite.NE))) {
			t.Fatal("SE must be the vertical mirror of NE")
		}
	})
	t.Run("happy: an east barrel is a long gun, a north barrel is a tall one", func(t *testing.T) {
		g := New()
		g.Start(stageW, stageH)
		minC, minR, maxC, maxR, n := bounds(g.Frame(sprite.E))
		if n == 0 {
			t.Fatal("east gun is empty")
		}
		if w, h := maxC-minC+1, maxR-minR+1; w <= h {
			t.Fatalf("east shotgun should be wider than tall, got %dx%d", w, h)
		}
		minC, minR, maxC, maxR, n = bounds(g.Frame(sprite.N))
		if n == 0 {
			t.Fatal("north gun is empty")
		}
		if w, h := maxC-minC+1, maxR-minR+1; h <= w {
			t.Fatalf("north shotgun should be taller than wide, got %dx%d", w, h)
		}
	})
	t.Run("unhappy: a heading off the compass has no frame", func(t *testing.T) {
		g := New()
		g.Start(stageW, stageH)
		if n := opaqueCount(g.Frame(sprite.Heading("up"))); n != 0 {
			t.Fatalf("off-compass frame painted %d cells", n)
		}
	})
}

func TestAim(t *testing.T) {
	t.Run("happy: Aim points the barrel and Render follows it", func(t *testing.T) {
		g := New()
		g.Start(stageW, stageH)
		east := g.Render()
		if !g.Aim(sprite.N) {
			t.Fatal("Aim N must be accepted")
		}
		if g.Heading() != sprite.N {
			t.Fatalf("heading %q, want N", g.Heading())
		}
		north := g.Render()
		if sameSprite(east, north) {
			t.Fatal("aiming north must change the stage — the barrel has to turn")
		}
	})
	t.Run("happy: Step walks the compass clockwise and wraps", func(t *testing.T) {
		g := New()
		g.Start(stageW, stageH)
		_ = g.Aim(sprite.N)
		if got := g.Step(1); got != sprite.NE {
			t.Fatalf("one step from N landed on %q, want NE", got)
		}
		_ = g.Aim(sprite.NW)
		if got := g.Step(1); got != sprite.N {
			t.Fatalf("step off NW must wrap to N, got %q", got)
		}
		_ = g.Aim(sprite.N)
		if got := g.Step(-1); got != sprite.NW {
			t.Fatalf("back from N must wrap to NW, got %q", got)
		}
	})
	t.Run("unhappy: Aim off the compass is a refused no-op", func(t *testing.T) {
		g := New()
		g.Start(stageW, stageH)
		before := g.Heading()
		if g.Aim(sprite.Heading("sideways")) {
			t.Fatal("a heading off the compass must be refused")
		}
		if g.Heading() != before {
			t.Fatalf("refused Aim mutated the heading to %q", g.Heading())
		}
	})
}

func TestFire(t *testing.T) {
	t.Run("happy: Fire bursts the muzzle flame along the aimed heading", func(t *testing.T) {
		g := New()
		g.Start(stageW, stageH)
		_ = g.Aim(sprite.NE)
		if !g.Fire() {
			t.Fatal("Fire after Start must pull the trigger")
		}
		if liveBlast(g) == 0 {
			t.Fatal("the trigger must throw particles")
		}
		g.Update(1.0 / 30)
		stage := g.Render()
		if n := opaqueCount(stage); n < 12 {
			t.Fatalf("a fired gun must still paint the shotgun, got %d cells", n)
		}
	})
	t.Run("unhappy: the trigger needs a stage — Fire before Start is refused", func(t *testing.T) {
		g := New()
		if g.Fire() {
			t.Fatal("Fire before Start must report the refused trigger")
		}
		g.Start(stageW, stageH)
		g.Update(1.0 / 30)
		if liveBlast(g) != 0 {
			t.Fatalf("a pre-Start trigger must not fire later, found %d particles", liveBlast(g))
		}
	})
}

func TestAtlas(t *testing.T) {
	t.Run("happy: BuildAtlas carries all eight headings and the shipped file matches", func(t *testing.T) {
		built, err := BuildAtlas()
		if err != nil {
			t.Fatalf("BuildAtlas: %v", err)
		}
		for _, h := range sprite.Headings {
			if _, ok := built.Frame(Size, h); !ok {
				t.Fatalf("atlas missing heading %s", h)
			}
		}
		path := FindAtlas()
		if path == "" {
			t.Fatal("FindAtlas must locate assets/shotgun.json")
		}
		loaded, err := Load()
		if err != nil {
			t.Fatalf("the shipped atlas must load: %v", err)
		}
		for _, h := range sprite.Headings {
			a, _ := built.Frame(Size, h)
			b, ok := loaded.Frame(Size, h)
			if !ok {
				t.Fatalf("shipped atlas missing %s", h)
			}
			if !sameSprite(a, b) {
				t.Fatalf("shipped %s does not match BuildAtlas", h)
			}
		}
	})
	t.Run("unhappy: LoadPath of a missing file is an error, never a blank gun", func(t *testing.T) {
		if _, err := LoadPath(t.TempDir() + "/no-such-shotgun.json"); err == nil {
			t.Fatal("a missing atlas must error")
		}
	})
}
