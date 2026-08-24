package editor

import (
	"fmt"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/lander-lab/sprite"
)

// Window is one vim-style split.
type Window int

const (
	WinCanvas Window = iota
	WinPalette
	WinFrames
)

func (w Window) String() string {
	switch w {
	case WinCanvas:
		return "canvas"
	case WinPalette:
		return "palette"
	case WinFrames:
		return "frames"
	default:
		return "?"
	}
}

type cellKey struct{ R, C int }

// Model is the lander sprite editor.
type Model struct {
	Atlas   *sprite.Atlas
	Size    sprite.Size
	Heading sprite.Heading
	Path    string

	Win     Window
	PalIdx  int
	CursorR int
	CursorC int
	TermW   int
	TermH   int

	CanvasX int
	CanvasY int

	pendingWin bool
	sel        map[cellKey]bool
	err        string
	status     string
}

// New boots the editor on an atlas. path is where :w / Save writes.
func New(a *sprite.Atlas, path string) Model {
	if a == nil {
		a = sprite.Default()
	}
	pal := 1
	if len(a.Palette) == 0 {
		pal = -1
	} else if len(a.Palette) == 1 {
		pal = 0
	}
	return Model{
		Atlas:    a,
		Size:     sprite.Size4,
		Heading:  sprite.N,
		Path:     path,
		Win:      WinCanvas,
		PalIdx:   pal,
		TermW:    80,
		TermH:    24,
		CanvasX:  1,
		CanvasY:  2,
		sel:      map[cellKey]bool{},
		status:   "i paint · d delete · f/b fg/bg · ^A/^B shade · space select · ^W hjkl windows · q quit",
	}
}

// Current is the sprite under edit.
func (m Model) Current() sprite.Sprite {
	return m.Atlas.MustFrame(m.Size, m.Heading)
}

func (m *Model) setCurrent(sp sprite.Sprite) {
	m.Atlas.SetFrame(m.Size, m.Heading, sp)
}

// Selected reports whether a canvas cell is in the selection.
func (m Model) Selected(r, c int) bool {
	return m.sel[cellKey{r, c}]
}

func (m *Model) SetStatus(s string) {
	m.status = s
	m.err = ""
}

func (m *Model) SetErr(s string) { m.err = s }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.TermW, m.TermH = msg.Width, msg.Height
		return m, nil
	case tea.MouseMsg:
		m.handleMouse(msg)
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.pendingWin {
		m.pendingWin = false
		switch msg.Type {
		case tea.KeyEsc:
			return m, nil
		}
		r := runeFrom(msg)
		switch unicode.ToLower(r) {
		case 'h':
			m.Win = WinCanvas
		case 'l':
			if m.Win == WinCanvas {
				m.Win = WinPalette
			}
		case 'j':
			if m.Win == WinPalette || m.Win == WinCanvas {
				m.Win = WinFrames
			}
		case 'k':
			if m.Win == WinFrames {
				m.Win = WinPalette
			}
		}
		return m, nil
	}

	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyCtrlW:
		m.pendingWin = true
		return m, nil
	case tea.KeyEsc:
		m.sel = map[cellKey]bool{}
		return m, nil
	case tea.KeyCtrlA:
		if m.Win == WinCanvas {
			m.shade(+1)
		}
		return m, nil
	case tea.KeyCtrlB:
		if m.Win == WinCanvas {
			m.shade(-1)
		}
		return m, nil
	case tea.KeySpace:
		m.space()
		return m, nil
	}

	r := runeFrom(msg)
	switch r {
	case 'q':
		return m, tea.Quit
	case 'h', 'j', 'k', 'l':
		m.move(r)
	case 'i', 'I':
		if m.Win == WinCanvas {
			m.paint('i')
		}
	case 'd', 'D', 'x':
		if m.Win == WinCanvas {
			m.paint('d')
		}
	case 'f':
		if m.Win == WinCanvas {
			m.paint('f')
		}
	case 'b':
		if m.Win == WinCanvas {
			m.paint('b')
		}
	}
	return m, nil
}

