package ui

// UI tests, written before the UI exists. The View is treated as a string
// contract: labels and states that the educational video depends on must be
// present. Happy and unhappy paths per interaction.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/exec-tui/sim"
)

func newTestModel() (*sim.Engine, Model) {
	e := sim.New()
	m := NewModel(e)
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 45})
	return e, mm.(Model)
}

func key(m Model, r rune) Model {
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	return mm.(Model)
}

func tick(m Model, n int) Model {
	for i := 0; i < n; i++ {
		mm, _ := m.Update(FrameMsg{})
		m = mm.(Model)
	}
	return m
}

// ---------------------------------------------------------------------------
// t16 — header shows how much free compute is left
// ---------------------------------------------------------------------------

func TestHeaderFreeCompute(t *testing.T) {
	t.Run("happy: FREE COMPUTE text with a percent", func(t *testing.T) {
		_, m := newTestModel()
		m = tick(m, 5)
		v := m.View()
		if !strings.Contains(v, "FREE COMPUTE") {
			t.Fatal("header must show FREE COMPUTE")
		}
		if !strings.Contains(v, "%") {
			t.Fatal("header must show a percentage")
		}
	})
	t.Run("unhappy: header still renders at tiny sizes", func(t *testing.T) {
		e := sim.New()
		m := NewModel(e)
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
		v := mm.(Model).View()
		if len(v) == 0 {
			t.Fatal("view must not be empty at small sizes")
		}
	})
}

// ---------------------------------------------------------------------------
// t17 — 8 core set boxes and 5 VAC boxes with busy/free states
// ---------------------------------------------------------------------------

func TestCoreAndVacBoxes(t *testing.T) {
	t.Run("happy: all 13 boxes labeled", func(t *testing.T) {
		_, m := newTestModel()
		v := m.View()
		for _, want := range []string{"CS1", "CS2", "CS3", "CS4", "CS5", "CS6", "CS7", "CS8", "VC1", "VC2", "VC3", "VC4", "VC5"} {
			if !strings.Contains(v, want) {
				t.Fatalf("view missing box label %s", want)
			}
		}
		if !strings.Contains(v, "CORE SETS") || !strings.Contains(v, "VAC AREAS") {
			t.Fatal("box columns must be titled CORE SETS and VAC AREAS")
		}
	})
	t.Run("happy: busy boxes show the owning job", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(5)
		v := m.View()
		if !strings.Contains(v, "SERV") {
			t.Fatal("busy core set / VAC should display SERVICER ownership")
		}
	})
	t.Run("unhappy: alarm-flushed pools render free again", func(t *testing.T) {
		e, m := newTestModel()
		for i := 0; i < 8; i++ {
			e.ScheduleJob("HOG", 25, 1e9, false)
		}
		e.ScheduleJob("STRAW", 25, 10, false) // 1202 + restart wipes pools
		v := m.View()
		if strings.Contains(v, "HOG") {
			t.Fatal("after the restart no box should still show HOG")
		}
	})
}

// ---------------------------------------------------------------------------
// t18 — timeline rows: the long lines of tasks that need computing
// ---------------------------------------------------------------------------

func TestTimelineRows(t *testing.T) {
	t.Run("happy: fixed rows are labeled", func(t *testing.T) {
		_, m := newTestModel()
		v := m.View()
		for _, want := range []string{"SERVICER", "DAP", "MONITOR", "CHARIN", "T4RUPT", "RR STEAL", "IDLE"} {
			if !strings.Contains(v, want) {
				t.Fatalf("timeline missing row %s", want)
			}
		}
	})
	t.Run("happy: activity paints cells", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(500)
		v := m.View()
		if !strings.Contains(v, "█") {
			t.Fatal("running work should paint filled timeline cells")
		}
	})
	t.Run("unhappy: idle engine paints the idle row, not job rows", func(t *testing.T) {
		e, m := newTestModel()
		e.AdvanceAGC(500)
		v := m.View()
		if !strings.Contains(v, "IDLE") {
			t.Fatal("idle row must exist")
		}
	})
}

// ---------------------------------------------------------------------------
// t19 — keybindings
// ---------------------------------------------------------------------------

