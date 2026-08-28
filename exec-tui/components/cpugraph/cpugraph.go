// Package cpugraph is the CPU portrait as its own scene component: 2.5
// seconds of "here is what the CPU operates with" under the current
// switch states, never animated, and NOTHING else — no legend text, no
// switch row, no surrounding information. One row per process that
// consumed CPU, grouped under VAC JOBS / CORESET JOBS / NO-PRIORITY OPS /
// COUNTER THEFT headers (the last is the RR CDU steal itself — hardware,
// no job, no memory), names inside a 20-column gutter, light-gray
// vertical gridlines every 100 ms (brighter on the seconds), a HARD
// WHITE line on the 2.00 s guidance boundary, and a millisecond axis
// underneath. The SERVICER is entered exactly ONCE per portrait, so its
// row is the single pass stretching toward the white line as load is
// switched on — and turning RED past it.
//
// The component owns the msim snapshot and the whole switch API, so any
// composite can wire its own toggles: SetDescent, SetMonitor, SetRadar
// and SetApproach re-simulate a fresh 2.5 s still (the 1668 monitor and
// the P64 approach cannot share the DSKY — keying either through the API
// drops the other), while Running, Stolen and Engine expose everything a
// legend needs to describe the picture.
//
// Two projections of the one portrait: Rows(width) is the styled-string
// render a Bubble Tea screen embeds line for line, and Art(width) /
// Render() is the sprite a scene blits — a still screenplay.Component
// whose Update moves nothing. The switches are the graph's identity and
// survive Stop/Start; the stage does not.
package cpugraph

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	msim "github.com/theprimeagen/apollo-11/msim"
)

// windowMS is the portrait span: 2.5 s across the plot columns.
const windowMS = 2500

// boundaryMS is the guidance boundary the hard white line marks: the
// instant the next READACCS arrives and a finished SERVICER would have
// already reached ENDOFJOB.
const boundaryMS = 2000

// gutter is the label column budget.
const gutter = 20

// maxPlot is the plot column budget.
const maxPlot = 180

// The lane inks, xterm-256 — exported so composites and scenes can match
// the portrait's colors elsewhere on their stage.
const (
	LabelInk      = 250 // group headers
	VacInk        = 46  // jobs holding a VAC
	CoreInk       = 214 // jobs holding a core set only
	OpsInk        = 75  // the no-priority operations
	TheftInk      = 135 // the RR CDU counter theft
	OverrunInk    = 196 // the SERVICER tail past the boundary
	GridInk       = 238 // 100 ms gridlines
	GridBrightInk = 245 // second gridlines and the axis
	BoundaryInk   = 231 // the hard white 2.00 s line
)

// ink is one drawing color in both projections: the xterm index the
// sprite cells wear and the lipgloss style the string render wears.
type ink struct {
	idx int
	st  lipgloss.Style
}

func plainInk(idx int) ink {
	return ink{idx: idx, st: lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprint(idx)))}
}

func boldInk(idx int) ink {
	return ink{idx: idx, st: lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprint(idx))).Bold(true)}
}

var (
	inkLabel      = boldInk(LabelInk)
	inkVac        = plainInk(VacInk)
	inkCore       = plainInk(CoreInk)
	inkOps        = plainInk(OpsInk)
	inkTheft      = plainInk(TheftInk)
	inkOver       = plainInk(OverrunInk)
	inkGrid       = plainInk(GridInk)
	inkGridBright = plainInk(GridBrightInk)
	inkBound      = boldInk(BoundaryInk)
)

var blockRunes = []rune("▁▂▃▄▅▆▇█")

// process groups: who holds what while consuming the CPU. The last group
// is not software at all — the RR CDU counter theft, which holds nothing
// and answers to nobody.
const (
	groupVac = iota
	groupCore
	groupOps
	groupTheft
)

var groupLabels = [...]string{"VAC JOBS", "CORESET JOBS", "NO-PRIORITY OPS", "COUNTER THEFT"}
var groupInks = [...]ink{inkVac, inkCore, inkOps, inkTheft}

// proc is one describable process: its lane group under the current
// switches, how it is activated, and how often.
type proc struct {
	name   string
	period string
	count  func(*msim.Engine) int
	group  func(approach bool) int
}

