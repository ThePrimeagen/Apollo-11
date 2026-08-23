package ui

// t41 — "what's on right now" must be readable from the key bar itself:
// every latching control shows a ✓ while its state is active. (The header
// badges give the same information, but the bar is where the eyes are when
// choosing the next key.)

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
	t.Run("happy: each latched control gains a ✓ when switched on", func(t *testing.T) {
		_, m := newTestModel()
		v := m.View()
		for _, k := range []string{"d", "l", "n", "r", "6", "a"} {
			if activeMarked(v, k) {
				t.Fatalf("[%s] must not be marked active before it is on", k)
			}
		}
		m = key(m, 'd')
		if v := m.View(); !activeMarked(v, "d") {
			t.Fatal("[d] descent must show ✓ once the descent is running")
		}
		m = key(m, 'l')
		if v := m.View(); !activeMarked(v, "l") {
			t.Fatal("[l] radar lock must show ✓ once locked")
		}
		m = key(m, 'r')
		if v := m.View(); !activeMarked(v, "r") {
			t.Fatal("[r] RR bug must show ✓ while stealing")
		}
		m = key(m, '6')
		v = m.View()
		if !activeMarked(v, "6") {
			t.Fatal("[6] P64 must show ✓ in P64")
		}
		if !activeMarked(v, "d") {
			t.Fatal("[d] descent stays ✓ in P64 — the descent is still running")
		}
		m = key(m, 'a')
		if v := m.View(); !activeMarked(v, "a") {
			t.Fatal("[a] att-hold must show ✓ in P66")
		}
	})
	t.Run("happy: n shows ✓ while the V16N68 monitor refreshes", func(t *testing.T) {
		e, m := newTestModel()
		for _, k := range []byte("V16N68E") {
			e.PressKey(k)
		}
		if v := m.View(); !activeMarked(v, "n") {
			t.Fatal("[n] must show ✓ while the monitor is active")
		}
	})
	t.Run("unhappy: r toggled off and x reset clear the marks", func(t *testing.T) {
		_, m := newTestModel()
		m = key(m, 'r')
		m = key(m, 'r')
		if v := m.View(); activeMarked(v, "r") {
			t.Fatal("[r] must lose its ✓ when the bug is switched off")
		}
		m = key(m, 'd')
		m = key(m, 'l')
		m = key(m, 'r')
		m = key(m, 'x')
		v := m.View()
		for _, k := range []string{"d", "l", "n", "r", "6", "a"} {
			if activeMarked(v, k) {
				t.Fatalf("[%s] must be cleared by reset", k)
			}
		}
	})
}
