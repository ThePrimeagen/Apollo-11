// Package director is the screenplay editor, and MAIN owns its own
// numbers. The whole bill plays on one stage; ctrl+n and ctrl+p (or
// plain n and p) scroll the scenes both ways and never end the show.
// Browsing is the quiet face: the marquee, one hold row — how long
// the scene lasts in play mode before the cut — trimmed directly with
// h/l, and the help line. A scene may also pin a knob to that face
// (the fire's fall duration) so the operator can read it without
// hunting; other knobs still wait behind e. e opens the MAIN CONFIG
// panel for the scene now playing: the hold first, then every one of
// the scene's own knobs; j/k pick a row, h/l turn it, e or esc hands
// the quiet face back. Hold and fall stay two numbers — a short hold
// can cut mid-fall; that is the operator's choice, never clamped.
// The panel wears the MAIN CONFIG name so it never reads like
// the scene's standalone tuner — these numbers are the show's copy,
// laid onto each scene instance, never the scene package's config or
// its Active. No knob is ever clamped: a zero or negative hold is the
// operator's number and cuts at once. Space is the play button: the
// scene restarts and the bill cuts itself forward on each hold; past
// the last hold the play just stops. f is the fullscreen premiere:
// the chrome drops, the show rewinds to the top, and it plays
// through; f or esc hands the chrome back, and the end of the show
// does too. r replays the scene from its top. s writes one file —
// MAIN's own config, every scene in bill order with its hold and its
// knobs — syncing only the moonwalk's sibling beats, one performance
// that they are. q and ctrl+c are the only ways out.
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

// Model is the editor: the composed screenplay, MAIN's own config,
// the quiet browse face or the MAIN CONFIG panel over the scene now
// playing, and the play/premiere state.
type Model struct {
	w, h    int
	title   string
	bill    screenplay.Bill
	play    *screenplay.Screenplay
	screen  *screenplay.Screen
	cfg     Config
	cfgPath string
	resolve func(string) string
	knobs   []*knobs
	editing bool
	cursor  int
	playing bool
	full    bool
	clock   float64
	seconds float64
	elapsed float64
	note    string
}

