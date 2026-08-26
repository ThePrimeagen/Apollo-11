package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func (m *Model) openShipPicker() {
	if m.PickerOpen || m.GlyphGridOpen || m.ColorPaletteOpen || m.Inserting {
		return
	}
	m.preloadSizeAtlases()
	m.ShipPickerSize = m.Size
	m.ShipPickerHead = m.Heading
	if !validSize(m.ShipPickerSize) {
		m.ShipPickerSize = sprite.Size4
	}
	if !validHeading(m.ShipPickerHead) {
		m.ShipPickerHead = sprite.N
	}
	m.ShipPickerOpen = true
	m.status = "ships  hjkl size/heading  1-4 size  enter open  esc close"
}

func (m *Model) closeShipPicker() {
	m.ShipPickerOpen = false
}

func validSize(sz sprite.Size) bool {
	for _, s := range sprite.Sizes {
		if s == sz {
			return true
		}
	}
	return false
}

func validHeading(h sprite.Heading) bool {
	for _, x := range sprite.Headings {
		if x == h {
			return true
		}
	}
	return false
}

func sizeIndex(sz sprite.Size) int {
	for i, s := range sprite.Sizes {
		if s == sz {
			return i
		}
	}
	return 0
}

func headingIndex(h sprite.Heading) int {
	for i, x := range sprite.Headings {
		if x == h {
			return i
		}
	}
	return 0
}

func (m *Model) stepPickerHeading(delta int) {
	n := len(sprite.Headings)
	i := headingIndex(m.ShipPickerHead)
	m.ShipPickerHead = sprite.Headings[(i+delta+n)%n]
}

func (m *Model) stepPickerSize(delta int) {
	n := len(sprite.Sizes)
	i := sizeIndex(m.ShipPickerSize)
	m.ShipPickerSize = sprite.Sizes[(i+delta+n)%n]
}

func atlasHasSize(a *sprite.Atlas, sz sprite.Size) bool {
	if a == nil {
		return false
	}
	for _, h := range sprite.Headings {
		if _, ok := a.Frame(sz, h); ok {
			return true
		}
	}
	return false
}

func (m Model) sizeAssetPath(sz sprite.Size) string {
	name := fmt.Sprintf("lm-%d.json", int(sz))
	var cands []string
	if dir := m.assetsDir(); dir != "" {
		cands = append(cands, filepath.Join(dir, name))
	}
	if m.Path != "" {
		cands = append(cands, filepath.Join(filepath.Dir(m.Path), name))
	}
	seen := map[string]bool{}
	for _, p := range cands {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func (m *Model) preloadSizeAtlases() {
	if m.sizeCache == nil {
		m.sizeCache = map[sprite.Size]*sprite.Atlas{}
	}
	if m.sizePaths == nil {
		m.sizePaths = map[sprite.Size]string{}
	}
	for _, sz := range sprite.Sizes {
		if path := m.sizeAssetPath(sz); path != "" {
			m.sizePaths[sz] = path
		}
		if atlasHasSize(m.Atlas, sz) || m.sizeCache[sz] != nil {
			continue
		}
		path := m.sizePaths[sz]
		if path == "" {
			continue
		}
		a, err := LoadShip(path)
		if err != nil {
			continue
		}
		m.sizeCache[sz] = a
	}
}

func (m Model) frameAt(sz sprite.Size, h sprite.Heading) (sprite.Sprite, bool) {
	if m.Atlas != nil {
		if sp, ok := m.Atlas.Frame(sz, h); ok {
			return sp, true
		}
	}
	if a := m.sizeCache[sz]; a != nil {
		return a.Frame(sz, h)
	}
	return sprite.Sprite{}, false
}

func (m *Model) applySizeHeading(sz sprite.Size, h sprite.Heading) bool {
	if m.Atlas != nil {
		if _, ok := m.Atlas.Frame(sz, h); ok {
			m.Size = sz
			m.Heading = h
			m.clampCursor()
			m.sel = map[cellKey]bool{}
			m.status = fmt.Sprintf("size %d heading %s", sz, h)
			return true
		}
	}
	a := m.sizeCache[sz]
	if a == nil {
		return false
	}
	if _, ok := a.Frame(sz, h); !ok {
		return false
	}
	if m.sizeCache == nil {
		m.sizeCache = map[sprite.Size]*sprite.Atlas{}
	}
	if m.Atlas != nil {
		if _, ok := m.Atlas.Frame(m.Size, m.Heading); ok {
			m.Atlas.SetFrame(m.Size, m.Heading, cloneSprite(m.Current()))
		}
		m.sizeCache[m.Size] = m.Atlas
	}
	m.Atlas = a
	if path := m.sizePaths[sz]; path != "" {
		m.Path = path
	} else if path := m.sizeAssetPath(sz); path != "" {
		m.Path = path
	}
	m.Size = sz
	m.Heading = h
	m.sel = map[cellKey]bool{}
	m.clampCursor()
	m.status = fmt.Sprintf("size %d heading %s", sz, h)
	return true
}

func (m Model) handleShipPickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeShipPicker()
		m.status = "ship picker cancelled"
		return m, nil
	case "enter":
		if !m.applySizeHeading(m.ShipPickerSize, m.ShipPickerHead) {
			return m, nil
		}
		m.closeShipPicker()
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "left":
		m.stepPickerHeading(-1)
		return m, nil
	case "right":
		m.stepPickerHeading(1)
		return m, nil
	case "up":
		m.stepPickerSize(-1)
		return m, nil
	case "down":
		m.stepPickerSize(1)
		return m, nil
	}

	r := runeFrom(msg)
	switch r {
	case 'h':
		m.stepPickerHeading(-1)
	case 'l':
		m.stepPickerHeading(1)
	case 'k':
		m.stepPickerSize(-1)
	case 'j':
		m.stepPickerSize(1)
	case '[':
		m.stepPickerHeading(-1)
	case ']':
		m.stepPickerHeading(1)
	case '1', '2', '3', '4':
		m.ShipPickerSize = sprite.Size(r - '0')
	}
	return m, nil
}

