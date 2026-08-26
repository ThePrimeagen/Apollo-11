package dsky

// Tests written FIRST: the DSKY panel is a scene component that docks
// on the right third of the stage. Over WipeSeconds it reveals one
// column at a time from the right edge, painting the electroluminescent
// display (VERB/NOUN/PROG and the registers) into the blanked sky. The
// panel hugs the right edge and clips on a stage smaller than itself.

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

func columnLit(sp sprite.Sprite, col int) bool {
	for r := 0; r < sp.Height; r++ {
		if !sp.At(r, col).Transparent() {
			return true
		}
	}
	return false
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
	t.Run("happy: after the wipe the DSKY hugs the right edge", func(t *testing.T) {
		p := NewPanel(MonitorState())
		p.Start(stageW, stageH)
		p.Update(WipeSeconds)
		sp := p.Render()
		if sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("stage %dx%d, want %dx%d", sp.Width, sp.Height, stageW, stageH)
		}
		text := stageText(sp)
		for _, want := range []string{"VERB", "NOUN", "PROG"} {
			if !strings.Contains(text, want) {
				t.Fatalf("revealed panel is missing %q", want)
			}
		}
		// VERB sits on the panel; the panel's right edge is the stage's.
		left := stageW - lab.Width
		found := false
		for r := 0; r < sp.Height; r++ {
			row := glyphRow(sp, r)
			if i := strings.Index(row, "VERB"); i >= 0 {
				found = true
				if i < left {
					t.Fatalf("VERB starts at col %d, want ≥ %d (right-hugging panel)", i, left)
				}
			}
		}
		if !found {
			t.Fatal("VERB must be on stage after the wipe")
		}
	})
	t.Run("happy: at t=0 nothing of the panel has been revealed", func(t *testing.T) {
		p := NewPanel(MonitorState())
		p.Start(stageW, stageH)
		if n := opaqueCount(p.Render()); n != 0 {
			t.Fatalf("opening frame lit %d cells — the wipe has not started", n)
		}
	})
	t.Run("happy: columns appear from the right, one at a time", func(t *testing.T) {
		p := NewPanel(MonitorState())
		p.Start(stageW, stageH)
		want := DockCols(stageW)
		prev := 0
		for i := 1; i <= 10; i++ {
			p.Update(WipeSeconds / 10)
			sp := p.Render()
			lit := 0
			for c := stageW - 1; c >= 0; c-- {
				if !columnLit(sp, c) {
					break
				}
				lit++
			}
			if lit < prev {
				t.Fatalf("revealed columns shrank from %d to %d — the wipe only grows", prev, lit)
			}
			prev = lit
		}
		if prev == 0 {
			t.Fatal("after the wipe the right edge must hold DSKY cells")
		}
		if prev > want {
			t.Fatalf("revealed %d columns past the dock of %d", prev, want)
		}
	})
	t.Run("happy: stop clears the staging; a fresh start keeps the wipe clock", func(t *testing.T) {
		p := NewPanel(MonitorState())
		p.Start(stageW, stageH)
		p.Update(WipeSeconds)
		if opaqueCount(p.Render()) == 0 {
			t.Fatal("test premise: a wiped panel must show")
		}
		p.Stop()
		if sp := p.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a stopped panel rendered %dx%d", sp.Width, sp.Height)
		}
		p.Start(stageW, stageH)
		if opaqueCount(p.Render()) == 0 {
			t.Fatal("a restaged panel must keep its wipe clock — already revealed")
		}
	})
	t.Run("unhappy: a stage smaller than the panel clips instead of panicking", func(t *testing.T) {
		p := NewPanel(MonitorState())
		p.Start(10, 5)
		p.Update(WipeSeconds)
		if n := opaqueCount(p.Render()); n > 10*5 {
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

func TestDockCols(t *testing.T) {
	t.Run("happy: a 72-wide stage docks 25 — enough for the panel, ~one third", func(t *testing.T) {
		if got := DockCols(72); got != lab.Width {
			t.Fatalf("DockCols(72)=%d, want %d", got, lab.Width)
		}
	})
	t.Run("happy: a wide stage docks a full third", func(t *testing.T) {
		if got := DockCols(120); got != 40 {
			t.Fatalf("DockCols(120)=%d, want 40", got)
		}
	})
	t.Run("unhappy: a stage narrower than the panel docks the whole width", func(t *testing.T) {
		if got := DockCols(10); got != 10 {
			t.Fatalf("DockCols(10)=%d, want 10", got)
		}
	})
	t.Run("unhappy: a zero stage docks nothing", func(t *testing.T) {
		if got := DockCols(0); got != 0 {
			t.Fatalf("DockCols(0)=%d, want 0", got)
		}
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
