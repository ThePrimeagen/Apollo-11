package main

// Demo harness tests, written first: cmd/astronaut plays the moonwalk
// loop — an original NES-styled astronaut built on the classic 16×16
// side-scroller envelope. One cycle: he runs in from the left on the
// three-frame stride, jumps a joy hop, runs on, leaps onto the flag
// pole, slides down it on the two alternating grips, and stands at the
// base before the loop restarts. The timeline is a pure function of
// time, so every phase is checkable without a terminal.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/astro"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	tw = 72
	th = 26
)

type sample struct {
	t    float64
	pose sprite.Heading
	x, y int
}

func sweep(w, h int) []sample {
	var out []sample
	cyc := cycleSeconds(w, h)
	for t := 0.0; t < cyc; t += 1.0 / 60 {
		pose, x, y := timelineAt(w, h, t)
		out = append(out, sample{t, pose, x, y})
	}
	return out
}

func isRun(p sprite.Heading) bool {
	return p == astro.PoseRun1 || p == astro.PoseRun2 || p == astro.PoseRun3
}

func isPole(p sprite.Heading) bool {
	return p == astro.PosePole1 || p == astro.PosePole2
}

func frames(m model, n int) model {
	for i := 0; i < n; i++ {
		mm, _ := m.Update(frameMsg{})
		m = mm.(model)
	}
	return m
}

func press(m model, msg tea.Msg) model {
	mm, _ := m.Update(msg)
	return mm.(model)
}

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func newTestModel(t *testing.T, seconds float64) model {
	t.Helper()
	m, err := newModel(seconds)
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	mm, _ := m.Update(tea.WindowSizeMsg{Width: tw, Height: th})
	return mm.(model)
}

func TestTimeline(t *testing.T) {
	grounded := groundedY(th - statusRows)
	t.Run("happy: the cycle opens running on the ground, striding right", func(t *testing.T) {
		samples := sweep(tw, th-statusRows)
		first := samples[0]
		if !isRun(first.pose) {
			t.Fatalf("the cycle must open running, got %q", first.pose)
		}
		if first.y != grounded {
			t.Fatalf("the opening run must be on the ground: y %d, want %d", first.y, grounded)
		}
		poses := map[sprite.Heading]bool{}
		lastX := first.x
		for _, s := range samples {
			if isPole(s.pose) {
				break
			}
			if s.x < lastX {
				t.Fatalf("t=%.2f: x went backward (%d after %d) — he only runs left to right", s.t, s.x, lastX)
			}
			lastX = s.x
			if isRun(s.pose) {
				poses[s.pose] = true
				if s.y != grounded {
					t.Fatalf("t=%.2f: running but off the ground (y %d)", s.t, s.y)
				}
			}
		}
		for _, want := range []sprite.Heading{astro.PoseRun1, astro.PoseRun2, astro.PoseRun3} {
			if !poses[want] {
				t.Fatalf("the run never showed stride %q — the three-frame cycle is broken", want)
			}
		}
	})
	t.Run("happy: the jump leaves the ground and comes back down", func(t *testing.T) {
		samples := sweep(tw, th-statusRows)
		minY, found, landed := grounded, false, false
		for _, s := range samples {
			if s.pose == astro.PoseJump {
				found = true
				if s.y < minY {
					minY = s.y
				}
			}
			if found && !isPole(s.pose) && s.pose != astro.PoseJump && s.y == grounded {
				landed = true
			}
		}
		if !found {
			t.Fatal("the cycle never jumps")
		}
		if minY >= grounded {
			t.Fatalf("the jump never left the ground: min y %d, ground %d", minY, grounded)
		}
		if !landed {
			t.Fatal("what goes up must land back on the stride")
		}
	})
	t.Run("happy: the pole slide grips the flagpole column and rides it down", func(t *testing.T) {
		samples := sweep(tw, th-statusRows)
		var pole []sample
		for _, s := range samples {
			if isPole(s.pose) {
				pole = append(pole, s)
			}
		}
		if len(pole) == 0 {
			t.Fatal("the cycle never reaches the flagpole")
		}
		wantX := poleCol(tw) - astro.GripCol
		grips := map[sprite.Heading]bool{}
		lastY := pole[0].y
		for _, s := range pole {
			if s.x != wantX {
				t.Fatalf("t=%.2f: sliding at x %d, want %d — the hands must stay on the pole", s.t, s.x, wantX)
			}
			if s.y < lastY {
				t.Fatalf("t=%.2f: the slide went up (y %d after %d)", s.t, s.y, lastY)
			}
			lastY = s.y
			grips[s.pose] = true
		}
		if !grips[astro.PosePole1] || !grips[astro.PosePole2] {
			t.Fatal("the slide must alternate both grips or the shimmy freezes")
		}
		if pole[len(pole)-1].y != grounded {
			t.Fatalf("the slide must end on the ground: y %d, want %d", pole[len(pole)-1].y, grounded)
		}
	})
	t.Run("happy: he stands at the base, then the loop starts over", func(t *testing.T) {
		samples := sweep(tw, th-statusRows)
		last := samples[len(samples)-1]
		if last.pose != astro.PoseStand {
			t.Fatalf("the cycle must end standing, got %q", last.pose)
		}
		if last.y != grounded {
			t.Fatalf("standing off the ground: y %d, want %d", last.y, grounded)
		}
		cyc := cycleSeconds(tw, th-statusRows)
		p0, x0, y0 := timelineAt(tw, th-statusRows, 0.4)
		p1, x1, y1 := timelineAt(tw, th-statusRows, 0.4+cyc)
		if p0 != p1 || x0 != x1 || y0 != y1 {
			t.Fatal("one full cycle later the timeline must repeat exactly")
		}
	})
	t.Run("unhappy: time before the curtain clamps to the opening run", func(t *testing.T) {
		pose, _, y := timelineAt(tw, th-statusRows, -5)
		if !isRun(pose) || y != grounded {
			t.Fatalf("t<0 must clamp to the opening stride, got %q y %d", pose, y)
		}
	})
	t.Run("unhappy: a stage too small for the route still answers", func(t *testing.T) {
		pose, _, _ := timelineAt(3, 2, 1.0)
		if pose == "" {
			t.Fatal("a tiny stage must still name a pose")
		}
	})
}

