// Package dsky is the DSKY as a scene component: the Apollo Display
// and Keyboard docks against the right edge of the stage, whole from
// its first frame, and types like the real unit — Press feeds it VERB,
// NOUN, digits, ENTR, CLR, and RSET. The component holds no animation
// — any entrance choreography belongs to the scene that casts it.
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
)

// Panel is the DSKY as a scene component. Start pins the stage, Render
// paints the whole panel into a stage-sized sprite hugging the right
// edge, and Stop clears the staging. Press types on it — VERB, NOUN,
// digits, ENTR, CLR, RSET (see keys.go). The panel keeps no clocks, so
// every frame — the first included — shows the complete display.
type Panel struct {
	State  lab.State
	w, h   int
	staged bool
	// The open keypad entry: which field digits land in, what has been
	// typed, and the value to fall back to (keys.go).
	entry Key
	buf   string
	prev  string
}

// NewPanel opens a DSKY. Nothing is built until Start.
func NewPanel(st lab.State) *Panel {
	return &Panel{State: st}
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

// Start pins the stage the panel docks on.
func (p *Panel) Start(w, h int) {
	if p == nil {
		return
	}
	p.w, p.h = w, h
	p.staged = true
}

// Update holds: the panel has no clocks to advance. It exists so a
// Panel plays as a screenplay component.
func (p *Panel) Update(dt float64) {}

// Render paints the whole DSKY into a stage-sized sprite, hugging the
// right edge and vertically centered. Before Start and after Stop the
// panel is off.
func (p *Panel) Render() sprite.Sprite {
	if p == nil || !p.staged || p.w < 1 || p.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(p.w, p.h)
	panel := SpriteOf(p.State)
	x := p.w - panel.Width
	y := (p.h - panel.Height) / 2
	for r := 0; r < panel.Height; r++ {
		for c := 0; c < panel.Width; c++ {
			cell := panel.At(r, c)
			if cell.Transparent() {
				continue
			}
			stage.Set(y+r, x+c, cell)
		}
	}
	return stage
}

// Stop clears the staging; the State is the panel's identity and
// stays, so a fresh Start shows it again.
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