func spawns(name string) func(*msim.Engine) int {
	return func(e *msim.Engine) int { return e.SpawnCount(name) }
}

func tasks(name string) func(*msim.Engine) int {
	return func(e *msim.Engine) int { return e.TaskFires(name) }
}

func rupts(name string) func(*msim.Engine) int {
	return func(e *msim.Engine) int { return e.InterruptFires(name) }
}

func fixed(g int) func(bool) int { return func(bool) int { return g } }

// procs is the catalog, in display order. A row (and its Running entry)
// appears only when the process consumed CPU in the window. MAKEPLAY moves
// with the display form: the P63 static V06N63 is NOVAC, the approach's
// flashing V06N64 holds a VAC while it sleeps awaiting PRO.
var procs = []proc{
	{"SERVICER", "2s", spawns("SERVICER"), fixed(groupVac)},
	{"MAKEPLAY", "2s", spawns("MAKEPLAY"), func(approach bool) int {
		if approach {
			return groupVac
		}
		return groupCore
	}},
	{"HIGATJOB", "high gate", spawns("HIGATJOB"), fixed(groupVac)},
	{"LRHJOB", "2s", spawns("LRHJOB"), fixed(groupCore)},
	{"LRVJOB", "2s", spawns("LRVJOB"), fixed(groupCore)},
	{"MONDO", "1s", spawns("MONDO"), fixed(groupCore)},
	{"CHARIN", "keystroke", spawns("CHARIN"), fixed(groupCore)},
	{"1/GYRO", "2s", spawns("1/GYRO"), fixed(groupCore)},
	{"READACCS", "2s", tasks("READACCS"), fixed(groupOps)},
	{"R10,R11", "250ms", tasks("R10,R11"), fixed(groupOps)},
	{"LRHTASK", "2s", tasks("LRHTASK"), fixed(groupOps)},
	{"LRVTASK", "2s", tasks("LRVTASK"), fixed(groupOps)},
	{"HIGATASK", "high gate", tasks("HIGATASK"), fixed(groupOps)},
	{"MONREQ", "1s", tasks("MONREQ"), fixed(groupOps)},
	{"DAP", "100ms", rupts("DAP"), fixed(groupOps)},
	{"T4RUPT", "120ms", rupts("T4RUPT"), fixed(groupOps)},
	{"DOWNRUPT", "20ms", rupts("DOWNRUPT"), fixed(groupOps)},
}

// Graph is the CPU portrait: one frozen 2.5 s snapshot per switch
// configuration, owning the simulation and the switches that shape it.
type Graph struct {
	live     *msim.Live
	descent  bool
	monitor  bool
	radar    bool
	approach bool
	w, h     int
	staged   bool
}

// New opens on the healthy portrait: descent on, monitor off, steal off,
// approach off — pre-simulated and ready to draw.
func New() *Graph {
	g := &Graph{descent: true}
	g.rebuild()
	return g
}

// rebuild re-simulates a fresh 2.5 s snapshot under the current switches.
// The portrait's fixed rules: the SERVICER is entered once (everything
// else keeps its timer), and the theft sweep rides its worst-case crest —
// the RESEARCH.md "worst 2 s window" — instead of the flight window's
// floor dwell.
func (g *Graph) rebuild() {
	l := msim.NewLive()
	l.SetRadar(g.radar)
	l.SetDescent(g.descent)
	l.SetServicerOneShot(true)
	l.SetApproach(g.approach)
	l.Engine().SetTheftPhaseMS(msim.TheftPeakPhaseMS)
	if g.monitor {
		// the monitor is already up as the portrait opens, on the flight's
		// ENTR phase: each 1 Hz refresh lands .985 into its second, the
		// second one straddling the white line
		msim.StartMonitor(l.Engine(), -15*msim.Millisecond)
	}
	l.StepMS(windowMS)
	g.live = l
}

// Descent reports the descent chain switch.
func (g *Graph) Descent() bool { return g != nil && g.descent }

// Monitor reports the 1668 DELTAH monitor switch.
func (g *Graph) Monitor() bool { return g != nil && g.monitor }

