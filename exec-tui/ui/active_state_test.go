package ui

// t41 — "what's on right now" must always be readable: the three toggle
// cards carry their own ● ON state, and the remaining latched key-bar
// controls (P64, ATT HOLD) show a ✓ while active. Reset clears everything.

import (
	"strings"
	"testing"
)

func barLineOf(v, key string) string {
	for _, line := range strings.Split(v, "\n") {
		if strings.Contains(line, "["+key+"]") {
			return line
		}
	}
	return ""
}

func activeMarked(v, key string) bool {
	line := barLineOf(v, key)
	if line == "" {
		return false
	}
	// the ✓ must follow this key's bracket before the next hint's bracket
	after := line[strings.Index(line, "["+key+"]"):]
	if i := strings.Index(after[1:], "["); i >= 0 {
		after = after[:i+1]
	}
	return strings.Contains(after, "✓")
}

func TestKeyBarActiveStates(t *testing.T) {
	t.Run("happy: P64 and ATT HOLD gain a ✓ when entered", func(t *testing.T) {
		_, m := newTestModel()
		v := m.View()
		for _, k := range []string{"6", "a"} {
			if activeMarked(v, k) {
				t.Fatalf("[%s] must not be marked active before it is on", k)
			}
		}
		m = key(m, 'd')
		m = key(m, '6')
		if v := m.View(); !activeMarked(v, "6") {
			t.Fatal("[6] P64 must show ✓ in P64")
		}
		m = key(m, 'a')
		if v := m.View(); !activeMarked(v, "a") {
			t.Fatal("[a] att-hold must show ✓ in P66")
		}
	})
	t.Run("unhappy: reset clears the marks and the card states", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, 'd')
		m = key(m, 'r')
		m = key(m, '6')
		m = key(m, 'x')
		v := m.View()
		for _, k := range []string{"6", "a"} {
			if activeMarked(v, k) {
				t.Fatalf("[%s] must be cleared by reset", k)
			}
		}
		if strings.Contains(v, "● ON") {
			t.Fatal("reset must clear every card's ON state")
		}
		_ = e
	})
}
