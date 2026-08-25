// Package adjustflame is the TUI that edits fire heat thresholds.
// j/k walk the rungs, h/l change the selected amount (0..500), s
// writes the JSON config and quits. The same page plays all eight
// headings so the runner is the demo.
package adjustflame

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/lander-lab/components/fire"
	"github.com/theprimeagen/apollo-11/lander-lab/sprite"
)

const (
	ListCols = 36
	PageCols = ListCols + fire.CompassCols
	PageRows = 1 + fire.CompassRows
	fps      = 20
)

// TickMsg advances the eight plumes one frame.
type TickMsg struct{}

// Model is the adjusting-flame item: sliders plus the live rose.
type Model struct {
	Path       string
	Thresholds []int
	Cursor     int
	Err        string
	Saved      bool
	Rose       *fire.Compass
}

// Open reads the JSON config at path. A missing or invalid file is an error.
func Open(path string) (Model, error) {
	c, err := fire.LoadHeat(path)
	if err != nil {
		return Model{}, err
	}
	m := Model{
		Path:       path,
		Thresholds: append([]int(nil), c.Thresholds...),
		Rose:       fire.NewCompass(1),
	}
	m.applyHeat()
	return m, nil
}

func (m Model) Init() tea.Cmd { return tick() }

func tick() tea.Cmd {
	return tea.Tick(time.Second/fps, func(time.Time) tea.Msg { return TickMsg{} })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case TickMsg:
		m.applyHeat()
		if m.Rose != nil {
			m.Rose.Update(1.0 / float64(fps))
		}
		return m, tick()
	case tea.KeyPressMsg:
		switch strings.ToLower(msg.String()) {
		case "j", "down":
			if m.Cursor < len(m.Thresholds)-1 {
				m.Cursor++
			}
		case "k", "up":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "l", "right":
			m = m.nudge(1)
		case "h", "left":
			m = m.nudge(-1)
		case "s":
			if err := m.save(); err != nil {
				m.Err = err.Error()
				m.Saved = false
				return m, nil
			}
			m.Err = ""
			m.Saved = true
			return m, tea.Quit
		}
		m.applyHeat()
	}
	return m, nil
}

func (m Model) nudge(delta int) Model {
	if m.Cursor < 0 || m.Cursor >= len(m.Thresholds) {
		return m
	}
	n := m.Thresholds[m.Cursor] + delta
	if n < fire.MinThreshold {
		n = fire.MinThreshold
	}
	if n > fire.MaxThreshold {
		n = fire.MaxThreshold
	}
	m.Thresholds[m.Cursor] = n
	return m
}

func (m Model) save() error {
	return fire.HeatConfig{Thresholds: append([]int(nil), m.Thresholds...)}.Save(m.Path)
}

func (m Model) applyHeat() {
	_ = fire.UseHeat(fire.HeatConfig{Thresholds: append([]int(nil), m.Thresholds...)})
}

func (m Model) View() tea.View {
	v := tea.NewView(sprite.Render(m.Page()))
	v.AltScreen = true
	return v
}

// Page is the runner canvas: threshold sliders on the left, all eight
// headings playing on the right.
func (m Model) Page() sprite.Sprite {
	m.applyHeat()
	board := sprite.New(PageCols, PageRows)
	put(board, 0, 0, "adjust flame  +15%  j/k select  h/l change  s save+quit", 250)
	if m.Rose != nil {
		blit(board, 1, ListCols, m.Rose.View())
	}
	rungs := fire.Bands()
	if len(m.Thresholds) == 0 {
		put(board, 1, 0, "(no thresholds)", 244)
		return board
	}
	for i, n := range m.Thresholds {
		mark := " "
		if i == m.Cursor {
			mark = ">"
		}
		name, glyph := "?", ' '
		if i < len(rungs) {
			name = rungs[i].Name
			glyph = rungs[i].Glyph
		}
		fg := 250
		if i == m.Cursor {
			fg = 229
		}
		put(board, 1+i, 0, fmt.Sprintf("%s %c  %-20s %3d", mark, glyph, name, n), fg)
	}
	if m.Err != "" {
		put(board, PageRows-1, 0, "error: "+m.Err, 196)
	}
	return board
}

// WriteTape writes n PNG frames of the runner page at 20 fps.
func (m Model) WriteTape(dir string, n, cellW int) ([]string, error) {
	if n <= 0 {
		return nil, fmt.Errorf("need at least one frame")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	paths := make([]string, 0, n)
	for i := 0; i < n; i++ {
		got, _ := m.Update(TickMsg{})
		m = got.(Model)
		p := filepath.Join(dir, fmt.Sprintf("frame-%04d.png", i))
		if err := fire.WritePNG(p, m.Page(), cellW); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}

func put(sp sprite.Sprite, row, col int, s string, fg int) {
	for i, r := range []rune(s) {
		if col+i >= sp.Width || row < 0 || row >= sp.Height {
			break
		}
		sp.Set(row, col+i, sprite.Cell{Ch: r, FG: fg, BG: -1})
	}
}

func blit(dst sprite.Sprite, row, col int, src sprite.Sprite) {
	for r := 0; r < src.Height; r++ {
		for c := 0; c < src.Width; c++ {
			cell := src.At(r, c)
			if cell.Transparent() {
				continue
			}
			dst.Set(row+r, col+c, cell)
		}
	}
}