// New opens the editor on a bill wearing MAIN's saved numbers. The
// config is reseeded in bill order — every scene gets a row, file
// values kept, the rest stock — and each saved knob set is laid onto
// its scene's own Cfg; a blob that does not fit its scene is ignored
// with a word on the status line. seconds > 0 auto-quits, handy for
// tapes.
func New(title string, bill screenplay.Bill, cfg Config, cfgPath string, seconds float64) Model {
	seeded := Config{}
	ks := make([]*knobs, len(bill))
	note := ""
	for i, e := range bill {
		seeded.SetHold(e.Name, cfg.HoldFor(e.Name))
		ks[i] = knobsFor(e.Scene)
		if ks[i] == nil {
			continue
		}
		raw := cfg.KnobsFor(e.Name)
		if len(raw) == 0 {
			continue
		}
		if err := ks[i].apply(raw); err != nil {
			note = "config: " + e.Name + " knobs ignored"
			continue
		}
		seeded.SetKnobs(e.Name, raw)
	}
	play := screenplay.Compose(bill)
	play.Start()
	return Model{
		w:       defaultW,
		h:       defaultH,
		title:   title,
		bill:    bill,
		play:    play,
		screen:  screenplay.NewScreen(defaultW, defaultH-1),
		cfg:     seeded,
		cfgPath: cfgPath,
		resolve: menu.Resolve,
		knobs:   ks,
		seconds: seconds,
		note:    note,
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

// rows is the MAIN CONFIG panel height for the scene now playing: the
// hold plus its own knobs.
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

// nudge turns the selected knob. Row zero — browsing or the panel —
// is the hold, never clamped. Editing walks every scene knob; a
// quiet-face front knob (the fire's fall) is the only scene number
// h/l can turn without e.
func (m *Model) nudge(dir int) {
	if m.cursor > 0 {
		if k := m.currentKnobs(); k != nil {
			idx := m.cursor - 1
			if !m.editing {
				if idx >= len(k.front) {
					return
				}
				idx = k.front[idx]
			}
			k.nudge(idx, dir)
			return
		}
	}
	name := m.play.CurrentName()
	m.cfg.SetHold(name, m.cfg.HoldFor(name)+HoldStepSeconds*float64(dir))
}

// browseRows is the quiet face height: the hold plus any knobs pinned
// to the browse face. Most scenes are hold alone.
func (m Model) browseRows() int {
	n := 1
	if k := m.currentKnobs(); k != nil {
		n += len(k.front)
	}
	return n
}

// moveCursor walks the panel, wrapping both ways. Browsing with only
// the hold has no cursor — h/l always mean the hold. A scene that
// pins a knob to the quiet face (fire's fall) lets j/k pick it.
func (m *Model) moveCursor(delta int) {
	n := m.browseRows()
	if m.editing {
		n = m.rows()
	} else if n <= 1 {
		return
	}
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

// exitFull hands the chrome back and stops the play. The panel comes
// back as it was.
func (m *Model) exitFull() {
	m.full = false
	m.playing = false
	m.fitScreen()
}

// save writes MAIN's one file: every scene in bill order, its hold
// and — for a knobbed scene — its knobs in the scene's own JSON
// shape. Saving from a moonwalk beat first copies its Cfg across the
// sibling beats: the three are one performance. Nothing here touches
// a scene package's config file or its Active. Any failure lands on
// the status line and the editor keeps going.
func (m *Model) save() {
	idx := m.play.SceneIndex()
	if cur := m.currentKnobs(); cur != nil && cur.syncs {
		if raw, err := cur.marshal(); err == nil {
			for i, other := range m.knobs {
				if i == idx || other == nil || other.kind != cur.kind {
					continue
				}
				_ = other.apply(raw)
			}
		}
	}
	for i, e := range m.bill {
		k := m.knobs[i]
		if k == nil {
			continue
		}
		raw, err := k.marshal()
		if err != nil {
			m.note = "save failed: " + err.Error()
			return
		}
		m.cfg.SetKnobs(e.Name, raw)
	}
	if err := m.cfg.Save(m.resolve(m.cfgPath)); err != nil {
		m.note = "save failed: " + err.Error()
		return
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
			if m.clock >= m.cfg.HoldFor(m.play.CurrentName()) {
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
		m.note = ""
		if dir := sceneCut(msg); dir != 0 {
			m.cut(dir)
			return m, nil
		}
		switch strings.ToLower(msg.String()) {
		case "ctrl+c", "q":
			m.play.Stop()
			return m, tea.Quit
		case " ", "space":
			if m.playing {
				m.playing = false
			} else {
				m.playing = true
				m.replay()
			}
		case "e":
			m.editing = !m.editing
			m.cursor = 0
		case "f":
			if m.full {
				m.exitFull()
			} else {
				m.enterFull()
			}
		case "esc":
			if m.full {
				m.exitFull()
			} else if m.editing {
				m.editing = false
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

// paintBrowseRow is one quiet-face knob line. The selected row wears
// the gold pointer; the rest stay dim. Hold and fall stay two rows.
func (m Model) paintBrowseRow(x, y, row int, label string, val float64) {
	marker, ink := "  ", dimInk
	if row == m.cursor {
		marker, ink = "▸ ", goldInk
	}
	paintText(m.screen, x, y, fmt.Sprintf("%s%-14s %10.3f", marker, label, val), ink)
}

// paintText lays a line of ink onto the screen, clipped at the edges.
func paintText(scr *screenplay.Screen, x, y int, text string, ink int) {
	for i, ch := range []rune(text) {
		scr.PutCell(x+i, y, ch, ink, -1)
	}
}

// paintChrome puts the marquee over the stage, then the quiet hold
// row and any knobs pinned to the browse face (the fire's fall) —
// or, editing, the MAIN CONFIG panel: the show's own copy of the
// scene's every knob, dressed in its own name so it never reads
// like the scene's standalone tuner.
func (m Model) paintChrome() {
	w, h := m.screen.Size()
	if w < 1 || h < 1 {
		return
	}
	name := m.play.CurrentName()
	marquee := fmt.Sprintf(" %s · %d/%d %s", m.title, m.play.SceneIndex()+1, m.play.Len(), name)
	mode, ink := "  ■ hold", dimInk
	if m.playing {
		mode, ink = fmt.Sprintf("  ▶ %.1f/%.1fs", m.clock, m.cfg.HoldFor(name)), goldInk
	}
	paintText(m.screen, 0, 0, marquee, titleInk)
	paintText(m.screen, len([]rune(marquee)), 0, mode, ink)

	if !m.editing {
		m.paintBrowseRow(1, 2, 0, "hold", m.cfg.HoldFor(name))
		if k := m.currentKnobs(); k != nil {
			for i, ki := range k.front {
				m.paintBrowseRow(1, 3+i, i+1, k.label(ki), k.value(ki))
			}
		}
		return
	}
	paintText(m.screen, 1, 2, "MAIN CONFIG · "+name, goldInk)
	k := m.currentKnobs()
	rows := m.rows()
	avail := h - 3
	if avail < 1 {
		return
	}
	cols := 1
	if rows > avail {
		cols = (rows + avail - 1) / avail
	}
	colW := 30
	if cols > 1 && w/cols > 0 {
		colW = w / cols
	}
	for i := 0; i < rows; i++ {
		col, row := 0, i
		if cols > 1 {
			col = i / avail
			row = i % avail
		}
		label, val := "hold", m.cfg.HoldFor(name)
		if i > 0 {
			label, val = k.label(i-1), k.value(i-1)
		}
		marker, rowInk := "  ", dimInk
		if i == m.cursor {
			marker, rowInk = "▸ ", goldInk
		}
		text := fmt.Sprintf("%s%-14s %10.3f", marker, label, val)
		if rs := []rune(text); colW > 0 && len(rs) > colW {
			text = fmt.Sprintf("%s%s %.3f", marker, label, val)
		}
		if rs := []rune(text); colW > 0 && len(rs) > colW {
			text = string(rs[:colW])
		}
		x := 1
		if cols > 1 {
			x = col * colW
		}
		paintText(m.screen, x, 3+row, text, rowInk)
	}
}

// sceneCut is +1 / -1 when the key walks the bill. Plain n/p and
// C-n / C-p both cut — a real pty reports Mod+Code, not always the
// "ctrl+n" string.
func sceneCut(msg tea.KeyPressMsg) int {
	if ctrlLetter(msg, 'n') {
		return 1
	}
	if ctrlLetter(msg, 'p') {
		return -1
	}
	switch strings.ToLower(msg.String()) {
	case "n", "ctrl+n":
		return 1
	case "p", "ctrl+p":
		return -1
	}
	return 0
}

func ctrlLetter(msg tea.KeyPressMsg, r rune) bool {
	if msg.Mod&tea.ModCtrl == 0 {
		return false
	}
	c := msg.Code
	if c >= 'A' && c <= 'Z' {
		c = c - 'A' + 'a'
	}
	return c == r
}

func (m Model) statusLine(w int) string {
	text := " ctrl+n/p scene · h/l hold · e edit · space play · f premiere · s save · q quit"
	if m.editing {
		text = " j/k h/l knobs · e done · s save · r replay · ctrl+n/p scene · q quit"
	}
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
// the marquee and its face over the scene and one status line under
// it.
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
