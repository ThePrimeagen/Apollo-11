package ui

// The DANZIG card sits in the center gap and shows how the Executive
// picks a job: packed PRIORITY words, SETLOC/DANZIG preempt, EJSCAN
// rescan. Rose Pine highlighting comes from the danzig component.
// 'c' pins or hides it. During a flight the lander keeps the gap
// unless the card is pinned.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/danzig"
	"github.com/theprimeagen/apollo-11/exec-tui/sim"
)

func TestDanzigCardInGap(t *testing.T) {
	t.Run("happy: a wide idle view shows FINDVAC and the packed 20455 word", func(t *testing.T) {
		_, m := newWideTestModel()
		v := stripAnsi(m.View().Content)
		for _, want := range []string{"HOW THE EXEC PICKS A JOB", "HASNEWJOB", "EJSCAN", "SETLOC", "20455"} {
			if !strings.Contains(v, want) {
				t.Fatalf("idle view missing %q:\n%s", want, v)
			}
		}
		if !strings.Contains(m.View().Content, "38;2;246;193;119") {
			t.Fatal("the card must keep Rose Pine gold on the packed numbers")
		}
	})
	t.Run("unhappy: a narrow terminal omits the card instead of wrapping", func(t *testing.T) {
		e := sim.New()
		m := NewModel(e)
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		v := stripAnsi(mm.(Model).View().Content)
		if strings.Contains(v, "HOW THE EXEC PICKS A JOB") {
			t.Fatal("narrow terminals must skip the card")
		}
		if !strings.Contains(v, "SERVICER") {
			t.Fatal("the rest of the executive view must still render")
		}
	})
}

func TestDanzigToggle(t *testing.T) {
	t.Run("happy: c hides the card, c brings it back", func(t *testing.T) {
		_, m := newWideTestModel()
		if !strings.Contains(stripAnsi(m.View().Content), "DANZIG") {
			t.Fatal("precondition: the card is on at boot")
		}
		m = key(m, 'c')
		if strings.Contains(stripAnsi(m.View().Content), "HOW THE EXEC PICKS A JOB") {
			t.Fatal("c must hide the card")
		}
		m = key(m, 'c')
		if !strings.Contains(stripAnsi(m.View().Content), "DANZIG") {
			t.Fatal("c again must show the card")
		}
	})
	t.Run("unhappy: c in typing mode is a DSKY key, not a toggle", func(t *testing.T) {
		_, m := newWideTestModel()
		m = key(m, 't')
		if !m.TypingMode() {
			t.Fatal("precondition: typing mode")
		}
		m = key(m, 'c')
		if !strings.Contains(stripAnsi(m.View().Content), "HOW THE EXEC PICKS A JOB") {
			t.Fatal("typing c must not hide the scheduler card")
		}
	})
}

func TestDanzigYieldsToLander(t *testing.T) {
	t.Run("happy: a flight takes the gap; pinning c keeps the card", func(t *testing.T) {
		e, m := newWideTestModel()
		m = key(m, 'f')
		m = flyTo(t, e, m, 3)
		v := stripAnsi(m.View().Content)
		if strings.Contains(v, "HOW THE EXEC PICKS A JOB") {
			t.Fatal("the lander must own the gap during a flight")
		}
		if !strings.Contains(m.View().Content, "ft/s") {
			t.Fatal("precondition: the lander is up")
		}
		m = key(m, 'c')
		v = stripAnsi(m.View().Content)
		if !strings.Contains(v, "DANZIG") {
			t.Fatal("pinning c during a flight must show the card")
		}
		if strings.Contains(m.View().Content, "ft/s") {
			t.Fatal("the pinned card replaces the lander")
		}
	})
	t.Run("unhappy: hiding the card after pin returns the lander", func(t *testing.T) {
		e, m := newWideTestModel()
		m = key(m, 'f')
		m = flyTo(t, e, m, 3)
		m = key(m, 'c')
		m = key(m, 'c')
		if !strings.Contains(m.View().Content, "ft/s") {
			t.Fatal("unpinning must give the gap back to the lander")
		}
		if strings.Contains(stripAnsi(m.View().Content), "HOW THE EXEC PICKS A JOB") {
			t.Fatal("unpinned flight must not keep the card")
		}
	})
}

func TestDanzigFootprintFitsTheGap(t *testing.T) {
	t.Run("happy: the card is narrower than a wide-terminal gap", func(t *testing.T) {
		// 170-wide: left ~64, right 25, gap ~80. Card must fit.
		if danzig.CardWidth() > 70 {
			t.Fatalf("card width %d is too wide for the center gap", danzig.CardWidth())
		}
	})
	t.Run("unhappy: compact height still holds with the card beside the rows", func(t *testing.T) {
		_, m := newWideTestModel()
		if got := len(strings.Split(m.View().Content, "\n")); got > 33 {
			t.Fatalf("idle+card must still fit in 33 lines, got %d", got)
		}
	})
}
