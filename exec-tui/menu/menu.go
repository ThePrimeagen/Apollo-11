// Package menu is the exec-tui launcher: a scrollable list of every lab
// and configurator in the repo, grouped by category — the main program,
// the configurators, the demo labs, and the legacy TUIs. j/k (or arrows)
// move over entries (headers are never selectable), enter runs the
// highlighted program, q quits. Running exec-tui with no arguments opens
// this menu instead of the sim — the sim is the LEGACY EXEC entry.
package menu

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Entry is one runnable program. In-process programs (built into the
// exec-tui binary) leave Module and Pkg empty. Programs in this module
// leave Module empty and carry the ./cmd package path `go run` needs
// from the module root. External labs carry the sibling module
// directory name plus the package path inside it. Section is the
// category header the entry renders under; entries with the same
// section must sit together, and an empty section renders as a plain,
// header-less list.
type Entry struct {
	ID      string
	Title   string
	Desc    string
	Module  string
	Pkg     string
	Section string
}

// Catalog lists every runnable program by category: the main program,
// then the config editors, then the demos, then the legacy TUIs.
// Everything this module owns launches out of its own cmd/ folder;
// only the truly separate labs still run as sibling modules.
func Catalog() []Entry {
	return []Entry{
		{ID: "screenplay", Section: "MAIN PROGRAM", Title: "SCREENPLAY", Desc: "the three-scene premiere: arrival, DSKY dock, then THE END", Pkg: "./cmd/premiere"},
		{ID: "flame", Section: "CONFIG", Title: "FLAME CONFIG", Desc: "tune the booster heat rungs (in-process)"},
		{ID: "stars-config", Section: "CONFIG", Title: "STARS CONFIG", Desc: "tune sky density and fly delays per star layer", Pkg: "./cmd/adjuststars/main"},
		{ID: "editor", Section: "CONFIG", Title: "SPRITE EDITOR", Desc: "vim-ish editor for ASCII ships in assets/ (C-p sizes×headings, C-e glyphs)"},
		{ID: "particle", Section: "CONFIG", Title: "PARTICLE CONFIG", Desc: "tune the nyan rainbow trail (bands, life, spawn)", Pkg: "./cmd/adjustparticle/main"},
		{ID: "lander", Section: "DEMO", Title: "LANDER DEMO", Desc: "the continuous descent with alarms at their true moments", Pkg: "./cmd/lander"},
		{ID: "stars", Section: "DEMO", Title: "STARS DEMO", Desc: "browse the starfield fly strategies", Pkg: "./cmd/stars"},
		{ID: "nyan", Section: "DEMO", Title: "NYAN CAT", Desc: "pop-tart cat with a live rainbow particle trail", Pkg: "./cmd/nyan"},
		{ID: "dsky", Section: "DEMO", Title: "DSKY DEMO", Desc: "a lone DSKY with keypad — press 0-9, replay the descent displays", Module: "dsky-lab", Pkg: "."},
		{ID: "button", Section: "DEMO", Title: "BUTTON DEMO", Desc: "the cockpit toggle switch playground", Module: "button-lab", Pkg: "."},
		{ID: "legacy", Section: "LEGACY TUIS", Title: "LEGACY EXEC", Desc: "the AGC Executive sim during the powered descent (in-process)"},
		{ID: "timeline", Section: "LEGACY TUIS", Title: "TIMELINE", Desc: "one 2-second Executive cycle, step by step", Module: "timeline-tui", Pkg: "."},
	}
}

// LocateModule walks up from startDir looking for module/go.mod, so the
// launcher finds its sibling labs whether it runs from the repo root,
// from exec-tui/, or anywhere deeper.
func LocateModule(startDir, module string) (string, error) {
	dir := startDir
	for {
		cand := filepath.Join(dir, module)
		if _, err := os.Stat(filepath.Join(cand, "go.mod")); err == nil {
			return cand, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("module %s not found above %s — run from the apollo-11 checkout", module, startDir)
		}
		dir = parent
	}
}

// ModuleRoot walks up from startDir to this module's own go.mod, so
// in-module programs and their relative config paths work no matter
// how deep the launcher was started.
func ModuleRoot(startDir string) (string, error) {
	dir := startDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod above %s — run from inside the module", startDir)
		}
		dir = parent
	}
}

