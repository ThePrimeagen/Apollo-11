// astronaut: the moonwalk loop. An original NES-styled astronaut —
// big helmet, gold visor, life-support pack — runs in from the left
// on the classic three-frame stride, hops a joy jump, runs on, leaps
// onto the flagpole, slides down it on the two alternating grips, and
// stands at the base before the loop restarts.
//
//	space / enter / p   replay from the top
//	q / ctrl+c          quit
//
//	go run ./cmd/astronaut
//	go run ./cmd/astronaut -seconds 12
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/astro"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

const (
	defaultW   = 72
	defaultH   = 26
	frameMs    = 1000.0 / 30
	statusRows = 1

	// runSpeed is ground speed in cells per second.
	runSpeed = 16.0
	// hopDX/hopSec/hopPeak shape the joy hop mid-route.
	hopDX   = 10
	hopSec  = 0.7
	hopPeak = 4
	// leapDX/leapSec/leapPeak shape the leap onto the pole.
	leapDX   = 12
	leapSec  = 0.8
	leapPeak = 2
	// slideSec rides the pole down; standSec holds the bow.
	slideSec = 1.2
	standSec = 1.8
	// grabAbove is how many rows above his standing head he grabs on.
	grabAbove = 5
	// clockEps soaks float noise off modulo-rebuilt clocks.
	clockEps = 1e-7
)

// floorRow is the ground line's stage row.
func floorRow(stageH int) int { return stageH - 2 }

// groundedY is the sprite top row that parks the boots on the floor.
func groundedY(stageH int) int { return floorRow(stageH) - astro.Rows }

// poleCol is the flagpole's stage column.
func poleCol(stageW int) int { return stageW - 14 }

// route is one loop of choreography, precomputed for a stage size:
// where each leg of the run starts and lands, and how long it takes.
type route struct {
	grounded, grabY        int
	x0, x1, x2, x3, grabX  int
	d1, d3                 float64
	hopAt, leapAt, slideAt float64
	standAt, cycle         float64
}

func routeFor(stageW, stageH int) route {
	r := route{grounded: groundedY(stageH)}
	r.grabY = r.grounded - grabAbove
	r.grabX = poleCol(stageW) - astro.GripCol
	r.x0 = -astro.Cols
	r.x1 = stageW / 4
	r.x2 = r.x1 + hopDX
	r.x3 = r.grabX - leapDX
	r.d1 = runFor(r.x1 - r.x0)
	r.d3 = runFor(r.x3 - r.x2)
	r.hopAt = r.d1
	r.leapAt = r.hopAt + hopSec + r.d3
	r.slideAt = r.leapAt + leapSec
	r.standAt = r.slideAt + slideSec
	r.cycle = r.standAt + standSec
	return r
}

// runFor is how long a ground sprint over cells takes; a degenerate
// stage still gets a beat so the piecewise clock never divides by zero.
func runFor(cells int) float64 {
	d := float64(cells) / runSpeed
	if d < 0.05 {
		return 0.05
	}
	return d
}

// cycleSeconds is how long one full loop of the route takes.
func cycleSeconds(stageW, stageH int) float64 { return routeFor(stageW, stageH).cycle }

// runPose is the stride playing at t on the shared run clock, so the
// step never skips across route segments.
func runPose(t float64) sprite.Heading {
	return astro.RunPoses[int(t*astro.RunFPS+clockEps)%len(astro.RunPoses)]
}

// arc is a parabolic lift: 0 at p=0 and p=1, peak in the middle.
func arc(peak int, p float64) int {
	return int(math.Round(float64(peak) * 4 * p * (1 - p)))
}

func lerp(a, b int, p float64) int {
	return a + int(math.Round(p*float64(b-a)))
}