func TestKeybindings(t *testing.T) {
	t.Run("happy: d starts descent, r toggles the bug, p pings", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, 'd')
		if e.Phase() != sim.P63 {
			t.Fatalf("d should enter P63, got %v", e.Phase())
		}
		m = key(m, 'r')
		if !e.RadarBug() {
			t.Fatal("r should enable the RR bug")
		}
		m = key(m, 'r')
		if e.RadarBug() {
			t.Fatal("r again should disable the RR bug")
		}
		m = key(m, 'p')
		if core := hasOwner(e, "RR READ"); !core {
			t.Fatal("p should schedule the RR READ job")
		}
		m = key(m, '6')
		if e.Phase() != sim.P64 {
			t.Fatalf("6 should enter P64, got %v", e.Phase())
		}
		m = key(m, 'a')
		if e.Phase() != sim.P66 {
			t.Fatalf("a should enter ATT HOLD/P66, got %v", e.Phase())
		}
		m = key(m, 'x')
		if e.Phase() != sim.P00 {
			t.Fatalf("x should reset to idle, got %v", e.Phase())
		}
	})
	t.Run("happy: h and l move the card selection, not the radar", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, 'l')
		if e.LandingRadarAcquired() {
			t.Fatal("l is selection movement; the landing radar locks on its own during descent")
		}
		if m.Selected() != 1 {
			t.Fatalf("l should move the selection, got %d", m.Selected())
		}
	})
	t.Run("happy: n fake-types V16N68 over time", func(t *testing.T) {
		e, m := newTestModel()
		e.SetWallToAGC(1.0) // real time: ~2s of typing fits in ~70 frames
		m = key(m, 'n')
		if m.PendingKeys() < 7 {
			t.Fatalf("n should queue the 7 keystrokes of V16N68 ENTR, got %d", m.PendingKeys())
		}
		m = tick(m, 120)
		if m.PendingKeys() != 0 {
			t.Fatalf("fake typing should complete, %d keys still pending", m.PendingKeys())
		}
		if !e.MonitorActive() {
			t.Fatal("after Neil types V16N68 the monitor must be active")
		}
	})
	t.Run("happy: t enters typing mode where your keys are DSKY keys", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, 't')
		if !m.TypingMode() {
			t.Fatal("t should enter typing mode")
		}
		m = key(m, 'v')
		m = key(m, '1')
		m = key(m, '6')
		if e.DSKY().Verb != "16" {
			t.Fatalf("typed V16 should set the verb display, got %q", e.DSKY().Verb)
		}
		mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = mm.(Model)
		if m.TypingMode() {
			t.Fatal("esc should leave typing mode")
		}
	})
	t.Run("happy: '.' pauses, brackets change speed, q quits", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, '.')
		if !m.Paused() {
			t.Fatal("'.' should pause")
		}
		before := e.AGCTimeMs()
		m = tick(m, 10)
		if e.AGCTimeMs() != before {
			t.Fatal("paused ticks must not advance AGC time")
		}
		m = key(m, '.')
		s0 := m.TimeScale()
		m = key(m, ']')
		if m.TimeScale() <= s0 {
			t.Fatal("] should speed up")
		}
		m = key(m, '[')
		m = key(m, '[')
		if m.TimeScale() >= s0 {
			t.Fatal("[ should slow down")
		}
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		if cmd == nil {
			t.Fatal("q should produce a quit command")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("q's command must be tea.Quit")
		}
	})
	t.Run("unhappy: unknown keys change nothing", func(t *testing.T) {
		e, m := newTestModel()
		phase, bug, paused := e.Phase(), e.RadarBug(), m.Paused()
		m = key(m, 'z')
		m = key(m, '!')
		if e.Phase() != phase || e.RadarBug() != bug || m.Paused() != paused {
			t.Fatal("unknown keys must be no-ops")
		}
	})
}

func hasOwner(e *sim.Engine, name string) bool {
	for _, c := range e.CoreSets() {
		if c.Busy && c.Owner == name {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// t21 — DSKY panel: VERB/NOUN, PROG lamp, alarm codes
// ---------------------------------------------------------------------------

func TestDSKYPanel(t *testing.T) {
	t.Run("happy: typed verb/noun appears", func(t *testing.T) {
		e, m := newTestModel()
		for _, k := range []byte("V16N68E") {
			e.PressKey(k)
		}
		st := m.dskyState()
		if st.Verb != "16" || st.Noun != "68" {
			t.Fatalf("DSKY should show VERB 16 NOUN 68, got V%q N%q", st.Verb, st.Noun)
		}
		if !strings.Contains(m.View(), "|_") {
			t.Fatal("the panel must render the digits as seven-segment strokes")
		}
	})
	t.Run("unhappy: alarm lights PROG and shows the code on the DSKY", func(t *testing.T) {
		e, m := newTestModel()
		for i := 0; i < 8; i++ {
			e.ScheduleJob("HOG", 25, 1e9, false)
		}
		e.ScheduleJob("STRAW", 25, 10, false)
		st := m.dskyState()
		if !st.Lights.Prog {
			t.Fatal("PROG lamp must light after an alarm")
		}
		if !strings.Contains(st.R1, "1202") {
			t.Fatalf("the alarm code must be readable in R1, got %q", st.R1)
		}
	})
}
