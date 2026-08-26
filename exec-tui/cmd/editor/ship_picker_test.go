package editor

// Ctrl-P opens a size×heading gallery. Enter loads the highlighted
// combination; escape leaves the current frame alone. A missing frame
// must not panic or switch the canvas.

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestShipPicker(t *testing.T) {
	t.Run("happy: ctrl-p opens a size×heading gallery", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 120, 40
		m = send(m, keyCtrl('p'))
		if !m.ShipPickerOpen {
			t.Fatal("ctrl-p must open the ship picker")
		}
		v := m.View().Content
		for _, h := range sprite.Headings {
			if !strings.Contains(v, string(h)) {
				t.Fatalf("picker view missing heading %q", h)
			}
		}
		for _, sz := range sprite.Sizes {
			if !strings.Contains(v, fmt.Sprintf("%d", sz)) {
				t.Fatalf("picker view missing size %d", sz)
			}
		}
	})
	t.Run("happy: hjkl and enter select a size and heading", func(t *testing.T) {
		m := newEd(t)
		m.Size = sprite.Size4
		m.Heading = sprite.N
		m = send(m, keyCtrl('p'))
		m = send(m, key('l'))
		m = send(m, key('k'))
		m = send(m, keyType(tea.KeyEnter))
		if m.ShipPickerOpen {
			t.Fatal("enter must close the picker")
		}
		if m.Heading != sprite.NE {
			t.Fatalf("enter must load heading NE, got %s", m.Heading)
		}
		if m.Size != sprite.Size3 {
			t.Fatalf("enter must load size 3, got %d", m.Size)
		}
	})
	t.Run("unhappy: escape closes without switching size or heading", func(t *testing.T) {
		m := newEd(t)
		m.Size = sprite.Size4
		m.Heading = sprite.N
		before := m.Current()
		m = send(m, keyCtrl('p'))
		m = send(m, key('l'))
		m = send(m, key('j'))
		m = send(m, keyType(tea.KeyEsc))
		if m.ShipPickerOpen {
			t.Fatal("esc must close the picker")
		}
		if m.Size != sprite.Size4 || m.Heading != sprite.N {
			t.Fatalf("esc must keep size 4 heading N, got %d %s", m.Size, m.Heading)
		}
		if sprite.Render(m.Current()) != sprite.Render(before) {
			t.Fatal("esc must not paint or replace the canvas")
		}
	})
	t.Run("unhappy: enter on a missing frame stays open", func(t *testing.T) {
		path := writeMiniShip(t, t.TempDir(), "only-n", 'Q')
		m, err := Open(path)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		m = send(m, keyCtrl('p'))
		m = send(m, key('l'))
		m.TermW, m.TermH = 80, 24
		_ = m.View().Content
		m = send(m, keyType(tea.KeyEnter))
		if !m.ShipPickerOpen {
			t.Fatal("enter on a missing frame must stay open, not load a ghost ship")
		}
		if m.Size != sprite.Size1 || m.Heading != sprite.N {
			t.Fatalf("must keep the only real frame, got size %d heading %s", m.Size, m.Heading)
		}
	})
}

func TestShipPickerDoesNotFightOtherModals(t *testing.T) {
	t.Run("unhappy: ctrl-p while the color picker is open does not switch ships", func(t *testing.T) {
		m := newEd(t)
		beforeSize, beforeHead := m.Size, m.Heading
		m = send(m, key('c'))
		if !m.PickerOpen {
			t.Fatal("c must open the color picker")
		}
		beforeCanvas := m.Current()
		m = send(m, keyCtrl('p'))
		if m.ShipPickerOpen {
			t.Fatal("ctrl-p must not steal an already-open color picker")
		}
		if m.Size != beforeSize || m.Heading != beforeHead {
			t.Fatal("ctrl-p in another modal must not switch size or heading")
		}
		if sprite.Render(m.Current()) != sprite.Render(beforeCanvas) {
			t.Fatal("ctrl-p in another modal must not paint the canvas")
		}
	})
}
