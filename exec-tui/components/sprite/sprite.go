package sprite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"unicode/utf8"
)

// Size is one of the four LM scales: 1 is the current 13×5 sprite, 4 is
// twice that (26×10), and 2/3 are the in-between shrink steps.
type Size int

const (
	Size1 Size = 1
	Size2 Size = 2
	Size3 Size = 3
	Size4 Size = 4
)

// Sizes is largest-last so ranging it walks small → big.
var Sizes = []Size{Size1, Size2, Size3, Size4}

// Dim returns the fixed canvas for a size. Every heading shares it so the
// craft rotates in place, the way the current 13×5 attitudes already do.
func (s Size) Dim() (w, h int) {
	switch s {
	case Size1:
		return 13, 5
	case Size2:
		return 17, 7
	case Size3:
		return 22, 8
	case Size4:
		return 26, 10
	default:
		return 0, 0
	}
}

// Heading is the cabin-pointing direction, 45° steps from north.
type Heading string

const (
	N  Heading = "N"
	NE Heading = "NE"
	E  Heading = "E"
	SE Heading = "SE"
	S  Heading = "S"
	SW Heading = "SW"
	W  Heading = "W"
	NW Heading = "NW"
)

// Headings is clockwise from north, cardinals plus the 45° offsets.
var Headings = []Heading{N, NE, E, SE, S, SW, W, NW}

// PaletteEntry is one named color. FG/BG are xterm-256 indexes; -1 means
// "default / transparent" for that channel.
type PaletteEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	FG   int    `json:"fg"`
	BG   int    `json:"bg"`
}

// DefaultPalette is the LM's materials: silver ascent, gold kapton, dark
// windows, steel bell, the plume. Backgrounds are optional and used on the
// larger sizes for a bit of depth.
var DefaultPalette = []PaletteEntry{
	{ID: ".", Name: "empty", FG: -1, BG: -1},
	{ID: "S", Name: "silver", FG: 252, BG: -1},
	{ID: "G", Name: "gold", FG: 178, BG: 94},
	{ID: "W", Name: "window", FG: 24, BG: 232},
	{ID: "E", Name: "engine", FG: 245, BG: -1},
	{ID: "P", Name: "plume", FG: 208, BG: 52},
}

// Cell is one terminal cell of a sprite.
type Cell struct {
	Ch     rune
	FG, BG int
}

// Transparent reports a cell that does not paint. A blank glyph with
// no background is empty sky. A background color is a floor — it
// paints even without a glyph, so fire can sit on top of it.
func (c Cell) Transparent() bool {
	if c.BG >= 0 {
		return false
	}
	return c.Ch == 0 || c.Ch == ' '
}

// Sprite is a rectangular grid of cells.
type Sprite struct {
	Width, Height int
	Cells         [][]Cell
}

// New allocates a transparent w×h sprite.
func New(w, h int) Sprite {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	s := Sprite{Width: w, Height: h, Cells: make([][]Cell, h)}
	for r := range s.Cells {
		s.Cells[r] = make([]Cell, w)
		for c := range s.Cells[r] {
			s.Cells[r][c] = Cell{Ch: ' ', FG: -1, BG: -1}
		}
	}
	return s
}

// At returns the cell or a transparent one if out of bounds.
func (s Sprite) At(r, c int) Cell {
	if r < 0 || c < 0 || r >= s.Height || c >= s.Width {
		return Cell{Ch: ' ', FG: -1, BG: -1}
	}
	return s.Cells[r][c]
}

// Set writes a cell; out of bounds is ignored.
func (s Sprite) Set(r, c int, cell Cell) {
	if r < 0 || c < 0 || r >= s.Height || c >= s.Width {
		return
	}
	if cell.Ch == 0 {
		cell.Ch = ' '
	}
	s.Cells[r][c] = cell
}

// GlyphRows is the editable ASCII, one string per row.
func (s Sprite) GlyphRows() []string {
	out := make([]string, s.Height)
	for r := 0; r < s.Height; r++ {
		row := make([]rune, s.Width)
		for c := 0; c < s.Width; c++ {
			ch := s.At(r, c).Ch
			if ch == 0 {
				ch = ' '
			}
			row[c] = ch
		}
		out[r] = string(row)
	}
	return out
}

// Validate checks the grid is rectangular and matches Width/Height.
func (s Sprite) Validate() error {
	if s.Width < 0 || s.Height < 0 {
		return fmt.Errorf("negative dimension %dx%d", s.Width, s.Height)
	}
	if len(s.Cells) != s.Height {
		return fmt.Errorf("have %d rows, height %d", len(s.Cells), s.Height)
	}
	for i, row := range s.Cells {
		if len(row) != s.Width {
			return fmt.Errorf("row %d: %d cols, width %d", i, len(row), s.Width)
		}
	}
	if s.Width == 0 || s.Height == 0 {
		return fmt.Errorf("empty sprite")
	}
	return nil
}

