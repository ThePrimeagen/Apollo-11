package agctop

// The command screen: every process on the millisecond Executive, live,
// in the three groups that matter —
//
//   VAC JOBS                 jobs holding a core set AND a VAC area
//   CORESET JOBS             jobs holding a core set only
//   NO-PRIORITY OPERATIONS   tasks & interrupts: cpu only, no memory
//
// — with three switches on the bottom: [d] DESCENT (the P63 job chain),
// [1] 1668 (Buzz's V16N68 monitor), [r] RADAR STEAL (the RR CDU theft).

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	msim "github.com/theprimeagen/apollo-11/msim"
)

func keyed(m Model, code rune) (Model, tea.Cmd) {
	mm, cmd := m.Update(tea.KeyPressMsg{Code: code})
	return mm.(Model), cmd
}

func sized(m Model, w, h int) Model {
	mm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return mm.(Model)
}

func view(m Model) string { return m.View().Content }

func TestGroupsShowTheRunningProcesses(t *testing.T) {
	// happy: with descent + 1668 + steal on, the three groups carry their
	// processes — SERVICER copies under VAC JOBS, the monitor and the radar
	// gates under CORESET JOBS, the cadences under NO-PRIORITY OPERATIONS
	l := msim.NewLive()
	l.SetMonitor(true)
	l.StepMS(21_000) // deep into the build: stubs parked, gates sleeping
	m := sized(New(l), 100, 40)
	v := view(m)

	iVac := strings.Index(v, "VAC JOBS")
	iCore := strings.Index(v, "CORESET JOBS")
	iOps := strings.Index(v, "NO-PRIORITY OPERATIONS")
	if iVac < 0 || iCore < 0 || iOps < 0 {
		t.Fatalf("missing a group header: vac=%d core=%d ops=%d\n%s", iVac, iCore, iOps, v)
	}
	if !(iVac < iCore && iCore < iOps) {
		t.Fatalf("group order wrong: vac=%d core=%d ops=%d", iVac, iCore, iOps)
	}
	vacSec, coreSec, opsSec := v[iVac:iCore], v[iCore:iOps], v[iOps:]
	if !strings.Contains(vacSec, "SERVICER") {
		t.Fatalf("VAC JOBS missing SERVICER:\n%s", vacSec)
	}
	for _, want := range []string{"MONDO", "LRHJOB", "LRVJOB", "MAKEPLAY", "1/GYRO"} {
		if !strings.Contains(coreSec, want) {
			t.Fatalf("CORESET JOBS missing %s:\n%s", want, coreSec)
		}
	}
	for _, want := range []string{"DAP", "T4RUPT", "DOWNRUPT", "READACCS", "R10,R11"} {
		if !strings.Contains(opsSec, want) {
			t.Fatalf("NO-PRIORITY OPERATIONS missing %s:\n%s", want, opsSec)
		}
	}
	// SERVICER is a VAC job — it must not be listed as a coreset job
	if strings.Contains(coreSec, "SERVICER") {
		t.Fatalf("SERVICER leaked into CORESET JOBS:\n%s", coreSec)
	}
}

func TestGroupsRenderIdle(t *testing.T) {
	// unhappy: descent off, nothing running — the headers still render and
	// the rows show placeholders instead of vanishing
	l := msim.NewLive()
	l.SetRadar(false)
	l.SetDescent(false)
	l.StepMS(5_000)
	m := sized(New(l), 100, 40)
	v := view(m)
	for _, want := range []string{"VAC JOBS", "CORESET JOBS", "NO-PRIORITY OPERATIONS"} {
		if !strings.Contains(v, want) {
			t.Fatalf("idle screen missing %q", want)
		}
	}
	if !strings.Contains(v, "—") {
		t.Fatalf("idle screen must show placeholder rows")
	}
}

func TestTogglesDriveTheEngine(t *testing.T) {
	// happy: d / 1 / r flip the switches and the live engine follows
	l := msim.NewLive()
	m := sized(New(l), 100, 40)

	m, _ = keyed(m, 'd')
	if l.DescentOn() {
		t.Fatalf("'d' must switch DESCENT off")
	}
	m, _ = keyed(m, '1')
	if !l.MonitorOn() {
		t.Fatalf("'1' must key the 1668 monitor up")
	}
	m, _ = keyed(m, 'r')
	if l.RadarOn() {
		t.Fatalf("'r' must switch the radar steal off")
	}
	v := view(m)
	for _, want := range []string{"DESCENT", "1668", "RADAR STEAL"} {
		if !strings.Contains(v, want) {
			t.Fatalf("footer missing switch %q:\n%s", want, v)
		}
	}
	if !strings.Contains(v, "OFF") || !strings.Contains(v, "ON") {
		t.Fatalf("footer must show switch states:\n%s", v)
	}
	// flip back
	m, _ = keyed(m, 'd')
	if !l.DescentOn() {
		t.Fatalf("'d' again must switch DESCENT back on")
	}
	_ = m
}

func TestOtherKeysInertAndQQuits(t *testing.T) {
	// unhappy: unrelated keys change nothing; q quits
	l := msim.NewLive()
	m := sized(New(l), 100, 40)
	m, cmd := keyed(m, 'x')
	if cmd != nil {
		t.Fatalf("'x' must be inert")
	}
	if !l.DescentOn() || l.MonitorOn() || !l.RadarOn() {
		t.Fatalf("'x' disturbed the switches")
	}
	_, cmd = keyed(m, 'q')
	if cmd == nil {
		t.Fatalf("'q' must quit")
	}
}

func TestFrameTickAdvancesTheSim(t *testing.T) {
	// happy: each frame message advances the machine by the frame budget
	l := msim.NewLive()
	m := sized(New(l), 100, 40)
	before := l.Engine().Now()
	mm, cmd := m.Update(frameMsg{})
	if cmd == nil {
		t.Fatalf("a frame must schedule the next frame")
	}
	m = mm.(Model)
	if got := l.Engine().Now() - before; got != msim.Nanos(frameMS)*msim.Millisecond {
		t.Fatalf("frame advanced %d ns, want %d ms", got, frameMS)
	}
}

func TestHeaderPoolsAndAlarmFlash(t *testing.T) {
	// happy: the header carries the GET clock and both pool bars; after the
	// monitor drives the pools to the wall the 1202 flashes on screen
	l := msim.NewLive()
	l.SetMonitor(true)
	m := sized(New(l), 100, 40)
	l.StepMS(27_000) // past the wall: the first 1202 has fired
	v := view(m)
	if !strings.Contains(v, "GET 102:") {
		t.Fatalf("header missing the GET clock:\n%s", v)
	}
	if !strings.Contains(v, "cores") || !strings.Contains(v, "vacs") {
		t.Fatalf("header missing the pool bars:\n%s", v)
	}
	if len(l.Engine().Alarms()) == 0 {
		t.Fatalf("scenario never alarmed — the flash cannot be tested")
	}
	if !strings.Contains(v, "1202") {
		t.Fatalf("screen must flash the 1202:\n%s", v)
	}
}

func TestTinyWindowDoesNotPanic(t *testing.T) {
	// unhappy: a 5x3 terminal still renders something
	l := msim.NewLive()
	l.StepMS(3_000)
	m := sized(New(l), 5, 3)
	if v := view(m); v == "" {
		t.Fatalf("tiny window rendered nothing")
	}
}