// Radar reports the rendezvous-radar counter-steal switch.
func (g *Graph) Radar() bool { return g != nil && g.radar }

// Approach reports the P64 approach-phase switch.
func (g *Graph) Approach() bool { return g != nil && g.approach }

// SetDescent switches the descent chain and re-simulates. A set that
// changes nothing is refused: same switches, same portrait.
func (g *Graph) SetDescent(on bool) {
	if g == nil || g.descent == on {
		return
	}
	g.descent = on
	g.rebuild()
}

// SetMonitor keys the 1668 DELTAH monitor and re-simulates. The monitor
// owns the DSKY — keying it drops the P64 approach display. A set that
// changes nothing is refused.
func (g *Graph) SetMonitor(on bool) {
	if g == nil || g.monitor == on {
		return
	}
	g.monitor = on
	if on {
		g.approach = false
	}
	g.rebuild()
}

// SetRadar switches the counter steal and re-simulates. A set that
// changes nothing is refused.
func (g *Graph) SetRadar(on bool) {
	if g == nil || g.radar == on {
		return
	}
	g.radar = on
	g.rebuild()
}

// SetApproach keys the P64 approach phase and re-simulates. P64's
// flashing V06N64 owns the DSKY — keying it drops the 1668 monitor. A
// set that changes nothing is refused.
func (g *Graph) SetApproach(on bool) {
	if g == nil || g.approach == on {
		return
	}
	g.approach = on
	if on {
		g.monitor = false
	}
	g.rebuild()
}

// Engine is read access to the frozen snapshot, for composites that
// describe the portrait beyond what Running and Stolen carry.
func (g *Graph) Engine() *msim.Engine {
	if g == nil || g.live == nil {
		return nil
	}
	return g.live.Engine()
}

// Process is one running process as a legend would list it: its lane
// name, activation period, window busy total, and activation count.
type Process struct {
	Name   string
	Period string
	Busy   msim.Nanos
	Fires  int
}

// Running lists every process that consumed CPU in the window, in lane
// order — the same order the rows draw in.
func (g *Graph) Running() []Process {
	if g == nil {
		return nil
	}
	e := g.Engine()
	var out []Process
	for grp := range groupLabels {
		for _, p := range procs {
			if p.group(g.approach) != grp || !running(e, p.name) {
				continue
			}
			out = append(out, Process{Name: p.name, Period: p.period, Busy: e.BusyNs(p.name), Fires: p.count(e)})
		}
	}
	return out
}

// Stolen is the theft ledger's window total: the time the RR CDU counter
// steal skimmed across the whole 2.5 s, zero when the steal is off.
func (g *Graph) Stolen() msim.Nanos {
	if g == nil {
		return 0
	}
	return g.Engine().TheftNsBefore(windowMS * msim.Millisecond)
}

// running reports whether the process consumed any CPU in the window.
func running(e *msim.Engine, name string) bool { return e.BusyNs(name) > 0 }

// column is one plotted slice: the bar level (0..8 eighths of one row) and
// its grid marking.
type column struct {
	level int
	grid  int // 0 none, 1 light (100 ms), 2 strong (1 s)
}

// columns buckets the snapshot's window into `plot` columns, levelling
// each column's busy time (0..8 eighths of one row) from the given
// busy(loMs, hiMs) accumulator.
func (g *Graph) columns(plot int, busy func(loMs, hiMs int) msim.Nanos) []column {
	out := make([]column, plot)
	for i := 0; i < plot; i++ {
		loMs := i * windowMS / plot
		hiMs := (i + 1) * windowMS / plot
		if hiMs == loMs {
			hiMs = loMs + 1
		}
		if b := (loMs + 99) / 100 * 100; b >= loMs && b < hiMs {
			out[i].grid = 1
			if b%1000 == 0 {
				out[i].grid = 2
			}
		}
		b := busy(loMs, hiMs)
		span := msim.Nanos(hiMs-loMs) * msim.Millisecond
		lvl := int((b*8 + span/2) / span)
		if b > 0 && lvl == 0 {
			lvl = 1 // sub-slice work must stay visible
		}
		if lvl > 8 {
			lvl = 8
		}
		out[i].level = lvl
	}
	return out
}

