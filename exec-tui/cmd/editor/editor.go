package editor

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/terminal-fonts/termfont"
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

// Model is the ASCII sprite editor.
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
	toast      string
	toastID    int

	Brush        Swatch
	PaintCh      rune
	SymIdx       int
	Inserting    bool
	RecentColors []Swatch
	RecentGlyphs []rune
	PickerOpen   bool
	PickerIdx    int
	PickerCube   bool
	PickerSystem bool
	CubeRed      int

	AssetsDir string
	Files     []Asset

	FilePickerOpen  bool
	FilePickerIdx   int
	FilePickerQuery string
	atlases         map[string]*sprite.Atlas

	GlyphGridOpen  bool
	GlyphGridDigit rune
	GlyphGridRow   int
	GlyphGridCol   int

	ColorPaletteOpen bool
	ColorPaletteIdx  int

	Layer EditLayer
}

// New boots the editor on an atlas. path is where :w / Save writes.
func New(a *sprite.Atlas, path string) Model {
	if a == nil {
		a = blankAtlas()
	}
	pal := 1
	if len(a.Palette) == 0 {
		pal = -1
	} else if len(a.Palette) == 1 {
		pal = 0
	}
	return Model{
		Atlas:     a,
		Size:      sprite.Size4,
		Heading:   sprite.N,
		Path:      path,
		Win:       WinCanvas,
		PalIdx:    pal,
		TermW:     80,
		TermH:     24,
		CanvasX:   1,
		CanvasY:   2,
		sel:       map[cellKey]bool{},
		Brush:     Swatch{FG: 252, BG: -1},
		PaintCh:   '█',
		SymIdx:    0,
		PickerIdx: 23,
		Layer:     LayerOutline,
		status:    "layer outline  ^H/^L layers  hjkl move  p paste  ^E glyphs  ^K colors  ^P files  s save  q quit",
	}
}

