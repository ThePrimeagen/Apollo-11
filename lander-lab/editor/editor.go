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
	WinSymbols
	WinPalette
	WinFrames
)

func (w Window) String() string {
	switch w {
	case WinCanvas:
		return "canvas"
	case WinSymbols:
		return "symbols"
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

	Brush        Swatch
	PaintCh      rune
	SymIdx       int
	Inserting    bool
	RecentColors []Swatch
	RecentGlyphs []rune
	PickerOpen   bool
	PickerIdx    int
	PickerCube   bool
	CubeRed      int
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
		Atlas:        a,
		Size:         sprite.Size4,
		Heading:      sprite.N,
		Path:         path,
		Win:          WinCanvas,
		PalIdx:       pal,
		TermW:        80,
		TermH:        24,
		CanvasX:      1,
		CanvasY:      2,
		sel:          map[cellKey]bool{},
		Brush:        Swatch{FG: 252, BG: -1},
		PaintCh:      '█',
		SymIdx:       0,
		RecentColors: seedColors(),
		RecentGlyphs: seedGlyphs(),
		PickerIdx:    23,
		status:       "i insert one  P paste  p cycle  1-0 colors  !@# glyphs  c 8-bit  d delete  ^W hjkl  q quit",
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
			m.winLeft()
		case 'l':
			m.winRight()
		case 'j':
			m.winDown()
		case 'k':
			m.winUp()
		}
		return m, nil
	}

	if m.PickerOpen {
		return m.handlePickerKey(msg)
	}

	if m.Inserting {
		return m.handleInsert(msg)
	}

	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyCtrlW:
		m.pendingWin = true
		return m, nil
	case tea.KeyEsc:
		m.sel = map[cellKey]bool{}
		m.Inserting = false
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
	if m.applyColorKey(r) || m.applyGlyphKey(r) {
		return m, nil
	}
	switch r {
	case 'q':
		return m, tea.Quit
	case 'c':
		m.openPicker()
	case 'p':
		m.cyclePaint()
	case 'P':
		m.pasteSymbol()
	case 'h', 'j', 'k', 'l':
		m.move(r)
	case 'i', 'I':
		m.Inserting = true
		m.status = "-- INSERT one char (esc cancel) --"
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

func (m *Model) winRight() {
	switch m.Win {
	case WinCanvas:
		m.Win = WinSymbols
	case WinSymbols:
		m.Win = WinPalette
	}
}

func (m *Model) winLeft() {
	switch m.Win {
	case WinPalette, WinFrames:
		m.Win = WinSymbols
	case WinSymbols:
		m.Win = WinCanvas
	}
}

func (m *Model) winDown() {
	switch m.Win {
	case WinCanvas, WinSymbols:
		m.Win = WinPalette
	case WinPalette:
		m.Win = WinFrames
	}
}

func (m *Model) winUp() {
	switch m.Win {
	case WinFrames:
		m.Win = WinPalette
	case WinPalette:
		m.Win = WinSymbols
	}
}

func (m Model) handleInsert(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.Inserting = false
		m.status = "insert cancelled"
		return m, nil
	case tea.KeyCtrlC:
		return m, tea.Quit
	}
	r := runeFrom(msg)
	if r == 0 {
		m.Inserting = false
		m.status = "insert cancelled"
		return m, nil
	}
	m.paintGlyph(r)
	m.PaintCh = r
	m.syncSymIdx()
	m.Inserting = false
	m.status = fmt.Sprintf("inserted %s", string(r))
	return m, nil
}