// sampleBusy accumulates one named consumer's per-millisecond attribution.
func (g *Graph) sampleBusy(name string) func(int, int) msim.Nanos {
	samples := g.Engine().Samples()
	return func(lo, hi int) msim.Nanos {
		var b msim.Nanos
		for ms := lo; ms < hi && ms < len(samples); ms++ {
			if ms >= 0 {
				b += samples[ms].ByName[name]
			}
		}
		return b
	}
}

// theftBusy accumulates the hardware skim, exact per the engine's ledger.
func (g *Graph) theftBusy() func(int, int) msim.Nanos {
	e := g.Engine()
	return func(lo, hi int) msim.Nanos {
		return e.TheftNsBefore(msim.Nanos(hi)*msim.Millisecond) -
			e.TheftNsBefore(msim.Nanos(lo)*msim.Millisecond)
	}
}

// geometry splits a width budget into the gutter and plot columns: the
// 20-column gutter unless the budget is too tight to keep it, and a plot
// capped at maxPlot but never below one column.
func geometry(width int) (gut, plot int) {
	gut = gutter
	if width < gut+10 {
		gut = 0
	}
	plot = width - gut
	if plot > maxPlot {
		plot = maxPlot
	}
	if plot < 1 {
		plot = 1
	}
	return
}

// lane is one planned row of the portrait: the gutter text, its ink, the
// plotted columns, and the bar inks on either side of the boundary.
type lane struct {
	gutter string
	gink   ink
	cols   []column
	bar    ink
	over   ink
}

// plan lays the portrait out for a width budget — the single source both
// projections draw from: the lanes in display order, the axis characters,
// and the geometry that framed them.
func (g *Graph) plan(width int) (lanes []lane, axis string, gut, plot, bcol int) {
	gut, plot = geometry(width)
	bcol = boundaryMS * plot / windowMS
	e := g.Engine()
	none := func(int, int) msim.Nanos { return 0 }
	for gi, label := range groupLabels {
		lanes = append(lanes, lane{gutter: label, gink: inkLabel, cols: g.columns(plot, none), bar: inkGrid, over: inkGrid})
		if gi == groupTheft {
			if g.Stolen() > 0 {
				lanes = append(lanes, lane{gutter: " RR CDU", gink: inkTheft, cols: g.columns(plot, g.theftBusy()), bar: inkTheft, over: inkTheft})
			}
			continue
		}
		for _, p := range procs {
			if p.group(g.approach) != gi || !running(e, p.name) {
				continue
			}
			over := groupInks[gi]
			if p.name == "SERVICER" {
				// a single-cycle pass still running past its own boundary
				// is the overflow: paint the overrun red
				over = inkOver
			}
			lanes = append(lanes, lane{gutter: " " + p.name, gink: groupInks[gi], cols: g.columns(plot, g.sampleBusy(p.name)), bar: groupInks[gi], over: over})
		}
	}
	return lanes, axisCells(plot), gut, plot, bcol
}

// axisCells anchors an "Nms" label at every other gridline column (every
// 200 ms), skipping any label that would not fit or would collide.
func axisCells(plot int) string {
	cells := make([]rune, plot)
	for i := range cells {
		cells[i] = ' '
	}
	for t := 0; t < windowMS; t += 200 {
		col := t * plot / windowMS
		label := []rune(fmt.Sprintf("%dms", t))
		if col+len(label) > plot {
			continue
		}
		free := true
		lo := col - 1
		if lo < 0 {
			lo = 0
		}
		for _, r := range cells[lo : col+len(label)] {
			if r != ' ' {
				free = false
				break
			}
		}
		if !free {
			continue
		}
		copy(cells[col:], label)
	}
	return string(cells)
}