// Current is the sprite under edit. A missing frame is an empty
// footprint, never a panic — a bad atlas must not kill the TUI.
func (m Model) Current() sprite.Sprite {
	if m.Atlas != nil {
		if sp, ok := m.Atlas.Frame(m.Size, m.Heading); ok {
			return sp
		}
	}
	w, h := m.Size.Dim()
	if w < 1 || h < 1 {
		w, h = 1, 1
	}
	return sprite.New(w, h)
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

const saveToastTTL = 5 * time.Second

type saveToastClearMsg struct{ id int }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.TermW, m.TermH = msg.Width, msg.Height
		return m, nil
	case tea.MouseClickMsg:
		m.handleMouse(msg)
		return m, nil
	case saveToastClearMsg:
		if msg.id == m.toastID {
			m.toast = ""
		}
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.pendingWin {
		m.pendingWin = false
		if msg.Code == tea.KeyEsc {
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

	// ctrl-s saves even when a popup would otherwise swallow keys.
	// Plain s stays below so the glyph grid can still use row "s".
	if !m.Inserting && msg.String() == "ctrl+s" {
		return m, m.SaveWithToast()
	}

	if m.PickerOpen {
		return m.handlePickerKey(msg)
	}

	if m.FilePickerOpen {
		return m.handleFilePickerKey(msg)
	}

	if m.GlyphGridOpen {
		return m.handleGlyphGridKey(msg)
	}

	if m.ColorPaletteOpen {
		return m.handleColorPaletteKey(msg)
	}

	if m.Inserting {
		return m.handleInsert(msg)
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+s":
		return m, m.SaveWithToast()
	case "ctrl+w":
		m.pendingWin = true
		return m, nil
	case "ctrl+p":
		m.openFilePicker()
		return m, nil
	case "ctrl+e":
		m.openGlyphGrid()
		return m, nil
	case "ctrl+k":
		m.openColorPalette()
		return m, nil
	case "ctrl+h":
		m.stepLayer(-1)
		return m, nil
	case "ctrl+l":
		m.stepLayer(1)
		return m, nil
	case "esc":
		if m.Win != WinCanvas {
			m.Win = WinCanvas
			m.status = "canvas"
			return m, nil
		}
		m.sel = map[cellKey]bool{}
		m.Inserting = false
		return m, nil
	case "ctrl+a":
		if m.Win == WinCanvas {
			m.shade(+1)
		}
		return m, nil
	case "ctrl+b":
		if m.Win == WinCanvas {
			m.shade(-1)
		}
		return m, nil
	case "left":
		m.moveCursor(-1, 0)
		return m, nil
	case "right":
		m.moveCursor(1, 0)
		return m, nil
	case "up":
		m.moveCursor(0, -1)
		return m, nil
	case "down":
		m.moveCursor(0, 1)
		return m, nil
	case "space":
		m.space()
		return m, nil
	}

	r := runeFrom(msg)
	if m.applyColorKey(r) || m.applyGlyphKey(r) {
		return m, nil
	}
	switch r {
	case 's':
		return m, m.SaveWithToast()
	case 'q':
		return m, tea.Quit
	case 'c':
		m.openPicker()
	case 'p', 'P':
		m.pasteOnLayer()
	case 'h', 'l', 'j', 'k':
		m.move(r)
	case '[':
		m.stepHeading(-1)
	case ']':
		m.stepHeading(1)
	case 'i', 'I':
		m.Inserting = true
		m.status = "-- INSERT one char (esc cancel) --"
	case 'd', 'D':
		if m.Win == WinCanvas {
			m.paint('d')
		}
	case 'x':
		m.cutPickup()
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
	case WinCanvas:
		m.Win = WinFrames
	case WinFrames:
		m.Win = WinPalette
	case WinPalette:
		m.Win = WinSymbols
	}
}

func (m Model) handleInsert(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Inserting = false
		m.status = "insert cancelled"
		return m, nil
	case "ctrl+c":
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

func (m Model) handlePickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closePicker(false)
		return m, nil
	case "space", "enter":
		m.closePicker(true)
		return m, nil
	case "ctrl+c":
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

func runeFrom(msg tea.KeyPressMsg) rune {
	rs := []rune(msg.Text)
	if len(rs) == 1 {
		return rs[0]
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
			m.applyNamedColor(p)
		}
	case WinFrames:
	case WinSymbols:
		if m.SymIdx >= 0 && m.SymIdx < len(SymbolList) {
			m.PaintCh = SymbolList[m.SymIdx].Ch
			m.RecentGlyphs = rememberGlyph(m.RecentGlyphs, m.PaintCh, 10)
			m.status = fmt.Sprintf("symbol %s", SymbolList[m.SymIdx].Name)
		}
	}
}

func (m *Model) move(r rune) {
	switch m.Win {
	case WinCanvas:
		switch r {
		case 'k':
			m.moveCursor(0, -1)
		case 'j':
			m.moveCursor(0, 1)
		case 'h':
			m.moveCursor(-1, 0)
		case 'l':
			m.moveCursor(1, 0)
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
			m.stepHeading(-1)
		case 'l':
			m.stepHeading(1)
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

func (m *Model) moveCursor(dx, dy int) {
	if m.Win != WinCanvas {
		return
	}
	sp := m.Current()
	m.CursorC += dx
	m.CursorR += dy
	if m.CursorC < 0 {
		m.CursorC = 0
	}
	if m.CursorR < 0 {
		m.CursorR = 0
	}
	if sp.Width > 0 && m.CursorC >= sp.Width {
		m.CursorC = sp.Width - 1
	}
	if sp.Height > 0 && m.CursorR >= sp.Height {
		m.CursorR = sp.Height - 1
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

func (m *Model) stepHeading(delta int) {
	n := len(sprite.Headings)
	if n == 0 || m.Atlas == nil {
		return
	}
	start := 0
	for i, h := range sprite.Headings {
		if h == m.Heading {
			start = i
			break
		}
	}
	for i := 1; i <= n; i++ {
		idx := start + delta*i
		idx %= n
		if idx < 0 {
			idx += n
		}
		h := sprite.Headings[idx]
		if _, ok := m.Atlas.Frame(m.Size, h); ok {
			m.Heading = h
			m.clampCursor()
			m.sel = map[cellKey]bool{}
			m.status = "heading " + string(h)
			return
		}
	}
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

// cutPickup is x: on outline, delete the cell and make that glyph plus
// color the brush. On fg/bg, strip that color only — the ASCII stays —
// and pick the deleted channel up as the brush. An empty cell stays
// empty and leaves the brush alone.
func (m *Model) cutPickup() {
	if m.Win != WinCanvas {
		return
	}
	sp := cloneSprite(m.Current())
	c := sp.At(m.CursorR, m.CursorC)
	switch m.Layer {
	case LayerFG:
		if c.FG >= 0 {
			m.Brush.FG = c.FG
			m.PalIdx = -1
			m.RecentColors = rememberSwatch(m.RecentColors, m.Brush, 10)
			m.status = fmt.Sprintf("cut fg %d", c.FG)
		}
		c.FG = -1
		sp.Set(m.CursorR, m.CursorC, c)
	case LayerBG:
		if c.BG >= 0 {
			m.Brush.BG = c.BG
			m.PalIdx = -1
			m.RecentColors = rememberSwatch(m.RecentColors, m.Brush, 10)
			m.status = fmt.Sprintf("cut bg %d", c.BG)
		}
		c.BG = -1
		sp.Set(m.CursorR, m.CursorC, c)
	default:
		if !c.Transparent() {
			m.PaintCh = c.Ch
			m.syncSymIdx()
			m.RecentGlyphs = rememberGlyph(m.RecentGlyphs, c.Ch, 10)
			m.Brush = Swatch{FG: c.FG, BG: c.BG}
			m.PalIdx = -1
			m.RecentColors = rememberSwatch(m.RecentColors, m.Brush, 10)
			m.status = fmt.Sprintf("cut %q fg %d bg %d", string(c.Ch), c.FG, c.BG)
		}
		sp.Set(m.CursorR, m.CursorC, sprite.Cell{Ch: ' ', FG: -1, BG: -1})
	}
	m.setCurrent(sp)
}

func (m *Model) paint(mode rune) {
	sp := cloneSprite(m.Current())
	col := m.color()
	for _, k := range m.targets() {
		c := sp.At(k.R, k.C)
		switch mode {
		case 'd':
			switch m.Layer {
			case LayerFG:
				c.FG = -1
			case LayerBG:
				c.BG = -1
			default:
				c = sprite.Cell{Ch: ' ', FG: -1, BG: -1}
			}
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

func (m *Model) handleMouse(msg tea.MouseClickMsg) {
	if msg.Button != tea.MouseLeft {
		return
	}
	x, y := msg.X, msg.Y
	sp := m.Current()
	tw, th := m.termSize()
	cx, cy := canvasOrigin(tw, th, sp.Width, sp.Height)
	m.CanvasX, m.CanvasY = cx, cy
	if x >= cx && x < cx+sp.Width && y >= cy && y < cy+sp.Height {
		m.Win = WinCanvas
		m.CursorC = x - cx
		m.CursorR = y - cy
		m.clampCursor()
		return
	}
	if m.handleRecentsClick(x, y) {
		return
	}
	m.clampCursor()
}

// Save writes the current canvas to the title-bar path, then flushes
// every other atlas still warm in the cache. Ctrl-P swaps files;
// saving only the one on screen would leave the others' edits in RAM.
func (m Model) Save() error {
	if m.Path == "" {
		return fmt.Errorf("no path to save")
	}
	if m.Atlas == nil {
		return fmt.Errorf("no atlas to save")
	}
	m.Atlas.SetFrame(m.Size, m.Heading, cloneSprite(m.Current()))
	if err := m.Atlas.WriteFile(m.Path); err != nil {
		return err
	}
	return m.flushAtlases()
}

func (m Model) flushAtlases() error {
	for path, a := range m.atlases {
		if a == nil || a == m.Atlas || path == "" || path == m.Path {
			continue
		}
		if err := a.WriteFile(path); err != nil {
			return err
		}
	}
	return nil
}

// SaveWithToast writes the atlas and flashes a 3-height confirmation
// above the art for saveToastTTL. A stale tick from an earlier save
// cannot dismiss a newer toast.
func (m *Model) SaveWithToast() tea.Cmd {
	m.toastID++
	id := m.toastID
	m.status = ""
	m.err = ""
	if err := m.Save(); err != nil {
		m.toast = "ERR"
	} else {
		m.toast = "SAVED"
	}
	return tea.Tick(saveToastTTL, func(time.Time) tea.Msg {
		return saveToastClearMsg{id: id}
	})
}

func (m Model) saveToastLines() []string {
	if m.toast == "" {
		return nil
	}
	lines, err := termfont.Lines(3, m.toast)
	if err != nil {
		return []string{m.toast}
	}
	return lines
}

func (m Model) termSize() (w, h int) {
	w, h = m.TermW, m.TermH
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	return w, h
}

// canvasOrigin is the inner top-left of the sprite box after a 1-line
// title and before a 2-line footer, with the box centered in the rest.
func canvasOrigin(termW, termH, spriteW, spriteH int) (x, y int) {
	boxW, boxH := spriteW+2, spriteH+2
	padX := (termW - boxW) / 2
	if padX < 0 {
		padX = 0
	}
	avail := termH - 3
	if avail < 1 {
		avail = 1
	}
	padY := (avail - boxH) / 2
	if padY < 0 {
		padY = 0
	}
	return padX + 1, 1 + padY + 1
}

func (m Model) centeredCanvas() string {
	sp := m.Current()
	tw, th := m.termSize()
	ox, oy := canvasOrigin(tw, th, sp.Width, sp.Height)
	padX, padY := ox-1, oy-2
	if padX < 0 {
		padX = 0
	}
	if padY < 0 {
		padY = 0
	}
	raw := renderCanvas(sp, m)
	lines := strings.Split(raw, "\n")
	boxW := 0
	if len(lines) > 0 {
		boxW = displayLen(lines[0])
		if padX+boxW > tw {
			padX = tw - boxW
			if padX < 0 {
				padX = 0
			}
		}
	}
	toast := m.saveToastLines()
	blanks := padY
	if n := len(toast); n > 0 && blanks >= n {
		blanks -= n
	}
	var b strings.Builder
	for i := 0; i < blanks; i++ {
		b.WriteByte('\n')
	}
	for _, line := range toast {
		tp := padX + (boxW-displayLen(line))/2
		if tp < 0 {
			tp = 0
		}
		b.WriteString(clipTo(strings.Repeat(" ", tp)+line, tw))
		b.WriteByte('\n')
	}
	for i, line := range lines {
		out := strings.Repeat(" ", padX) + line
		b.WriteString(clipTo(out, tw))
		if i+1 < len(lines) {
			b.WriteByte('\n')
		}
	}
	tabs := renderLayerTabs(m.Layer)
	tp := padX + (boxW-displayLen(tabs))/2
	if tp < 0 {
		tp = 0
	}
	b.WriteByte('\n')
	b.WriteString(clipTo(strings.Repeat(" ", tp)+tabs, tw))
	return b.String()
}

func (m Model) View() tea.View {
	tw, th := m.termSize()
	m.TermW, m.TermH = tw, th
	sp := m.Current()
	title := fmt.Sprintf("ASCII EDITOR  size %d  heading %s  layer %s  %s", m.Size, m.Heading, m.Layer, m.Path)
	if m.Path == "" {
		title = fmt.Sprintf("ASCII EDITOR  size %d  heading %s  layer %s", m.Size, m.Heading, m.Layer)
	}

	body := m.centeredCanvas()
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

	content := clipTo(title, tw) + "\n" + body + "\n" +
		clipLines(renderRecents(m), tw) + "\n" +
		clipTo(meta, tw) + "\n" +
		clipTo(status, tw)
	switch m.Win {
	case WinSymbols:
		content = renderSymbols(m) + "\n" + content
	case WinPalette:
		content = renderPalette(m) + "\n" + content
	case WinFrames:
		content = renderFrames(m) + "\n" + content
	}
	if m.PickerOpen {
		content = renderEightBitOverlay(m) + "\n" + content
	}
	if m.FilePickerOpen {
		content = renderFilePicker(m) + "\n" + content
	}
	if m.GlyphGridOpen {
		content = renderGlyphGrid(m) + "\n" + content
	}
	if m.ColorPaletteOpen {
		content = renderColorPalette(m) + "\n" + content
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func renderEightBitOverlay(m Model) string {
	const w = 36
	lines := []string{padPlain(fmt.Sprintf("8-bit ▾  %s", pickerLabel(m)), w)}
	lines = append(lines, renderPickerGrid(m, w)...)
	return box(" 8-bit ", lines, w)
}

func renderCanvas(sp sprite.Sprite, m Model) string {
	inner := make([]string, sp.Height)
	for r := 0; r < sp.Height; r++ {
		var b strings.Builder
		skip := 0
		used := 0
		for c := 0; c < sp.Width; c++ {
			if skip > 0 {
				skip--
				continue
			}
			cell := sp.At(r, c)
			view := layerCell(m.Layer, cell)
			ch := view.Ch
			if ch == 0 || ch == ' ' {
				if m.Selected(r, c) || (m.Win == WinCanvas && r == m.CursorR && c == m.CursorC) {
					ch = '·'
				} else {
					ch = ' '
				}
			}
			cols := runeCols(ch)
			if used+cols > sp.Width {
				b.WriteString(strings.Repeat(" ", sp.Width-used))
				used = sp.Width
				break
			}
			s := string(ch)
			if m.Selected(r, c) {
				s = "\x1b[7m" + s + "\x1b[0m"
			} else if m.Win == WinCanvas && r == m.CursorR && c == m.CursorC {
				s = "\x1b[7m" + s + "\x1b[0m"
			} else if !view.Transparent() {
				s = sprite.Render(oneCell(view))
			}
			b.WriteString(s)
			used += cols
			if cols > 1 {
				skip = cols - 1
			}
		}
		if used < sp.Width {
			b.WriteString(strings.Repeat(" ", sp.Width-used))
		}
		inner[r] = clipPad(b.String(), sp.Width)
	}
	label := fmt.Sprintf(" canvas %dx%d %s  %s ", sp.Width, sp.Height, m.Heading, m.Layer)
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
		b.WriteString("  recent glyphs ")
		n := len(m.RecentGlyphs)
		if n > 10 {
			n = 10
		}
		for i := 0; i < n; i++ {
			b.WriteRune(m.RecentGlyphs[i])
		}
	}
	return b.String()
}

const (
	recentGlyphsLabel = "recent glyphs  "
	recentColorsLabel = "recent colors  "
)

func renderRecentGlyphs(m Model) string {
	var b strings.Builder
	b.WriteString(recentGlyphsLabel)
	n := len(m.RecentGlyphs)
	if n > 10 {
		n = 10
	}
	for i := 0; i < n; i++ {
		ch := m.RecentGlyphs[i]
		if m.PaintCh == ch {
			b.WriteString("\x1b[7m")
			b.WriteRune(ch)
			b.WriteString("\x1b[0m")
		} else {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

func renderRecentColors(m Model) string {
	return recentColorsLabel + renderClutch(m)
}

func renderRecents(m Model) string {
	return renderRecentGlyphs(m) + "\n" + renderRecentColors(m)
}

func recentsOrigin(termW, termH, spriteW, spriteH int) (glyphY, colorY int) {
	_, oy := canvasOrigin(termW, termH, spriteW, spriteH)
	glyphY = oy + spriteH + 2
	colorY = glyphY + 1
	return glyphY, colorY
}

func (m *Model) handleRecentsClick(x, y int) bool {
	if m.PickerOpen || m.FilePickerOpen || m.GlyphGridOpen || m.ColorPaletteOpen {
		return false
	}
	if m.Win != WinCanvas {
		return false
	}
	sp := m.Current()
	tw, th := m.termSize()
	glyphY, colorY := recentsOrigin(tw, th, sp.Width, sp.Height)
	label := len([]rune(recentGlyphsLabel))
	if y == glyphY && x >= label {
		return m.pickRecentGlyph(x - label)
	}
	if y == colorY && x >= label {
		// clutch cells are "1█ " (digit, swatch, space)
		rel := x - label
		if rel < 0 {
			return false
		}
		return m.pickRecentColor(rel / 3)
	}
	return false
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
		b.WriteString(clipPad(line, innerW))
		b.WriteString("│\n")
	}
	b.WriteString(bot)
	return b.String()
}

func padTitle(title string, innerW int) string {
	t := clipTo(title, innerW)
	dash := innerW - displayLen(t)
	if dash < 0 {
		dash = 0
	}
	left := dash / 2
	right := dash - left
	return strings.Repeat("─", left) + t + strings.Repeat("─", right)
}

func padPlain(s string, w int) string {
	return clipPad(s, w)
}

func visibleLen(s string) int {
	return displayLen(s)
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
