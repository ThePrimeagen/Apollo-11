// astronaut: the moonwalk tuner. Plays the scenes/moonwalk show — the
// crate climb, the pole-top landing, the American flag rising as he
// slides, and the closing camera pan to the lunar rover — with every
// scene knob adjustable live.
//
//	j / k               select a knob
//	h / l               nudge it down / up
//	s                   save knobs to scenes/moonwalk/config.json
//	space / enter / p   replay from the top
//	q / ctrl+c          quit
//
//	go run ./cmd/astronaut
//	go run ./cmd/astronaut -seconds 20
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/astro"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/menu"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/moonwalk"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
	"github.com/theprimeagen/apollo-11/terminal-fonts/termfont"
)

const (
	defaultW   = 84
	defaultH   = 30
	frameMs    = 1000.0 / 30
	footerRows = 4
)

type model struct {
	w, h    int
	clock   float64
	seconds float64
	atlas   *sprite.Atlas
	cfg     moonwalk.Config
	knob    moonwalk.Knob
	path    string
	note    string
	toast   string
	toastID int
}

// saveToastClearMsg drops the banner armed by the save that carries
// the same id — a stale curtain never wipes a fresh banner.
type saveToastClearMsg struct{ id int }

func newModel(seconds float64) (model, error) {
	atlas, err := astro.Load()
	if err != nil {
		return model{}, err
	}
	path := menu.Resolve(moonwalk.DefaultConfigPath)
	cfg, err := moonwalk.LoadOrDefault(path)
	if err != nil {
		return model{}, err
	}
	return model{
		w:       defaultW,
		h:       defaultH,
		seconds: seconds,
		atlas:   atlas,
		cfg:     cfg,
		path:    path,
	}, nil
}

type frameMsg struct{}

func tick() tea.Cmd {
	ns := float64(frameMs) * 1e6
	return tea.Tick(time.Duration(ns)*time.Nanosecond, func(time.Time) tea.Msg {
		return frameMsg{}
	})
}

func (m model) Init() tea.Cmd { return tick() }

// saveToastTTL is how long the banner flies — the operator asked for
// three seconds, top center, too big to miss.
const saveToastTTL = 3 * time.Second

// save writes the knobs and raises the banner: SAVED on success, ERR
// with the failure named in the footer otherwise.
func (m model) save() model {
	m.toastID++
	if err := m.cfg.Save(m.path); err != nil {
		m.note = "save failed: " + err.Error()
		m.toast = "ERR"
		return m
	}
	m.note = ""
	m.toast = "SAVED"
	return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case frameMsg:
		m.clock += frameMs / 1000
		if m.seconds > 0 && m.clock >= m.seconds {
			return m, tea.Quit
		}
		return m, tick()
	case saveToastClearMsg:
		if msg.id == m.toastID {
			m.toast = ""
		}
		return m, nil
	case tea.KeyPressMsg:
		switch strings.ToLower(msg.String()) {
		case "ctrl+c", "q":
			return m, tea.Quit
		case " ", "space", "enter", "p":
			m.clock = 0
			return m, nil
		case "j", "down":
			m.knob = (m.knob + 1) % moonwalk.KnobCount
			return m, nil
		case "k", "up":
			m.knob = (m.knob - 1 + moonwalk.KnobCount) % moonwalk.KnobCount
			return m, nil
		case "l", "right":
			m.cfg.Nudge(m.knob, 1)
			return m, nil
		case "h", "left":
			m.cfg.Nudge(m.knob, -1)
			return m, nil
		case "s":
			m = m.save()
			id := m.toastID
			return m, tea.Tick(saveToastTTL, func(time.Time) tea.Msg {
				return saveToastClearMsg{id: id}
			})
		}
	}
	return m, nil
}

// knobCell is one footer entry: a marker, the knob's name, and its
// value — ints plain, floats to two places.
func (m model) knobCell(k moonwalk.Knob) string {
	mark := "  "
	if k == m.knob {
		mark = "> "
	}
	v := m.cfg.Value(k)
	val := fmt.Sprintf("%.2f", v)
	switch k {
	case moonwalk.KnobPoleRows, moonwalk.KnobPanCols, moonwalk.KnobBoxStart, moonwalk.KnobLMGap:
		val = fmt.Sprintf("%d", int(v))
	}
	return fmt.Sprintf("%s%-10s %6s", mark, k.String(), val)
}

func (m model) footer(w int) []string {
	dim := "\x1b[38;5;240m"
	hot := "\x1b[38;5;214m"
	reset := "\x1b[0m"
	// The help line survives everything — only a failure detail may
	// borrow it, so "s save" never disappears after a save.
	help := "moonwalk  j/k knob  h/l nudge  s save  space replay  q quit"
	if m.note != "" {
		help = "moonwalk  " + m.note
	}
	lines := []string{dim + clip(help, w) + reset}
	for row := 0; row < 3; row++ {
		var cells []string
		for i := 0; i < 4; i++ {
			k := moonwalk.Knob(row*4 + i)
			if k >= moonwalk.KnobCount {
				break
			}
			cell := m.knobCell(k)
			if k == m.knob {
				cell = hot + cell + reset
			} else {
				cell = dim + cell + reset
			}
			cells = append(cells, cell)
		}
		lines = append(lines, clipANSI(strings.Join(cells, " "), w))
	}
	return lines
}

func (m model) View() tea.View {
	w, h := m.w, m.h
	if w < 1 {
		w = defaultW
	}
	if h < 1 {
		h = defaultH
	}
	stageH := h - footerRows
	if stageH < 1 {
		stageH = 1
	}
	stage := moonwalk.Frame(m.cfg, m.atlas, w, stageH, m.clock)
	lines := strings.Split(sprite.Render(stage), "\n")
	overlayToast(lines, m.toast, w)
	lines = append(lines, m.footer(w)...)
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	return v
}

// overlayToast rides the save banner over the top of the stage: five
// rows of termfont, centered, gold and bold — too big to miss.
func overlayToast(lines []string, toast string, w int) {
	if toast == "" {
		return
	}
	rows, err := termfont.Lines(5, toast)
	if err != nil {
		rows = []string{toast}
	}
	for i, row := range rows {
		target := 1 + i
		if target >= len(lines) {
			break
		}
		pad := (w - len([]rune(row))) / 2
		if pad < 0 {
			pad = 0
		}
		lines[target] = "\x1b[1;38;5;214m" + clip(strings.Repeat(" ", pad)+row, w) + "\x1b[0m"
	}
}

func clip(s string, w int) string {
	r := []rune(s)
	if len(r) > w {
		return string(r[:w])
	}
	return s
}

// clipANSI trims a styled line to w visible cells, keeping escape
// sequences intact and closing with a reset.
func clipANSI(s string, w int) string {
	var b strings.Builder
	visible := 0
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		if rs[i] == 0x1b {
			for i < len(rs) {
				b.WriteRune(rs[i])
				if rs[i] == 'm' {
					break
				}
				i++
			}
			continue
		}
		if visible >= w {
			break
		}
		b.WriteRune(rs[i])
		visible++
	}
	return b.String() + "\x1b[0m"
}

func forcedColorProfile() (colorprofile.Profile, bool) {
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return colorprofile.ANSI256, true
	}
	return 0, false
}

func main() {
	seconds := flag.Float64("seconds", 0, "auto-quit after N seconds (0 = interactive)")
	flag.Parse()
	m, err := newModel(*seconds)
	if err != nil {
		fmt.Fprintln(os.Stderr, "astronaut:", err)
		os.Exit(1)
	}
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "astronaut:", err)
		os.Exit(1)
	}
}
