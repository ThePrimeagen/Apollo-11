// Package dsky is the DSKY as a scene component: the Apollo Display
// and Keyboard docks on the right third of the stage and reveals one
// column at a time from the right edge, so a cut to this scene wipes
// the sky and plants the panel in its place.
package dsky

import (
	"strconv"
	"strings"
	"unicode/utf8"

	lab "github.com/theprimeagen/apollo-11/dsky-lab/dsky"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	// Width/Height re-export the lab's fixed footprint.
	Width  = lab.Width
	Height = lab.Height
	// WipeSeconds is how long the right-edge column reveal takes.
	WipeSeconds = 0.5
)

// Panel is the DSKY as a scene component. Start pins the stage, Update
// runs the wipe clock, Render paints the revealed columns into a
// stage-sized sprite hugging the right edge, and Stop clears the
// staging. The clock carries across restages so a resize never replays
// the wipe.
type Panel struct {
	State       lab.State
	clock       float64
	w, h        int
	staged      bool
	wipeSeconds float64
}

// NewPanel opens a DSKY that will wipe in over WipeSeconds. Nothing is
// built until Start.
func NewPanel(st lab.State) *Panel {
	return &Panel{State: st, wipeSeconds: WipeSeconds}
}

// MonitorState is the iconic descent monitor: V16 N68 on P63, with
// ALT/VEL still burning.
func MonitorState() lab.State {
	return lab.State{
		Prog: "63", Verb: "16", Noun: "68",
		R1: "+01405", R2: "+00335", R3: "-02900",
		Lights: lab.Lights{Alt: true, Vel: true},
	}
}

// DockCols is how many columns from the right edge the dock occupies:
// the larger of one third of the stage and the panel width, clipped to
// the stage.
func DockCols(width int) int {
	if width < 1 {
		return 0
	}
	n := width / 3
	if Width > n {
		n = Width
	}
	if n > width {
		n = width
	}
	return n
}

func wipeCols(total int, t, seconds float64) int {
	if total < 1 {
		return 0
	}
	if seconds <= 0 || t+1e-9 >= seconds {
		return total
	}
	if t <= 0 {
		return 0
	}
	n := int(float64(total) * t / seconds)
	if n > total {
		return total
	}
	return n
}

// Start pins the stage the panel docks on.
func (p *Panel) Start(w, h int) {
	if p == nil {
		return
	}
	p.w, p.h = w, h
	p.staged = true
}

// Update advances the wipe. dt <= 0 holds.
func (p *Panel) Update(dt float64) {
	if p == nil || dt <= 0 {
		return
	}
	p.clock += dt
}

// Render paints the revealed slice of the DSKY into a stage-sized
// sprite. Before Start and after Stop the panel is off.
func (p *Panel) Render() sprite.Sprite {
	if p == nil || !p.staged || p.w < 1 || p.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(p.w, p.h)
	panel := SpriteOf(p.State)
	dark := wipeCols(DockCols(p.w), p.clock, p.wipeSeconds)
	cutoff := p.w - dark
	x := p.w - panel.Width
	y := (p.h - panel.Height) / 2
	for r := 0; r < panel.Height; r++ {
		for c := 0; c < panel.Width; c++ {
			col := x + c
			if col < cutoff {
				continue
			}
			cell := panel.At(r, c)
			if cell.Transparent() {
				continue
			}
			stage.Set(y+r, col, cell)
		}
	}
	return stage
}

// Stop clears the staging; the State is the panel's identity and
// stays, so a fresh Start shows it again. The clock carries on.
func (p *Panel) Stop() {
	if p == nil {
		return
	}
	p.staged = false
}

// SpriteOf paints one DSKY state as a Width×Height sprite. Spaces that
// carry a background (the caution lights, COMP ACTY) become blocks so
// they survive sprite.Blit, which treats a space as transparent.
func SpriteOf(st lab.State) sprite.Sprite {
	return spriteFromANSI(lab.Render(st, true))
}

func spriteFromANSI(s string) sprite.Sprite {
	lines := strings.Split(s, "\n")
	h := len(lines)
	w := Width
	sp := sprite.New(w, h)
	for r, line := range lines {
		col := 0
		fg, bg := -1, -1
		i := 0
		for i < len(line) {
			if line[i] == '\x1b' && i+1 < len(line) && line[i+1] == '[' {
				j := i + 2
				for j < len(line) && line[j] != 'm' {
					j++
				}
				if j <= len(line) {
					end := j
					if j < len(line) {
						applySGR(line[i+2:end], &fg, &bg)
						i = j + 1
						continue
					}
				}
			}
			ch, size := utf8.DecodeRuneInString(line[i:])
			cellFG, cellBG := fg, bg
			if ch == ' ' && bg >= 0 {
				ch = '█'
				cellFG = bg
			}
			if col < w {
				sp.Set(r, col, sprite.Cell{Ch: ch, FG: cellFG, BG: cellBG})
			}
			col++
			i += size
		}
	}
	return sp
}

func applySGR(params string, fg, bg *int) {
	if params == "" || params == "0" {
		*fg, *bg = -1, -1
		return
	}
	parts := strings.Split(params, ";")
	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "0":
			*fg, *bg = -1, -1
		case "38":
			if i+2 < len(parts) && parts[i+1] == "5" {
				*fg = atoi(parts[i+2])
				i += 2
			}
		case "48":
			if i+2 < len(parts) && parts[i+1] == "5" {
				*bg = atoi(parts[i+2])
				i += 2
			}
		}
	}
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return n
}
