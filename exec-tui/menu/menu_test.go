package menu

// The exec-tui launcher: running exec-tui with no arguments opens a
// scrollable menu of every lab and configurator instead of the sim.
// j/k (or arrows) move, enter runs the highlighted program, q quits.
// Tests written before the implementation.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsi(s string) string { return ansiPat.ReplaceAllString(s, "") }

// fakeEntries builds n synthetic entries so navigation math stays stable
// no matter how the real catalog grows.
func fakeEntries(n int) []Entry {
	es := make([]Entry, n)
	for i := range es {
		es[i] = Entry{
			ID:    fmt.Sprintf("e%d", i),
			Title: fmt.Sprintf("ENTRY %02d", i),
			Desc:  fmt.Sprintf("description %d", i),
		}
	}
	return es
}

func sized(m Model, w, h int) Model {
	mm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return mm.(Model)
}

func key(m Model, r rune) Model {
	mm, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	return mm.(Model)
}

func keyCode(m Model, code rune) Model {
	mm, _ := m.Update(tea.KeyPressMsg{Code: code})
	return mm.(Model)
}

func TestMenuBoot(t *testing.T) {
	t.Run("happy: lists the programs with the first one selected", func(t *testing.T) {
		m := sized(New(Catalog(), ""), 100, 40)
		v := stripAnsi(m.View().Content)
		for _, want := range []string{"SCREENPLAY", "FLAME", "STARS", "LEGACY"} {
			if !strings.Contains(v, want) {
				t.Fatalf("menu missing %q:\n%s", want, v)
			}
		}
		if !strings.Contains(strings.ToLower(v), "what do you want to run") {
			t.Fatal("the menu must ask what to run")
		}
		if !strings.Contains(v, "enter run") {
			t.Fatal("the footer must explain enter")
		}
		first := Catalog()[0].Title
		marked := false
		for _, line := range strings.Split(v, "\n") {
			if strings.Contains(line, first) && strings.Contains(line, "▸") {
				marked = true
			}
		}
		if !marked {
			t.Fatalf("the first entry %q must carry the selection marker", first)
		}
	})
	t.Run("unhappy: a tiny terminal still renders the selection", func(t *testing.T) {
		m := sized(New(Catalog(), ""), 30, 6)
		v := stripAnsi(m.View().Content)
		if v == "" {
			t.Fatal("tiny terminals must still render")
		}
		if !strings.Contains(v, Catalog()[0].Title) {
			t.Fatal("the selected entry must stay visible on tiny terminals")
		}
	})
}

func TestMenuNavigation(t *testing.T) {
	t.Run("happy: j and k move with wrap at both ends", func(t *testing.T) {
		m := sized(New(fakeEntries(5), ""), 80, 30)
		m = key(m, 'j')
		if m.sel != 1 {
			t.Fatalf("j must move down, sel=%d", m.sel)
		}
		m = key(m, 'k')
		m = key(m, 'k')
		if m.sel != 4 {
			t.Fatalf("k from the top must wrap to the last entry, sel=%d", m.sel)
		}
		m = key(m, 'j')
		if m.sel != 0 {
			t.Fatalf("j from the bottom must wrap to the first entry, sel=%d", m.sel)
		}
	})
	t.Run("happy: arrow keys move too", func(t *testing.T) {
		m := sized(New(fakeEntries(5), ""), 80, 30)
		m = keyCode(m, tea.KeyDown)
		if m.sel != 1 {
			t.Fatalf("down arrow must move down, sel=%d", m.sel)
		}
		m = keyCode(m, tea.KeyUp)
		if m.sel != 0 {
			t.Fatalf("up arrow must move up, sel=%d", m.sel)
		}
	})
	t.Run("unhappy: unknown keys never move the selection", func(t *testing.T) {
		m := sized(New(fakeEntries(5), ""), 80, 30)
		for _, r := range []rune{'z', 'x', '1', ' '} {
			m = key(m, r)
		}
		if m.sel != 0 {
			t.Fatalf("unknown keys moved the selection to %d", m.sel)
		}
	})
}

func TestMenuScrolling(t *testing.T) {
	// height 10 leaves a 5-row window (2 title + blank + blank + footer).
	t.Run("happy: the window slides to keep the cursor visible", func(t *testing.T) {
		m := sized(New(fakeEntries(11), ""), 80, 10)
		if m.visible() >= 11 {
			t.Fatalf("test premise: window %d must be smaller than the list", m.visible())
		}
		for i := 0; i < 7; i++ {
			m = key(m, 'j')
		}
		if m.sel != 7 {
			t.Fatalf("sel=%d", m.sel)
		}
		if m.sel < m.offset || m.sel >= m.offset+m.visible() {
			t.Fatalf("cursor %d escaped the window [%d,%d)", m.sel, m.offset, m.offset+m.visible())
		}
		v := stripAnsi(m.View().Content)
		if !strings.Contains(v, "ENTRY 07") {
			t.Fatal("the selected entry must be rendered after scrolling down")
		}
		if strings.Contains(v, "ENTRY 00") {
			t.Fatal("entries scrolled off the top must not render")
		}
		// wrap to the top snaps the window back
		for i := 0; i < 4; i++ {
			m = key(m, 'j')
		}
		if m.sel != 0 || m.offset != 0 {
			t.Fatalf("wrapping to the top must reset the window, sel=%d offset=%d", m.sel, m.offset)
		}
		// wrapping backwards from the top shows the tail
		m = key(m, 'k')
		if m.sel != 10 {
			t.Fatalf("sel=%d", m.sel)
		}
		if m.sel < m.offset || m.sel >= m.offset+m.visible() {
			t.Fatalf("cursor %d escaped the window [%d,%d)", m.sel, m.offset, m.offset+m.visible())
		}
	})
	t.Run("unhappy: a tall terminal never scrolls", func(t *testing.T) {
		m := sized(New(fakeEntries(11), ""), 80, 40)
		for i := 0; i < 15; i++ {
			m = key(m, 'j')
		}
		if m.offset != 0 {
			t.Fatalf("a window taller than the list must never scroll, offset=%d", m.offset)
		}
		v := stripAnsi(m.View().Content)
		for i := 0; i < 11; i++ {
			if !strings.Contains(v, fmt.Sprintf("ENTRY %02d", i)) {
				t.Fatalf("tall terminals must show every entry, missing %02d", i)
			}
		}
	})
}