func (m Model) handlePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.closePicker(false)
		return m, nil
	case tea.KeySpace, tea.KeyEnter:
		m.closePicker(true)
		return m, nil
	case tea.KeyCtrlC:
		return m, tea.Quit
	}
	r := runeFrom(msg)
	switch r {
	case 'c', 'q':
		if r == 'q' {
			return m, tea.Quit
		}
		m.closePicker(false)
	case 'h', 'j', 'k', 'l', '[', ']', 'g':
		m.movePicker(r)
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
		if m.PalIdx >= 0 && m.PalIdx < len(m.Atlas.Palette) {
			p := m.Atlas.Palette[m.PalIdx]
			m.Brush = Swatch{FG: p.FG, BG: p.BG}
			m.RecentColors = rememberSwatch(m.RecentColors, m.Brush, 10)
			m.status = "color " + p.Name
		}
	case WinFrames:
	case WinSymbols:
		if m.SymIdx >= 0 && m.SymIdx < len(SymbolList) {
			m.PaintCh = SymbolList[m.SymIdx].Ch
			m.status = fmt.Sprintf("symbol %s", SymbolList[m.SymIdx].Name)
		}
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
		if m.PalIdx >= 0 && m.PalIdx < n {
			p := m.Atlas.Palette[m.PalIdx]
			m.Brush = Swatch{FG: p.FG, BG: p.BG}
		}
	case WinSymbols:
		n := len(SymbolList)
		if n == 0 {
			return
		}
		switch r {
		case 'k', 'h':
			m.SymIdx = (m.SymIdx - 1 + n) % n
		case 'j', 'l':
			m.SymIdx = (m.SymIdx + 1) % n
		}
		m.PaintCh = SymbolList[m.SymIdx].Ch
		m.status = fmt.Sprintf("symbol %s", SymbolList[m.SymIdx].Name)
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

func (m *Model) color() Swatch {
	if m.PalIdx >= 0 && m.PalIdx < len(m.Atlas.Palette) {
		p := m.Atlas.Palette[m.PalIdx]
		return Swatch{FG: p.FG, BG: p.BG}
	}
	return m.Brush
}

func (m *Model) paint(mode rune) {
	sp := cloneSprite(m.Current())
	col := m.color()
	for _, k := range m.targets() {
		c := sp.At(k.R, k.C)
		switch mode {
		case 'd':
			c = sprite.Cell{Ch: ' ', FG: -1, BG: -1}
		case 'f':
			if c.Ch == ' ' {
				c.Ch = m.PaintCh
				if c.Ch == 0 || c.Ch == ' ' {
					c.Ch = '█'
				}
			}
			c.FG = col.FG
			m.RecentColors = rememberSwatch(m.RecentColors, col, 10)
		case 'b':
			if c.Ch == ' ' {
				c.Ch = m.PaintCh
				if c.Ch == 0 || c.Ch == ' ' {
					c.Ch = '█'
				}
			}
			c.BG = col.BG
			m.RecentColors = rememberSwatch(m.RecentColors, col, 10)
		}
		sp.Set(k.R, k.C, c)
	}
	m.setCurrent(sp)
}

func (m *Model) pasteSymbol() {
	if len(SymbolList) == 0 {
		return
	}
	if m.SymIdx < 0 || m.SymIdx >= len(SymbolList) {
		m.SymIdx = 0
	}
	m.PaintCh = SymbolList[m.SymIdx].Ch
	m.paintGlyph(m.PaintCh)
}

func (m *Model) paintGlyph(ch rune) {
	sp := cloneSprite(m.Current())
	col := m.color()
	if ch == 0 {
		ch = '█'
	}
	for _, k := range m.targets() {
		c := sp.At(k.R, k.C)
		if ch == ' ' {
			c = sprite.Cell{Ch: ' ', FG: -1, BG: -1}
		} else {
			c.Ch = ch
			c.FG, c.BG = col.FG, col.BG
			m.RecentGlyphs = rememberGlyph(m.RecentGlyphs, ch, 10)
			m.RecentColors = rememberSwatch(m.RecentColors, col, 10)
		}
		sp.Set(k.R, k.C, c)
	}
	m.setCurrent(sp)
}

func (m *Model) shade(dir int) {
	sp := cloneSprite(m.Current())
	for _, k := range m.targets() {
		c := sp.At(k.R, k.C)
		wasEmpty := c.Transparent()
		if dir > 0 {
			c = sprite.IncrementShade(c)
			if wasEmpty {
				col := m.color()
				c.FG = col.FG
				c.BG = col.BG
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
	// symbols list sits in the right sidebar, starting at the same row as
	// the canvas box (title is line 0, box top is line 1, inner is line 2).
	sideX := sp.Width + 2 + 1
	if x >= sideX {
		rows := symbolRows()
		// header line inside the symbols box is "full · half · quarter"
		innerY := y - 2
		if innerY >= 1 && innerY < 1+len(rows) {
			row := rows[innerY-1]
			m.Win = WinSymbols
			if row.idx >= 0 {
				m.SymIdx = row.idx
				m.PaintCh = SymbolList[row.idx].Ch
			}
			return
		}
		if innerY == 0 || (y >= 1 && y < 2+1+len(rows)) {
			m.Win = WinSymbols
			return
		}
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
	symbols := renderSymbols(m)
	palette := renderPalette(m)
	frames := renderFrames(m)
	sidebar := joinVert(joinVert(symbols, palette), frames)

	body := joinHoriz(canvas, sidebar)
	status := m.status
	if m.Inserting && status == "" {
		status = "-- INSERT one char (esc cancel) --"
	}
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

func symbolRows() []symRow {
	var rows []symRow
	last := ""
	for i, s := range SymbolList {
		if s.Kind != last {
			last = s.Kind
			rows = append(rows, symRow{idx: -1, text: " " + s.Kind})
		}
		rows = append(rows, symRow{idx: i, text: fmt.Sprintf(" %c %s", s.Ch, s.Name)})
	}
	return rows
}

type symRow struct {
	idx  int
	text string
}

func renderSymbols(m Model) string {
	const w = 22
	rows := symbolRows()
	lines := make([]string, 0, len(rows)+1)
	lines = append(lines, padPlain("full · half · quarter", w))
	for _, row := range rows {
		text := row.text
		if row.idx == m.SymIdx {
			text = "\x1b[7m>" + strings.TrimPrefix(row.text, " ") + "\x1b[0m"
		}
		lines = append(lines, padPlain(text, w))
	}
	focus := ""
	if m.Win == WinSymbols {
		focus = "*"
	}
	return box(focus+" symbols ", lines, w)
}

func oneCell(c sprite.Cell) sprite.Sprite {
	s := sprite.New(1, 1)
	s.Set(0, 0, c)
	return s
}

func renderPalette(m Model) string {
	const w = 36
	var lines []string
	lines = append(lines, padPlain("colors  1-0 clutch", w))
	lines = append(lines, padPlain(renderClutch(m), w))
	lines = append(lines, padPlain("paints  !@# $%^ &*( )", w))
	lines = append(lines, padPlain(renderGlyphs(m), w))
	if m.PickerOpen {
		lines = append(lines, padPlain(fmt.Sprintf("8-bit ▾  %s", pickerLabel(m)), w))
		lines = append(lines, renderPickerGrid(m, w)...)
	} else {
		sw := swatchCell(m.color())
		lines = append(lines, padPlain(fmt.Sprintf("8-bit ▸  %s fg %-3d  c opens", sw, m.color().FG), w))
	}
	lines = append(lines, padPlain("named", w))
	for i, p := range m.Atlas.Palette {
		mark := "  "
		if i == m.PalIdx {
			mark = "> "
		}
		swatch := " "
		if p.FG >= 0 {
			swatch = sprite.Render(oneCell(sprite.Cell{Ch: '█', FG: p.FG, BG: p.BG}))
		}
		bg := "-"
		if p.BG >= 0 {
			bg = fmt.Sprintf("%d", p.BG)
		}
		lines = append(lines, padPlain(fmt.Sprintf("%s%s %-7s fg %-3d bg %s", mark, swatch, p.Name, p.FG, bg), w))
	}
	if len(m.Atlas.Palette) == 0 {
		lines = append(lines, padPlain("(empty)", w))
	}
	focus := ""
	if m.Win == WinPalette {
		focus = "*"
	}
	return box(focus+" palette ", lines, w)
}

func swatchCell(s Swatch) string {
	if s.FG < 0 && s.BG < 0 {
		return " "
	}
	return sprite.Render(oneCell(sprite.Cell{Ch: '█', FG: s.FG, BG: s.BG}))
}

func renderClutch(m Model) string {
	var b strings.Builder
	for i, k := range ColorKeys {
		b.WriteRune(k)
		if i < len(m.RecentColors) {
			b.WriteString(swatchCell(m.RecentColors[i]))
		} else {
			b.WriteByte('.')
		}
		if i+1 < len(ColorKeys) {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

func renderGlyphs(m Model) string {
	var b strings.Builder
	for i, k := range GlyphKeys {
		b.WriteRune(k)
		ch := DefaultGlyphs[i]
		mark := string(ch)
		if m.PaintCh == ch {
			mark = "\x1b[7m" + mark + "\x1b[0m"
		}
		b.WriteString(mark)
		if i+1 < len(GlyphKeys) {
			b.WriteByte(' ')
		}
	}
	b.WriteString("  more ")
	for _, ch := range ExtraGlyphs {
		if m.PaintCh == ch {
			b.WriteString("\x1b[7m")
			b.WriteRune(ch)
			b.WriteString("\x1b[0m")
		} else {
			b.WriteRune(ch)
		}
	}
	if len(m.RecentGlyphs) > 0 {
		b.WriteString("  past ")
		n := len(m.RecentGlyphs)
		if n > 6 {
			n = 6
		}
		for i := 0; i < n; i++ {
			b.WriteRune(m.RecentGlyphs[i])
		}
	}
	return b.String()
}

func pickerLabel(m Model) string {
	if m.PickerCube {
		return fmt.Sprintf("cube r=%d  [ ] slice", m.CubeRed)
	}
	fg := 255
	if m.PickerIdx >= 0 && m.PickerIdx < len(Greys) {
		fg = Greys[m.PickerIdx]
	}
	return fmt.Sprintf("grey %d  g greys", fg)
}

func renderPickerGrid(m Model, w int) []string {
	var lines []string
	if m.PickerCube {
		var row strings.Builder
		for i := 0; i < 36; i++ {
			if i > 0 && i%6 == 0 {
				lines = append(lines, padPlain(row.String(), w))
				row.Reset()
			}
			fg := cubeColor(m.CubeRed, i)
			cell := sprite.Render(oneCell(sprite.Cell{Ch: '█', FG: fg, BG: -1}))
			if i == m.PickerIdx {
				cell = "\x1b[7m" + cell + "\x1b[0m"
			}
			row.WriteString(cell)
		}
		if row.Len() > 0 {
			lines = append(lines, padPlain(row.String(), w))
		}
		return lines
	}
	var row strings.Builder
	for i, g := range Greys {
		if i > 0 && i%12 == 0 {
			lines = append(lines, padPlain(row.String(), w))
			row.Reset()
		}
		cell := sprite.Render(oneCell(sprite.Cell{Ch: '█', FG: g, BG: -1}))
		if i == m.PickerIdx {
			cell = "\x1b[7m" + cell + "\x1b[0m"
		}
		row.WriteString(cell)
	}
	if row.Len() > 0 {
		lines = append(lines, padPlain(row.String(), w))
	}
	return lines
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
	w := 36
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
