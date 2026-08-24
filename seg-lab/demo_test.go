package main

// Demo harness tests, written first. The lab is a Go font viewer: pass a
// string, tab toggles small / large writing. No Python, no TTF.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/seg-lab/font"
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
	t.Run("happy: boots HELLO WORLD in large writing", func(t *testing.T) {
		m := newDemo()
		if m.size != font.Large {
			t.Fatalf("boot size %s, want large", m.size)
		}
		if m.text != "HELLO WORLD" {
			t.Fatalf("boot text %q", m.text)
		}
		v := m.View()
		if !strings.Contains(v, "large") {
			t.Fatal("the UI must name the size")
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

func TestViewerSize(t *testing.T) {
	t.Run("happy: tab flips large ↔ small", func(t *testing.T) {
		m := newDemo()
		if m.size != font.Large {
			t.Fatal("boot large")
		}
		m = keyType(m, tea.KeyTab)
		if m.size != font.Small {
			t.Fatalf("tab → %s, want small", m.size)
		}
		m = keyType(m, tea.KeyTab)
		if m.size != font.Large {
			t.Fatalf("tab again → %s, want large", m.size)
		}
	})
	t.Run("unhappy: unknown keys do not wipe the text", func(t *testing.T) {
		m := newDemo()
		before := m.text
		m = keyType(m, tea.KeyUp)
		if m.text != before {
			t.Fatal("arrow keys must not clear the message")
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
