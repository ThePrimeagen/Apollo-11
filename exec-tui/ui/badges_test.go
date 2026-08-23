package ui

// t29 — armed loads must be visible at a glance: persistent header badges
// for landing-radar lock and the RR bug (the monitor already shows MON 1Hz).

import (
	"strings"
	"testing"
)

func TestHeaderBadges(t *testing.T) {
	t.Run("happy: landing-radar lock lights an LR LOCK badge", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, 'd')
		if strings.Contains(m.View(), "LR LOCK") {
			t.Fatal("LR LOCK badge must not show before the radar locks")
		}
		e.AcquireLandingRadar()
		if !strings.Contains(m.View(), "LR LOCK") {
			t.Fatal("LR LOCK badge must show once the radar reports data good")
		}
	})
	t.Run("happy: r lights an RR BUG badge, r again clears it", func(t *testing.T) {
		_, m := newTestModel()
		if strings.Contains(m.View(), "RR BUG") {
			t.Fatal("RR BUG badge must not show before the bug is on")
		}
		m = key(m, 'r')
		if !strings.Contains(m.View(), "RR BUG") {
			t.Fatal("RR BUG badge must show while the bug is stealing")
		}
		m = key(m, 'r')
		if strings.Contains(m.View(), "RR BUG") {
			t.Fatal("RR BUG badge must clear when the bug is toggled off")
		}
	})
	t.Run("unhappy: reset clears every badge", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, 'd')
		e.AcquireLandingRadar()
		m = key(m, 'r')
		m = key(m, 'x')
		v := m.View()
		if strings.Contains(v, "LR LOCK") || strings.Contains(v, "RR BUG") {
			t.Fatal("reset must clear the LR LOCK and RR BUG badges")
		}
	})
}
