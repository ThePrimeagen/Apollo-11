package dsky

// Tests written FIRST: the DSKY panel is a scene component that docks
// against the right edge of the stage. There is no entrance animation:
// the very first render after Start already paints the whole
// electroluminescent display (VERB/NOUN/PROG and the registers), and
// time never changes what the panel shows. The panel hugs the right
// edge and clips on a stage smaller than itself.

import (
	"strings"
	"testing"

	lab "github.com/theprimeagen/apollo-11/dsky-lab/dsky"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 72
	stageH = 28
)

// The compile-time pin: a Panel plays as a screenplay component.
var _ screenplay.Component = (*Panel)(nil)

func glyphRow(sp sprite.Sprite, r int) string {
	out := make([]rune, sp.Width)
	for c := 0; c < sp.Width; c++ {
		ch := sp.At(r, c).Ch
		if ch == 0 || ch == ' ' {
			out[c] = ' '
			continue
		}
		out[c] = ch
	}
	return string(out)
}

func stageText(sp sprite.Sprite) string {
	rows := make([]string, sp.Height)
	for r := 0; r < sp.Height; r++ {
		rows[r] = glyphRow(sp, r)
	}
	return strings.Join(rows, "\n")
}

func TestPanelSprite(t *testing.T) {
	t.Run("happy: the panel sprite carries the DSKY labels", func(t *testing.T) {
		sp := SpriteOf(MonitorState())
		if sp.Width != lab.Width || sp.Height != lab.Height {
			t.Fatalf("panel sprite %dx%d, want %dx%d", sp.Width, sp.Height, lab.Width, lab.Height)
		}
		text := stageText(sp)
		for _, want := range []string{"VERB", "NOUN", "PROG", "COMP"} {
			if !strings.Contains(text, want) {
				t.Fatalf("panel sprite is missing %q", want)
			}
		}
	})
	t.Run("unhappy: an idle panel still has its labels, never panics", func(t *testing.T) {
		sp := SpriteOf(lab.State{})
		if sp.Width != lab.Width {
			t.Fatalf("idle panel %dx%d, want width %d", sp.Width, sp.Height, lab.Width)
		}
		if !strings.Contains(stageText(sp), "VERB") {
			t.Fatal("even a blank DSKY must show the VERB label")
		}
	})
}

func TestPanelOnStage(t *testing.T) {
	t.Run("happy: the very first render is the whole DSKY, hugging the right edge", func(t *testing.T) {
		p := NewPanel(MonitorState())
		p.Start(stageW, stageH)
		sp := p.Render() // no Update: the panel has no entrance to wait out
		if sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("stage %dx%d, want %dx%d", sp.Width, sp.Height, stageW, stageH)
		}
		text := stageText(sp)
		for _, want := range []string{"VERB", "NOUN", "PROG"} {
			if !strings.Contains(text, want) {
				t.Fatalf("the opening frame is missing %q", want)
			}
		}
		// Every opaque panel cell is on stage, right-hugging and
		// vertically centered — and nothing else is painted.
		panel := SpriteOf(MonitorState())
		x := stageW - panel.Width
		y := (stageH - panel.Height) / 2
		want := 0
		for r := 0; r < panel.Height; r++ {
			for c := 0; c < panel.Width; c++ {
				cell := panel.At(r, c)
				if cell.Transparent() {
					continue
				}
				want++
				if got := sp.At(y+r, x+c); got != cell {
					t.Fatalf("panel cell (%d,%d) at stage (%d,%d): %+v, want %+v", r, c, y+r, x+c, got, cell)
				}
			}
		}
		if got := opaqueCount(sp); got != want {
			t.Fatalf("stage lit %d cells, the panel alone has %d — the component paints only itself", got, want)
		}
	})
	t.Run("happy: time never changes the panel — every update holds the same picture", func(t *testing.T) {
		p := NewPanel(MonitorState())
		p.Start(stageW, stageH)
		first := stageText(p.Render())
		if !strings.Contains(first, "VERB") {
			t.Fatal("test premise: the opening frame must already show the panel")
		}
		// Forward time, a single frame, held time, and (unhappy) time
		// running backwards: none of them may repaint a single cell.
		for _, dt := range []float64{1.0 / 30, 5, 0, -1} {
			p.Update(dt)
			if got := stageText(p.Render()); got != first {
				t.Fatalf("Update(%v) changed the panel:\n%s\nwant:\n%s", dt, got, first)
			}
		}
	})
	t.Run("happy: stop clears the staging; a fresh start shows the whole panel at once", func(t *testing.T) {
		p := NewPanel(MonitorState())
		p.Start(stageW, stageH)
		if opaqueCount(p.Render()) == 0 {
			t.Fatal("test premise: a started panel must show")
		}
		p.Stop()
		if sp := p.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a stopped panel rendered %dx%d", sp.Width, sp.Height)
		}
		p.Start(stageW, stageH)
		if !strings.Contains(stageText(p.Render()), "VERB") {
			t.Fatal("a restaged panel must be whole on its first frame — no entrance replays")
		}
	})
	t.Run("unhappy: a stage smaller than the panel clips instead of panicking", func(t *testing.T) {
		p := NewPanel(MonitorState())
		p.Start(10, 5)
		n := opaqueCount(p.Render())
		if n == 0 {
			t.Fatal("a clipped panel must still show the slice that fits")
		}
		if n > 10*5 {
			t.Fatalf("tiny stage lit %d cells, has only %d", n, 10*5)
		}
	})
	t.Run("unhappy: rendering before the first start is an empty stage", func(t *testing.T) {
		p := NewPanel(MonitorState())
		if sp := p.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("an unstarted panel rendered %dx%d", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a nil panel skips every cue", func(t *testing.T) {
		var ghost *Panel
		ghost.Start(4, 2)
		ghost.Update(1)
		ghost.Render()
		ghost.Stop()
	})
}

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