func TestMenuSelect(t *testing.T) {
	t.Run("happy: enter chooses the highlighted entry and quits", func(t *testing.T) {
		m := sized(New(fakeEntries(5), ""), 80, 30)
		m = key(m, 'j')
		m = key(m, 'j')
		mm, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = mm.(Model)
		e, ok := m.Chosen()
		if !ok || e.ID != "e2" {
			t.Fatalf("enter must choose entry e2, got %v ok=%v", e.ID, ok)
		}
		if cmd == nil {
			t.Fatal("enter must quit the menu program")
		}
		if _, isQuit := cmd().(tea.QuitMsg); !isQuit {
			t.Fatal("enter's command must be tea.Quit")
		}
	})
	t.Run("unhappy: q quits with no choice", func(t *testing.T) {
		m := sized(New(fakeEntries(5), ""), 80, 30)
		mm, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
		m = mm.(Model)
		if _, ok := m.Chosen(); ok {
			t.Fatal("q must not choose anything")
		}
		if cmd == nil {
			t.Fatal("q must quit")
		}
		if _, isQuit := cmd().(tea.QuitMsg); !isQuit {
			t.Fatal("q's command must be tea.Quit")
		}
	})
	t.Run("unhappy: ctrl+c quits with no choice", func(t *testing.T) {
		m := sized(New(fakeEntries(5), ""), 80, 30)
		mm, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		m = mm.(Model)
		if _, ok := m.Chosen(); ok {
			t.Fatal("ctrl+c must not choose anything")
		}
		if cmd == nil {
			t.Fatal("ctrl+c must quit")
		}
	})
}

func TestMenuStatus(t *testing.T) {
	t.Run("happy: a launch error renders on the status line", func(t *testing.T) {
		m := sized(New(fakeEntries(3), "screenplay: exit status 1"), 80, 30)
		if !strings.Contains(stripAnsi(m.View().Content), "screenplay: exit status 1") {
			t.Fatal("the status line must surface launch errors")
		}
	})
	t.Run("unhappy: an empty status adds no line", func(t *testing.T) {
		with := sized(New(fakeEntries(3), "boom"), 80, 30)
		without := sized(New(fakeEntries(3), ""), 80, 30)
		lw := len(strings.Split(with.View().Content, "\n"))
		lo := len(strings.Split(without.View().Content, "\n"))
		if lw != lo+1 {
			t.Fatalf("status must cost exactly one line: %d vs %d", lw, lo)
		}
	})
}

func TestLocateModule(t *testing.T) {
	t.Run("happy: walks up from a nested dir to a sibling module", func(t *testing.T) {
		root := t.TempDir()
		mod := filepath.Join(root, "screenplay-lab")
		deep := filepath.Join(root, "exec-tui", "cmd", "deep")
		if err := os.MkdirAll(mod, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(deep, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(mod, "go.mod"), []byte("module x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := LocateModule(deep, "screenplay-lab")
		if err != nil {
			t.Fatalf("LocateModule: %v", err)
		}
		if got != mod {
			t.Fatalf("got %q, want %q", got, mod)
		}
	})
	t.Run("unhappy: a missing module names itself in the error", func(t *testing.T) {
		_, err := LocateModule(t.TempDir(), "nope-lab")
		if err == nil {
			t.Fatal("a missing module must error")
		}
		if !strings.Contains(err.Error(), "nope-lab") {
			t.Fatalf("the error must name the module, got %v", err)
		}
	})
}

func TestCatalog(t *testing.T) {
	t.Run("happy: the four named programs lead the list", func(t *testing.T) {
		c := Catalog()
		if len(c) < 4 {
			t.Fatalf("catalog too small: %d", len(c))
		}
		want := []string{"screenplay", "flame", "stars-config", "legacy"}
		for i, id := range want {
			if c[i].ID != id {
				t.Fatalf("entry %d must be %q, got %q", i, id, c[i].ID)
			}
		}
	})
	t.Run("happy: every entry is fully described", func(t *testing.T) {
		seen := map[string]bool{}
		for _, e := range Catalog() {
			if e.ID == "" || e.Title == "" || e.Desc == "" {
				t.Fatalf("entry %+v must carry id, title and description", e)
			}
			if seen[e.ID] {
				t.Fatalf("duplicate entry id %q", e.ID)
			}
			seen[e.ID] = true
		}
	})
	t.Run("unhappy: launch specs are consistent", func(t *testing.T) {
		for _, e := range Catalog() {
			if e.Module == "" && e.Pkg != "" {
				t.Fatalf("in-process entry %q must not carry a pkg path", e.ID)
			}
			if e.Module != "" && !strings.HasPrefix(e.Pkg, ".") {
				t.Fatalf("external entry %q needs a ./-relative pkg path, got %q", e.ID, e.Pkg)
			}
		}
	})
}
