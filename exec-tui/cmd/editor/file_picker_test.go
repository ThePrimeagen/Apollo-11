package editor

// Tests written FIRST. Ctrl-P is the file picker — the quick-open over
// every *.json atlas in the assets folder. Typing filters the list,
// ctrl-j/ctrl-k (or arrows) move, enter opens the highlighted file,
// esc leaves everything alone. The file being left stays warm in the
// editor, so unsaved edits survive the switch and the next save
// flushes them to disk. A corrupt file must never crash the picker.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// pickerEd opens an editor on a folder of three distinct one-glyph
// ships: alpha Ω, bravo Ψ, charlie Δ.
func pickerEd(t *testing.T) (Model, string) {
	t.Helper()
	dir := t.TempDir()
	writeMiniShip(t, dir, "alpha", 'Ω')
	writeMiniShip(t, dir, "bravo", 'Ψ')
	writeMiniShip(t, dir, "charlie", 'Δ')
	m, err := Open(dir)
	if err != nil {
		t.Fatalf("Open(dir): %v", err)
	}
	m.TermW, m.TermH = 120, 40
	return m, dir
}

func TestFilePickerOpens(t *testing.T) {
	t.Run("happy: ctrl-p lists every atlas in the folder and starts on the open file", func(t *testing.T) {
		m, _ := pickerEd(t)
		m = send(m, keyCtrl('p'))
		if !m.FilePickerOpen {
			t.Fatal("ctrl-p must open the file picker")
		}
		if m.FilePickerQuery != "" {
			t.Fatalf("the picker must open with an empty filter, got %q", m.FilePickerQuery)
		}
		v := m.View().Content
		for _, name := range []string{"alpha", "bravo", "charlie"} {
			if !strings.Contains(v, name) {
				t.Fatalf("picker view missing file %q", name)
			}
		}
		files := m.filteredFiles()
		if len(files) != 3 {
			t.Fatalf("picker must hold all 3 files, got %d", len(files))
		}
		if files[m.FilePickerIdx].Name != "alpha" {
			t.Fatalf("the highlight must start on the open file, got %q", files[m.FilePickerIdx].Name)
		}
		if !strings.Contains(v, "* alpha") {
			t.Fatal("the open file must be marked in the list")
		}
	})
	t.Run("unhappy: ctrl-p while the color picker is open does not open the files", func(t *testing.T) {
		m, _ := pickerEd(t)
		beforePath := m.Path
		m = send(m, key('c'))
		if !m.PickerOpen {
			t.Fatal("c must open the color picker")
		}
		beforeCanvas := m.Current()
		m = send(m, keyCtrl('p'))
		if m.FilePickerOpen {
			t.Fatal("ctrl-p must not steal an already-open color picker")
		}
		if m.Path != beforePath {
			t.Fatal("ctrl-p in another modal must not switch files")
		}
		if sprite.Render(m.Current()) != sprite.Render(beforeCanvas) {
			t.Fatal("ctrl-p in another modal must not paint the canvas")
		}
	})
}

func TestFilePickerFilter(t *testing.T) {
	t.Run("happy: typing narrows the list and backspace widens it back", func(t *testing.T) {
		m, _ := pickerEd(t)
		m = send(m, keyCtrl('p'))
		m = send(m, key('b'))
		if m.FilePickerQuery != "b" {
			t.Fatalf("typed letters must build the filter, got %q", m.FilePickerQuery)
		}
		files := m.filteredFiles()
		if len(files) != 1 || files[0].Name != "bravo" {
			t.Fatalf("filter b must keep only bravo, got %#v", files)
		}
		m = send(m, keyType(tea.KeyBackspace))
		if m.FilePickerQuery != "" {
			t.Fatalf("backspace must trim the filter, got %q", m.FilePickerQuery)
		}
		if got := len(m.filteredFiles()); got != 3 {
			t.Fatalf("an empty filter must show every file again, got %d", got)
		}
	})
	t.Run("unhappy: enter with no matching file keeps the picker and the file", func(t *testing.T) {
		m, _ := pickerEd(t)
		beforePath := m.Path
		m = send(m, keyCtrl('p'))
		for _, r := range "zzz" {
			m = send(m, key(r))
		}
		if got := len(m.filteredFiles()); got != 0 {
			t.Fatalf("filter zzz must match nothing, got %d", got)
		}
		m = send(m, keyType(tea.KeyEnter))
		if !m.FilePickerOpen {
			t.Fatal("enter on no matches must keep the picker open")
		}
		if m.Path != beforePath {
			t.Fatalf("enter on no matches must not switch files, got %q", m.Path)
		}
		_ = m.View().Content
	})
}