func runeFrom(msg tea.KeyMsg) rune {
	if len(msg.Runes) == 1 {
		return msg.Runes[0]
	}
	s := msg.String()
	if s == " " {
		return ' '
	}
	if len(s) == 1 {
		return rune(s[0])
	}
	return 0
}

func (m *Model) space() {
	switch m.Win {
	case WinCanvas:
		k := cellKey{m.CursorR, m.CursorC}
		if m.sel[k] {
			delete(m.sel, k)
		} else {
			m.sel[k] = true
		}
	case WinPalette:
		// space selects the highlighted palette entry — PalIdx already
		// follows hjkl, so this is a confirm no-op besides status.
		if m.PalIdx >= 0 && m.PalIdx < len(m.Atlas.Palette) {
			m.status = "color " + m.Atlas.Palette[m.PalIdx].Name
		}
	case WinFrames:
		// space confirms the highlighted frame (already current)
	}
}

func (m *Model) move(r rune) {
	switch m.Win {
	case WinCanvas:
		sp := m.Current()
		switch r {
		case 'h':
			if m.CursorC > 0 {
				m.CursorC--
			}
		case 'l':
			if m.CursorC+1 < sp.Width {
				m.CursorC++
			}
		case 'k':
			if m.CursorR > 0 {
				m.CursorR--
			}
		case 'j':
			if m.CursorR+1 < sp.Height {
				m.CursorR++
			}
		}
	case WinPalette:
		n := len(m.Atlas.Palette)
		if n == 0 {
			return
		}
		switch r {
		case 'k', 'h':
			m.PalIdx = (m.PalIdx - 1 + n) % n
		case 'j', 'l':
			m.PalIdx = (m.PalIdx + 1) % n
		}
	case WinFrames:
		switch r {
		case 'h':
			m.Heading = prevHeading(m.Heading)
			m.clampCursor()
			m.sel = map[cellKey]bool{}
		case 'l':
			m.Heading = nextHeading(m.Heading)
			m.clampCursor()
			m.sel = map[cellKey]bool{}
		case 'k':
			if m.Size > sprite.Size1 {
				m.Size--
			}
			m.clampCursor()
			m.sel = map[cellKey]bool{}
		case 'j':
			if m.Size < sprite.Size4 {
				m.Size++
			}
			m.clampCursor()
			m.sel = map[cellKey]bool{}
		}
	}
}

func (m *Model) clampCursor() {
	sp := m.Current()
	if m.CursorR >= sp.Height {
		m.CursorR = sp.Height - 1
	}
	if m.CursorC >= sp.Width {
		m.CursorC = sp.Width - 1
	}
	if m.CursorR < 0 {
		m.CursorR = 0
	}
	if m.CursorC < 0 {
		m.CursorC = 0
	}
}

func nextHeading(h sprite.Heading) sprite.Heading {
	for i, x := range sprite.Headings {
		if x == h {
			return sprite.Headings[(i+1)%len(sprite.Headings)]
		}
	}
	return sprite.N
}

func prevHeading(h sprite.Heading) sprite.Heading {
	for i, x := range sprite.Headings {
		if x == h {
			return sprite.Headings[(i-1+len(sprite.Headings))%len(sprite.Headings)]
		}
	}
	return sprite.N
}

func (m *Model) targets() []cellKey {
	if len(m.sel) > 0 {
		out := make([]cellKey, 0, len(m.sel))
		for k := range m.sel {
			out = append(out, k)
		}
		return out
	}
	return []cellKey{{m.CursorR, m.CursorC}}
}

func (m *Model) pal() *sprite.PaletteEntry {
	if m.PalIdx < 0 || m.PalIdx >= len(m.Atlas.Palette) {
		return nil
	}
	return &m.Atlas.Palette[m.PalIdx]
}

func (m *Model) paint(mode rune) {
	if mode != 'd' && m.pal() == nil {
		return
	}
	sp := cloneSprite(m.Current())
	for _, k := range m.targets() {
		c := sp.At(k.R, k.C)
		switch mode {
		case 'd':
			c = sprite.Cell{Ch: ' ', FG: -1, BG: -1}
		case 'i':
			p := m.pal()
			if c.Ch == ' ' {
				c.Ch = '█'
			}
			c.FG, c.BG = p.FG, p.BG
		case 'f':
			p := m.pal()
			if c.Ch == ' ' {
				c.Ch = '█'
			}
			c.FG = p.FG
		case 'b':
			p := m.pal()
			if c.Ch == ' ' {
				c.Ch = '█'
			}
			c.BG = p.BG
		}
		sp.Set(k.R, k.C, c)
	}
	m.setCurrent(sp)
}