func TestAstronautDemo(t *testing.T) {
	t.Run("happy: the opening view shows the scene and fills the window", func(t *testing.T) {
		m := newTestModel(t, 0)
		v := m.View().Content
		if got := len(strings.Split(v, "\n")); got != th {
			t.Fatalf("view has %d lines for a %d-line window", got, th)
		}
		if !strings.Contains(v, "astronaut") {
			t.Fatal("the status line must name the show")
		}
		if !strings.Contains(v, "│") {
			t.Fatal("the flagpole must be on stage from the first frame")
		}
	})
	t.Run("happy: the run animates — ticks change the frame on screen", func(t *testing.T) {
		m := newTestModel(t, 0)
		m = frames(m, 30)
		before := m.View().Content
		m = frames(m, 6)
		if m.View().Content == before {
			t.Fatal("six ticks at 30fps must advance the stride")
		}
	})
	t.Run("happy: space replays from the top", func(t *testing.T) {
		m := newTestModel(t, 0)
		m = frames(m, 90)
		if m.clock == 0 {
			t.Fatal("ninety frames must move the clock")
		}
		m = press(m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
		if m.clock != 0 {
			t.Fatalf("space must rewind the clock, got %v", m.clock)
		}
	})
	t.Run("happy: each frame schedules the next; Init starts the clock", func(t *testing.T) {
		m := newTestModel(t, 0)
		if m.Init() == nil {
			t.Fatal("Init must start the ticker")
		}
		_, cmd := m.Update(frameMsg{})
		if cmd == nil {
			t.Fatal("a frame must schedule the next tick")
		}
	})
	t.Run("happy: -seconds brings the curtain down on time", func(t *testing.T) {
		m := newTestModel(t, 0.05)
		mm, cmd := m.Update(frameMsg{})
		m = mm.(model)
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Fatal("one frame is 0.033s — too early for a 0.05s curtain")
		}
		_, cmd = m.Update(frameMsg{})
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("two frames pass 0.05s — the curtain must fall")
		}
	})
	t.Run("unhappy: q and ctrl+c quit from any point", func(t *testing.T) {
		for _, msg := range []tea.Msg{
			runeKey('q'),
			tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
		} {
			m := newTestModel(t, 0)
			m = frames(m, 10)
			_, cmd := m.Update(msg)
			if cmd == nil {
				t.Fatalf("%v must quit", msg)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("%v must issue tea.Quit", msg)
			}
		}
	})
	t.Run("unhappy: a tiny window never panics and still fills its height", func(t *testing.T) {
		m := newTestModel(t, 0)
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 5, Height: 3})
		m = mm.(model)
		m = frames(m, 60)
		if got := len(strings.Split(m.View().Content, "\n")); got != 3 {
			t.Fatalf("view has %d lines for a 3-line window", got)
		}
	})
}
