// dskypad: the DSKY panel played as a live keypad — the terminal's
// keys go straight to the component. The pad boots on PROG 63 with a
// blank verb and noun; type V16 N68 and ENTR to wake the descent
// monitor (the registers fill and the ALT/VEL lamps light), any other
// commit puts it back to sleep.
//
//	v / n       VERB / NOUN — open a two-digit entry
//	0-9         digits fill the open entry
//	enter / e   ENTR — commit the entry
//	c / bksp    CLR — cancel the entry
//	r           RSET — extinguish the caution lights
//	q           quit
//
//	go run ./cmd/dskypad
package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	lab "github.com/theprimeagen/apollo-11/dsky-lab/dsky"
	"github.com/theprimeagen/apollo-11/exec-tui/components/dsky"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

const (
	defaultW = 72
	defaultH = 28
	minW     = 10
	minH     = 4
)

type model struct {
	w, h   int
	panel  *dsky.Panel
	screen *screenplay.Screen
}

// newModel boots the pad on PROG 63 with a blank verb and noun. The
// stage is exactly the panel; the window just frames it.
func newModel() model {
	p := dsky.NewPanel(lab.State{Prog: "63"})
	p.Start(dsky.Width, dsky.Height)
	return model{
		w:      defaultW,
		h:      defaultH,
		panel:  p,
		screen: screenplay.NewScreen(dsky.Width, dsky.Height),
	}
}

// Init schedules nothing: the pad is event-driven — the panel keeps no
// clocks, so there is no frame loop to run.
func (m model) Init() tea.Cmd { return nil }

// press feeds one keypad key to the component. ENTR is where the pad
// itself reacts: V16 N68 wakes the descent monitor, any other commit
// puts it back to sleep.
func (m *model) press(k dsky.Key) {
	m.panel.Press(k)
	if k != dsky.KeyEnter {
		return
	}
	st := &m.panel.State
	if st.Verb == "16" && st.Noun == "68" {
		mon := dsky.MonitorState()
		st.R1, st.R2, st.R3, st.Lights = mon.R1, mon.R2, mon.R3, mon.Lights
		return
	}
	st.R1, st.R2, st.R3 = "", "", ""
	st.Lights.Alt, st.Lights.Vel = false, false
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = max(msg.Width, minW), max(msg.Height, minH)
		return m, nil
	case tea.KeyPressMsg:
		switch {
		case msg.String() == "ctrl+c":
			return m, tea.Quit
		case msg.Code == tea.KeyEnter:
			m.press(dsky.KeyEnter)
			return m, nil
		case msg.Code == tea.KeyBackspace:
			m.press(dsky.KeyClear)
			return m, nil
		}
		if rs := []rune(msg.Text); len(rs) == 1 {
			if rs[0] == 'q' {
				return m, tea.Quit
			}
			if k, ok := dsky.KeyFromRune(rs[0]); ok {
				m.press(k)
			}
		}
		return m, nil
	}
	return m, nil
}

// View frames the panel in the middle of the window over one dim hint
// line — always exactly window-height lines.
func (m model) View() tea.View {
	m.screen.Clear()
	m.screen.Blit(0, 0, m.panel.Render())
	placed := lipgloss.Place(m.w, m.h-1, lipgloss.Center, lipgloss.Center, m.screen.Render())
	lines := strings.Split(placed, "\n")
	if len(lines) > m.h-1 {
		lines = lines[:m.h-1]
	}
	for len(lines) < m.h-1 {
		lines = append(lines, "")
	}
	hint := " v verb · n noun · 0-9 · enter entr · c clr · r rset · q quit"
	dim := "\x1b[38;5;240m"
	reset := "\x1b[0m"
	v := tea.NewView(strings.Join(lines, "\n") + "\n" + dim + pad(hint, m.w) + reset)
	v.AltScreen = true
	return v
}

func pad(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return string(r[:w])
	}
	return s + strings.Repeat(" ", w-len(r))
}

func main() {
	var opts []tea.ProgramOption
	if os.Getenv("CLICOLOR_FORCE") != "" {
		opts = append(opts, tea.WithColorProfile(colorprofile.ANSI256))
	}
	if _, err := termreset.Run(newModel(), opts...); err != nil {
		fmt.Fprintln(os.Stderr, "dskypad:", err)
		os.Exit(1)
	}
}
