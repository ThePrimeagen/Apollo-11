// Package durdraw reads and writes native durdraw .dur files
// (gzipped DurMovie JSON, format version 7). Optional import helper;
// the sprite editor and premiere use JSON atlases.
package durdraw

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const FormatVersion = 7

// Extra is stored in DurMovie.extra so a multi-frame movie can name
// each frame as an LM heading without leaving durdraw's format.
type Extra struct {
	Size     int      `json:"size,omitempty"`
	Headings []string `json:"headings,omitempty"`
}

// Movie is one DurMovie: a canvas plus one or more frames.
type Movie struct {
	FormatVersion int
	ColorFormat   string
	PreferredFont string
	Encoding      string
	Name          string
	Artist        string
	Framerate     float64
	SizeX         int
	SizeY         int
	Extra         *Extra
	Frames        []Frame
}

// Frame is one DurMovie frame. ColorMap is column-major, the way
// durdraw v7 actually writes it: colorMap[col][row] = [fg, bg].
type Frame struct {
	FrameNumber int
	Delay       float64
	Contents    []string
	ColorMap    [][][2]int
}

type fileDoc struct {
	DurMovie movieDoc `json:"DurMovie"`
}

type movieDoc struct {
	FormatVersion int             `json:"formatVersion"`
	ColorFormat   string          `json:"colorFormat"`
	PreferredFont string          `json:"preferredFont"`
	Encoding      string          `json:"encoding"`
	Name          string          `json:"name"`
	Artist        string          `json:"artist"`
	Framerate     float64         `json:"framerate"`
	SizeX         int             `json:"sizeX"`
	SizeY         int             `json:"sizeY"`
	Columns       int             `json:"columns,omitempty"`
	Lines         int             `json:"lines,omitempty"`
	Extra         json.RawMessage `json:"extra"`
	Frames        []frameDoc      `json:"frames"`
}

type frameDoc struct {
	FrameNumber int       `json:"frameNumber"`
	Delay       float64   `json:"delay"`
	Contents    []string  `json:"contents"`
	ColorMap    [][][2]int `json:"colorMap"`
}

