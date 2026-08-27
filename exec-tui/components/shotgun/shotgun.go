// Package shotgun is the Doom pump gun as a scene component: a
// stock-and-barrel sprite on the eight-point compass the gunfire blast
// already speaks. Start builds every heading for a w×h stage; Aim
// points the barrel; Fire pulls the trigger so the muzzle flame leaps
// from this heading's barrel tip; Update burns the blast; Render
// paints the aimed gun with whatever flame is still in the air. W is
// the horizontal mirror of E, S the vertical mirror of N. A fresh
// Start rises idle, aimed east — the Doom side-on stock shot.
package shotgun

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/gunfire"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// Gun is the component. Start builds the eight frames and arms a
// quiet blast; Stop drops them so a stopped gun holds nothing.
type Gun struct {
	Blast *gunfire.Blast

	heading sprite.Heading
	frames  map[sprite.Heading]sprite.Sprite
	seed    int64
	w, h    int
}

// New binds a gun aimed east. Nothing is built until Start.
func New() *Gun {
	return &Gun{heading: sprite.E, seed: 11}
}

// Heading is the compass point the barrel faces.
func (g *Gun) Heading() sprite.Heading {
	if g == nil {
		return ""
	}
	return g.heading
}

// Frame is the sprite for one compass point. Missing headings and
// unstarted guns hand back an empty sprite.
func (g *Gun) Frame(h sprite.Heading) sprite.Sprite {
	if g == nil || g.frames == nil {
		return sprite.Sprite{}
	}
	return g.frames[h]
}

// Aim points the barrel at h. Headings off the compass are refused
// and the gun keeps its last aim.
func (g *Gun) Aim(h sprite.Heading) bool {
	if g == nil {
		return false
	}
	for _, hh := range sprite.Headings {
		if h == hh {
			g.heading = h
			return true
		}
	}
	return false
}

// Step walks the compass delta steps — clockwise for positive, wrapping
// at the ends — and returns the heading it landed on.
func (g *Gun) Step(delta int) sprite.Heading {
	if g == nil {
		return ""
	}
	idx := 0
	for i, h := range sprite.Headings {
		if h == g.heading {
			idx = i
			break
		}
	}
	n := len(sprite.Headings)
	g.heading = sprite.Headings[((idx+delta)%n+n)%n]
	return g.heading
}

// Start builds every heading and a quiet blast for a w×h stage.
func (g *Gun) Start(w, h int) {
	if g == nil {
		return
	}
	g.w, g.h = w, h
	a, err := BuildAtlas()
	if err != nil {
		g.frames = nil
		return
	}
	g.frames = map[sprite.Heading]sprite.Sprite{}
	for _, heading := range sprite.Headings {
		if sp, ok := a.Frame(Size, heading); ok {
			g.frames[heading] = sp
		}
	}
	g.Blast = gunfire.NewBlast(g.seed)
	g.Blast.Start(w, h)
}

// Update burns the muzzle flame dt seconds. dt <= 0 holds.
func (g *Gun) Update(dt float64) {
	if g == nil || g.Blast == nil || dt <= 0 {
		return
	}
	g.Blast.Update(dt)
}

// Fire pulls the trigger: the active blast is aimed along this gun's
// heading, the muzzle sits on the barrel tip, and the flame bursts.
// The trigger needs a stage — before Start it is refused.
func (g *Gun) Fire() bool {
	if g == nil || g.Blast == nil || g.frames == nil {
		return false
	}
	body := g.frames[g.heading]
	if body.Width < 1 || body.Height < 1 {
		return false
	}
	c := gunfire.ActiveBlast()
	c.Heading = g.heading
	mx, my := muzzleOf(body, g.heading)
	left := (g.w - body.Width) / 2
	top := (g.h - body.Height) / 2
	if g.w > 0 {
		c.MuzzleX = (float64(left+mx) + 0.5) / float64(g.w)
	}
	if g.h > 0 {
		c.MuzzleY = (float64(top+my) + 0.5) / float64(g.h)
	}
	if err := gunfire.UseBlast(c); err != nil {
		return false
	}
	return g.Blast.Fire()
}

// muzzleOf is the opaque cell furthest along heading h — the barrel tip.
func muzzleOf(sp sprite.Sprite, h sprite.Heading) (x, y int) {
	dx, dy := 0, 0
	switch h {
	case sprite.N:
		dy = -1
	case sprite.NE:
		dx, dy = 1, -1
	case sprite.E:
		dx = 1
	case sprite.SE:
		dx, dy = 1, 1
	case sprite.S:
		dy = 1
	case sprite.SW:
		dx, dy = -1, 1
	case sprite.W:
		dx = -1
	case sprite.NW:
		dx, dy = -1, -1
	}
	best := -1 << 30
	x, y = sp.Width/2, sp.Height/2
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if sp.At(r, c).Transparent() {
				continue
			}
			score := c*dx + r*dy
			if score > best {
				best = score
				x, y = c, r
			}
		}
	}
	return
}

// Render paints the aimed gun, then the muzzle flame on top, onto a
// stage-sized sprite. Before Start and after Stop the stage is empty.
func (g *Gun) Render() sprite.Sprite {
	if g == nil || g.frames == nil || g.w < 1 || g.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(g.w, g.h)
	body := g.frames[g.heading]
	left := (g.w - body.Width) / 2
	top := (g.h - body.Height) / 2
	sprite.Blit(stage, left, top, body)
	if g.Blast != nil {
		sprite.Blit(stage, 0, 0, g.Blast.Render())
	}
	return stage
}

// Stop drops the frames and the blast; a fresh Start rebuilds them idle.
func (g *Gun) Stop() {
	if g == nil {
		return
	}
	if g.Blast != nil {
		g.Blast.Stop()
		g.Blast = nil
	}
	g.frames = nil
}
