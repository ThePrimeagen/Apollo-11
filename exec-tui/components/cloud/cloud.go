package cloud

import (
	"math"
	"math/rand"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sky"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// puff is one blob the generator plants inside a cloud: a pool
// origin relative to the cloud centre, and the seed that parks it.
type puff struct {
	origin particle.Vec2
	seed   int64
}

// layout is the unique puff arrangement for seed+cfg. The same pair
// always plants the same blobs; a different seed plants another.
func layout(seed int64, cfg Config) []puff {
	if cfg.Puffs <= 0 {
		return nil
	}
	rng := rand.New(rand.NewSource(seed))
	out := make([]puff, cfg.Puffs)
	for i := 0; i < cfg.Puffs; i++ {
		theta := rng.Float64() * 2 * math.Pi
		r := cfg.Spread * math.Sqrt(rng.Float64())
		s, c := math.Sincos(theta)
		out[i] = puff{
			origin: particle.Vec2{X: c * r, Y: s * r},
			seed:   seed + int64(i)*19 + 1,
		}
	}
	return out
}

// Cloud is one generated puff of parked pool particles. Start parks
// the specks for a w×h stage; Update keeps them put; Render paints
// concentration; Stop drops the engines.
type Cloud struct {
	Engines []*particle.Engine

	seed   int64
	cfg    Config
	puffs  []puff
	w, h   int
	staged bool
}

// Generate builds a unique Cloud from cfg and seed. Nothing is built
// until Start; the puff layout is fixed here so two Generates with
// the same pair always Start the same sky.
func Generate(seed int64, cfg Config) *Cloud {
	return &Cloud{seed: seed, cfg: cfg, puffs: layout(seed, cfg)}
}

// New is Generate on the active knobs.
func New(seed int64) *Cloud {
	return Generate(seed, Active())
}

func (c *Cloud) Start(w, h int) {
	if c == nil {
		return
	}
	c.w, c.h = w, h
	uw := float64(w)*particle.CellWidthUnits - 0.01
	uh := float64(h)*particle.CellHeightUnits - 0.01
	cx, cy := uw/2, uh/2
	c.Engines = c.Engines[:0]
	if c.puffs == nil {
		c.puffs = layout(c.seed, c.cfg)
	}
	for _, p := range c.puffs {
		origin := particle.Vec2{X: clamp(cx+p.origin.X, 0, uw), Y: clamp(cy+p.origin.Y, 0, uh)}
		cfg := c.cfg.Engine(uw, uh, origin)
		e := particle.New(p.seed, cfg)
		e.Burst()
		c.Engines = append(c.Engines, e)
	}
	c.staged = true
}

func (c *Cloud) Update(dt float64) {
	if c == nil || dt <= 0 {
		return
	}
	for _, e := range c.Engines {
		if e != nil {
			e.Update(dt)
		}
	}
}

func (c *Cloud) Render() sprite.Sprite {
	if c == nil || !c.staged || c.w < 1 || c.h < 1 {
		return sprite.Sprite{}
	}
	return paint(c.cfg, c.w, c.h, c.Engines...)
}

func (c *Cloud) Stop() {
	if c == nil {
		return
	}
	c.Engines = nil
	c.staged = false
}

// Field is a generated sky of clouds that rides the same rise pan
// the blue sky uses. At pan 0 they wait in the upper world, off a
// horizon shot; over Rise seconds they drift into view.
type Field struct {
	placed []placedCloud
	seed   int64
	cfg    Config
	rise   float64
	clock  float64
	w, h   int
	staged bool
}

type placedCloud struct {
	cloud *Cloud
	x, y  int
}

// NewField plants Active().Field unique clouds in the upper sky.
func NewField(seed int64) *Field {
	return &Field{seed: seed, cfg: Active()}
}

func (f *Field) Rise(seconds float64) *Field {
	if f == nil {
		return nil
	}
	if seconds > 0 {
		f.rise = seconds
	}
	return f
}

func (f *Field) Pan() float64 {
	if f == nil {
		return 0
	}
	if f.rise <= 0 {
		return 0
	}
	p := f.clock / f.rise
	if p > 1 {
		return 1
	}
	if p < 0 {
		return 0
	}
	return p
}

func (f *Field) Start(w, h int) {
	if f == nil {
		return
	}
	f.w, f.h = w, h
	rng := rand.New(rand.NewSource(f.seed))
	n := f.cfg.Field
	f.placed = make([]placedCloud, 0, n)
	for i := 0; i < n; i++ {
		cl := Generate(f.seed+int64(i)*31+3, f.cfg)
		cl.Start(w, h)
		cx := rng.Intn(max(w, 1))
		cy := rng.Intn(max(h, 1))
		f.placed = append(f.placed, placedCloud{cloud: cl, x: cx, y: cy})
	}
	f.staged = true
}

func (f *Field) Update(dt float64) {
	if f == nil || dt <= 0 {
		return
	}
	f.clock += dt
	for i := range f.placed {
		f.placed[i].cloud.Update(dt)
	}
}

func (f *Field) Render() sprite.Sprite {
	if f == nil || !f.staged || f.w < 1 || f.h < 1 {
		return sprite.Sprite{}
	}
	worldH := f.h * sky.WorldScale
	viewTop := (1 - f.Pan()) * float64(worldH-f.h)
	stage := sprite.New(f.w, f.h)
	for _, p := range f.placed {
		if p.cloud == nil {
			continue
		}
		body := p.cloud.Render()
		// The puff is generated around the sprite centre; land that
		// centre on the world cell, then shift into the viewport.
		dx := p.x - body.Width/2
		dy := p.y - body.Height/2 - int(viewTop+0.5)
		sprite.Blit(stage, dx, dy, body)
	}
	return stage
}

func (f *Field) Stop() {
	if f == nil {
		return
	}
	for i := range f.placed {
		f.placed[i].cloud.Stop()
	}
	f.placed = nil
	f.staged = false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