var (
	sTitle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	sDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sSel    = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	sName   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	sHead   = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true)
	sStatus = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

// chrome is the fixed line budget around the list: two title lines, a
// blank above the list, a blank below it, and the footer.
const chrome = 5

// Model is the launcher menu.
type Model struct {
	entries []Entry
	status  string
	sel     int
	offset  int
	w, h    int
	chosen  int
}

// New builds a menu over the given entries. status, when non-empty, is
// shown under the list — the launch loop feeds errors back through it.
func New(entries []Entry, status string) Model {
	return Model{entries: entries, status: status, w: 80, h: 24, chosen: -1}
}

// Chosen reports the entry picked with enter, if any.
func (m Model) Chosen() (Entry, bool) {
	if m.chosen < 0 || m.chosen >= len(m.entries) {
		return Entry{}, false
	}
	return m.entries[m.chosen], true
}

// row is one rendered list line: a section header, a spacer between
// sections, or an entry (an index into m.entries).
type row struct {
	head  string
	entry int
}

// rows lays the entries out with a header above each section and a
// blank spacer between sections. Sectionless entries stay plain rows,
// so a synthetic flat list renders exactly as before.
func (m Model) rows() []row {
	rs := make([]row, 0, len(m.entries))
	last := ""
	for i, e := range m.entries {
		if e.Section != "" && e.Section != last {
			if len(rs) > 0 {
				rs = append(rs, row{entry: -1})
			}
			rs = append(rs, row{head: e.Section, entry: -1})
		}
		last = e.Section
		rs = append(rs, row{entry: i})
	}
	return rs
}

// selRow is the list row carrying the selected entry.
func (m Model) selRow(rs []row) int {
	for i, r := range rs {
		if r.entry == m.sel {
			return i
		}
	}
	return 0
}

// visible is how many list rows fit under the fixed chrome.
func (m Model) visible() int {
	rows := m.h - chrome
	if m.status != "" {
		rows--
	}
	if rows < 1 {
		rows = 1
	}
	if n := len(m.rows()); rows > n {
		rows = n
	}
	return rows
}

// clampWindow slides the window of list rows so the cursor stays
// visible, pulling a section header along when the cursor sits on the
// first entry of its section.
func (m *Model) clampWindow() {
	rs := m.rows()
	vis := m.visible()
	sel := m.selRow(rs)
	top := sel
	for top > 0 && rs[top-1].entry < 0 {
		top--
	}
	if top < m.offset {
		m.offset = top
	}
	if sel >= m.offset+vis {
		m.offset = sel - vis + 1
	}
	if max := len(rs) - vis; m.offset > max {
		m.offset = max
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.clampWindow()
		return m, nil
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch msg.Code {
		case 'j', tea.KeyDown:
			m.sel = (m.sel + 1) % len(m.entries)
			m.clampWindow()
		case 'k', tea.KeyUp:
			m.sel = (m.sel + len(m.entries) - 1) % len(m.entries)
			m.clampWindow()
		case 'q':
			return m, tea.Quit
		case tea.KeyEnter:
			m.chosen = m.sel
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	var b strings.Builder
	b.WriteString(sTitle.Render("APOLLO-11 LABS") + "\n")
	b.WriteString(sDim.Render("what do you want to run?") + "\n\n")

	titleW := 0
	for _, e := range m.entries {
		if len(e.Title) > titleW {
			titleW = len(e.Title)
		}
	}
	rs := m.rows()
	vis := m.visible()
	for i := m.offset; i < m.offset+vis && i < len(rs); i++ {
		switch r := rs[i]; {
		case r.head != "":
			b.WriteString(sHead.Render(r.head) + "\n")
		case r.entry < 0:
			b.WriteString("\n")
		default:
			e := m.entries[r.entry]
			name := fmt.Sprintf("%-*s", titleW, e.Title)
			if r.entry == m.sel {
				b.WriteString(sSel.Render("▸ "+name) + "  " + sDim.Render(e.Desc) + "\n")
			} else {
				b.WriteString("  " + sName.Render(name) + "  " + sDim.Render(e.Desc) + "\n")
			}
		}
	}

	b.WriteString("\n")
	if m.status != "" {
		b.WriteString(sStatus.Render(m.status) + "\n")
	}
	b.WriteString(sDim.Render("j/k move · enter run · q quit"))

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}
