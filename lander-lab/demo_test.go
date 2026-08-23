package main

// Demo harness tests, written first: the descent now plays CONTINUOUSLY —
// the LM stays upright (no rotation) and lowers smoothly as mission time
// advances, with ALT/VEL interpolated between the flight events. The
// playback rate is adjustable ([ slower, ] faster), '.' pauses, alarms
// appear as time passes their moments, and the flight ends landed.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/lander-lab/lander"
)

func key(m demoModel, r rune) demoModel {
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	return mm.(demoModel)
}

func TestContinuousDescent(t *testing.T) {
	t.Run("happy: boots at PDI and lowers smoothly with time", func(t *testing.T) {
		m := newDemoModel()
		s := m.state()
		if s.AltFt != 49971 || !strings.Contains(s.Phase, "P63") {
			t.Fatalf("boot must be PDI, got %v ft %q", s.AltFt, s.Phase)
		}
		prev := s.AltFt
		for i := 0; i < 20; i++ {
			m.advance(1000) // one wall second at default 20x = 20 mission sec
			alt := m.state().AltFt
			if alt > prev {
				t.Fatalf("altitude must never climb, %v -> %v", prev, alt)
			}
			prev = alt
		}
		if prev >= 49971 {
			t.Fatal("twenty wall-seconds must visibly lower the craft")
		}
	})
	t.Run("happy: meters interpolate between events, not jump", func(t *testing.T) {
		m := newDemoModel()
		m.advance(129 * 1000 / 20) // mission t = 129s, between yaw(232? no: between 26 and 232)
		s := m.state()
		if !(s.AltFt < 48000 && s.AltFt > 42426) {
			t.Fatalf("at t=129s altitude must sit between the neighboring events, got %v", s.AltFt)
		}
		if !(s.VelFps < 5460 && s.VelFps > 3366) {
			t.Fatalf("velocity must interpolate too, got %v", s.VelFps)
		}
	})
	t.Run("happy: zero rotation — one upright silhouette the whole flight", func(t *testing.T) {
		m := newDemoModel()
		for i := 0; i < 36; i++ { // sample the entire flight second by second
			m.advance(1000)
			if s := m.state(); s.AltFt > 0 && s.Attitude != lander.Vertical {
				t.Fatalf("at t=%.0fs the craft must be upright, got %v", s.TimeSec, s.Attitude)
			}
		}
		m.advance(30 * 1000)
		s := m.state()
		if s.Attitude != lander.Landed || s.AltFt != 0 {
			t.Fatalf("the flight must end landed at 0 ft, got %v at %v ft", s.Attitude, s.AltFt)
		}
	})
	t.Run("happy: the state carries a live countdown to touchdown", func(t *testing.T) {
		m := newDemoModel()
		if got := m.state().LandInSec; got != 757 {
			t.Fatalf("the countdown starts at 757s, got %v", got)
		}
		m.advance(1000) // 20 mission seconds
		if got := m.state().LandInSec; got != 737 {
			t.Fatalf("the countdown must tick with mission time, got %v", got)
		}
		m.advance(120 * 1000)
		if got := m.state().LandInSec; got != 0 {
			t.Fatalf("the countdown must clamp at zero, got %v", got)
		}
	})
	t.Run("happy: the demo ticks the plume animation frame counter", func(t *testing.T) {
		m := newDemoModel()
		t0 := m.state().Tick
		mm, _ := m.Update(frameMsg{})
		m = mm.(demoModel)
		if m.state().Tick == t0 {
			t.Fatal("each frame must advance the animation tick")
		}
	})
	t.Run("happy: alarms appear as their moments pass, all five by the end", func(t *testing.T) {
		m := newDemoModel()
		m.advance(320 * 1000 / 20) // just past the first 1202 at t=316
		if got := len(m.state().Alarms); got != 1 {
			t.Fatalf("one alarm after t=316s, got %d", got)
		}
		m.advance(60 * 1000)
		if got := len(m.state().Alarms); got != 5 {
			t.Fatalf("all five alarms by touchdown, got %d", got)
		}
	})
}

func TestPlaybackControls(t *testing.T) {
	t.Run("happy: ] doubles the rate, [ halves it", func(t *testing.T) {
		m := newDemoModel()
		s0 := m.scale
		m = key(m, ']')
		if m.scale != s0*2 {
			t.Fatalf("] must double the rate, %v -> %v", s0, m.scale)
		}
		m = key(m, '[')
		m = key(m, '[')
		if m.scale != s0/2 {
			t.Fatalf("[ must halve the rate, got %v", m.scale)
		}
	})
	t.Run("happy: '.' pauses and resumes the fall", func(t *testing.T) {
		m := newDemoModel()
		m = key(m, '.')
		before := m.state().AltFt
		m.advance(5000)
		if m.state().AltFt != before {
			t.Fatal("a paused descent must hold its altitude")
		}
		m = key(m, '.')
		m.advance(5000)
		if m.state().AltFt >= before {
			t.Fatal("resuming must let the craft fall again")
		}
	})
	t.Run("happy: r toggles true realtime (1x) and back", func(t *testing.T) {
		m := newDemoModel()
		m = key(m, 'r')
		if m.scale != 1 {
			t.Fatalf("r must drop to 1x realtime, got %v", m.scale)
		}
		m = key(m, 'r')
		if m.scale != defaultScale {
			t.Fatalf("r again must restore the default rate, got %v", m.scale)
		}
	})
	t.Run("unhappy: rate clamps at sane bounds", func(t *testing.T) {
		m := newDemoModel()
		for i := 0; i < 12; i++ {
			m = key(m, ']')
		}
		if m.scale > 320 {
			t.Fatalf("rate must clamp high, got %v", m.scale)
		}
		for i := 0; i < 20; i++ {
			m = key(m, '[')
		}
		if m.scale < 1 {
			t.Fatalf("rate must clamp low, got %v", m.scale)
		}
	})
	t.Run("happy: q quits", func(t *testing.T) {
		m := newDemoModel()
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		if cmd == nil {
			t.Fatal("q must quit")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("q's command must be tea.Quit")
		}
	})
}

func TestScriptStillTruthful(t *testing.T) {
	t.Run("happy: five alarms in flight order, ending on the surface", func(t *testing.T) {
		codes := []string{}
		for _, ev := range script {
			if ev.alarm != "" {
				codes = append(codes, ev.alarm)
			}
		}
		if strings.Join(codes, ",") != "1202,1202,1201,1202,1202" {
			t.Fatalf("alarm sequence wrong: %v", codes)
		}
		if last := script[len(script)-1]; last.altFt != 0 {
			t.Fatalf("the script must end at 0 ft, got %v", last.altFt)
		}
	})
}