func (m *Model) shade(dir int) {
	sp := cloneSprite(m.Current())
	p := m.pal()
	for _, k := range m.targets() {
		c := sp.At(k.R, k.C)
		wasEmpty := c.Transparent()
		if dir > 0 {
			c = sprite.IncrementShade(c)
			if wasEmpty && p != nil {
				c.FG = p.FG
				c.BG = p.BG
			}
		} else {
			c = sprite.DecrementShade(c)
		}
		sp.Set(k.R, k.C, c)
	}
	m.setCurrent(sp)
}

func cloneSprite(s sprite.Sprite) sprite.Sprite {
	out := sprite.New(s.Width, s.Height)
	for r := 0; r < s.Height; r++ {
		copy(out.Cells[r], s.Cells[r])
	}
	return out
}

func (m *Model) handleMouse(msg tea.MouseMsg) {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return
	}
	x, y := msg.X, msg.Y
	sp := m.Current()
	// canvas
	cx, cy := m.CanvasX, m.CanvasY
	if x >= cx && x < cx+sp.Width && y >= cy && y < cy+sp.Height {
		m.Win = WinCanvas
		m.CursorC = x - cx
		m.CursorR = y - cy
		m.clampCursor()
		return
	}
	// clicks off the canvas must not invent an out-of-range cursor
	m.clampCursor()
}

// Save writes the atlas JSON to Path.
func (m Model) Save() error {
	if m.Path == "" {
		return fmt.Errorf("no path to save")
	}
	return m.Atlas.WriteFile(m.Path)
}

func (m Model) View() string {
	if m.TermW < 1 {
		m.TermW = 80
	}
	if m.TermH < 1 {
		m.TermH = 24
	}
	sp := m.Current()
	title := fmt.Sprintf("LM EDITOR  size %d  heading %s  %s", m.Size, m.Heading, m.Path)
	if m.Path == "" {
		title = fmt.Sprintf("LM EDITOR  size %d  heading %s", m.Size, m.Heading)
	}

	canvas := renderCanvas(sp, m)
	palette := renderPalette(m)
	frames := renderFrames(m)
	sidebar := joinVert(palette, frames)

	body := joinHoriz(canvas, sidebar)
	status := m.status
	if m.err != "" {
		status = m.err
	}
	cur := sp.At(m.CursorR, m.CursorC)
	meta := fmt.Sprintf("cell (%d,%d) %q fg %d bg %d  [%s]",
		m.CursorR, m.CursorC, string(cur.Ch), cur.FG, cur.BG, m.Win)

	return title + "\n" + body + "\n" + meta + "\n" + status
}

func renderCanvas(sp sprite.Sprite, m Model) string {
	inner := make([]string, sp.Height)
	for r := 0; r < sp.Height; r++ {
		var b strings.Builder
		for c := 0; c < sp.Width; c++ {
			cell := sp.At(r, c)
			ch := cell.Ch
			if ch == 0 || ch == ' ' {
				if m.Selected(r, c) || (m.Win == WinCanvas && r == m.CursorR && c == m.CursorC) {
					ch = '·'
				} else {
					ch = ' '
				}
			}
			s := string(ch)
			if m.Selected(r, c) {
				s = "\x1b[7m" + s + "\x1b[0m"
			} else if m.Win == WinCanvas && r == m.CursorR && c == m.CursorC {
				s = "\x1b[7m" + s + "\x1b[0m"
			} else if !cell.Transparent() {
				s = sprite.Render(oneCell(cell))
			}
			b.WriteString(s)
		}
		inner[r] = b.String()
	}
	label := fmt.Sprintf(" canvas %dx%d %s ", sp.Width, sp.Height, m.Heading)
	return box(label, inner, sp.Width)
}

