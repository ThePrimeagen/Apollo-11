// stars-lab: a standalone UI for the reusable starfield component.
//
//	n / p     cycle fly strategies
//	space     pause
//	q         quit
//
//	go run . -strategy dust-rush
//	go run . -strategy hyperspace -seconds 4
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/stars-lab/stars"
)

type demoModel struct {
	w, h    int
	tick    int
	paused  bool
	idx     int
	strats  []stars.Strategy
	seconds float64
	elapsed float64
}

func newDemo(s stars.Strategy) demoModel {
	strats := stars.Strategies()
	idx := 0
	for i, st := range strats {
		if st.Name == s.Name {
			idx = i
			break
		}
	}
	return demoModel{
		w: 72, h: 28,
		strats: strats,
		idx:    idx,
	}
}

func (m demoModel) strategy() stars.Strategy { return m.strats[m.idx] }

func (m *demoModel) advance() {
	if !m.paused {
		m.tick++
	}
}

func strategyOrDefault(name string) stars.Strategy {
	if s, ok := stars.Lookup(name); ok {
		return s
	}
	return stars.DustRush
}

func mustStrategy(name string) stars.Strategy {
	s, ok := stars.Lookup(name)
	if !ok {
		return stars.DustRush
	}
	return s
}

type frameMsg struct{}

const frameMs = 33.34

func tick() tea.Cmd {
	return tea.Tick(time.Duration(frameMs*1e6)*time.Nanosecond, func(time.Time) tea.Msg {
		return frameMsg{}
	})
}

func (m demoModel) Init() tea.Cmd { return tick() }

func (m demoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case frameMsg:
		m.advance()
		m.elapsed += frameMs / 1000
		if m.seconds > 0 && m.elapsed >= m.seconds {
			return m, tea.Quit
		}
		return m, tick()
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if msg.Type == tea.KeySpace {
			m.paused = !m.paused
			return m, nil
		}
		if len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case 'q':
				return m, tea.Quit
			case 'n':
				m.idx = (m.idx + 1) % len(m.strats)
			case 'p':
				m.idx = (m.idx + len(m.strats) - 1) % len(m.strats)
			case ' ':
				m.paused = !m.paused
			}
		}
	}
	return m, nil
}

func (m demoModel) View() string {
	s := m.strategy()
	skyH := m.h - 2
	if skyH < 4 {
		skyH = 4
	}
	if m.w < 10 {
		m.w = 10
	}
	field := stars.Field{
		Width:    m.w,
		Height:   skyH,
		Tick:     m.tick,
		Strategy: s,
	}
	title := fmt.Sprintf("STARFIELD  %s   delays ·%d ˚%d *%d ✦%d",
		s.Name, s.Delay[0], s.Delay[1], s.Delay[2], s.Delay[3])
	if s.Name == stars.Still.Name {
		title = "STARFIELD  still   no motion"
	}
	help := "n/p strategy  space pause  q quit"
	if m.paused {
		help = "PAUSED  " + help
	}
	if len(title) > m.w {
		title = title[:m.w]
	}
	if len(help) > m.w {
		help = help[:m.w]
	}
	title = pad(title, m.w)
	help = pad(help, m.w)
	dim := "\x1b[38;5;240m"
	amber := "\x1b[38;5;214m"
	reset := "\x1b[0m"
	var b strings.Builder
	b.WriteString(amber)
	b.WriteString(title)
	b.WriteString(reset)
	b.WriteString("\n")
	b.WriteString(field.Render())
	b.WriteString("\n")
	b.WriteString(dim)
	b.WriteString(help)
	b.WriteString(reset)
	return b.String()
}

func pad(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return string(r[:w])
	}
	return s + strings.Repeat(" ", w-len(r))
}

func main() {
	name := flag.String("strategy", "dust-rush", "opening fly style")
	seconds := flag.Float64("seconds", 0, "auto-quit after N seconds (0 = interactive)")
	flag.Parse()
	m := newDemo(strategyOrDefault(*name))
	m.seconds = *seconds
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "stars-lab:", err)
		os.Exit(1)
	}
}
