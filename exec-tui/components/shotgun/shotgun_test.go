package shotgun

// Tests written FIRST: Shotgun is the Doom pump gun as a scene
// component, aimed on the same eight-point compass the gunfire blast
// already speaks. It is one 2D side-on asset — the east stock shot —
// spun in the plane of the screen around the Y-axis coming out of it
// (east 0°, then counterclockwise: NE 45°, N 90°, …). The spin
// happens in square-pixel space — a terminal cell is two stacked
// square pixels — so the gun is the same length on screen whichever
// way it points: upright is never twice as long as side-on. And the
// gun always faces up: the left half of the compass (W, NW, SW) is
// the right half (E, NE, SE) mirrored left-right, never a spin past
// vertical that would hang the sights toward the floor. Start builds
// every heading for a w×h stage; Aim points the barrel; Fire pulls
// the trigger so the muzzle flame leaps from this heading's barrel
// tip using the shipped gunfire config
// (components/gunfire/config.json) for that heading only — never the
// whole eight-way rose. The firing is tied to this gun: Start pins
// the shipped config onto the gun's own blast, and Fire (or FireFrom,
// for a gun a scene mounts anywhere) is one shot per call on that
// blast alone — the package-wide active blast belongs to the tuner
// and is never touched, and nothing ever fires on its own clock, so
// the caller controls exactly how often a shotgun fires. Update burns
// the blast; Render paints the aimed gun with whatever flame is still
// in the air. Before Start and after Stop the stage is empty; a
// heading off the compass is refused; the trigger needs a stage.

import (
	"math"
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

// cellPixels reads one half-block cell as its two stacked square
// pixels — the colors the terminal actually shows, top then bottom.
func cellPixels(c sprite.Cell) (top, bot int) {
	if c.Transparent() {
		return -1, -1
	}
	switch c.Ch {
	case '▀':
		return c.FG, c.BG
	case '▄':
		return c.BG, c.FG
	}
	return c.FG, c.FG
}

// visualLength is the gun's on-screen span in square-pixel units —
// the distance from the muzzle to the opaque cell farthest from it,
// with a cell one unit wide and two units tall. An empty sprite has
// no length.
func visualLength(sp sprite.Sprite, h sprite.Heading) float64 {
	mx, my := Muzzle(sp, h)
	best := 0.0
	found := false
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if sp.At(r, c).Transparent() {
				continue
			}
			found = true
			if d := math.Hypot(float64(c-mx), 2*float64(r-my)); d > best {
				best = d
			}
		}
	}
	if !found {
		return 0
	}
	return best
}

// goldColor is the brass palette entry the bead and the trigger wear.
const goldColor = 178

// framePixels expands a frame into its square-pixel colors, two rows
// per cell, -1 for empty sky.
func framePixels(sp sprite.Sprite) [][]int {
	px := make([][]int, sp.Height*2)
	for r := range px {
		px[r] = make([]int, sp.Width)
		for c := range px[r] {
			px[r][c] = -1
		}
	}
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			top, bot := cellPixels(sp.At(r, c))
			px[2*r][c] = top
			px[2*r+1][c] = bot
		}
	}
	return px
}

// goldCount is how many square pixels of brass a frame shows.
func goldCount(sp sprite.Sprite) int {
	n := 0
	for _, row := range framePixels(sp) {
		for _, col := range row {
			if col == goldColor {
				n++
			}
		}
	}
	return n
}

// touchesGold reports whether the gold pixel at (r,c) has a gold
// 4-neighbour — brass travels in masses, never crumbs.
func touchesGold(px [][]int, r, c int) bool {
	for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		rr, cc := r+d[0], c+d[1]
		if rr < 0 || rr >= len(px) || cc < 0 || cc >= len(px[rr]) {
			continue
		}
		if px[rr][cc] == goldColor {
			return true
		}
	}
	return false
}