// Atlas is the full set of lander frames plus the palette they share.
type Atlas struct {
	Palette []PaletteEntry
	frames  map[Size]map[Heading]Sprite
}

// Frame looks up one size×heading. The bool is false when missing.
func (a *Atlas) Frame(sz Size, h Heading) (Sprite, bool) {
	if a == nil || a.frames == nil {
		return Sprite{}, false
	}
	byH, ok := a.frames[sz]
	if !ok {
		return Sprite{}, false
	}
	sp, ok := byH[h]
	return sp, ok
}

// MustFrame is Frame that panics — for tests and the editor's current view.
func (a *Atlas) MustFrame(sz Size, h Heading) Sprite {
	sp, ok := a.Frame(sz, h)
	if !ok {
		panic(fmt.Sprintf("missing frame size=%d heading=%s", sz, h))
	}
	return sp
}

// SetFrame stores a sprite. It copies the grid so later edits don't alias.
func (a *Atlas) SetFrame(sz Size, h Heading, sp Sprite) {
	if a.frames == nil {
		a.frames = map[Size]map[Heading]Sprite{}
	}
	if a.frames[sz] == nil {
		a.frames[sz] = map[Heading]Sprite{}
	}
	a.frames[sz][h] = clone(sp)
}

// Clone deep-copies the atlas (palette + every frame).
func (a *Atlas) Clone() *Atlas {
	out := &Atlas{Palette: append([]PaletteEntry(nil), a.Palette...)}
	for sz, byH := range a.frames {
		for h, sp := range byH {
			out.SetFrame(sz, h, sp)
		}
	}
	return out
}

func clone(s Sprite) Sprite {
	out := New(s.Width, s.Height)
	for r := 0; r < s.Height; r++ {
		copy(out.Cells[r], s.Cells[r])
	}
	return out
}

// ShrinkSequence is the animation from twice-as-big down to the current
// sprite: size 4, 3, 2, 1. Missing frames are skipped, never invented.
func ShrinkSequence(a *Atlas, h Heading) []Sprite {
	if a == nil {
		return nil
	}
	var out []Sprite
	for _, sz := range []Size{Size4, Size3, Size2, Size1} {
		if sp, ok := a.Frame(sz, h); ok {
			out = append(out, sp)
		}
	}
	return out
}

type fileFormat struct {
	Palette []PaletteEntry                 `json:"palette"`
	Frames  map[string]map[string]frameDoc `json:"frames"`
}

type frameDoc struct {
	Glyphs []string `json:"glyphs"`
	FG     []string `json:"fg"`
	BG     []string `json:"bg"`
}

// paletteIDChars are single-rune ids the fg/bg masks can store. Named
// materials occupy .SGWEP; extras are allocated from the rest.
const paletteIDChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghijklmnopqrstuvwxyz"

// rememberColors adds palette entries for painted fg/bg values that
// are not already named. Without this, Marshal silently writes "." and
// a reload drops the color.
func (a *Atlas) rememberColors() {
	if a == nil {
		return
	}
	used := map[string]bool{}
	for _, p := range a.Palette {
		used[p.ID] = true
	}
	nextID := func() string {
		for _, r := range paletteIDChars {
			id := string(r)
			if !used[id] {
				used[id] = true
				return id
			}
		}
		return ""
	}
	add := func(fg, bg int) {
		needFG := fg >= 0 && a.idForFG(fg) == "."
		needBG := bg >= 0 && a.idForBG(bg) == "."
		if !needFG && !needBG {
			return
		}
		id := nextID()
		if id == "" {
			return
		}
		name := fmt.Sprintf("c%d", fg)
		if bg >= 0 {
			name = fmt.Sprintf("c%d_%d", fg, bg)
		}
		a.Palette = append(a.Palette, PaletteEntry{ID: id, Name: name, FG: fg, BG: bg})
	}
	for _, byH := range a.frames {
		for _, sp := range byH {
			for r := 0; r < sp.Height; r++ {
				for c := 0; c < sp.Width; c++ {
					cell := sp.At(r, c)
					if cell.Transparent() {
						continue
					}
					add(cell.FG, cell.BG)
				}
			}
		}
	}
}

