package shootingstar

// Tests written FIRST: the shooting-star scene is a larger star plus a
// persist-particle trail, flying under a still sky. New always falls
// right-to-left, high on the right to low on the left. NewPreview
// follows the config path — fall (the scene), or circle/square so the
// tuner can still read the tail. The trail emits at the star; as the
// star moves the specks stay put and fade.

import (
	"math"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/bigstar"
	"github.com/theprimeagen/apollo-11/exec-tui/components/startrail"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 72
	stageH = 27
)

func paint(sc screenplay.Scene) *screenplay.Screen {
	scr := screenplay.NewScreen(stageW, stageH)
	sc.Render(scr)
	return scr
}

func tick(sc screenplay.Scene, seconds float64) {
	const dt = 1.0 / 30
	for t := 0.0; t < seconds-dt/2; t += dt {
		sc.Update(dt)
	}
}

func coreCell(scr *screenplay.Screen) (x, y int, ok bool) {
	for y = 0; y < stageH; y++ {
		for x = 0; x < stageW; x++ {
			c := scr.Cell(x, y)
			if c != nil && c.Content == string(bigstar.CoreGlyph) {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

func sparkCells(scr *screenplay.Screen) int {
	n := 0
	for y := 0; y < stageH; y++ {
		for x := 0; x < stageW; x++ {
			c := scr.Cell(x, y)
			if c == nil || c.Content == "" || c.Content == " " {
				continue
			}
			n++
		}
	}
	return n
}

func TestShootingStarBill(t *testing.T) {
	t.Run("happy: the bill is the one shooting-star scene", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the shooting-star bill holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "shootingstar" {
			t.Fatalf("the scene is %q, want shootingstar", b[0].Name)
		}
		if b[0].Scene == nil {
			t.Fatal("the shooting star has no performer")
		}
	})
	t.Run("unhappy: a second scene is not hiding on the bill", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		defer p.Stop()
		if p.Len() != 1 || p.CurrentName() != "shootingstar" {
			t.Fatalf("the show opens on %d %q, want one shootingstar", p.Len(), p.CurrentName())
		}
		if p.Next() {
			t.Fatal("after the shooting star there is nothing left")
		}
	})
}

func TestShootingStarScene(t *testing.T) {
	t.Cleanup(Reset)
	t.Cleanup(startrail.Reset)
	t.Run("happy: New opens on a right-to-left fall and the star is already on stage", func(t *testing.T) {
		sc := New(nil)
		sc.Seed = 11
		sc.Start()
		defer sc.Stop()
		opening := paint(sc)
		x, _, ok := coreCell(opening)
		if !ok {
			t.Fatal("the shooting star must open with the larger star already on (or entering) stage")
		}
		if x < stageW/2 {
			t.Fatalf("the fall must enter from the right, star at col %d", x)
		}
		if sc.cross.Start.X <= sc.cross.End.X {
			t.Fatalf("the scene must run right-to-left, %+v → %+v", sc.cross.Start, sc.cross.End)
		}
		if sc.cross.Start.Y >= sc.cross.End.Y {
			t.Fatalf("the scene must fall downward, %+v → %+v", sc.cross.Start, sc.cross.End)
		}
	})
	t.Run("happy: the meteor moves left and leaves a persist trail behind it", func(t *testing.T) {
		sc := New(nil)
		sc.Seed = 7
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.2)
		a := paint(sc)
		x0, y0, ok := coreCell(a)
		if !ok {
			t.Fatal("need the star on stage to watch it fly")
		}
		tick(sc, 0.5)
		b := paint(sc)
		x1, y1, ok := coreCell(b)
		if !ok {
			t.Fatal("the star must still be on stage half a second later")
		}
		if x0 == x1 && y0 == y1 {
			t.Fatal("the shooting star must move")
		}
		if x1 >= x0 {
			t.Fatalf("the scene must travel right-to-left, col %d → %d", x0, x1)
		}
		if sparkCells(b) <= 1 {
			t.Fatal("the moving star must leave a trail, not just its core")
		}
	})
	t.Run("happy: NewPreview on fall uses the same right-to-left meteor as the scene", func(t *testing.T) {
		sc := NewPreview(nil)
		sc.Cfg.Path = PathFall
		sc.Seed = 9
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		if sc.closedLoop() {
			t.Fatal("fall is the scene path, not a closed loop")
		}
		if sc.cross.Start.X <= sc.cross.End.X || sc.cross.Start.Y >= sc.cross.End.Y {
			t.Fatalf("preview fall must run top-right to bottom-left, %+v → %+v", sc.cross.Start, sc.cross.End)
		}
		x0, _, ok := coreCell(paint(sc))
		if !ok {
			t.Fatal("the fall preview must put the star on stage")
		}
		tick(sc, 0.5)
		x1, _, ok := coreCell(paint(sc))
		if !ok {
			t.Fatal("the fall preview must keep the star on stage")
		}
		if x1 >= x0 {
			t.Fatalf("the fall preview must travel right-to-left, col %d → %d", x0, x1)
		}
	})
	t.Run("happy: NewPreview on a circle keeps a near-constant radius from center", func(t *testing.T) {
		sc := NewPreview(nil)
		sc.Cfg.Path = PathCircle
		sc.Cfg.Speed = 20
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.2)
		x0, y0, ok := coreCell(paint(sc))
		if !ok {
			t.Fatal("the circle preview must put the star on stage")
		}
		tick(sc, 0.8)
		x1, y1, ok := coreCell(paint(sc))
		if !ok {
			t.Fatal("the circle preview must keep the star on stage")
		}
		if x0 == x1 && y0 == y1 {
			t.Fatal("the circle preview must move")
		}
		cx, cy := float64(stageW)/2, float64(stageH)/2
		r0 := math.Hypot(float64(x0)-cx, (float64(y0)-cy)*2)
		r1 := math.Hypot(float64(x1)-cx, (float64(y1)-cy)*2)
		if math.Abs(r0-r1) > 3 {
			t.Fatalf("circle radius drifted %.1f -> %.1f — the preview must hold the loop", r0, r1)
		}
	})
	t.Run("happy: NewPreview on a square stays near the inset rectangle", func(t *testing.T) {
		sc := NewPreview(nil)
		sc.Cfg.Path = PathSquare
		sc.Cfg.Speed = 24
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.15)
		x, y, ok := coreCell(paint(sc))
		if !ok {
			t.Fatal("the square preview must put the star on stage")
		}
		// inset ~8 cells; allow a generous band so the star's span
		// and the unit-space mapping do not fail a tight rail
		if x < 2 || x > stageW-3 || y < 1 || y > stageH-2 {
			t.Fatalf("square preview parked at (%d,%d) — that is off the inset rectangle", x, y)
		}
		tick(sc, 0.6)
		x2, y2, ok := coreCell(paint(sc))
		if !ok {
			t.Fatal("the square preview must keep the star on stage")
		}
		if x == x2 && y == y2 {
			t.Fatal("the square preview must move")
		}
	})
	t.Run("happy: Use is what NewPreview plays on the first Start", func(t *testing.T) {
		t.Cleanup(Reset)
		c := DefaultConfig()
		c.Path = PathSquare
		c.Size = 3
		if err := Use(c); err != nil {
			t.Fatal(err)
		}
		sc := NewPreview(nil)
		sc.Start()
		defer sc.Stop()
		if sc.Cfg.Path != PathSquare || sc.Cfg.Size != 3 {
			t.Fatalf("the first play must ride the active knobs, got %+v", sc.Cfg)
		}
		if _, _, ok := coreCell(paint(sc)); !ok {
			t.Fatal("the preview must still paint the star")
		}
	})
	t.Run("happy: Start after Stop rebuilds from the current knobs", func(t *testing.T) {
		sc := NewPreview(nil)
		sc.Cfg.Path = PathCircle
		sc.Cfg.Size = 1
		sc.Start()
		_ = paint(sc)
		sc.Cfg.Size = 4
		sc.Stop()
		sc.Start()
		_ = paint(sc)
		if sc.flyer == nil || sc.flyer.star == nil {
			t.Fatal("play must rebuild the star")
		}
		if sc.flyer.star.Size != 4 {
			t.Fatalf("play must rebuild from the current size, got %d want 4", sc.flyer.star.Size)
		}
		sc.Stop()
	})
	t.Run("unhappy: New's fall ignores a circle Path and never runs left-to-right", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.Path = PathCircle
		sc.Seed = 3
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		if sc.preview {
			t.Fatal("New is the scene, not the tuner preview")
		}
		if sc.cross.Start.X <= sc.cross.End.X {
			t.Fatalf("a circle Path must not turn the scene around: %+v → %+v", sc.cross.Start, sc.cross.End)
		}
		if sc.cross.Start.Y >= sc.cross.End.Y {
			t.Fatalf("the scene must still fall: %+v → %+v", sc.cross.Start, sc.cross.End)
		}
	})
	t.Run("happy: a size past 5 still paints, and the stored size is not rewritten", func(t *testing.T) {
		sc := NewPreview(nil)
		sc.Cfg.Size = 6
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		if sc.flyer == nil || sc.flyer.star == nil {
			t.Fatal("play must build the star")
		}
		if sc.flyer.star.Size != 6 {
			t.Fatalf("size 6 was rewritten to %d", sc.flyer.star.Size)
		}
		if _, _, ok := coreCell(paint(sc)); !ok {
			t.Fatal("a size-6 star must still paint its core")
		}
	})
	t.Run("unhappy: speed is used as given — below 1 is not rewritten to 1, and a negative speed is kept", func(t *testing.T) {
		slow := New(nil)
		slow.Seed = 11
		slow.Cfg.Speed = 0.5
		slow.Start()
		defer slow.Stop()
		fast := New(nil)
		fast.Seed = 11
		fast.Cfg.Speed = 4
		fast.Start()
		defer fast.Stop()
		_ = paint(slow)
		_ = paint(fast)
		tick(slow, 0.4)
		tick(fast, 0.4)
		if slow.Cfg.Speed != 0.5 {
			t.Fatalf("slow speed was rewritten to %v", slow.Cfg.Speed)
		}
		sx, _, sok := coreCell(paint(slow))
		fx, _, fok := coreCell(paint(fast))
		if !sok || !fok {
			t.Fatal("both flights must still paint the star")
		}
		if fx >= sx {
			t.Fatalf("speed 4 must travel farther left than 0.5, cols %d vs %d — a floor of 1 would make them match", fx, sx)
		}
		rev := NewPreview(nil)
		rev.Cfg.Path = PathCircle
		rev.Cfg.Speed = -20
		rev.Start()
		defer rev.Stop()
		_ = paint(rev)
		if rev.Cfg.Speed != -20 {
			t.Fatalf("negative speed was rewritten to %v", rev.Cfg.Speed)
		}
		tick(rev, 0.3)
		if rev.Cfg.Speed != -20 {
			t.Fatalf("the circle must keep speed -20, got %v", rev.Cfg.Speed)
		}
		if _, _, ok := coreCell(paint(rev)); !ok {
			t.Fatal("a negative-speed circle must still paint the star")
		}
	})
	t.Run("unhappy: a scene stopped before its first render never panics, and dt<=0 holds", func(t *testing.T) {
		sc := New(nil)
		sc.Start()
		sc.Update(1)
		sc.Stop()
		sc.Update(1)
		sc.Render(nil)
		sc.Stop()

		held := NewPreview(nil)
		held.Start()
		defer held.Stop()
		_ = paint(held)
		x0, y0, _ := coreCell(paint(held))
		held.Update(0)
		held.Update(-3)
		x1, y1, _ := coreCell(paint(held))
		if x0 != x1 || y0 != y1 {
			t.Fatal("dt<=0 must hold the star")
		}
	})
}
