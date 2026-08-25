package ui

// t48 — layout round two:
//   - core sets render as two STACKS of four (CS1–CS4 beside CS5–CS8),
//     VACs as one stack of five
//   - the help/key bar is gone (keys still work)
//   - the three switches live on the right side, under the DSKY

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func lineWith(v, needle string) string {
	for _, line := range strings.Split(v, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	return ""
}

func TestCoreSetsStacked(t *testing.T) {
	t.Run("happy: CS1 shares a row with CS5, not with CS2", func(t *testing.T) {
		_, m := newTestModel()
		l := lineWith(m.View().Content, "CS1")
		if l == "" {
			t.Fatal("CS1 missing")
		}
		if !strings.Contains(l, "CS5") {
			t.Fatalf("CS1 and CS5 must sit side by side (two stacks of four), got %q", stripAnsi(l))
		}
		if strings.Contains(l, "CS2") {
			t.Fatal("CS2 must be stacked below CS1, not beside it")
		}
	})
	t.Run("happy: all four stack rows pair up (CSn with CSn+4)", func(t *testing.T) {
		_, m := newTestModel()
		v := m.View().Content
		for i := 1; i <= 4; i++ {
			l := lineWith(v, "CS"+itoa(i))
			if !strings.Contains(l, "CS"+itoa(i+4)) {
				t.Fatalf("CS%d must sit beside CS%d", i, i+4)
			}
		}
	})
	t.Run("happy: VACs are one stack — VC1 alone on its row", func(t *testing.T) {
		_, m := newTestModel()
		v := m.View().Content
		l := lineWith(v, "VC1")
		if l == "" || strings.Contains(l, "VC2") {
			t.Fatalf("VC1 must sit alone in the VAC stack, got %q", stripAnsi(l))
		}
		if lineWith(v, "VC5") == "" {
			t.Fatal("all five VACs must render")
		}
	})
}

func TestNoHelpMenu(t *testing.T) {
	t.Run("happy: the key-bar hints are gone", func(t *testing.T) {
		_, m := newTestModel()
		v := m.View().Content
		for _, gone := range []string{"[h/l]", "[space]", "[.]", "[q] quit", "[x] reset"} {
			if strings.Contains(v, gone) {
				t.Fatalf("the help menu must be gone, found %q", gone)
			}
		}
	})
	t.Run("unhappy: the keys still work without the menu", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, 'l')
		if m.Selected() != 1 {
			t.Fatal("l must still move the selection")
		}
		mm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
		m = mm.(Model)
		if !e.MonitorActive() && m.PendingKeys() == 0 {
			t.Fatal("space must still flip the selected switch")
		}
		m = key(m, '.')
		if !m.Paused() {
			t.Fatal("'.' must still pause")
		}
	})
}

func TestSwitchesOnRight(t *testing.T) {
	t.Run("happy: the switch labels sit in the right half, under the DSKY", func(t *testing.T) {
		_, m := newTestModel()
		// DESCENT is unique to the switch panel; its line carries all three
		// labels side by side ("RR STEAL" alone would match the timeline row).
		l := stripAnsi(lineWith(m.View().Content, "DESCENT"))
		if l == "" {
			t.Fatal("switch label line missing")
		}
		for _, label := range []string{"DESCENT", "DELTAH", "RR STEAL"} {
			idx := strings.Index(l, label)
			if idx < 0 {
				t.Fatalf("switch label %q missing from the panel line %q", label, l)
			}
			if idx < 70 {
				t.Fatalf("%q must sit on the right side, found at col %d", label, idx)
			}
		}
	})
	t.Run("happy: switches render below the DSKY registers", func(t *testing.T) {
		_, m := newTestModel()
		lines := strings.Split(m.View().Content, "\n")
		verbRow, descentRow := -1, -1
		for i, l := range lines {
			p := stripAnsi(l)
			if strings.Contains(p, "VERB") && verbRow < 0 {
				verbRow = i
			}
			if strings.Contains(p, "DESCENT") && descentRow < 0 {
				descentRow = i
			}
		}
		if verbRow < 0 || descentRow < 0 {
			t.Fatal("VERB or DESCENT missing")
		}
		if descentRow <= verbRow {
			t.Fatal("the switches must sit below the DSKY")
		}
	})
}

func itoa(n int) string { return string(rune('0' + n)) }
