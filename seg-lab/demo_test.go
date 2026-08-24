package main

// Demo harness tests, written first. The lab is a Go font viewer: pass a
// string and a height unit 1–5. Tab cycles the unit. Digits 1–5 set it.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func key(m demoModel, r rune) demoModel {
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	return mm.(demoModel)
}

func keyType(m demoModel, t tea.KeyType) demoModel {
	mm, _ := m.Update(tea.KeyMsg{Type: t})
	return mm.(demoModel)
}

func TestViewerBoot(t *testing.T) {
	t.Run("happy: boots HELLO WORLD at height 3", func(t *testing.T) {
		m := newDemo()
		if m.height != 3 {
			t.Fatalf("boot height %d, want 3", m.height)
		}
		if m.text != "HELLO WORLD" {
			t.Fatalf("boot text %q", m.text)
		}
		v := m.View()
		if !strings.Contains(v, "3") {
			t.Fatal("the UI must name the height unit")
		}
		if strings.Contains(v, "font.Small") || strings.Contains(v, "font.Large") {
			t.Fatal("Small/Large is gone; height is an int")
		}
		if strings.Contains(v, "genfont") || strings.Contains(v, ".py") || strings.Contains(v, "U+E000") {
			t.Fatal("the Python / PUA font path is gone")
		}
	})
	t.Run("unhappy: a lone q is a letter, not a quit", func(t *testing.T) {
		m := newDemo()
		m = keyType(m, tea.KeyEsc)
		m = key(m, 'q')
		if m.text != "Q" && m.text != "q" {
			t.Fatalf("q must type, got %q", m.text)
		}
	})
}

func TestViewerTyping(t *testing.T) {
	t.Run("happy: typed runes append and backspace deletes", func(t *testing.T) {
		m := newDemo()
		m = keyType(m, tea.KeyEsc)
		for _, r := range "HI" {
			m = key(m, r)
		}
		if m.text != "HI" {
			t.Fatalf("typed %q", m.text)
		}
		m = keyType(m, tea.KeyBackspace)
		if m.text != "H" {
			t.Fatalf("backspace left %q", m.text)
		}
	})
	t.Run("unhappy: backspace on empty text is a no-op", func(t *testing.T) {
		m := newDemo()
		m = keyType(m, tea.KeyEsc)
		m = keyType(m, tea.KeyBackspace)
		if m.text != "" {
			t.Fatalf("empty backspace must stay empty, got %q", m.text)
		}
	})
}

func TestViewerHeight1Plain(t *testing.T) {
	t.Run("happy: height 1 shows the message in the default font", func(t *testing.T) {
		m := newDemo()
		m = key(m, '1')
		if m.height != 1 {
			t.Fatalf("digit 1 → %d", m.height)
		}
		v := m.View()
		if !strings.Contains(v, "HELLO WORLD") {
			t.Fatal("height 1 must show the terminal's own letters")
		}
	})
	t.Run("unhappy: the viewer never asks for height 6", func(t *testing.T) {
		m := newDemo()
		m.height = 6
		v := m.View()
		if !strings.Contains(strings.ToLower(v), "height") {
			t.Fatal("an illegal height must still be named so the error is visible")
		}
		if strings.Contains(v, "█") {
			t.Fatal("height 6 must not draw bars")
		}
	})
}

func TestViewerHeight(t *testing.T) {
	t.Run("happy: tab walks 3→4→5→1 and skips 2", func(t *testing.T) {
		m := newDemo()
		if m.height != 3 {
			t.Fatal("boot 3")
		}
		m = keyType(m, tea.KeyTab)
		if m.height != 4 {
			t.Fatalf("tab → %d, want 4", m.height)
		}
		m = keyType(m, tea.KeyTab)
		if m.height != 5 {
			t.Fatalf("tab → %d, want 5", m.height)
		}
		m = keyType(m, tea.KeyTab)
		if m.height != 1 {
			t.Fatalf("tab wraps → %d, want 1", m.height)
		}
		m = keyType(m, tea.KeyTab)
		if m.height != 3 {
			t.Fatalf("tab from 1 skips 2 → %d, want 3", m.height)
		}
		m = key(m, '5')
		if m.height != 5 || m.text != "HELLO WORLD" {
			t.Fatalf("digit 5 must set height, not type; height=%d text=%q", m.height, m.text)
		}
	})
	t.Run("unhappy: digit 2 does not select an impossible height", func(t *testing.T) {
		m := newDemo()
		m = key(m, '2')
		if m.height == 2 {
			t.Fatal("height 2 is skipped; digit 2 must not select it")
		}
		if m.text != "HELLO WORLD" {
			t.Fatalf("digit 2 must not type, text=%q", m.text)
		}
	})
	t.Run("unhappy: unknown keys do not wipe the text or the height", func(t *testing.T) {
		m := newDemo()
		before := m.text
		h := m.height
		m = keyType(m, tea.KeyUp)
		if m.text != before || m.height != h {
			t.Fatal("arrow keys must not change the message or the height")
		}
	})
}

func TestViewerCatalog(t *testing.T) {
	t.Run("happy: clearing the text shows the A-Z catalog", func(t *testing.T) {
		m := newDemo()
		m = keyType(m, tea.KeyEsc)
		if m.text != "" {
			t.Fatal("esc must clear")
		}
		v := strings.ToLower(m.View())
		if !strings.Contains(v, "catalog") {
			t.Fatalf("empty viewer must announce the alphabet catalog:\n%s", m.View())
		}
	})
	t.Run("unhappy: catalog is not shown while a message is typed", func(t *testing.T) {
		m := newDemo()
		v := strings.ToLower(m.View())
		if strings.Contains(v, "catalog") {
			t.Fatal("a typed message must not show the catalog caption")
		}
	})
}
