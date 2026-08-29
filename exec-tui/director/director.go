// Package director is the screenplay editor: the whole bill on one
// stage. n and p walk the cuts both ways and never end the show. A
// knob panel rides over the scene now playing — the editor's own hold
// first (how long the scene plays in play mode before the cut), then
// the scene's own knobs — j/k pick a row, h/l turn it, and no knob is
// ever clamped: a zero or negative hold is the operator's number and
// cuts at once. Space is the play button: the scene restarts and the
// bill cuts itself forward on each hold; past the last hold the play
// just stops — the editor never quits on its own. f is the fullscreen
// premiere: the chrome drops, the show rewinds to the top, and it
// plays through; f or esc hands the chrome back, and the end of the
// show does too. r replays the scene from its top. s saves the holds
// file and the current scene's knobs to their own homes, syncing only
// the moonwalk's sibling beats — they are one performance. q and
// ctrl+c are the only ways out.
package director

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/menu"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	defaultW = 84
	defaultH = 30
	minW     = 10
	minH     = 4
	frameMs  = 1000.0 / 30

	titleInk = 252
	goldInk  = 214
	dimInk   = 240
)

// Model is the editor: the composed screenplay, the knob panel over
// the scene now playing, and the play/premiere state.
type Model struct {
	w, h      int
	title     string
	bill      screenplay.Bill
	play      *screenplay.Screenplay
	screen    *screenplay.Screen
	holds     Config
	holdsPath string
	resolve   func(string) string
	knobs     []*knobs
	cursor    int
	playing   bool
	full      bool
	clock     float64
	seconds   float64
	elapsed   float64
	note      string
}

// New opens the editor on a bill. The loaded holds are reseeded in
// bill order — every scene gets a row, file values kept, the rest
// stock — so a save always writes the whole show. seconds > 0
// auto-quits, handy for tapes.
func New(title string, bill screenplay.Bill, holds Config, holdsPath string, seconds float64) Model {
	seeded := Config{}
	ks := make([]*knobs, len(bill))
	for i, e := range bill {
		seeded.SetHold(e.Name, holds.HoldFor(e.Name))
		ks[i] = knobsFor(e.Scene)
	}
	play := screenplay.Compose(bill)
	play.Start()
	return Model{
		w:         defaultW,
		h:         defaultH,
		title:     title,
		bill:      bill,
		play:      play,
		screen:    screenplay.NewScreen(defaultW, defaultH-1),
		holds:     seeded,
		holdsPath: holdsPath,
		resolve:   menu.Resolve,
		knobs:     ks,
		seconds:   seconds,
	}
}

type frameMsg struct{}

func tick() tea.Cmd {
	ns := float64(frameMs) * 1e6
	return tea.Tick(time.Duration(ns)*time.Nanosecond, func(time.Time) tea.Msg {
		return frameMsg{}
	})
}

// Init schedules the first frame.
func (m Model) Init() tea.Cmd { return tick() }

// currentKnobs is the adapter for the scene now playing, or nil for a
// scene with no knobs.
func (m Model) currentKnobs() *knobs {
	idx := m.play.SceneIndex()
	if idx < 0 || idx >= len(m.knobs) {
		return nil
	}
	return m.knobs[idx]
}

// rows is the panel height for the scene now playing: the hold plus
// its own knobs.
func (m Model) rows() int {
	n := 1
	if k := m.currentKnobs(); k != nil {
		n += k.count
	}
	return n
}

// fitScreen sizes the stage under the window: the whole window in the
// premiere, everything above the status line in the editor.
func (m *Model) fitScreen() {
	w, h := max(m.w, minW), max(m.h, minH)
	if !m.full {
		h--
	}
	m.screen.Resize(w, h)
}

// cut walks the bill one scene in dir and resets the hold clock and
// the knob cursor. The ends hold — the editor never ends the show.
func (m *Model) cut(dir int) {
	moved := false
	if dir > 0 {
		moved = m.play.Next()
	} else {
		moved = m.play.Prev()
	}
	if moved {
		m.clock, m.cursor = 0, 0
	}
}

// replay restarts the scene now playing from its top.
func (m *Model) replay() {
	idx := m.play.SceneIndex()
	if idx >= 0 && idx < len(m.bill) {
		if sc := m.bill[idx].Scene; sc != nil {
			sc.Stop()
			sc.Start()
		}
	}
	m.clock = 0
}

// nudge turns the selected knob: the hold on row zero — never clamped
// — or the scene's own knob under it.
func (m *Model) nudge(dir int) {
	if m.cursor == 0 {
		name := m.play.CurrentName()
		m.holds.SetHold(name, m.holds.HoldFor(name)+HoldStepSeconds*float64(dir))
		return
	}
	if k := m.currentKnobs(); k != nil {
		k.nudge(m.cursor-1, dir)
	}
}

// moveCursor walks the panel, wrapping both ways.
func (m *Model) moveCursor(delta int) {
	n := m.rows()
	m.cursor = ((m.cursor+delta)%n + n) % n
}

// enterFull is the premiere: chrome down, the show rewound to the
// top, playing.
func (m *Model) enterFull() {
	m.full = true
	m.playing = true
	m.clock, m.cursor = 0, 0
	m.play.Rewind()
	m.fitScreen()
}

