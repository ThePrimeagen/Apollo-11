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
		for _, want := range []string{"MAIN", "01. Moon Orbit", "02. Walkthrough", "Landing", "America", "FLAME", "STARS", "LEGACY"} {
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

// headerLine finds the line rendered as exactly the section name (so
// "CONFIG" never matches the "STARS CONFIG" entry), or -1.
func headerLine(v, name string) int {
	for i, l := range strings.Split(v, "\n") {
		if strings.TrimSpace(l) == name {
			return i
		}
	}
	return -1
}

// entryLine finds the first line containing the entry title, or -1.
func entryLine(v, title string) int {
	for i, l := range strings.Split(v, "\n") {
		if strings.Contains(l, title) {
			return i
		}
	}
	return -1
}

func TestMenuSections(t *testing.T) {
	t.Run("happy: each category header renders once, above its entries", func(t *testing.T) {
		v := stripAnsi(sized(New(Catalog(), ""), 100, 40).View().Content)
		order := []struct{ header, first string }{
			{"Screenplays", "MAIN"},
			{"Scenes", "Landing"},
			{"CONFIG", "FLAME CONFIG"},
			{"Particles", "PARTICLE CONFIG"},
			{"LEGACY TUIS", "LEGACY EXEC"},
		}
		prev := -1
		for _, s := range order {
			hi := headerLine(v, s.header)
			if hi < 0 {
				t.Fatalf("menu missing the %q header:\n%s", s.header, v)
			}
			if hi <= prev {
				t.Fatalf("header %q out of order at line %d:\n%s", s.header, hi, v)
			}
			ei := entryLine(v, s.first)
			if ei < hi {
				t.Fatalf("entry %q renders above its %q header:\n%s", s.first, s.header, v)
			}
			seen := 0
			for _, l := range strings.Split(v, "\n") {
				if strings.TrimSpace(l) == s.header {
					seen++
				}
			}
			if seen != 1 {
				t.Fatalf("header %q must render exactly once, saw %d", s.header, seen)
			}
			prev = hi
		}
	})
	t.Run("happy: MAIN then 01. Moon Orbit then 02. Walkthrough sit directly under Screenplays", func(t *testing.T) {
		v := stripAnsi(sized(New(Catalog(), ""), 100, 40).View().Content)
		hi := headerLine(v, "Screenplays")
		main := entryLine(v, "MAIN")
		orbit := entryLine(v, "01. Moon Orbit")
		walk := entryLine(v, "02. Walkthrough")
		if hi < 0 || main < 0 || orbit < 0 || walk < 0 {
			t.Fatalf("menu missing Screenplays / MAIN / 01. Moon Orbit / 02. Walkthrough:\n%s", v)
		}
		if main <= hi {
			t.Fatalf("MAIN must sit under Screenplays, header=%d entry=%d:\n%s", hi, main, v)
		}
		if orbit != main+1 {
			t.Fatalf("01. Moon Orbit must sit directly below MAIN, MAIN=%d orbit=%d:\n%s", main, orbit, v)
		}
		if walk != orbit+1 {
			t.Fatalf("02. Walkthrough must sit directly below 01. Moon Orbit, orbit=%d walk=%d:\n%s", orbit, walk, v)
		}
		if scenes := headerLine(v, "Scenes"); scenes >= 0 && walk > scenes {
			t.Fatalf("02. Walkthrough rendered under Scenes, not Screenplays:\n%s", v)
		}
	})
	t.Run("happy: headers are never selectable — j walks entry to entry", func(t *testing.T) {
		m := sized(New(Catalog(), ""), 100, 40)
		m = key(m, 'j')
		if got := Catalog()[m.sel].ID; got != "moon" {
			t.Fatalf("j from MAIN must land on moon, got %q", got)
		}
		m = key(m, 'j')
		if got := Catalog()[m.sel].ID; got != "closeup" {
			t.Fatalf("j from moon must land on closeup, got %q", got)
		}
		m = key(m, 'j')
		if got := Catalog()[m.sel].ID; got != "landing" {
			t.Fatalf("j from closeup must land on the landing scene, got %q", got)
		}
		m = key(m, 'k')
		m = key(m, 'k')
		m = key(m, 'k')
		m = key(m, 'k')
		if got := Catalog()[m.sel].ID; got != "timeline" {
			t.Fatalf("k from the top must wrap to timeline, got %q", got)
		}
		v := stripAnsi(m.View().Content)
		for _, line := range strings.Split(v, "\n") {
			if strings.Contains(line, "▸") && !strings.Contains(line, "TIMELINE") {
				t.Fatalf("the marker sits on a non-selected line: %q", line)
			}
		}
	})
	t.Run("unhappy: sectionless entries render without headers or spacers", func(t *testing.T) {
		es := fakeEntries(5)
		v := stripAnsi(sized(New(es, ""), 80, 30).View().Content)
		if got, want := len(strings.Split(v, "\n")), len(es)+chrome; got != want {
			t.Fatalf("a sectionless menu must add no rows: %d lines, want %d:\n%s", got, want, v)
		}
	})
	t.Run("unhappy: a tiny terminal keeps the selection visible through every section", func(t *testing.T) {
		m := sized(New(Catalog(), ""), 30, 6)
		for range Catalog() {
			want := Catalog()[m.sel].Title
			if v := stripAnsi(m.View().Content); !strings.Contains(v, want) {
				t.Fatalf("selected entry %q fell out of the tiny window:\n%s", want, v)
			}
			m = key(m, 'j')
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
		mod := filepath.Join(root, "dsky-lab")
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
		got, err := LocateModule(deep, "dsky-lab")
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
	t.Run("happy: the catalog runs screenplays, scenes, config, particles, legacy", func(t *testing.T) {
		c := Catalog()
		want := []string{
			"screenplay", "moon", "closeup",
			"landing", "america",
			"flame", "stars-config", "editor",
			"particle", "dust-config", "gunfire-config",
			"legacy", "timeline",
		}
		if len(c) != len(want) {
			t.Fatalf("catalog holds %d entries, want %d", len(c), len(want))
		}
		for i, id := range want {
			if c[i].ID != id {
				t.Fatalf("entry %d must be %q, got %q", i, id, c[i].ID)
			}
		}
	})
	t.Run("happy: entries group under their category headers in order", func(t *testing.T) {
		wantSections := map[string]string{
			"screenplay":     "Screenplays",
			"moon":           "Screenplays",
			"closeup":        "Screenplays",
			"landing":        "Scenes",
			"america":        "Scenes",
			"flame":          "CONFIG",
			"stars-config":   "CONFIG",
			"editor":         "CONFIG",
			"particle":       "Particles",
			"dust-config":    "Particles",
			"gunfire-config": "Particles",
			"legacy":         "LEGACY TUIS",
			"timeline":       "LEGACY TUIS",
		}
		seen := map[string]bool{}
		last := ""
		for _, e := range Catalog() {
			if want := wantSections[e.ID]; e.Section != want {
				t.Fatalf("entry %q sits in section %q, want %q", e.ID, e.Section, want)
			}
			if e.Section != last && seen[e.Section] {
				t.Fatalf("section %q is split — categories must stay contiguous", e.Section)
			}
			seen[e.Section] = true
			last = e.Section
		}
	})
	t.Run("happy: screenplays are named MAIN then 01. Moon Orbit then 02. Walkthrough", func(t *testing.T) {
		c := Catalog()
		if len(c) < 3 {
			t.Fatal("catalog must hold MAIN, 01. Moon Orbit, and 02. Walkthrough")
		}
		if c[0].ID != "screenplay" || c[0].Title != "MAIN" || c[0].Section != "Screenplays" {
			t.Fatalf("first entry must be MAIN under Screenplays, got %+v", c[0])
		}
		if c[1].ID != "moon" || c[1].Title != "01. Moon Orbit" || c[1].Section != "Screenplays" {
			t.Fatalf("second entry must be 01. Moon Orbit under Screenplays, got %+v", c[1])
		}
		if c[2].ID != "closeup" || c[2].Title != "02. Walkthrough" || c[2].Section != "Screenplays" || c[2].Pkg != "./cmd/lunarcloseup" {
			t.Fatalf("third entry must be 02. Walkthrough (closeup → ./cmd/lunarcloseup) under Screenplays, got %+v", c[2])
		}
	})
	t.Run("happy: Scenes lists the portable landing", func(t *testing.T) {
		c := Catalog()
		if len(c) < 4 {
			t.Fatal("catalog must hold the landing scene after the screenplays")
		}
		if c[3].ID != "landing" || c[3].Title != "Landing" || c[3].Section != "Scenes" || c[3].Pkg != "./cmd/landing" {
			t.Fatalf("fourth entry must be Landing under Scenes → ./cmd/landing, got %+v", c[3])
		}
	})
	t.Run("happy: Particles lists the nyan trail, dust-off, and gunfire tuners", func(t *testing.T) {
		c := Catalog()
		want := []struct {
			id, title, pkg string
		}{
			{"particle", "PARTICLE CONFIG", "./cmd/adjustparticle/main"},
			{"dust-config", "DUSTOFF CONFIG", "./cmd/adjustdust/main"},
			{"gunfire-config", "GUNFIRE CONFIG", "./cmd/adjustgunfire/main"},
		}
		got := make([]Entry, 0, 3)
		for _, e := range c {
			if e.Section == "Particles" {
				got = append(got, e)
			}
		}
		if len(got) != len(want) {
			t.Fatalf("Particles must list %d editable particle components, got %d: %+v", len(want), len(got), got)
		}
		for i, w := range want {
			if got[i].ID != w.id || got[i].Title != w.title || got[i].Pkg != w.pkg || got[i].Section != "Particles" {
				t.Fatalf("Particles entry %d must be %s (%s) → %s, got %+v", i, w.title, w.id, w.pkg, got[i])
			}
		}
	})
	t.Run("happy: Scenes lists America right after the landing", func(t *testing.T) {
		c := Catalog()
		if len(c) < 5 {
			t.Fatal("catalog must hold the America scene after the landing")
		}
		if c[4].ID != "america" || c[4].Title != "America" || c[4].Section != "Scenes" || c[4].Pkg != "./cmd/america" {
			t.Fatalf("fifth entry must be America under Scenes → ./cmd/america, got %+v", c[4])
		}
	})
	t.Run("unhappy: gunfire is not a scene and the one-shot demo stays off the launcher", func(t *testing.T) {
		for _, e := range Catalog() {
			if e.ID == "gunfire" {
				t.Fatalf("gunfire is a particle component, not a scene — the demo must not be listed, found %+v", e)
			}
			if e.Title == "Gunfire" && e.Section == "Scenes" {
				t.Fatalf("Scenes must not list Gunfire, found %+v", e)
			}
			if e.ID == "gunfire-config" && e.Section == "CONFIG" {
				t.Fatalf("the gunfire tuner belongs under Particles, not CONFIG, found %+v", e)
			}
			if (e.ID == "particle" || e.ID == "dust-config") && e.Section == "CONFIG" {
				t.Fatalf("particle tuners belong under Particles, not CONFIG, found %+v", e)
			}
		}
	})
	t.Run("unhappy: America is a scene, not a screenplay, and never doubles up", func(t *testing.T) {
		seen := 0
		for _, e := range Catalog() {
			if e.ID != "america" {
				continue
			}
			seen++
			if e.Section == "Screenplays" {
				t.Fatalf("America must sit under Scenes, found %+v", e)
			}
		}
		if seen != 1 {
			t.Fatalf("America must be listed exactly once, saw %d", seen)
		}
	})
	t.Run("unhappy: old MAIN PROGRAM / SCREENPLAY / MOON SCREENPLAY / LUNAR LANDER CLOSE-UP labels are gone", func(t *testing.T) {
		for _, e := range Catalog() {
			if e.Section == "MAIN PROGRAM" {
				t.Fatalf("section MAIN PROGRAM must be Screenplays, found %+v", e)
			}
			if e.Title == "SCREENPLAY" {
				t.Fatalf("premiere title must be MAIN, found %+v", e)
			}
			if e.Title == "MOON SCREENPLAY" {
				t.Fatalf("moon title must be 01. Moon Orbit, found %+v", e)
			}
			if e.Title == "LUNAR LANDER CLOSE-UP" {
				t.Fatalf("closeup title must be 02. Walkthrough, found %+v", e)
			}
			if e.Title == "2. Walkthrough" {
				t.Fatalf("closeup must match the zero-padded 01. style: 02. Walkthrough, found %+v", e)
			}
			if e.ID == "moon" && e.Section == "DEMO" {
				t.Fatalf("moon must live under Screenplays, not DEMO, found %+v", e)
			}
			if e.ID == "closeup" && e.Section == "DEMO" {
				t.Fatalf("closeup must live under Screenplays, not DEMO, found %+v", e)
			}
			if e.Section == "DEMO" {
				t.Fatalf("the DEMO section is gone — scenes replaced the demos, found %+v", e)
			}
			switch e.ID {
			case "lander", "stars", "nyan", "dustoff", "dsky", "button", "gunfire":
				t.Fatalf("demo %q must not be listed, found %+v", e.ID, e)
			}
		}
	})
	t.Run("unhappy: the seg lab is gone from the launcher", func(t *testing.T) {
		for _, e := range Catalog() {
			if e.ID == "seg" || strings.Contains(e.Title, "SEG") {
				t.Fatalf("the seg lab must not be listed, found %+v", e)
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
			switch {
			case e.Module != "":
				// A sibling module: go run Pkg inside Module's dir.
				if !strings.HasPrefix(e.Pkg, ".") {
					t.Fatalf("external entry %q needs a ./-relative pkg path, got %q", e.ID, e.Pkg)
				}
			case e.Pkg != "":
				// This module: go run Pkg from our own module root, so
				// everything we own launches out of cmd/.
				if !strings.HasPrefix(e.Pkg, "./cmd/") {
					t.Fatalf("in-module entry %q must live under ./cmd/, got %q", e.ID, e.Pkg)
				}
			}
		}
	})
	t.Run("happy: in-module entries point at real cmd packages", func(t *testing.T) {
		for _, e := range Catalog() {
			if e.Module != "" || e.Pkg == "" {
				continue
			}
			dir := filepath.Join("..", filepath.FromSlash(e.Pkg))
			info, err := os.Stat(dir)
			if err != nil || !info.IsDir() {
				t.Fatalf("entry %q points at %q which is not a package dir: %v", e.ID, e.Pkg, err)
			}
			glob, err := filepath.Glob(filepath.Join(dir, "*.go"))
			if err != nil || len(glob) == 0 {
				t.Fatalf("entry %q points at %q which holds no Go files", e.ID, e.Pkg)
			}
		}
	})
	t.Run("unhappy: the dissolved lab modules are gone from the launcher", func(t *testing.T) {
		for _, e := range Catalog() {
			switch e.Module {
			case "lander-lab", "stars-lab", "screenplay-lab":
				t.Fatalf("entry %q still launches into the dissolved %q module", e.ID, e.Module)
			}
		}
	})
}

func TestModuleRoot(t *testing.T) {
	t.Run("happy: walks up from a nested dir to this module's go.mod", func(t *testing.T) {
		root := t.TempDir()
		deep := filepath.Join(root, "cmd", "deep", "deeper")
		if err := os.MkdirAll(deep, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := ModuleRoot(deep)
		if err != nil {
			t.Fatalf("ModuleRoot: %v", err)
		}
		if got != root {
			t.Fatalf("got %q, want %q", got, root)
		}
	})
	t.Run("unhappy: no go.mod above means a clear error", func(t *testing.T) {
		if _, err := ModuleRoot(t.TempDir()); err == nil {
			t.Fatal("a dir outside any module must error")
		}
	})
	t.Run("happy: Resolve joins a relative path onto this module", func(t *testing.T) {
		got := Resolve("scenes/landing/config.json")
		if !filepath.IsAbs(got) || filepath.Base(got) != "config.json" {
			t.Fatalf("Resolve %q, want an absolute landing config path", got)
		}
		if Resolve("/tmp/x.json") != "/tmp/x.json" {
			t.Fatal("an absolute path must pass through")
		}
	})
	t.Run("unhappy: Resolve of empty is empty", func(t *testing.T) {
		if Resolve("") != "" {
			t.Fatal("empty must stay empty")
		}
	})
}