// Marshal writes a gzipped DurMovie JSON document.
func Marshal(m Movie) ([]byte, error) {
	if m.FormatVersion == 0 {
		m.FormatVersion = FormatVersion
	}
	if m.ColorFormat == "" {
		m.ColorFormat = "256"
	}
	if m.Encoding == "" {
		m.Encoding = "utf-8"
	}
	if m.PreferredFont == "" {
		m.PreferredFont = "fixed"
	}
	if m.Framerate == 0 {
		m.Framerate = 6
	}
	if m.SizeX < 1 || m.SizeY < 1 {
		return nil, fmt.Errorf("durdraw: invalid canvas %dx%d", m.SizeX, m.SizeY)
	}
	if len(m.Frames) == 0 {
		return nil, fmt.Errorf("durdraw: movie has no frames")
	}
	doc := fileDoc{DurMovie: movieDoc{
		FormatVersion: m.FormatVersion,
		ColorFormat:   m.ColorFormat,
		PreferredFont: m.PreferredFont,
		Encoding:      m.Encoding,
		Name:          m.Name,
		Artist:        m.Artist,
		Framerate:     m.Framerate,
		SizeX:         m.SizeX,
		SizeY:         m.SizeY,
		Frames:        make([]frameDoc, len(m.Frames)),
	}}
	if m.Extra != nil {
		raw, err := json.Marshal(m.Extra)
		if err != nil {
			return nil, err
		}
		doc.DurMovie.Extra = raw
	} else {
		doc.DurMovie.Extra = []byte("null")
	}
	for i, f := range m.Frames {
		if f.FrameNumber == 0 {
			f.FrameNumber = i + 1
		}
		doc.DurMovie.Frames[i] = frameDoc{
			FrameNumber: f.FrameNumber,
			Delay:       f.Delay,
			Contents:    f.Contents,
			ColorMap:    f.ColorMap,
		}
	}
	plain, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(plain); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal reads a gzipped (or raw JSON) DurMovie.
func Unmarshal(raw []byte) (Movie, error) {
	plain, err := gunzipOrJSON(raw)
	if err != nil {
		return Movie{}, err
	}
	var doc fileDoc
	if err := json.Unmarshal(plain, &doc); err != nil {
		return Movie{}, fmt.Errorf("durdraw: %w", err)
	}
	h := doc.DurMovie
	if h.FormatVersion == 0 && h.SizeX == 0 && h.SizeY == 0 && h.Columns == 0 && len(h.Frames) == 0 {
		return Movie{}, fmt.Errorf("durdraw: missing DurMovie")
	}
	w, ht := h.SizeX, h.SizeY
	if w < 1 {
		w = h.Columns
	}
	if ht < 1 {
		ht = h.Lines
	}
	if w < 1 || ht < 1 {
		return Movie{}, fmt.Errorf("durdraw: invalid canvas %dx%d", w, ht)
	}
	if len(h.Frames) == 0 {
		return Movie{}, fmt.Errorf("durdraw: movie has no frames")
	}
	m := Movie{
		FormatVersion: h.FormatVersion,
		ColorFormat:   h.ColorFormat,
		PreferredFont: h.PreferredFont,
		Encoding:      h.Encoding,
		Name:          h.Name,
		Artist:        h.Artist,
		Framerate:     h.Framerate,
		SizeX:         w,
		SizeY:         ht,
		Frames:        make([]Frame, len(h.Frames)),
	}
	if len(h.Extra) > 0 && string(h.Extra) != "null" {
		var extra Extra
		if err := json.Unmarshal(h.Extra, &extra); err != nil {
			return Movie{}, fmt.Errorf("durdraw: extra: %w", err)
		}
		m.Extra = &extra
	}
	for i, f := range h.Frames {
		m.Frames[i] = Frame{
			FrameNumber: f.FrameNumber,
			Delay:       f.Delay,
			Contents:    f.Contents,
			ColorMap:    f.ColorMap,
		}
	}
	return m, nil
}

func gunzipOrJSON(raw []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err == nil {
		defer zr.Close()
		plain, err := io.ReadAll(zr)
		if err != nil {
			return nil, fmt.Errorf("durdraw: gzip: %w", err)
		}
		return plain, nil
	}
	trim := bytes.TrimSpace(raw)
	if len(trim) > 0 && trim[0] == '{' {
		return raw, nil
	}
	return nil, fmt.Errorf("durdraw: not a gzip DurMovie")
}

// WriteFile marshals m to path.
func WriteFile(path string, m Movie) error {
	raw, err := Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// LoadFile reads a .dur from path.
func LoadFile(path string) (Movie, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Movie{}, err
	}
	return Unmarshal(raw)
}

// FromSprite builds a one-frame movie named after the heading.
func FromSprite(name string, sp sprite.Sprite) Movie {
	return Movie{
		FormatVersion: FormatVersion,
		ColorFormat:   "256",
		PreferredFont: "fixed",
		Encoding:      "utf-8",
		Name:          name,
		Framerate:     6,
		SizeX:         sp.Width,
		SizeY:         sp.Height,
		Frames:        []Frame{frameFromSprite(1, sp)},
	}
}

func frameFromSprite(n int, sp sprite.Sprite) Frame {
	cm := make([][][2]int, sp.Width)
	for c := 0; c < sp.Width; c++ {
		col := make([][2]int, sp.Height)
		for r := 0; r < sp.Height; r++ {
			cell := sp.At(r, c)
			fg, bg := cell.FG, cell.BG
			if fg < 0 {
				fg = 0
			}
			if bg < 0 {
				bg = 0
			}
			col[r] = [2]int{fg, bg}
		}
		cm[c] = col
	}
	return Frame{
		FrameNumber: n,
		Contents:    sp.GlyphRows(),
		ColorMap:    cm,
	}
}

// Sprite returns the first frame as a sprite.
func (m Movie) Sprite() (sprite.Sprite, error) {
	if len(m.Frames) == 0 {
		return sprite.Sprite{}, fmt.Errorf("durdraw: movie has no frames")
	}
	return m.Frames[0].Sprite(m.SizeX, m.SizeY)
}

// Sprite reconstructs one frame. w/h come from the movie canvas.
func (f Frame) Sprite(w, h int) (sprite.Sprite, error) {
	if w < 1 || h < 1 {
		return sprite.Sprite{}, fmt.Errorf("durdraw: invalid canvas %dx%d", w, h)
	}
	if len(f.Contents) == 0 {
		return sprite.Sprite{}, fmt.Errorf("durdraw: empty frame contents")
	}
	sp := sprite.New(w, h)
	for r := 0; r < h && r < len(f.Contents); r++ {
		row := []rune(f.Contents[r])
		for c := 0; c < w && c < len(row); c++ {
			fg, bg := colorAt(f.ColorMap, c, r)
			ch := row[c]
			if ch == 0 {
				ch = ' '
			}
			if ch == ' ' {
				fg, bg = -1, -1
			} else {
				if fg == 0 {
					fg = -1
				}
				if bg == 0 {
					bg = -1
				}
			}
			sp.Set(r, c, sprite.Cell{Ch: ch, FG: fg, BG: bg})
		}
	}
	if utf8.RuneCountInString(f.Contents[0]) == 0 && w > 0 {
		return sprite.Sprite{}, fmt.Errorf("durdraw: empty frame contents")
	}
	return sp, nil
}

func colorAt(cm [][][2]int, col, row int) (fg, bg int) {
	if col < 0 || row < 0 || col >= len(cm) || row >= len(cm[col]) {
		return 0, 0
	}
	pair := cm[col][row]
	return pair[0], pair[1]
}

// EncodeSize writes every heading present at sz, in lander display
// order (N S W for size 4, all eight otherwise).
func EncodeSize(a *sprite.Atlas, sz sprite.Size) (Movie, error) {
	if a == nil {
		return Movie{}, fmt.Errorf("durdraw: nil atlas")
	}
	heads := headingsFor(a, sz)
	if len(heads) == 0 {
		return Movie{}, fmt.Errorf("durdraw: no frames for size %d", sz)
	}
	w, h := sz.Dim()
	if w < 1 {
		sp, _ := a.Frame(sz, heads[0])
		w, h = sp.Width, sp.Height
	}
	mov := Movie{
		FormatVersion: FormatVersion,
		ColorFormat:   "256",
		PreferredFont: "fixed",
		Encoding:      "utf-8",
		Name:          fmt.Sprintf("lm-%d", sz),
		Framerate:     6,
		SizeX:         w,
		SizeY:         h,
		Extra:         &Extra{Size: int(sz), Headings: make([]string, len(heads))},
		Frames:        make([]Frame, len(heads)),
	}
	for i, hd := range heads {
		sp, ok := a.Frame(sz, hd)
		if !ok {
			return Movie{}, fmt.Errorf("durdraw: missing size %d %s", sz, hd)
		}
		mov.Extra.Headings[i] = string(hd)
		mov.Frames[i] = frameFromSprite(i+1, sp)
	}
	return mov, nil
}

func headingsFor(a *sprite.Atlas, sz sprite.Size) []sprite.Heading {
	pref := sprite.Headings
	if sz == sprite.Size4 {
		pref = []sprite.Heading{sprite.N, sprite.S, sprite.W}
	}
	var out []sprite.Heading
	for _, h := range pref {
		if _, ok := a.Frame(sz, h); ok {
			out = append(out, h)
		}
	}
	if len(out) > 0 {
		return out
	}
	for _, h := range sprite.Headings {
		if _, ok := a.Frame(sz, h); ok {
			out = append(out, h)
		}
	}
	return out
}

// DecodeAtlas turns a movie into a one-size atlas using extra.headings
// (or a sensible default order when extra is missing).
func DecodeAtlas(m Movie) (*sprite.Atlas, sprite.Size, error) {
	if len(m.Frames) == 0 {
		return nil, 0, fmt.Errorf("durdraw: movie has no frames")
	}
	sz := sprite.Size(0)
	heads := []sprite.Heading(nil)
	if m.Extra != nil {
		if m.Extra.Size != 0 {
			sz = sprite.Size(m.Extra.Size)
		}
		for _, name := range m.Extra.Headings {
			heads = append(heads, sprite.Heading(name))
		}
	}
	if sz == 0 {
		sz = sizeFromDim(m.SizeX, m.SizeY)
	}
	if len(heads) == 0 {
		heads = defaultHeadings(sz, len(m.Frames))
	}
	if len(heads) != len(m.Frames) {
		if len(heads) > len(m.Frames) {
			heads = heads[:len(m.Frames)]
		} else {
			for i := len(heads); i < len(m.Frames); i++ {
				heads = append(heads, sprite.Heading(fmt.Sprintf("F%d", i+1)))
			}
		}
	}
	a := &sprite.Atlas{Palette: append([]sprite.PaletteEntry(nil), sprite.DefaultPalette...)}
	for i, f := range m.Frames {
		sp, err := f.Sprite(m.SizeX, m.SizeY)
		if err != nil {
			return nil, 0, fmt.Errorf("durdraw: frame %d: %w", i+1, err)
		}
		a.SetFrame(sz, heads[i], sp)
	}
	return a, sz, nil
}

func sizeFromDim(w, h int) sprite.Size {
	for _, sz := range sprite.Sizes {
		dw, dh := sz.Dim()
		if dw == w && dh == h {
			return sz
		}
	}
	return sprite.Size4
}

func defaultHeadings(sz sprite.Size, n int) []sprite.Heading {
	pref := sprite.Headings
	if sz == sprite.Size4 {
		pref = []sprite.Heading{sprite.N, sprite.S, sprite.W}
	}
	if n <= len(pref) {
		return append([]sprite.Heading(nil), pref[:n]...)
	}
	out := append([]sprite.Heading(nil), pref...)
	for i := len(pref); i < n; i++ {
		out = append(out, sprite.Heading(fmt.Sprintf("F%d", i+1)))
	}
	return out
}
