package shotgun

// Tests written FIRST: Shotgun is the Doom pump gun as a scene
// component, aimed on the same eight-point compass the gunfire blast
// already speaks. It is one 2D side-on asset — the east stock shot —
// spun in the plane of the screen around the Y-axis coming out of it
// (east 0°, then counterclockwise: NE 45°, N 90°, …). That is not a
// 3D rotation: there is no separate rear/top/three-quarter drawing,
// and W is not FlipH(E). Start builds every heading for a w×h stage;
// Aim points the barrel; Fire pulls the trigger so the muzzle flame
// leaps from this heading's barrel tip using the shipped gunfire
// config (components/gunfire/config.json) for that heading only —
// never the whole eight-way rose. Update burns the blast; Render
// paints the aimed gun with whatever flame is still in the air.
// Before Start and after Stop the stage is empty; a heading off the
// compass is refused; the trigger needs a stage.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/gunfire"
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

func TestRotateGrid(t *testing.T) {
	t.Run("happy: 90° CCW around the Y-axis coming out of the screen stands a wide bar up", func(t *testing.T) {
		// X right, Y out of the screen: +90° is counterclockwise on
		// the page, so a bar along +X (east) stands along -row (north).
		in := []string{
			"BBB.",
			"....",
			"....",
			"....",
		}
		got := rotateGrid(in, 90)
		want := []string{
			"B...",
			"B...",
			"B...",
			"....",
		}
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("90° CCW:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	})
	t.Run("happy: 180° is the same 2D asset flipped both ways — in-plane, not a card-flip", func(t *testing.T) {
		in := []string{
			"BB..",
			"A...",
			"....",
			"....",
		}
		got := rotateGrid(in, 180)
		want := []string{
			"....",
			"....",
			"...A",
			"..BB",
		}
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("180°:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	})
	t.Run("unhappy: rotating an empty grid stays empty, and junk degrees still yield a grid of dots", func(t *testing.T) {
		empty := []string{"....", "...."}
		got := rotateGrid(empty, 45)
		for _, row := range got {
			if strings.Trim(row, ".") != "" {
				t.Fatalf("empty grid spun 45° painted %q", row)
			}
		}
		if got := rotateGrid(nil, 90); len(got) != 0 {
			t.Fatalf("nil grid must stay nil/empty, got %v", got)
		}
	})
}

func TestEightDirections(t *testing.T) {
	t.Run("happy: the eight compass frames are the same 2D gun spun in the screen plane", func(t *testing.T) {
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
		east := g.Frame(sprite.E)
		if sameSprite(g.Frame(sprite.W), sprite.FlipH(east)) {
			t.Fatal("W must not be FlipH(E) — that is a 3D card-flip, not an in-plane spin")
		}
		if !sameSprite(g.Frame(sprite.W), sprite.FlipH(sprite.FlipV(east))) {
			t.Fatal("W must be E spun 180° around the axis coming out of the screen (FlipH+FlipV)")
		}
		if !sameSprite(g.Frame(sprite.S), sprite.FlipH(sprite.FlipV(g.Frame(sprite.N)))) {
			t.Fatal("S must be N spun 180° around the axis coming out of the screen")
		}
	})
	t.Run("happy: an east barrel is a long side-on gun, north is that same gun stood on its stock", func(t *testing.T) {
		g := New()
		g.Start(stageW, stageH)
		minC, minR, maxC, maxR, n := bounds(g.Frame(sprite.E))
		if n == 0 {
			t.Fatal("east gun is empty")
		}
		ew, eh := maxC-minC+1, maxR-minR+1
		if ew <= eh {
			t.Fatalf("east shotgun should be wider than tall, got %dx%d", ew, eh)
		}
		minC, minR, maxC, maxR, n = bounds(g.Frame(sprite.N))
		if n == 0 {
			t.Fatal("north gun is empty")
		}
		nw, nh := maxC-minC+1, maxR-minR+1
		if nh <= nw {
			t.Fatalf("north shotgun should be taller than wide, got %dx%d", nw, nh)
		}
		// Same 2D pixels: a 90° spin swaps the bounding box, it does
		// not redraw a skinny 3D rear view.
		if nw > ew {
			t.Fatalf("north must be east stood up, not a wider 3D drawing: N %dx%d vs E %dx%d", nw, nh, ew, eh)
		}
	})
	t.Run("unhappy: a heading off the compass has no frame", func(t *testing.T) {
		g := New()
		g.Start(stageW, stageH)
		if n := opaqueCount(g.Frame(sprite.Heading("up"))); n != 0 {
			t.Fatalf("off-compass frame painted %d cells", n)
		}
	})
	t.Run("unhappy: NE is not a separately drawn 3D three-quarter — it is E spun 45°", func(t *testing.T) {
		g := New()
		g.Start(stageW, stageH)
		eN := opaqueCount(g.Frame(sprite.E))
		neN := opaqueCount(g.Frame(sprite.NE))
		if eN == 0 || neN == 0 {
			t.Fatal("east and NE guns must both paint")
		}
		// Nearest-neighbour 45° keeps most of the pixels; a skinny
		// 3D diagonal drawing would be far thinner than the side-on gun.
		if neN < eN/2 {
			t.Fatalf("NE painted %d cells, east %d — a 2D 45° spin must keep the same gun, not a thin 3D diagonal", neN, eN)
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
	t.Run("happy: Fire bursts only the aimed heading, using the shipped gunfire config", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		g := New()
		g.Start(stageW, stageH)
		shipped, err := gunfire.LoadBlast(gunfire.FindConfig())
		if err != nil {
			t.Fatalf("the shotgun must use the shipped gunfire config: %v", err)
		}
		_ = g.Aim(sprite.NE)
		if !g.Fire() {
			t.Fatal("Fire after Start must pull the trigger")
		}
		got := len(g.Blast.FlameAt(sprite.NE).Particles)
		want := shipped.ShotAt(sprite.NE).Count
		if got != want {
			t.Fatalf("NE flame burst %d, want the shipped gunfire config's %d", got, want)
		}
		for _, h := range sprite.Headings {
			if h == sprite.NE {
				continue
			}
			if n := len(g.Blast.FlameAt(h).Particles); n != 0 {
				t.Fatalf("Fire must not dump the whole rose: %s held %d particles", h, n)
			}
		}
		g.Update(1.0 / 30)
		stage := g.Render()
		if n := opaqueCount(stage); n < 12 {
			t.Fatalf("a fired gun must still paint the shotgun, got %d cells", n)
		}
	})
	t.Run("happy: Start loads components/gunfire/config.json so a prior UseBlast does not stick", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		weird := gunfire.DefaultBlast()
		shot := weird.ShotAt(sprite.E)
		shot.Count = 7
		weird.SetShot(sprite.E, shot)
		if err := gunfire.UseBlast(weird); err != nil {
			t.Fatalf("UseBlast: %v", err)
		}
		g := New()
		g.Start(stageW, stageH)
		shipped, err := gunfire.LoadBlast(gunfire.FindConfig())
		if err != nil {
			t.Fatalf("FindConfig must locate the shipped gunfire config: %v", err)
		}
		if !strings.HasSuffix(filepath.ToSlash(gunfire.FindConfig()), "components/gunfire/config.json") {
			t.Fatalf("FindConfig = %q, want …/components/gunfire/config.json", gunfire.FindConfig())
		}
		if gunfire.ActiveBlast().ShotAt(sprite.E).Count != shipped.ShotAt(sprite.E).Count {
			t.Fatalf("Start must arm the shipped gunfire config, E count %d want %d", gunfire.ActiveBlast().ShotAt(sprite.E).Count, shipped.ShotAt(sprite.E).Count)
		}
		if gunfire.ActiveBlast().ShotAt(sprite.E).Count == 7 {
			t.Fatal("Start must not leave a prior 7-count UseBlast in effect")
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
