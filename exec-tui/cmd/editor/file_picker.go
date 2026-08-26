package editor

// Ctrl-P is the file picker: a quick-open over every *.json atlas in
// the assets folder. Typing filters the list, ctrl-j/ctrl-k (or the
// arrows) move the highlight, enter opens it, esc leaves everything
// alone. A preview pane renders the highlighted ship's full composite
// — outline, foreground, and background together — and the file being
// left stays warm in the editor, so unsaved edits survive the switch.

import (
	"path/filepath"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func (m *Model) openFilePicker() {
	if m.FilePickerOpen || m.PickerOpen || m.GlyphGridOpen || m.ColorPaletteOpen || m.Inserting {
		return
	}
	files, err := ListAssets(m.assetsDir())
	if err != nil {
		m.SetErr("assets: " + err.Error())
		return
	}
	m.Files = files
	m.FilePickerQuery = ""
	m.FilePickerIdx = 0
	for i, f := range files {
		if f.Path == m.Path {
			m.FilePickerIdx = i
			break
		}
	}
	m.FilePickerOpen = true
	m.status = "files  type to filter  ^J/^K move  enter open  esc close"
}

func (m *Model) closeFilePicker() {
	m.FilePickerOpen = false
}

// filteredFiles is the working set under the current query: a case-
// insensitive substring match on the file name.
func (m Model) filteredFiles() []Asset {
	q := strings.ToLower(m.FilePickerQuery)
	if q == "" {
		return m.Files
	}
	var out []Asset
	for _, f := range m.Files {
		if strings.Contains(strings.ToLower(f.Name), q) {
			out = append(out, f)
		}
	}
	return out
}

// filePickerCursor clamps the highlight into the filtered list, which
// shrinks and grows as the query changes.
func (m Model) filePickerCursor() int {
	n := len(m.filteredFiles())
	if n == 0 || m.FilePickerIdx < 0 {
		return 0
	}
	if m.FilePickerIdx >= n {
		return n - 1
	}
	return m.FilePickerIdx
}

func (m *Model) stepFilePicker(delta int) {
	n := len(m.filteredFiles())
	if n == 0 {
		m.FilePickerIdx = 0
		return
	}
	m.FilePickerIdx = ((m.filePickerCursor()+delta)%n + n) % n
}

func (m Model) handleFilePickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+p":
		m.closeFilePicker()
		m.status = "file picker closed"
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		files := m.filteredFiles()
		if len(files) == 0 {
			m.status = "no matching file"
			return m, nil
		}
		pick := files[m.filePickerCursor()]
		if err := m.openAsset(pick.Path); err != nil {
			m.SetErr(pick.Name + ": " + err.Error())
			return m, nil
		}
		m.closeFilePicker()
		return m, nil
	case "up", "ctrl+k":
		m.stepFilePicker(-1)
		return m, nil
	case "down", "ctrl+j":
		m.stepFilePicker(1)
		return m, nil
	case "backspace":
		if q := []rune(m.FilePickerQuery); len(q) > 0 {
			m.FilePickerQuery = string(q[:len(q)-1])
		}
		m.FilePickerIdx = 0
		return m, nil
	case "space":
		m.FilePickerQuery += " "
		m.FilePickerIdx = 0
		return m, nil
	}
	if r := runeFrom(msg); r != 0 && !unicode.IsControl(r) {
		m.FilePickerQuery += string(r)
		m.FilePickerIdx = 0
	}
	return m, nil
}

func renderFilePicker(m Model) string {
	files := m.filteredFiles()
	cursor := m.filePickerCursor()

	helpMove := "type to filter  ^J/^K move"
	helpPick := "enter open  esc close"
	title := " files  " + filepath.Base(m.assetsDir()) + " "

	nameW := len(helpMove)
	for _, s := range []string{helpPick, title} {
		if n := len([]rune(s)); n > nameW {
			nameW = n
		}
	}
	for _, f := range m.Files {
		if n := len([]rune(f.Name)) + 4; n > nameW {
			nameW = n
		}
	}
	rows := []string{
		padPlain("> "+m.FilePickerQuery+"▌", nameW),
		strings.Repeat("─", nameW),
	}
	if len(files) == 0 {
		rows = append(rows, padPlain(" (no matching file)", nameW))
	}
	for i, f := range files {
		cur := " "
		if f.Path == m.Path {
			cur = "*"
		}
		row := padPlain(" "+cur+" "+f.Name, nameW)
		if i == cursor {
			row = "\x1b[7m" + row + "\x1b[0m"
		}
		rows = append(rows, row)
	}
	rows = append(rows,
		padPlain("", nameW),
		padPlain(helpMove, nameW),
		padPlain(helpPick, nameW))
	list := box(title, rows, nameW)

	if len(files) == 0 {
		return list
	}
	return joinHoriz(list, renderFilePreview(m, files[cursor]))
}

// renderFilePreview is the highlighted file's full composite. The open
// file previews its live canvas, so unsaved edits show true.
func renderFilePreview(m Model, f Asset) string {
	const minW = 16
	title := " " + f.Name + " "
	a := m.previewAtlas(f.Path)
	if a == nil {
		return box(title, []string{padPlain(" (unreadable)", minW)}, minW)
	}
	sp, ok := previewFrame(a, m.Size, m.Heading)
	if !ok {
		return box(title, []string{padPlain(" (no frames)", minW)}, minW)
	}
	lines := strings.Split(renderComposite(sp), "\n")
	w := sp.Width
	if w < minW {
		w = minW
	}
	for i := range lines {
		lines[i] = padPlain(lines[i], w)
	}
	return box(title, lines, w)
}

// previewAtlas is the atlas behind a picker row: the live canvas for
// the open file, a warm cache hit, or a fresh read. nil when the file
// cannot be read.
func (m Model) previewAtlas(path string) *sprite.Atlas {
	if path == m.Path && m.Atlas != nil {
		return m.Atlas
	}
	if a := m.atlases[path]; a != nil {
		return a
	}
	a, err := LoadAsset(path)
	if err != nil {
		return nil
	}
	return a
}

// previewFrame picks the frame a file would open on: the editor's
// current size and heading when the atlas has it, else the biggest
// frame it ships.
func previewFrame(a *sprite.Atlas, sz sprite.Size, h sprite.Heading) (sprite.Sprite, bool) {
	if sp, ok := a.Frame(sz, h); ok {
		return sp, true
	}
	for i := len(sprite.Sizes) - 1; i >= 0; i-- {
		s := sprite.Sizes[i]
		if sp, ok := a.Frame(s, h); ok {
			return sp, true
		}
		for _, hh := range sprite.Headings {
			if sp, ok := a.Frame(s, hh); ok {
				return sp, true
			}
		}
	}
	return sprite.Sprite{}, false
}
