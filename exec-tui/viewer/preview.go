package viewer

import (
	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/astro"
	"github.com/theprimeagen/apollo-11/exec-tui/components/cloud"
	"github.com/theprimeagen/apollo-11/exec-tui/components/fire"
	"github.com/theprimeagen/apollo-11/exec-tui/components/flag"
	"github.com/theprimeagen/apollo-11/exec-tui/components/gunfire"
	"github.com/theprimeagen/apollo-11/exec-tui/components/pools"
	"github.com/theprimeagen/apollo-11/exec-tui/components/rocket"
	"github.com/theprimeagen/apollo-11/exec-tui/components/shotgun"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sky"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/components/transition"
	"github.com/theprimeagen/apollo-11/exec-tui/menu"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/moonwalk"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// emptyPreview is a silent stand-in when a performer cannot be built.
type emptyPreview struct{}

func (emptyPreview) Start(int, int)        {}
func (emptyPreview) Update(float64)        {}
func (emptyPreview) Render() sprite.Sprite { return sprite.Sprite{} }
func (emptyPreview) Stop()                 {}

type cyclingGun struct {
	*shotgun.Gun
	clock float64
}

func newCyclingGun() *cyclingGun {
	return &cyclingGun{Gun: shotgun.New()}
}

func (g *cyclingGun) Start(w, h int) {
	g.Gun.Start(w, h)
	g.Fire()
}

func (g *cyclingGun) Update(dt float64) {
	g.Gun.Update(dt)
	if dt <= 0 {
		return
	}
	g.clock += dt
	if g.clock >= 1.2 {
		g.clock -= 1.2
		g.Step(1)
		g.Fire()
	}
}

type burstingBlast struct {
	*gunfire.Blast
	clock float64
}

func newBurstingBlast() *burstingBlast {
	return &burstingBlast{Blast: gunfire.NewBlast(11)}
}

func (b *burstingBlast) Start(w, h int) {
	b.Blast.Start(w, h)
	b.Fire()
}

func (b *burstingBlast) Update(dt float64) {
	b.Blast.Update(dt)
	if dt <= 0 {
		return
	}
	b.clock += dt
	if b.clock >= 1.6 {
		b.clock -= 1.6
		b.Fire()
	}
}

// poolStepSeconds is the beat between steps of the pool demo walk.
const poolStepSeconds = 0.55

// poolAlarmHoldBeats keeps the full pool on stage — chip up, count
// red — long enough to read before the drain starts.
const poolAlarmHoldBeats = 4

// poolHint names the play button while a pool demo sits idle.
const poolHint = "space plays: add 3 · drop 3 · add 4 · drop 4 · fill to the alarm · drain"

// poolStep is one beat of the walk: add the job, drop it by name, or
// wait — hold whatever is on stage for the beat.
type poolStep struct {
	add  bool
	wait bool
	job  pools.Job
}

// poolScript is the whole lifecycle the play button walks: add three
// jobs, drop those three, add four, drop those four, then fill every
// slot to raise the pool's alarm, hold it up for poolAlarmHoldBeats,
// and drain it back to empty. Each job wears the ink its lanes wear
// on the legacy sim and the graphs.
func poolScript(capacity int) []poolStep {
	trio := []pools.Job{
		{Name: "SERVICER", Prio: 20, Ink: 83},
		{Name: "CHARIN", Prio: 30, Ink: 213},
		{Name: "MONITOR", Prio: 26, Ink: 220},
	}
	quad := []pools.Job{
		{Name: "RR READ", Prio: 32, Ink: 87},
		{Name: "LR READ", Prio: 32, Ink: 75},
		{Name: "GYRO COMP", Prio: 22, Ink: 255},
		{Name: "DAP", Prio: 31, Ink: 214},
	}
	roster := append(append([]pools.Job{}, trio...), quad...)
	roster = append(roster, pools.Job{Name: "SELFCHK", Prio: 1, Ink: 245})
	if capacity > len(roster) {
		capacity = len(roster)
	}
	var script []poolStep
	for _, j := range trio {
		script = append(script, poolStep{add: true, job: j})
	}
	for _, j := range trio {
		script = append(script, poolStep{job: j})
	}
	for _, j := range quad {
		script = append(script, poolStep{add: true, job: j})
	}
	for _, j := range quad {
		script = append(script, poolStep{job: j})
	}
	for _, j := range roster[:capacity] {
		script = append(script, poolStep{add: true, job: j})
	}
	for i := 0; i < poolAlarmHoldBeats; i++ {
		script = append(script, poolStep{wait: true})
	}
	for _, j := range roster[:capacity] {
		script = append(script, poolStep{job: j})
	}
	return script
}

// poolDemo wraps a pool view for the viewer: space (the viewer's
// trigger key) plays the scripted lifecycle one step per
// poolStepSeconds, and the bottom row hints the button while idle and
// the current action while playing. Firing mid-walk is refused.
type poolDemo struct {
	view    *pools.View
	script  []poolStep
	playing bool
	step    int
	clock   float64
	last    string
	lastInk int
}

func newPoolDemo(v *pools.View) *poolDemo {
	return &poolDemo{view: v, script: poolScript(v.Cap())}
}

func (d *poolDemo) Start(w, h int) { d.view.Start(w, h) }
func (d *poolDemo) Stop()          { d.view.Stop() }

// Fire starts the walk from the top; a walk already playing keeps its
// place and refuses the trigger.
func (d *poolDemo) Fire() bool {
	if d.playing || len(d.script) == 0 {
		return false
	}
	d.playing = true
	d.step = 0
	d.clock = 0
	d.run()
	return true
}

// run performs the next script step and retires the walk on the last,
// so the trigger goes live again.
func (d *poolDemo) run() {
	if d.step >= len(d.script) {
		d.playing = false
		return
	}
	st := d.script[d.step]
	d.step++
	switch {
	case st.wait:
		// the held full pool is the message — the chip speaks
	case st.add:
		d.view.Add(st.job)
		d.last = "+ " + st.job.Name
		d.lastInk = st.job.Ink
	default:
		d.view.Remove(st.job.Name)
		d.last = "- " + st.job.Name
		d.lastInk = pools.DimInk
	}
	if d.step >= len(d.script) {
		d.playing = false
	}
}

func (d *poolDemo) Update(dt float64) {
	d.view.Update(dt)
	if !d.playing || dt <= 0 {
		return
	}
	d.clock += dt
	for d.clock >= poolStepSeconds && d.playing {
		d.clock -= poolStepSeconds
		d.run()
	}
}

func (d *poolDemo) Render() sprite.Sprite {
	stage := d.view.Render()
	if stage.Width < 1 || stage.Height < 1 {
		return stage
	}
	text, ink := poolHint, pools.DimInk
	if d.playing {
		text, ink = d.last, d.lastInk
	}
	col := 1
	for _, ch := range text {
		if col >= stage.Width {
			break
		}
		stage.Set(stage.Height-1, col, sprite.Cell{Ch: ch, FG: ink, BG: -1})
		col++
	}
	return stage
}

type centered struct {
	w, h int
	draw func() sprite.Sprite
	tick func(float64)
	stop func()
}

func (c *centered) Start(w, h int) { c.w, c.h = w, h }
func (c *centered) Update(dt float64) {
	if c.tick != nil {
		c.tick(dt)
	}
}
func (c *centered) Render() sprite.Sprite {
	if c.w < 1 || c.h < 1 || c.draw == nil {
		return sprite.Sprite{}
	}
	stage := sprite.New(c.w, c.h)
	src := c.draw()
	sprite.Blit(stage, (c.w-src.Width)/2, (c.h-src.Height)/2, src)
	return stage
}
func (c *centered) Stop() {
	if c.stop != nil {
		c.stop()
	}
}

func newFlamePreview() screenplay.Component {
	rose := fire.NewCompass(1)
	return &centered{
		draw: rose.View,
		tick: rose.Update,
		stop: func() { rose.Slots = nil },
	}
}

func newRocketPreview() screenplay.Component {
	r := rocket.New(1)
	return &centered{
		draw: r.View,
		tick: r.Update,
	}
}

func newSkyPreview() screenplay.Component {
	return sky.New().Rise(8)
}

type stacked struct {
	layers []screenplay.Component
	w, h   int
}

func newCloudPreview() screenplay.Component {
	return &stacked{layers: []screenplay.Component{
		sky.New().At(0.7),
		cloud.New(7),
	}}
}

func newTransitionPreview() screenplay.Component {
	return transition.Between(
		transition.Stack(sky.New().At(0.7), cloud.New(7)),
		flag.New(0),
	).Over(6)
}

func (s *stacked) Start(w, h int) {
	s.w, s.h = w, h
	for _, l := range s.layers {
		if l != nil {
			l.Start(w, h)
		}
	}
}

func (s *stacked) Update(dt float64) {
	for _, l := range s.layers {
		if l != nil {
			l.Update(dt)
		}
	}
}

func (s *stacked) Render() sprite.Sprite {
	stage := sprite.New(s.w, s.h)
	for _, l := range s.layers {
		if l == nil {
			continue
		}
		sprite.Blit(stage, 0, 0, l.Render())
	}
	return stage
}

func (s *stacked) Stop() {
	for _, l := range s.layers {
		if l != nil {
			l.Stop()
		}
	}
}

type astroRun struct {
	anim  sprite.Animation
	clock float64
	w, h  int
}

func newAstroRun() *astroRun { return &astroRun{} }

func (a *astroRun) Start(w, h int) {
	a.w, a.h = w, h
	atlas, err := astro.Load()
	if err != nil {
		return
	}
	anim, err := astro.Run(atlas)
	if err != nil {
		return
	}
	a.anim = anim
}

func (a *astroRun) Update(dt float64) {
	if dt > 0 {
		a.clock += dt
	}
}

func (a *astroRun) Render() sprite.Sprite {
	if a.w < 1 || a.h < 1 || a.anim.Len() == 0 {
		return sprite.Sprite{}
	}
	stage := sprite.New(a.w, a.h)
	frame := a.anim.At(a.clock)
	sprite.Blit(stage, (a.w-frame.Width)/2, (a.h-frame.Height)/2, frame)
	return stage
}

func (a *astroRun) Stop() { a.anim = sprite.Animation{} }

type moonwalkPreview struct {
	atlas *sprite.Atlas
	cfg   moonwalk.Config
	clock float64
	w, h  int
}

func newMoonwalkPreview() *moonwalkPreview { return &moonwalkPreview{} }

func (p *moonwalkPreview) Start(w, h int) {
	p.w, p.h = w, h
	p.atlas, _ = astro.Load()
	cfg, err := moonwalk.LoadOrDefault(menu.Resolve(moonwalk.DefaultConfigPath))
	if err != nil {
		cfg = moonwalk.DefaultConfig()
	}
	p.cfg = cfg
}

func (p *moonwalkPreview) Update(dt float64) {
	if dt > 0 {
		p.clock += dt
	}
}

func (p *moonwalkPreview) Render() sprite.Sprite {
	if p.w < 1 || p.h < 1 {
		return sprite.Sprite{}
	}
	return moonwalk.Frame(p.cfg, p.atlas, p.w, p.h, p.clock)
}

func (p *moonwalkPreview) Stop() { p.atlas = nil }

type sceneView struct {
	inner screenplay.Scene
	scr   *screenplay.Screen
	w, h  int
}

func wrapScene(sc screenplay.Scene) *sceneView {
	return &sceneView{inner: sc}
}

func (s *sceneView) Start(w, h int) {
	s.w, s.h = w, h
	s.scr = screenplay.NewScreen(w, h)
	if s.inner != nil {
		s.inner.Start()
	}
}

func (s *sceneView) Update(dt float64) {
	if s.inner != nil {
		s.inner.Update(dt)
	}
}

func (s *sceneView) Render() sprite.Sprite {
	if s.scr == nil || s.inner == nil {
		return sprite.Sprite{}
	}
	s.scr.Clear()
	s.inner.Render(s.scr)
	return cellsToSprite(s.scr)
}

func (s *sceneView) Stop() {
	if s.inner != nil {
		s.inner.Stop()
	}
	s.scr = nil
}

func cellsToSprite(scr *screenplay.Screen) sprite.Sprite {
	w, h := scr.Size()
	sp := sprite.New(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := scr.Cell(x, y)
			if c == nil {
				continue
			}
			fg := indexed(c.Style.Fg)
			bg := indexed(c.Style.Bg)
			ch := ' '
			if c.Content != "" {
				ch = []rune(c.Content)[0]
			}
			if ch == ' ' && bg < 0 {
				continue
			}
			sp.Set(y, x, sprite.Cell{Ch: ch, FG: fg, BG: bg})
		}
	}
	return sp
}

func indexed(c any) int {
	if c == nil {
		return -1
	}
	ic, ok := c.(ansi.IndexedColor)
	if !ok {
		return -1
	}
	return int(ic)
}
