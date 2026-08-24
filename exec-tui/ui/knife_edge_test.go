package ui

// t33 — the knife edge must be visible without any label text: the free
// number on the top line is pinned near zero (the flight's quiet ~5 minutes
// with the theft active), while a healthy descent shows a comfortable
// double-digit margin.

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var freeRE = regexp.MustCompile(`FREE COMPUTE +([+-] ?\d+\.\d)%`)

func freePct(t *testing.T, v string) float64 {
	t.Helper()
	m := freeRE.FindStringSubmatch(stripAnsi(strings.Split(v, "\n")[0]))
	if m == nil {
		t.Fatalf("free line not found in %q", stripAnsi(strings.Split(v, "\n")[0]))
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(m[1], " ", ""), 64)
	if err != nil {
		t.Fatalf("bad free value %q", m[1])
	}
	return f
}

func TestKnifeEdgeIndicator(t *testing.T) {
	t.Run("happy: the knife edge reads as a near-zero free number", func(t *testing.T) {
		e, m := newTestModel()
		e.AdvanceAGC(170)
		e.StartDescent()
		e.AcquireLandingRadar()
		e.SetRadarBug(true)
		e.AdvanceAGC(10000)
		if got := freePct(t, m.View()); got < -1 || got > 2 {
			t.Fatalf("knife edge must pin the free number near zero, got %v", got)
		}
	})
	t.Run("unhappy: healthy descent shows a comfortable margin, no edge", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(10000)
		if got := freePct(t, m.View()); got < 10 {
			t.Fatalf("healthy descent must show a double-digit margin, got %v", got)
		}
		if strings.Contains(m.View(), "knife edge") {
			t.Fatal("no knife-edge text may ever render")
		}
	})
}
