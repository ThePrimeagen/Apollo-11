package ui

// t31 — fake typing is anchored in AGC time, not wall frames: a human types
// ~230-330ms apart in REAL (AGC) time, so wall spacing must scale with the
// playback speed — slower playback types slower, faster playback faster.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/sim"
)

// keyruptGapsAGC returns the AGC-ms gaps between consecutive KEYRUPT events.
func keyruptGapsAGC(e *sim.Engine) []float64 {
	var times []float64
	for _, ev := range e.Events() {
		if ev.Kind == sim.EvKey {
			times = append(times, ev.AGCTimeMs)
		}
	}
	var gaps []float64
	for i := 1; i < len(times); i++ {
		gaps = append(gaps, times[i]-times[i-1])
	}
	return gaps
}

func assertHumanGaps(t *testing.T, gaps []float64) {
	t.Helper()
	for i, g := range gaps {
		if g < 180 || g > 400 {
			t.Fatalf("keystroke gap %d = %.1fms AGC, want human 180-400ms", i, g)
		}
	}
}

func TestTypingScalesWithPlayback(t *testing.T) {
	t.Run("happy: keystrokes land ~230-330ms of AGC time apart", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, 'n')
		m = tick(m, 1400) // default 20x slow-mo: ~2.3s AGC
		if m.PendingKeys() != 0 {
			t.Fatalf("typing should finish, %d keys pending", m.PendingKeys())
		}
		gaps := keyruptGapsAGC(e)
		if len(gaps) != 6 {
			t.Fatalf("want 6 gaps between 7 keystrokes, got %d", len(gaps))
		}
		assertHumanGaps(t, gaps)
		if !e.MonitorActive() {
			t.Fatal("V16N68 ENTR should start the monitor")
		}
	})
	t.Run("happy: fast playback types fast in wall terms, same AGC cadence", func(t *testing.T) {
		e, m := newTestModel()
		e.SetWallToAGC(0.4) // 8x the default speed
		m = key(m, 'n')
		m = tick(m, 200) // ~2.7s AGC — plenty
		if m.PendingKeys() != 0 {
			t.Fatalf("at 8x speed typing should finish in <200 frames, %d pending", m.PendingKeys())
		}
		assertHumanGaps(t, keyruptGapsAGC(e))
	})
	t.Run("happy: slow playback has not finished typing after the same frames", func(t *testing.T) {
		_, m := newTestModel() // default 0.05: 200 frames = ~333ms AGC
		m = key(m, 'n')
		m = tick(m, 200)
		if m.PendingKeys() == 0 {
			t.Fatal("at 20x slow-mo typing must still be in progress after 200 frames")
		}
	})
	t.Run("unhappy: pause stops the typing hand", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, 'n')
		m = key(m, '.') // pause
		m = tick(m, 300)
		if m.PendingKeys() != 7 {
			t.Fatalf("paused: all 7 keys should still be pending, got %d", m.PendingKeys())
		}
		if gaps := keyruptGapsAGC(e); len(gaps) != 0 {
			t.Fatal("paused: no KEYRUPT should fire")
		}
	})
	t.Run("unhappy: speed change mid-typing keeps the AGC cadence", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, 'n')
		m = tick(m, 300) // ~500ms AGC at default speed: a key or two
		m = key(m, ']')
		m = key(m, ']')
		m = key(m, ']') // now 0.4 AGC ms per wall ms
		m = tick(m, 200)
		if m.PendingKeys() != 0 {
			t.Fatalf("typing should finish after speeding up, %d pending", m.PendingKeys())
		}
		gaps := keyruptGapsAGC(e)
		if len(gaps) != 6 {
			t.Fatalf("want 6 gaps, got %d", len(gaps))
		}
		assertHumanGaps(t, gaps)
	})
}