// timelineAt is the whole choreography as a pure function of time:
// which pose plays and where its sprite's top-left sits on the stage.
// Time wraps at the cycle, so the loop plays forever; time before the
// curtain clamps to the opening stride.
func timelineAt(stageW, stageH int, t float64) (pose sprite.Heading, x, y int) {
	r := routeFor(stageW, stageH)
	if t < 0 {
		t = 0
	}
	t = math.Mod(t, r.cycle)
	if t < 0 {
		t += r.cycle
	}
	switch {
	case t < r.hopAt: // run in from the left wing
		p := t / r.d1
		return runPose(t), lerp(r.x0, r.x1, p), r.grounded
	case t < r.hopAt+hopSec: // the joy hop
		p := (t - r.hopAt) / hopSec
		return astro.PoseJump, lerp(r.x1, r.x2, p), r.grounded - arc(hopPeak, p)
	case t < r.leapAt: // run on toward the pole
		p := (t - r.hopAt - hopSec) / r.d3
		return runPose(t), lerp(r.x2, r.x3, p), r.grounded
	case t < r.slideAt: // leap up onto the pole
		p := (t - r.leapAt) / leapSec
		y := lerp(r.grounded, r.grabY, p) - arc(leapPeak, p)
		return astro.PoseJump, lerp(r.x3, r.grabX, p), y
	case t < r.standAt: // slide down, hands swapping grips
		p := (t - r.slideAt) / slideSec
		grip := astro.PolePoses[int((t-r.slideAt)*astro.PoleFPS+clockEps)%len(astro.PolePoses)]
		return grip, r.grabX, lerp(r.grabY, r.grounded, p)
	default: // stand at the base until the loop restarts
		return astro.PoseStand, r.grabX, r.grounded
	}
}

type model struct {
	w, h    int
	clock   float64
	seconds float64
	atlas   *sprite.Atlas
}

func newModel(seconds float64) (model, error) {
	atlas, err := astro.Load()
	if err != nil {
		return model{}, err
	}
	return model{w: defaultW, h: defaultH, seconds: seconds, atlas: atlas}, nil
}

type frameMsg struct{}

func tick() tea.Cmd {
	ns := float64(frameMs) * 1e6
	return tea.Tick(time.Duration(ns)*time.Nanosecond, func(time.Time) tea.Msg {
		return frameMsg{}
	})
}

func (m model) Init() tea.Cmd { return tick() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case frameMsg:
		m.clock += frameMs / 1000
		if m.seconds > 0 && m.clock >= m.seconds {
			return m, tea.Quit
		}
		return m, tick()
	case tea.KeyPressMsg:
		switch strings.ToLower(msg.String()) {
		case "ctrl+c", "q":
			return m, tea.Quit
		case " ", "space", "enter", "p":
			m.clock = 0
			return m, nil
		}
	}
	return m, nil
}

// paintScene lays the still life: a sparse starfield, the ground line
// with a dust of regolith under it, and the flagpole with its gold
// ball finial.
func paintScene(stage sprite.Sprite) {
	fr := floorRow(stage.Height)
	for i := 0; i < stage.Width/6; i++ {
		col := (i*23 + 7) % stage.Width
		row := (i*13 + 3) % maxInt(1, fr-3)
		stage.Set(row, col, sprite.Cell{Ch: '·', FG: 240, BG: -1})
	}
	for c := 0; c < stage.Width; c++ {
		stage.Set(fr, c, sprite.Cell{Ch: '▀', FG: 245, BG: -1})
		if (c*7+fr)%13 == 0 {
			stage.Set(fr+1, c, sprite.Cell{Ch: '·', FG: 240, BG: -1})
		}
	}
	pc := poleCol(stage.Width)
	top := groundedY(stage.Height) - grabAbove
	for row := top; row < fr; row++ {
		stage.Set(row, pc, sprite.Cell{Ch: '│', FG: 250, BG: -1})
	}
	stage.Set(top-1, pc, sprite.Cell{Ch: '●', FG: 220, BG: -1})
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m model) View() tea.View {
	w, h := m.w, m.h
	if w < 1 {
		w = defaultW
	}
	if h < 1 {
		h = defaultH
	}
	stageH := maxInt(1, h-statusRows)
	stage := sprite.New(w, stageH)
	paintScene(stage)
	pose, x, y := timelineAt(w, stageH, m.clock)
	if sp, ok := m.atlas.Frame(astro.Size, pose); ok {
		sprite.Blit(stage, x, y, sp)
	}
	lines := strings.Split(sprite.Render(stage), "\n")
	status := "\x1b[38;5;240m" + clip("astronaut  the moonwalk loop  space replay  q quit", w) + "\x1b[0m"
	lines = append(lines, status)
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	return v
}

func clip(s string, w int) string {
	r := []rune(s)
	if len(r) > w {
		return string(r[:w])
	}
	return s
}

func forcedColorProfile() (colorprofile.Profile, bool) {
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return colorprofile.ANSI256, true
	}
	return 0, false
}

func main() {
	seconds := flag.Float64("seconds", 0, "auto-quit after N seconds (0 = interactive)")
	flag.Parse()
	m, err := newModel(*seconds)
	if err != nil {
		fmt.Fprintln(os.Stderr, "astronaut:", err)
		os.Exit(1)
	}
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "astronaut:", err)
		os.Exit(1)
	}
}