// Marshal emits indented JSON a human can edit: palette, then per-size
// per-heading rows of glyphs plus fg/bg material masks.
func (a *Atlas) Marshal() ([]byte, error) {
	if a == nil {
		return nil, fmt.Errorf("nil atlas")
	}
	a.rememberColors()
	doc := fileFormat{
		Palette: a.Palette,
		Frames:  map[string]map[string]frameDoc{},
	}
	for _, sz := range Sizes {
		byH, ok := a.frames[sz]
		if !ok {
			continue
		}
		key := fmt.Sprintf("%d", sz)
		doc.Frames[key] = map[string]frameDoc{}
		for _, h := range Headings {
			sp, ok := byH[h]
			if !ok {
				continue
			}
			doc.Frames[key][string(h)] = frameDoc{
				Glyphs: sp.GlyphRows(),
				FG:     a.maskRows(sp, true),
				BG:     a.maskRows(sp, false),
			}
		}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (a *Atlas) maskRows(sp Sprite, fg bool) []string {
	out := make([]string, sp.Height)
	for r := 0; r < sp.Height; r++ {
		row := make([]rune, sp.Width)
		for c := 0; c < sp.Width; c++ {
			cell := sp.At(r, c)
			if cell.Transparent() {
				row[c] = '.'
				continue
			}
			if fg {
				row[c] = []rune(a.idForFG(cell.FG))[0]
			} else {
				row[c] = []rune(a.idForBG(cell.BG))[0]
			}
		}
		out[r] = string(row)
	}
	return out
}

func (a *Atlas) idForFG(fg int) string {
	if fg < 0 {
		return "."
	}
	for _, p := range a.Palette {
		if p.ID != "." && p.FG == fg {
			return p.ID
		}
	}
	return "."
}

func (a *Atlas) idForBG(bg int) string {
	if bg < 0 {
		return "."
	}
	for _, p := range a.Palette {
		if p.ID != "." && p.BG == bg {
			return p.ID
		}
	}
	return "."
}

func (a *Atlas) lookup(id string) PaletteEntry {
	for _, p := range a.Palette {
		if p.ID == id {
			return p
		}
	}
	return PaletteEntry{ID: ".", FG: -1, BG: -1}
}

// Unmarshal reads the editable JSON format.
func Unmarshal(raw []byte) (*Atlas, error) {
	var doc fileFormat
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	a := &Atlas{Palette: doc.Palette, frames: map[Size]map[Heading]Sprite{}}
	if len(a.Palette) == 0 {
		a.Palette = append([]PaletteEntry(nil), DefaultPalette...)
	}
	for szKey, byH := range doc.Frames {
		var sz Size
		if _, err := fmt.Sscanf(szKey, "%d", &sz); err != nil {
			return nil, fmt.Errorf("frame size %q: %w", szKey, err)
		}
		for hKey, fr := range byH {
			sp, err := fromMasks(a, fr.Glyphs, fr.FG, fr.BG)
			if err != nil {
				return nil, fmt.Errorf("size %s heading %s: %w", szKey, hKey, err)
			}
			a.SetFrame(sz, Heading(hKey), sp)
		}
	}
	return a, nil
}

func fromMasks(a *Atlas, glyphs, fg, bg []string) (Sprite, error) {
	if len(glyphs) == 0 {
		return Sprite{}, fmt.Errorf("empty glyphs")
	}
	if len(fg) != len(glyphs) || len(bg) != len(glyphs) {
		return Sprite{}, fmt.Errorf("glyphs %d / fg %d / bg %d rows, want equal", len(glyphs), len(fg), len(bg))
	}
	w := utf8.RuneCountInString(glyphs[0])
	h := len(glyphs)
	sp := New(w, h)
	for r, row := range glyphs {
		gr := []rune(row)
		fr := []rune(fg[r])
		br := []rune(bg[r])
		if len(gr) != w || len(fr) != w || len(br) != w {
			return Sprite{}, fmt.Errorf("row %d: glyph %d / fg %d / bg %d runes, want %d", r, len(gr), len(fr), len(br), w)
		}
		for c, ch := range gr {
			cell := Cell{Ch: ch, FG: -1, BG: -1}
			if ch != ' ' {
				cell.FG = a.lookup(string(fr[c])).FG
				cell.BG = a.lookup(string(br[c])).BG
			}
			sp.Set(r, c, cell)
		}
	}
	return sp, nil
}

// WriteFile marshals the atlas to path.
func (a *Atlas) WriteFile(path string) error {
	raw, err := a.Marshal()
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// LoadFile reads an atlas from path.
func LoadFile(path string) (*Atlas, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Unmarshal(raw)
}

// PaletteByID returns the entry and its index.
func (a *Atlas) PaletteByID(id string) (PaletteEntry, int, bool) {
	for i, p := range a.Palette {
		if p.ID == id {
			return p, i, true
		}
	}
	return PaletteEntry{}, -1, false
}