func TestFilePickerOpensFile(t *testing.T) {
	t.Run("happy: ctrl-j and arrows move the highlight; enter opens that file", func(t *testing.T) {
		m, dir := pickerEd(t)
		m = send(m, keyCtrl('p'))
		m = send(m, keyCtrl('j'))
		files := m.filteredFiles()
		if files[m.FilePickerIdx].Name != "bravo" {
			t.Fatalf("ctrl-j must step to bravo, got %q", files[m.FilePickerIdx].Name)
		}
		m = send(m, keyArrow(tea.KeyDown))
		if files[m.FilePickerIdx].Name != "charlie" {
			t.Fatalf("down arrow must step to charlie, got %q", files[m.FilePickerIdx].Name)
		}
		m = send(m, keyArrow(tea.KeyUp))
		m = send(m, keyCtrl('k'))
		if files[m.FilePickerIdx].Name != "alpha" {
			t.Fatalf("up and ctrl-k must walk back to alpha, got %q", files[m.FilePickerIdx].Name)
		}
		m = send(m, keyCtrl('j'))
		m = send(m, keyType(tea.KeyEnter))
		if m.FilePickerOpen {
			t.Fatal("enter must close the picker")
		}
		if want := filepath.Join(dir, "bravo.json"); m.Path != want {
			t.Fatalf("enter must open the highlighted file, got %q want %q", m.Path, want)
		}
		if got := m.Current().At(0, 0).Ch; got != 'Ψ' {
			t.Fatalf("the canvas must show the opened file, got %q want Ψ", string(got))
		}
	})
	t.Run("unhappy: esc closes without switching file or canvas", func(t *testing.T) {
		m, _ := pickerEd(t)
		beforePath := m.Path
		before := sprite.Render(m.Current())
		m = send(m, keyCtrl('p'))
		m = send(m, keyCtrl('j'))
		m = send(m, keyType(tea.KeyEsc))
		if m.FilePickerOpen {
			t.Fatal("esc must close the picker")
		}
		if m.Path != beforePath {
			t.Fatalf("esc must keep the open file, got %q", m.Path)
		}
		if sprite.Render(m.Current()) != before {
			t.Fatal("esc must not touch the canvas")
		}
	})
}

func TestFilePickerKeepsEdits(t *testing.T) {
	t.Run("happy: unsaved paint survives the switch and save flushes it to disk", func(t *testing.T) {
		m, dir := pickerEd(t)
		want := sprite.Cell{Ch: '♥', FG: 123, BG: 45}
		sp := cloneSprite(m.Current())
		sp.Set(1, 2, want)
		m.setCurrent(sp)

		m = send(m, keyCtrl('p'))
		m = send(m, keyCtrl('j'))
		m = send(m, keyType(tea.KeyEnter))
		if want := filepath.Join(dir, "bravo.json"); m.Path != want {
			t.Fatalf("the switch must land on bravo, got %q", m.Path)
		}
		if err := m.Save(); err != nil {
			t.Fatalf("save: %v", err)
		}
		alpha, err := sprite.LoadFile(filepath.Join(dir, "alpha.json"))
		if err != nil {
			t.Fatalf("reload alpha: %v", err)
		}
		got := alpha.MustFrame(sprite.Size1, sprite.N).At(1, 2)
		if got.Ch != want.Ch || got.FG != want.FG || got.BG != want.BG {
			t.Fatalf("alpha's unsaved edit was not flushed: disk %+v, want %+v", got, want)
		}
		bravo, err := sprite.LoadFile(m.Path)
		if err != nil {
			t.Fatalf("reload bravo: %v", err)
		}
		if bravo.MustFrame(sprite.Size1, sprite.N).At(0, 0).Ch != 'Ψ' {
			t.Fatal("bravo must still carry its own art after the flush")
		}
	})
	t.Run("unhappy: a corrupt sibling cannot be opened — the editor stays on the current file", func(t *testing.T) {
		m, dir := pickerEd(t)
		beforePath := m.Path
		if err := os.WriteFile(filepath.Join(dir, "zulu.json"), []byte("{nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		m = send(m, keyCtrl('p'))
		for _, r := range "zu" {
			m = send(m, key(r))
		}
		files := m.filteredFiles()
		if len(files) != 1 || files[0].Name != "zulu" {
			t.Fatalf("the picker must list the new sibling, got %#v", files)
		}
		m = send(m, keyType(tea.KeyEnter))
		if !m.FilePickerOpen {
			t.Fatal("a failed open must keep the picker up so another file can be picked")
		}
		if m.Path != beforePath {
			t.Fatalf("a failed open must not switch files, got %q", m.Path)
		}
		_ = m.View().Content
	})
}

func TestFilePickerPreview(t *testing.T) {
	t.Run("happy: the preview shows the highlighted file's art", func(t *testing.T) {
		m, _ := pickerEd(t)
		m = send(m, keyCtrl('p'))
		if v := m.View().Content; strings.Contains(v, "Ψ") {
			t.Fatal("bravo's art must not show before bravo is highlighted")
		}
		m = send(m, keyCtrl('j'))
		if v := m.View().Content; !strings.Contains(v, "Ψ") {
			t.Fatal("highlighting bravo must preview bravo's art")
		}
	})
	t.Run("unhappy: an unreadable file previews a warning instead of crashing", func(t *testing.T) {
		m, dir := pickerEd(t)
		if err := os.WriteFile(filepath.Join(dir, "zulu.json"), []byte("{nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		m = send(m, keyCtrl('p'))
		for _, r := range "zu" {
			m = send(m, key(r))
		}
		if v := m.View().Content; !strings.Contains(v, "(unreadable)") {
			t.Fatal("a corrupt file must preview (unreadable)")
		}
	})
}