func oneCell(c sprite.Cell) sprite.Sprite {
	s := sprite.New(1, 1)
	s.Set(0, 0, c)
	return s
}

func renderPalette(m Model) string {
	lines := make([]string, len(m.Atlas.Palette))
	for i, p := range m.Atlas.Palette {
		mark := "  "
		if i == m.PalIdx {
			mark = "> "
		}
		swatch := " "
		if p.FG >= 0 {
			cell := sprite.Cell{Ch: '█', FG: p.FG, BG: p.BG}
			swatch = sprite.Render(oneCell(cell))
		}
		bg := "-"
		if p.BG >= 0 {
			bg = fmt.Sprintf("%d", p.BG)
		}
		lines[i] = fmt.Sprintf("%s%s %-7s fg %-3d bg %s", mark, swatch, p.Name, p.FG, bg)
	}
	if len(lines) == 0 {
		lines = []string{"(empty)"}
	}
	w := 28
	for i := range lines {
		lines[i] = padPlain(lines[i], w)
	}
	focus := ""
	if m.Win == WinPalette {
		focus = "*"
	}
	return box(focus+" palette ", lines, w)
}

func renderFrames(m Model) string {
	var sizes []string
	for _, sz := range sprite.Sizes {
		s := fmt.Sprintf("%d", sz)
		if sz == m.Size {
			s = "[" + s + "]"
		}
		sizes = append(sizes, s)
	}
	var heads []string
	for _, h := range sprite.Headings {
		s := string(h)
		if h == m.Heading {
			s = "[" + s + "]"
		}
		heads = append(heads, s)
	}
	lines := []string{
		"size " + strings.Join(sizes, " "),
		" " + strings.Join(heads, " "),
		"shrink: 4 → 3 → 2 → 1",
	}
	w := 28
	for i := range lines {
		lines[i] = padPlain(lines[i], w)
	}
	focus := ""
	if m.Win == WinFrames {
		focus = "*"
	}
	return box(focus+" frames ", lines, w)
}

func box(title string, inner []string, innerW int) string {
	if innerW < 1 {
		innerW = 1
	}
	top := "╭" + padTitle(title, innerW) + "╮"
	bot := "╰" + strings.Repeat("─", innerW) + "╯"
	var b strings.Builder
	b.WriteString(top)
	b.WriteByte('\n')
	for _, line := range inner {
		b.WriteString("│")
		b.WriteString(line)
		b.WriteString("│\n")
	}
	b.WriteString(bot)
	return b.String()
}

func padTitle(title string, innerW int) string {
	t := []rune(title)
	if len(t) > innerW {
		t = t[:innerW]
	}
	dash := innerW - len(t)
	left := dash / 2
	right := dash - left
	return strings.Repeat("─", left) + string(t) + strings.Repeat("─", right)
}

func padPlain(s string, w int) string {
	n := visibleLen(s)
	if n >= w {
		return s
	}
	return s + strings.Repeat(" ", w-n)
}

func visibleLen(s string) int {
	plain := strip(s)
	return len([]rune(plain))
}

func strip(s string) string {
	var b strings.Builder
	i := 0
	rs := []rune(s)
	for i < len(rs) {
		if rs[i] == 0x1b && i+1 < len(rs) && rs[i+1] == '[' {
			i += 2
			for i < len(rs) && rs[i] != 'm' {
				i++
			}
			i++
			continue
		}
		b.WriteRune(rs[i])
		i++
	}
	return b.String()
}

func joinHoriz(left, right string) string {
	ls := strings.Split(left, "\n")
	rs := strings.Split(right, "\n")
	n := len(ls)
	if len(rs) > n {
		n = len(rs)
	}
	lw := 0
	for _, l := range ls {
		if w := visibleLen(l); w > lw {
			lw = w
		}
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		l, r := "", ""
		if i < len(ls) {
			l = ls[i]
		}
		if i < len(rs) {
			r = rs[i]
		}
		b.WriteString(padPlain(l, lw))
		b.WriteByte(' ')
		b.WriteString(r)
		if i+1 < n {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func joinVert(top, bot string) string {
	return top + "\n" + bot
}