// laneRow renders one lane's plot as a styled string: bars over
// gridlines, with the hard white boundary line cutting through everything
// — bars included. Bars in columns past the boundary render under `over`.
func laneRow(st, over lipgloss.Style, cols []column, bcol int) string {
	var b strings.Builder
	for i, c := range cols {
		switch {
		case i == bcol:
			b.WriteString(inkBound.st.Render("│"))
		case c.level > 0:
			if i > bcol {
				b.WriteString(over.Render(string(blockRunes[c.level-1])))
			} else {
				b.WriteString(st.Render(string(blockRunes[c.level-1])))
			}
		case c.grid == 2:
			b.WriteString(inkGridBright.st.Render("│"))
		case c.grid == 1:
			b.WriteString(inkGrid.st.Render("│"))
		default:
			b.WriteString(" ")
		}
	}
	return b.String()
}

// Rows is the styled-string projection at a width budget: the lane rows
// and the axis, exactly as a terminal screen embeds them — and nothing
// else. No legend, no switches: those belong to whoever composes the
// graph.
func (g *Graph) Rows(width int) []string {
	if g == nil {
		return nil
	}
	lanes, axis, gut, _, bcol := g.plan(width)
	pad := strings.Repeat(" ", gut)
	gutterCell := func(st lipgloss.Style, text string) string {
		if gut == 0 {
			return ""
		}
		if len(text) > gut {
			text = text[:gut]
		}
		return st.Render(fmt.Sprintf("%-*s", gut, text))
	}
	out := make([]string, 0, len(lanes)+1)
	for _, l := range lanes {
		out = append(out, gutterCell(l.gink.st, l.gutter)+laneRow(l.bar.st, l.over.st, l.cols, bcol))
	}
	out = append(out, pad+inkGridBright.st.Render(axis))
	return out
}

// Art is the sprite projection at a width budget: the same lanes and axis
// Rows draws, in the lane inks, on an exact-size sprite for scenes that
// place the portrait themselves. A budget below one column is refused
// with an empty sprite.
func (g *Graph) Art(width int) sprite.Sprite {
	if g == nil || width < 1 {
		return sprite.Sprite{}
	}
	lanes, axis, gut, plot, bcol := g.plan(width)
	sp := sprite.New(gut+plot, len(lanes)+1)
	for r, l := range lanes {
		if gut > 0 {
			text := l.gutter
			if len(text) > gut {
				text = text[:gut]
			}
			for c, ch := range []rune(text) {
				if ch == ' ' {
					continue
				}
				sp.Set(r, c, sprite.Cell{Ch: ch, FG: l.gink.idx, BG: -1})
			}
		}
		for i, c := range l.cols {
			x := gut + i
			switch {
			case i == bcol:
				sp.Set(r, x, sprite.Cell{Ch: '│', FG: BoundaryInk, BG: -1})
			case c.level > 0:
				k := l.bar
				if i > bcol {
					k = l.over
				}
				sp.Set(r, x, sprite.Cell{Ch: blockRunes[c.level-1], FG: k.idx, BG: -1})
			case c.grid == 2:
				sp.Set(r, x, sprite.Cell{Ch: '│', FG: GridBrightInk, BG: -1})
			case c.grid == 1:
				sp.Set(r, x, sprite.Cell{Ch: '│', FG: GridInk, BG: -1})
			}
		}
	}
	for i, ch := range []rune(axis) {
		if ch == ' ' {
			continue
		}
		sp.Set(len(lanes), gut+i, sprite.Cell{Ch: ch, FG: GridBrightInk, BG: -1})
	}
	return sp
}

// Start pins the stage: the curtain rises on a w×h slab.
func (g *Graph) Start(w, h int) {
	if g == nil {
		return
	}
	g.w, g.h = w, h
	g.staged = true
}

// Update is a deliberate no-op: the portrait is a STILL — only the
// switches change the picture, never the clock.
func (g *Graph) Update(float64) {}

// Render paints the portrait centered on a stage-sized sprite. Before
// Start and after Stop the stage is empty; a stage smaller than the art
// clips at the edges.
func (g *Graph) Render() sprite.Sprite {
	if g == nil || !g.staged || g.w < 1 || g.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(g.w, g.h)
	art := g.Art(g.w)
	x := (g.w - art.Width) / 2
	y := (g.h - art.Height) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	sprite.Blit(stage, x, y, art)
	return stage
}

// Stop clears the staging; the switches and the snapshot stay.
func (g *Graph) Stop() {
	if g == nil {
		return
	}
	g.staged = false
}