// exitFull hands the chrome back and stops the play.
func (m *Model) exitFull() {
	m.full = false
	m.playing = false
	m.fitScreen()
}

// save writes the holds beside the bill and the current scene's knobs
// to their own file, makes those knobs active, and syncs the sibling
// beats of a scene kind that volunteers one. Any failure lands on the
// status line and the editor keeps going.
func (m *Model) save() {
	if err := m.holds.Save(m.resolve(m.holdsPath)); err != nil {
		m.note = "save failed: " + err.Error()
		return
	}
	idx := m.play.SceneIndex()
	if k := m.currentKnobs(); k != nil {
		if err := k.save(m.resolve(k.path)); err != nil {
			m.note = "save failed: " + err.Error()
			return
		}
		for i, other := range m.knobs {
			if i == idx || other == nil || other.kind != k.kind || other.sync == nil {
				continue
			}
			other.sync()
		}
	}
	m.note = "saved"
}

// Update runs the clock and the keys.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.fitScreen()
		return m, nil
	case frameMsg:
		dt := frameMs / 1000
		m.elapsed += dt
		m.play.Update(dt)
		if m.playing {
			m.clock += dt
			if m.clock >= m.holds.HoldFor(m.play.CurrentName()) {
				if m.play.Next() {
					m.clock, m.cursor = 0, 0
				} else {
					m.playing = false
					if m.full {
						m.exitFull()
					}
				}
			}
		}
		if m.seconds > 0 && m.elapsed >= m.seconds {
			m.play.Stop()
			return m, tea.Quit
		}
		return m, tick()
	case tea.KeyPressMsg:
		switch strings.ToLower(msg.String()) {
		case "ctrl+c", "q":
			m.play.Stop()
			return m, tea.Quit
		case "n":
			m.cut(1)
		case "p":
			m.cut(-1)
		case " ", "space":
			if m.playing {
				m.playing = false
			} else {
				m.playing = true
				m.replay()
			}
		case "f":
			if m.full {
				m.exitFull()
			} else {
				m.enterFull()
			}
		case "esc":
			if m.full {
				m.exitFull()
			}
		case "j", "down":
			m.moveCursor(1)
		case "k", "up":
			m.moveCursor(-1)
		case "l", "right":
			m.nudge(1)
		case "h", "left":
			m.nudge(-1)
		case "r":
			m.replay()
		case "s":
			m.save()
		}
		return m, nil
	}
	return m, nil
}

// paintText lays a line of ink onto the screen, clipped at the edges.
func paintText(scr *screenplay.Screen, x, y int, text string, ink int) {
	for i, ch := range []rune(text) {
		scr.PutCell(x+i, y, ch, ink, -1)
	}
}

// paintChrome puts the marquee and the knob panel over the stage.
func (m Model) paintChrome() {
	w, h := m.screen.Size()
	if w < 1 || h < 1 {
		return
	}
	name := m.play.CurrentName()
	marquee := fmt.Sprintf(" %s · %d/%d %s", m.title, m.play.SceneIndex()+1, m.play.Len(), name)
	mode, ink := "  ■ hold", dimInk
	if m.playing {
		mode, ink = fmt.Sprintf("  ▶ %.1f/%.1fs", m.clock, m.holds.HoldFor(name)), goldInk
	}
	paintText(m.screen, 0, 0, marquee, titleInk)
	paintText(m.screen, len([]rune(marquee)), 0, mode, ink)

	k := m.currentKnobs()
	rows := m.rows()
	avail := h - 2
	if avail < 1 {
		return
	}
	off := 0
	if m.cursor >= avail {
		off = m.cursor - avail + 1
	}
	for r := 0; r < avail && off+r < rows; r++ {
		i := off + r
		label, val := "hold", m.holds.HoldFor(name)
		if i > 0 {
			label, val = k.label(i-1), k.value(i-1)
		}
		marker, rowInk := "  ", dimInk
		if i == m.cursor {
			marker, rowInk = "▸ ", goldInk
		}
		paintText(m.screen, 1, 2+r, fmt.Sprintf("%s%-14s %10.3f", marker, label, val), rowInk)
	}
}

func (m Model) statusLine(w int) string {
	text := " n/p scene · space play · f premiere · j/k h/l knobs · r replay · s save · q quit"
	if m.note != "" {
		text = " " + m.note
	}
	return "\x1b[38;5;240m" + pad(text, w) + "\x1b[0m"
}

func pad(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return string(r[:w])
	}
	return s + strings.Repeat(" ", w-len(r))
}

// View is the stage — the premiere owns every line; the editor keeps
// the marquee and panel over the scene and one status line under it.
func (m Model) View() tea.View {
	m.play.Render(m.screen)
	if !m.full {
		m.paintChrome()
	}
	w, h := m.screen.Size()
	lines := strings.Split(m.screen.Render(), "\n")
	for len(lines) < h {
		lines = append(lines, "")
	}
	if !m.full {
		lines = append(lines, m.statusLine(w))
	}
	winH := m.h
	if winH < 1 {
		winH = defaultH
	}
	for len(lines) < winH {
		lines = append(lines, "")
	}
	if len(lines) > winH {
		lines = lines[:winH]
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	return v
}