// samePicture compares two frames the way the screen shows them: cell
// by cell as stacked square pixels, so a ▀ and a ▄ that paint the
// same two colors are equal even when the cell structs differ.
func samePicture(a, b sprite.Sprite) bool {
	if a.Width != b.Width || a.Height != b.Height {
		return false
	}
	for r := 0; r < a.Height; r++ {
		for c := 0; c < a.Width; c++ {
			at, ab := cellPixels(a.At(r, c))
			bt, bb := cellPixels(b.At(r, c))
			if at != bt || ab != bb {
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
		// Canvas-centre spin: the 3-pixel bar is left-of-centre, so
		// after +90° it stands on the left, one row down from the top.
		want := []string{
			"....",
			"B...",
			"B...",
			"B...",
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
	t.Run("happy: the gun always faces up — W, NW and SW mirror E, NE and SE left-right", func(t *testing.T) {
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
		mirrors := []struct{ left, right sprite.Heading }{
			{sprite.W, sprite.E},
			{sprite.NW, sprite.NE},
			{sprite.SW, sprite.SE},
		}
		for _, m := range mirrors {
			if !samePicture(g.Frame(m.left), sprite.FlipH(g.Frame(m.right))) {
				t.Fatalf("%s must be FlipH(%s) so the sights stay up on the left half of the compass", m.left, m.right)
			}
		}
		if !samePicture(g.Frame(sprite.S), sprite.FlipH(sprite.FlipV(g.Frame(sprite.N)))) {
			t.Fatal("S must be N spun 180° around the axis coming out of the screen")
		}
	})
	t.Run("unhappy: W is never the 180° spin — that would hang the gun upside down", func(t *testing.T) {
		g := New()
		g.Start(stageW, stageH)
		east := g.Frame(sprite.E)
		if samePicture(g.Frame(sprite.W), sprite.FlipH(sprite.FlipV(east))) {
			t.Fatal("W must not be E spun 180° (FlipH+FlipV) — the sights would face the floor")
		}
	})
	t.Run("happy: on screen the upright gun matches the side-on gun — a cell is two square pixels tall", func(t *testing.T) {
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
		// Visual units: a cell is one unit wide and two units tall.
		eLen, eThick := ew, 2*eh
		for _, h := range []sprite.Heading{sprite.N, sprite.S} {
			minC, minR, maxC, maxR, n = bounds(g.Frame(h))
			if n == 0 {
				t.Fatalf("%s gun is empty", h)
			}
			w, cells := maxC-minC+1, maxR-minR+1
			if cells <= w {
				t.Fatalf("%s shotgun should be taller than wide, got %dx%d", h, w, cells)
			}
			vLen, vThick := 2*cells, w
			if diff := vLen - eLen; diff < -3 || diff > 3 {
				t.Fatalf("%s gun runs %d units on screen, east runs %d — an upright gun must not stretch", h, vLen, eLen)
			}
			if diff := vThick - eThick; diff < -3 || diff > 3 {
				t.Fatalf("%s gun is %d units thick on screen, east is %d", h, vThick, eThick)
			}
		}
	})
	t.Run("happy: every heading keeps the gun's on-screen length, muzzle to stock butt", func(t *testing.T) {
		g := New()
		g.Start(stageW, stageH)
		east := visualLength(g.Frame(sprite.E), sprite.E)
		if east <= 0 {
			t.Fatal("east gun is empty")
		}
		for _, h := range sprite.Headings {
			got := visualLength(g.Frame(h), h)
			if got <= 0 {
				t.Fatalf("%s gun is empty", h)
			}
			if diff := got - east; diff < -4 || diff > 4 {
				t.Fatalf("%s gun runs %.1f units muzzle to stock on screen, east runs %.1f — the gun must not stretch when it turns", h, got, east)
			}
		}
	})
	t.Run("happy: every heading carries the cardinal gold mass — the bead and trigger never flicker", func(t *testing.T) {
		g := New()
		g.Start(stageW, stageH)
		want := goldCount(g.Frame(sprite.E))
		if want < 4 {
			t.Fatalf("east gun carries %d gold pixels — the bead and the trigger must both be drawn", want)
		}
		for _, h := range sprite.Headings {
			if got := goldCount(g.Frame(h)); got != want {
				t.Fatalf("%s carries %d gold pixels, east carries %d — a rotating gun must not flicker its brass", h, got, want)
			}
		}
	})
	t.Run("unhappy: no heading strands an orphan gold speck", func(t *testing.T) {
		g := New()
		g.Start(stageW, stageH)
		for _, h := range sprite.Headings {
			px := framePixels(g.Frame(h))
			for r := range px {
				for c := range px[r] {
					if px[r][c] != goldColor {
						continue
					}
					if !touchesGold(px, r, c) {
						t.Fatalf("%s has a lone gold speck at pixel (%d,%d) — brass travels in masses, never crumbs", h, r, c)
					}
				}
			}
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
	t.Run("happy: Start arms this gun with the shipped config and leaves the tuner's active blast alone", func(t *testing.T) {
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
		if got := g.Blast.Config().ShotAt(sprite.E).Count; got != shipped.ShotAt(sprite.E).Count {
			t.Fatalf("this gun's blast counts %d on E, want the shipped %d — a prior UseBlast must not stick to the gun", got, shipped.ShotAt(sprite.E).Count)
		}
		if got := gunfire.ActiveBlast().ShotAt(sprite.E).Count; got != 7 {
			t.Fatalf("the package active blast counts %d on E after Start, want the tuner's 7 — arming a gun must not retune the world", got)
		}
	})
	t.Run("happy: Fire is one shot from this gun alone — the package active blast is never touched", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		before := gunfire.ActiveBlast()
		g := New()
		g.Start(stageW, stageH)
		_ = g.Aim(sprite.NE)
		if !g.Fire() {
			t.Fatal("Fire after Start must pull the trigger")
		}
		if gunfire.ActiveBlast() != before {
			t.Fatal("Fire must burn this gun's own blast, not retune the package active")
		}
		body := g.Frame(sprite.NE)
		mx, my := Muzzle(body, sprite.NE)
		left := (stageW - body.Width) / 2
		top := (stageH - body.Height) / 2
		c := g.Blast.Config()
		if c.Heading != sprite.NE {
			t.Fatalf("the gun's own blast aims %q, want NE", c.Heading)
		}
		wantX := (float64(left+mx) + 0.5) / float64(stageW)
		wantY := (float64(top+my) + 0.5) / float64(stageH)
		if math.Abs(c.MuzzleX-wantX) > 1e-9 || math.Abs(c.MuzzleY-wantY) > 1e-9 {
			t.Fatalf("the gun's own muzzle is (%v,%v), want the barrel tip (%v,%v)", c.MuzzleX, c.MuzzleY, wantX, wantY)
		}
	})
	t.Run("happy: one Fire is one shot — it burns out and stays out until the next trigger", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		g := New()
		g.Start(stageW, stageH)
		if !g.Fire() {
			t.Fatal("Fire after Start must pull the trigger")
		}
		for i := 0; i < 150; i++ { // five seconds: past every life and the pulse fuse
			g.Update(1.0 / 30)
		}
		if n := liveBlast(g); n != 0 {
			t.Fatalf("one shot must burn out, %d particles still in the air", n)
		}
		for i := 0; i < 30; i++ {
			g.Update(1.0 / 30)
		}
		if n := liveBlast(g); n != 0 {
			t.Fatalf("a spent gun re-fired on its own: %d particles — cadence belongs to the caller", n)
		}
		if !g.Fire() {
			t.Fatal("the next trigger must fire the next shot")
		}
		if liveBlast(g) == 0 {
			t.Fatal("the next shot must put flame back in the air")
		}
	})
	t.Run("unhappy: Update alone never pulls the trigger", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		g := New()
		g.Start(stageW, stageH)
		for i := 0; i < 90; i++ {
			g.Update(1.0 / 30)
		}
		if n := liveBlast(g); n != 0 {
			t.Fatalf("three untriggered seconds spawned %d particles", n)
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

// TestFireFrom: the trigger for a mounted gun. A scene blits the gun
// anywhere; FireFrom throws one shot from that mount's barrel tip on
// this gun's own blast. One call is one shot — cadence belongs to the
// caller — and no trigger ever touches the package active blast, so
// two mounted guns never clobber each other.
func TestFireFrom(t *testing.T) {
	t.Run("happy: FireFrom fires one shot from the mounted barrel tip, and two guns never clobber each other", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		shipped, err := gunfire.LoadBlast(gunfire.FindConfig())
		if err != nil {
			t.Fatalf("the shotgun must use the shipped gunfire config: %v", err)
		}
		a := New()
		a.Start(stageW, stageH)
		_ = a.Aim(sprite.NE)
		b := New()
		b.Start(stageW, stageH)
		_ = b.Aim(sprite.SW)
		if !a.FireFrom(2, 3) {
			t.Fatal("FireFrom must pull the trigger on a started gun")
		}
		if !b.FireFrom(30, 9) {
			t.Fatal("FireFrom must pull the trigger on the second gun")
		}
		if got := len(a.Blast.FlameAt(sprite.NE).Particles); got != shipped.ShotAt(sprite.NE).Count {
			t.Fatalf("gun A burst %d on NE, want the shipped %d", got, shipped.ShotAt(sprite.NE).Count)
		}
		if got := len(b.Blast.FlameAt(sprite.SW).Particles); got != shipped.ShotAt(sprite.SW).Count {
			t.Fatalf("gun B burst %d on SW, want the shipped %d", got, shipped.ShotAt(sprite.SW).Count)
		}
		if n := len(a.Blast.FlameAt(sprite.SW).Particles); n != 0 {
			t.Fatalf("gun B's trigger leaked %d particles into gun A", n)
		}
		ca, cb := a.Blast.Config(), b.Blast.Config()
		if ca.Heading != sprite.NE || cb.Heading != sprite.SW {
			t.Fatalf("each gun keeps its own aim, got A %q B %q", ca.Heading, cb.Heading)
		}
		bodyA := a.Frame(sprite.NE)
		mxA, myA := Muzzle(bodyA, sprite.NE)
		wantX := (float64(2+mxA) + 0.5) / float64(stageW)
		wantY := (float64(3+myA) + 0.5) / float64(stageH)
		if math.Abs(ca.MuzzleX-wantX) > 1e-9 || math.Abs(ca.MuzzleY-wantY) > 1e-9 {
			t.Fatalf("gun A's muzzle is (%v,%v), want its own mount's (%v,%v) — gun B must not drag it around", ca.MuzzleX, ca.MuzzleY, wantX, wantY)
		}
	})
	t.Run("unhappy: a mount off the stage still fires from the edge, and an unstarted or nil gun is refused", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		g := New()
		if g.FireFrom(0, 0) {
			t.Fatal("FireFrom before Start must be refused")
		}
		g.Start(stageW, stageH)
		if !g.FireFrom(-200, -200) {
			t.Fatal("a barrel poking off the stage still fires — from the edge")
		}
		c := g.Blast.Config()
		if c.MuzzleX != 0 || c.MuzzleY != 0 {
			t.Fatalf("a far off-stage mount fires from the stage edge (0,0), got (%v,%v)", c.MuzzleX, c.MuzzleY)
		}
		var ng *Gun
		if ng.FireFrom(1, 1) {
			t.Fatal("a nil gun must refuse the trigger")
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

// TestMuzzle: the exported barrel-tip finder a scene uses to mount
// the gun anywhere — the muzzle flame must leap from the barrel, not
// the stock, wherever the gun is blitted.
func TestMuzzle(t *testing.T) {
	t.Run("happy: the muzzle is the opaque cell furthest along the heading", func(t *testing.T) {
		a, err := BuildAtlas()
		if err != nil {
			t.Fatalf("BuildAtlas: %v", err)
		}
		east, ok := a.Frame(Size, sprite.E)
		if !ok {
			t.Fatal("the atlas must hold the east gun")
		}
		minC, _, maxC, _, n := bounds(east)
		if n == 0 {
			t.Fatal("test premise: the east gun has ink")
		}
		if x, y := Muzzle(east, sprite.E); x != maxC || east.At(y, x).Transparent() {
			t.Fatalf("the east muzzle is (%d,%d), want the rightmost opaque column %d", x, y, maxC)
		}
		if x, y := Muzzle(east, sprite.W); x != minC || east.At(y, x).Transparent() {
			t.Fatalf("aiming the east frame west, the muzzle is (%d,%d), want the leftmost opaque column %d", x, y, minC)
		}
		north, ok := a.Frame(Size, sprite.N)
		if !ok {
			t.Fatal("the atlas must hold the north gun")
		}
		_, nMinR, _, _, _ := bounds(north)
		if x, y := Muzzle(north, sprite.N); y != nMinR || north.At(y, x).Transparent() {
			t.Fatalf("the north muzzle is (%d,%d), want the topmost opaque row %d", x, y, nMinR)
		}
		for _, h := range sprite.Headings {
			body, ok := a.Frame(Size, h)
			if !ok {
				t.Fatalf("the atlas must hold the %s gun", h)
			}
			x, y := Muzzle(body, h)
			if x < 0 || x >= body.Width || y < 0 || y >= body.Height {
				t.Fatalf("%s muzzle (%d,%d) is off the %dx%d frame", h, x, y, body.Width, body.Height)
			}
			if body.At(y, x).Transparent() {
				t.Fatalf("%s muzzle (%d,%d) must sit on the gun", h, x, y)
			}
		}
	})
	t.Run("unhappy: an empty sprite or an off-compass heading never panics", func(t *testing.T) {
		x, y := Muzzle(sprite.Sprite{}, sprite.E)
		if x != 0 || y != 0 {
			t.Fatalf("an empty sprite's muzzle is (%d,%d), want its (0,0) centre", x, y)
		}
		a, err := BuildAtlas()
		if err != nil {
			t.Fatalf("BuildAtlas: %v", err)
		}
		east, _ := a.Frame(Size, sprite.E)
		bx, by := Muzzle(east, sprite.Heading("XX"))
		if bx < 0 || bx >= east.Width || by < 0 || by >= east.Height {
			t.Fatalf("an off-compass muzzle (%d,%d) must stay on the frame", bx, by)
		}
	})
}
