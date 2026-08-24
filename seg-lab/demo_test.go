package main

// Demo harness tests, written first. The lab is a standalone segmented
// letter viewer: type a string, tab cycles unicode / 7-seg / 14-seg, esc
// clears, ctrl-c quits. Empty text shows the A–Z catalog.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/seg-lab/seg"
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
	t.Run("happy: boots on alpha showing HELLO WORLD", func(t *testing.T) {
		m := newDemo()
		if m.style != seg.StyleAlpha {
			t.Fatalf("boot style %s, want alpha", m.style)
		}
		if m.text != "HELLO WORLD" {
			t.Fatalf("boot text %q", m.text)
		}
		v := m.View()
		if !strings.Contains(v, "alpha") {
			t.Fatal("the UI must name the active style")
		}
		if !strings.Contains(v, "U+1FBF0") && !strings.Contains(v, "1FBF0") {
			t.Fatal("the UI must mention the Unicode segmented-digit range")
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

func TestViewerStyles(t *testing.T) {
	t.Run("happy: tab walks alpha → unicode → 7-seg → 14-seg", func(t *testing.T) {
		m := newDemo()
		seen := []seg.Style{m.style}
		for i := 0; i < 4; i++ {
			m = keyType(m, tea.KeyTab)
			seen = append(seen, m.style)
		}
		want := []seg.Style{seg.StyleAlpha, seg.StyleUnicode, seg.StyleSeven, seg.StyleFourteen, seg.StyleAlpha}
		if len(seen) != len(want) {
			t.Fatalf("tab cycle %v", seen)
		}
		for i := range want {
			if seen[i] != want[i] {
				t.Fatalf("tab cycle %v, want %v", seen, want)
			}
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
		v := m.View()
		if !strings.Contains(v, "A") || !strings.Contains(strings.ToUpper(v), "CATALOG") && !strings.Contains(v, "ABCDEF") {
			// catalog may be segmented; the caption should still say so
			if !strings.Contains(strings.ToLower(v), "catalog") {
				t.Fatalf("empty viewer must announce the alphabet catalog:\n%s", v)
			}
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