func renderShipPicker(m Model) string {
	help := "hjkl move  1-4 size  enter open  esc close"
	grid := renderSizeHeadingGrid(m)
	thumbs := renderHeadingThumbs(m)
	inner := append([]string{}, grid...)
	inner = append(inner, "")
	inner = append(inner, help)
	inner = append(inner, "")
	inner = append(inner, strings.Split(thumbs, "\n")...)

	w := 40
	for _, line := range inner {
		if n := visibleLen(line); n > w {
			w = n
		}
	}
	for i := range inner {
		inner[i] = padPlain(inner[i], w)
	}
	title := fmt.Sprintf(" ships  size %d  %s ", m.ShipPickerSize, m.ShipPickerHead)
	return box(title, inner, w)
}

func renderSizeHeadingGrid(m Model) []string {
	var hdr strings.Builder
	hdr.WriteString("      ")
	for _, h := range sprite.Headings {
		hdr.WriteString(fmt.Sprintf("%4s", h))
	}
	lines := []string{hdr.String()}
	for _, sz := range sprite.Sizes {
		mark := " "
		if sz == m.ShipPickerSize {
			mark = ">"
		}
		var row strings.Builder
		row.WriteString(fmt.Sprintf("%s %d  ", mark, sz))
		for _, h := range sprite.Headings {
			_, present := m.frameAt(sz, h)
			selected := sz == m.ShipPickerSize && h == m.ShipPickerHead
			switch {
			case selected:
				row.WriteString(fmt.Sprintf("[%2s]", h))
			case !present:
				row.WriteString("  - ")
			default:
				row.WriteString(fmt.Sprintf(" %2s ", h))
			}
		}
		lines = append(lines, row.String())
	}
	return lines
}

func renderHeadingThumbs(m Model) string {
	innerW, innerH := m.ShipPickerSize.Dim()
	if innerW < 1 {
		innerW, innerH = 13, 5
	}
	cols := 2
	boxW := innerW + 2
	if tw := m.TermW; tw >= 4*boxW+6 {
		cols = 4
	}
	var cells []string
	for _, h := range sprite.Headings {
		cells = append(cells, headingThumb(m, h, innerW, innerH))
	}
	var rows []string
	for i := 0; i < len(cells); i += cols {
		end := i + cols
		if end > len(cells) {
			end = len(cells)
		}
		rows = append(rows, joinHorizAll(cells[i:end]...))
	}
	return strings.Join(rows, "\n")
}

func headingThumb(m Model, h sprite.Heading, innerW, innerH int) string {
	focus := ""
	if h == m.ShipPickerHead {
		focus = "*"
	}
	title := fmt.Sprintf(" %s%s ", focus, h)
	sp, ok := m.frameAt(m.ShipPickerSize, h)
	var lines []string
	if !ok {
		lines = make([]string, innerH)
		for i := range lines {
			lines[i] = padPlain("", innerW)
		}
		if innerH > 0 {
			lines[innerH/2] = padPlain(" (none)", innerW)
		}
	} else {
		raw := strings.Split(renderComposite(sp), "\n")
		lines = make([]string, innerH)
		for i := 0; i < innerH; i++ {
			row := ""
			if i < len(raw) {
				row = raw[i]
			}
			lines[i] = padPlain(row, innerW)
		}
	}
	return box(title, lines, innerW)
}

func joinHorizAll(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out = joinHoriz(out, parts[i])
	}
	return out
}
