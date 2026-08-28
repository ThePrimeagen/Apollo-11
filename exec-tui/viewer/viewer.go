package viewer

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/terminal-fonts/termfont"
)

const (
	defaultW = 80
	defaultH = 24
	minW     = 8
	minH     = 5
	titleH   = 3
	headerH  = 4
	footerH  = 1
	frameMs  = 1000.0 / 30

	titleInk = 252
	typeInk  = 240
)

// Model is the component viewer: one item on stage, chrome on top.
// F drops the chrome so the item owns every cell; F again restores it.
type Model struct {
	items   []Item
	idx     int
	edit    int
	full    bool
	w, h    int
	preview screenplay.Component
}

// New opens the catalog on item idx, wrapping if needed.
func New(idx int) Model {
	return NewWith(Catalog(), idx)
}

// NewWith opens a custom catalog. An empty list is a quiet viewer —
// nothing to cycle, nothing to edit.
func NewWith(items []Item, idx int) Model {
	m := Model{items: items, edit: -1, w: defaultW, h: defaultH}
	if n := len(items); n > 0 {
		m.idx = ((idx % n) + n) % n
	}
	m.restage()
	return m
}

// Index is the current catalog slot.
func (m Model) Index() int { return m.idx }

// Fullscreen reports whether the chrome is down and the item owns the window.
func (m Model) Fullscreen() bool { return m.full }

// Current is the item now on stage, or zero for an empty catalog.
func (m Model) Current() Item {
	if len(m.items) == 0 || m.idx < 0 || m.idx >= len(m.items) {
		return Item{}
	}
	return m.items[m.idx]
}

// ChosenEdit reports the edit e picked, if any.
func (m Model) ChosenEdit() (Edit, bool) {
	if m.edit < 0 || m.edit >= len(m.items) {
		return Edit{}, false
	}
	it := m.items[m.edit]
	return Edit{Kind: it.Kind, Path: it.Path, Program: it.Program}, true
}

func (m *Model) restage() {
	if m.preview != nil {
		m.preview.Stop()
		m.preview = nil
	}
	if len(m.items) == 0 {
		return
	}
	it := m.items[m.idx]
	if it.spawn == nil {
		return
	}
	pw, ph := m.previewSize()
	m.preview = it.spawn()
	if m.preview != nil {
		m.preview.Start(pw, ph)
	}
}

func (m Model) previewSize() (w, h int) {
	w, h = m.w, m.h
	if !m.full {
		h = m.h - headerH - footerH
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return
}

func (m *Model) resizePreview() {
	if m.preview == nil {
		return
	}
	pw, ph := m.previewSize()
	m.preview.Start(pw, ph)
}

func (m *Model) toggleFull() {
	m.full = !m.full
	m.resizePreview()
}

func (m *Model) cycle(delta int) {
	n := len(m.items)
	if n == 0 {
		return
	}
	m.idx = ((m.idx+delta)%n + n) % n
	m.restage()
}

type frameMsg struct{}

func tick() tea.Cmd {
	ns := float64(frameMs) * 1e6
	return tea.Tick(time.Duration(ns)*time.Nanosecond, func(time.Time) tea.Msg {
		return frameMsg{}
	})
}

func (m Model) Init() tea.Cmd { return tick() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		if m.w < minW {
			m.w = minW
		}
		if m.h < minH {
			m.h = minH
		}
		m.restage()
		return m, nil
	case frameMsg:
		if m.preview != nil {
			m.preview.Update(frameMs / 1000)
		}
		return m, tick()
	case tea.KeyPressMsg:
		s := strings.ToLower(msg.String())
		switch s {
		case "n", "j", "right":
			m.cycle(1)
			return m, nil
		case "p", "k", "left":
			m.cycle(-1)
			return m, nil
		case "e":
			if len(m.items) == 0 {
				return m, nil
			}
			m.edit = m.idx
			if m.preview != nil {
				m.preview.Stop()
			}
			return m, tea.Quit
		case "q", "ctrl+c":
			if m.preview != nil {
				m.preview.Stop()
			}
			return m, tea.Quit
		case "f":
			m.toggleFull()
			return m, nil
		case " ", "space":
			if f, ok := m.preview.(interface{ Fire() bool }); ok {
				f.Fire()
			}
			return m, nil
		}
		switch msg.Code {
		case tea.KeyRight:
			m.cycle(1)
		case tea.KeyLeft:
			m.cycle(-1)
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	body := sprite.Render(m.frame())
	v := tea.NewView(body)
	v.AltScreen = true
	return v
}

func (m Model) frame() sprite.Sprite {
	w, h := m.w, m.h
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	stage := sprite.New(w, h)
	if m.full {
		if m.preview != nil {
			sprite.Blit(stage, 0, 0, m.preview.Render())
		}
		return stage
	}
	it := m.Current()
	if it.Title != "" {
		paintTitle(stage, it.Title)
	}
	if it.Kind != "" {
		paintType(stage, it.Kind.String())
	}
	if m.preview != nil {
		sprite.Blit(stage, 0, headerH, m.preview.Render())
	}
	paintFooter(stage)
	return stage
}

func paintTitle(stage sprite.Sprite, text string) {
	lines, err := termfont.Lines(titleH, text)
	if err != nil || len(lines) == 0 {
		return
	}
	left := (stage.Width - len(lines[0])) / 2
	for r, line := range lines {
		for c, ch := range line {
			if ch == ' ' {
				continue
			}
			stage.Set(r, left+c, sprite.Cell{Ch: ch, FG: titleInk, BG: -1})
		}
	}
}

func paintType(stage sprite.Sprite, kind string) {
	left := (stage.Width - len(kind)) / 2
	for i, ch := range kind {
		stage.Set(titleH, left+i, sprite.Cell{Ch: ch, FG: typeInk, BG: -1})
	}
}

func paintFooter(stage sprite.Sprite) {
	help := " n/p cycle   e edit   f full   q quit"
	row := stage.Height - 1
	if row < 0 {
		return
	}
	for i, ch := range help {
		if i >= stage.Width {
			break
		}
		stage.Set(row, i, sprite.Cell{Ch: ch, FG: typeInk, BG: -1})
	}
}
