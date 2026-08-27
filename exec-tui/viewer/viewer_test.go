package viewer

// Tests written FIRST: the component viewer is one runner-list item
// that cycles every component, particle effect, and scene. The center
// top carries two lines — a 3-height termfont title of the current
// item, then a 1-height slightly-darker-gray type ("component",
// "particle", or "scene"). n/p (and j/k, arrows) walk the catalog,
// wrapping at both ends. e opens the matching editor: a particle
// effect launches that effect's tuner, a scene launches that scene's
// tuner, and a component opens the ASCII sprite editor on its atlas.
// Closing the sub-edit resumes on the same item. q leaves with no
// edit. An empty catalog never panics.

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustcloud"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustdust"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustflame"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustgunfire"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustparticle"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustsky"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/editor"
	"github.com/theprimeagen/apollo-11/terminal-fonts/termfont"
)

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsi(s string) string { return ansiPat.ReplaceAllString(s, "") }

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

func boot() Model {
	return sized(New(0), 80, 24)
}

func findItem(t *testing.T, kind Kind, id string) int {
	t.Helper()
	for i, it := range Catalog() {
		if it.ID == id {
			if it.Kind != kind {
				t.Fatalf("item %q is %s, want %s", id, it.Kind, kind)
			}
			return i
		}
	}
	t.Fatalf("catalog missing %s %q", kind, id)
	return -1
}

func TestCatalog(t *testing.T) {
	t.Run("happy: shotgun leads, then every component, particle and scene, each once", func(t *testing.T) {
		c := Catalog()
		if len(c) == 0 {
			t.Fatal("the viewer must list items")
		}
		if c[0].ID != "shotgun" || c[0].Kind != KindComponent || c[0].Title != "SHOTGUN" {
			t.Fatalf("first item must be the SHOTGUN component, got %+v", c[0])
		}
		want := map[string]Kind{
			"shotgun":   KindComponent,
			"stars":     KindComponent,
			"sky":       KindComponent,
			"cloud":     KindComponent,
			"lander":    KindComponent,
			"flag":        KindComponent,
			"transition": KindComponent,
			"eagle":     KindComponent,
			"moon":      KindComponent,
			"dsky":      KindComponent,
			"title":     KindComponent,
			"astronaut": KindComponent,
			"rocket":    KindComponent,
			"gunfire":   KindParticle,
			"flame":     KindParticle,
			"dust":      KindParticle,
			"nyan":      KindParticle,
			"landing":   KindScene,
			"america":   KindScene,
			"moonwalk":  KindScene,
			"skies":     KindScene,
		}
		seen := map[string]bool{}
		for _, it := range c {
			if it.ID == "" || it.Title == "" {
				t.Fatalf("item %+v must carry id and title", it)
			}
			if it.Kind != KindComponent && it.Kind != KindParticle && it.Kind != KindScene {
				t.Fatalf("item %q has kind %q — only component, particle, scene", it.ID, it.Kind)
			}
			if seen[it.ID] {
				t.Fatalf("duplicate catalog id %q", it.ID)
			}
			seen[it.ID] = true
			if k, ok := want[it.ID]; ok && it.Kind != k {
				t.Fatalf("%s must be a %s, got %s", it.ID, k, it.Kind)
			}
		}
		for id := range want {
			if !seen[id] {
				t.Fatalf("catalog missing %q", id)
			}
		}
		for i, it := range c {
			m := sized(New(i), 80, 24)
			if m.View().Content == "" {
				t.Fatalf("%s rendered an empty frame", it.ID)
			}
		}
	})
	t.Run("unhappy: kinds are the three words the type line prints, never a synonym", func(t *testing.T) {
		for _, it := range Catalog() {
			switch it.Kind {
			case KindComponent, KindParticle, KindScene:
			default:
				t.Fatalf("item %q kind %q is not component/particle/scene", it.ID, it.Kind)
			}
			if string(it.Kind) != it.Kind.String() {
				t.Fatalf("Kind.String must be the type-line word, %q vs %q", it.Kind, it.Kind.String())
			}
		}
		if KindComponent.String() != "component" || KindParticle.String() != "particle" || KindScene.String() != "scene" {
			t.Fatalf("type-line words: %q %q %q", KindComponent, KindParticle, KindScene)
		}
	})
}

func TestCycle(t *testing.T) {
	t.Run("happy: n/j/right walk forward and wrap to shotgun", func(t *testing.T) {
		m := boot()
		if m.Index() != 0 || m.Current().ID != "shotgun" {
			t.Fatalf("boot must open on shotgun, index %d id %q", m.Index(), m.Current().ID)
		}
		m = key(m, 'n')
		if m.Index() != 1 {
			t.Fatalf("n must move forward, index %d", m.Index())
		}
		m = key(m, 'j')
		if m.Index() != 2 {
			t.Fatalf("j must move forward, index %d", m.Index())
		}
		m = keyCode(m, tea.KeyRight)
		if m.Index() != 3 {
			t.Fatalf("right arrow must move forward, index %d", m.Index())
		}
		c := Catalog()
		m = sized(New(len(c)-1), 80, 24)
		m = key(m, 'n')
		if m.Index() != 0 || m.Current().ID != "shotgun" {
			t.Fatalf("n from the last item must wrap to shotgun, index %d id %q", m.Index(), m.Current().ID)
		}
	})
	t.Run("happy: p/k/left walk backward and wrap to the tail", func(t *testing.T) {
		m := boot()
		m = key(m, 'p')
		c := Catalog()
		if m.Index() != len(c)-1 {
			t.Fatalf("p from shotgun must wrap to the last item, index %d want %d", m.Index(), len(c)-1)
		}
		m = key(m, 'k')
		if m.Index() != len(c)-2 {
			t.Fatalf("k must move backward, index %d", m.Index())
		}
		m = keyCode(m, tea.KeyLeft)
		if m.Index() != len(c)-3 {
			t.Fatalf("left arrow must move backward, index %d", m.Index())
		}
	})
	t.Run("unhappy: unknown keys never move the cursor", func(t *testing.T) {
		m := boot()
		for _, r := range []rune{'z', 'x', '1', ' '} {
			m = key(m, r)
		}
		if m.Index() != 0 {
			t.Fatalf("unknown keys moved the cursor to %d", m.Index())
		}
	})
}

func headerLines(v string) (title []string, kind string) {
	rows := strings.Split(stripAnsi(v), "\n")
	if len(rows) < 4 {
		return nil, ""
	}
	return rows[:3], strings.TrimSpace(rows[3])
}

func TestHeader(t *testing.T) {
	t.Run("happy: center top is a 3-height title over a 1-height darker-gray type", func(t *testing.T) {
		m := boot()
		v := m.View().Content
		title, kind := headerLines(v)
		if len(title) != 3 {
			t.Fatalf("title must occupy 3 rows, got %d:\n%s", len(title), v)
		}
		want, err := termfont.Lines(3, "SHOTGUN")
		if err != nil {
			t.Fatalf("termfont: %v", err)
		}
		for i, line := range want {
			if !strings.Contains(title[i], strings.TrimSpace(line)) && !strings.Contains(title[i], line) {
				t.Fatalf("title row %d missing %q:\n got %q\nwant %q", i, line, title[i], line)
			}
		}
		if kind != "component" {
			t.Fatalf("type line %q, want component", kind)
		}
		if !strings.Contains(v, "\x1b[38;5;240m") {
			t.Fatal("the type line must wear slightly darker gray (xterm 240)")
		}
		if !strings.Contains(v, "\x1b[38;5;252m") {
			t.Fatal("the title must wear a lighter gray (xterm 252) so 240 reads darker")
		}
		idx := strings.Index(stripAnsi(title[1]), strings.TrimSpace(want[1]))
		if idx < 0 {
			t.Fatal("title glyphs must appear on the banner")
		}
		plain := stripAnsi(title[1])
		leftPad := idx
		rightPad := len(plain) - idx - len(strings.TrimRight(want[1], " "))
		if leftPad < 2 || rightPad < 2 {
			t.Fatalf("title must sit in the center, left pad %d right pad %d in %q", leftPad, rightPad, plain)
		}
	})
	t.Run("happy: cycling to a particle then a scene retitles both lines", func(t *testing.T) {
		m := sized(New(findItem(t, KindParticle, "gunfire")), 80, 24)
		v := m.View().Content
		title, kind := headerLines(v)
		if kind != "particle" {
			t.Fatalf("gunfire type %q, want particle", kind)
		}
		want, err := termfont.Lines(3, "GUNFIRE")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(title[1], strings.TrimSpace(want[1])) && !strings.Contains(stripAnsi(title[1]), strings.TrimSpace(want[1])) {
			t.Fatalf("gunfire title missing %q:\n%s", want[1], title[1])
		}
		m = sized(New(findItem(t, KindScene, "landing")), 80, 24)
		_, kind = headerLines(m.View().Content)
		if kind != "scene" {
			t.Fatalf("landing type %q, want scene", kind)
		}
	})
	t.Run("unhappy: a tiny terminal still renders the title and the type", func(t *testing.T) {
		m := sized(New(0), 20, 6)
		v := stripAnsi(m.View().Content)
		if v == "" {
			t.Fatal("tiny terminals must still render")
		}
		if !strings.Contains(strings.ToLower(v), "component") && !strings.Contains(v, "SHOT") {
			t.Fatalf("tiny view must still name the item:\n%s", v)
		}
	})
}

func TestEdit(t *testing.T) {
	t.Run("happy: e on a particle opens that particle's tuner", func(t *testing.T) {
		cases := []struct {
			id, wantPath, wantPkg string
		}{
			{"gunfire", adjustgunfire.DefaultConfigPath, "./cmd/adjustgunfire/main"},
			{"flame", adjustflame.DefaultConfigPath, "./cmd/adjustflame/main"},
			{"dust", adjustdust.DefaultConfigPath, "./cmd/adjustdust/main"},
			{"nyan", adjustparticle.DefaultConfigPath, "./cmd/adjustparticle/main"},
		}
		for _, tc := range cases {
			m := sized(New(findItem(t, KindParticle, tc.id)), 80, 24)
			mm, cmd := m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
			m = mm.(Model)
			ed, ok := m.ChosenEdit()
			if !ok {
				t.Fatalf("%s: e must choose an edit", tc.id)
			}
			if ed.Kind != KindParticle {
				t.Fatalf("%s: edit kind %s, want particle", tc.id, ed.Kind)
			}
			if ed.Path != tc.wantPath {
				t.Fatalf("%s: edit path %q, want %q", tc.id, ed.Path, tc.wantPath)
			}
			if ed.Program != tc.wantPkg {
				t.Fatalf("%s: edit program %q, want %q", tc.id, ed.Program, tc.wantPkg)
			}
			if cmd == nil {
				t.Fatalf("%s: e must quit so the tuner can take the terminal", tc.id)
			}
		}
	})
	t.Run("happy: e on a scene opens that scene's tuner", func(t *testing.T) {
		cases := []struct {
			id, wantPkg string
		}{
			{"landing", "./cmd/landing"},
			{"america", "./cmd/america"},
			{"moonwalk", "./cmd/astronaut"},
			{"skies", "./cmd/skies"},
		}
		for _, tc := range cases {
			m := sized(New(findItem(t, KindScene, tc.id)), 80, 24)
			mm, cmd := m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
			m = mm.(Model)
			ed, ok := m.ChosenEdit()
			if !ok {
				t.Fatalf("%s: e must choose an edit", tc.id)
			}
			if ed.Kind != KindScene {
				t.Fatalf("%s: edit kind %s, want scene", tc.id, ed.Kind)
			}
			if ed.Program != tc.wantPkg {
				t.Fatalf("%s: edit program %q, want %q", tc.id, ed.Program, tc.wantPkg)
			}
			if cmd == nil {
				t.Fatalf("%s: e must quit so the scene tuner can take the terminal", tc.id)
			}
		}
	})
	t.Run("happy: e on sky and cloud opens that component's tuner", func(t *testing.T) {
		cases := []struct {
			id, wantPath, wantPkg string
		}{
			{"sky", adjustsky.DefaultConfigPath, "./cmd/adjustsky/main"},
			{"cloud", adjustcloud.DefaultConfigPath, "./cmd/adjustcloud/main"},
		}
		for _, tc := range cases {
			m := sized(New(findItem(t, KindComponent, tc.id)), 80, 24)
			mm, cmd := m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
			m = mm.(Model)
			ed, ok := m.ChosenEdit()
			if !ok {
				t.Fatalf("%s: e must choose an edit", tc.id)
			}
			if ed.Kind != KindComponent {
				t.Fatalf("%s: edit kind %s, want component", tc.id, ed.Kind)
			}
			if ed.Path != tc.wantPath {
				t.Fatalf("%s: edit path %q, want %q", tc.id, ed.Path, tc.wantPath)
			}
			if ed.Program != tc.wantPkg {
				t.Fatalf("%s: edit program %q, want %q", tc.id, ed.Program, tc.wantPkg)
			}
			if cmd == nil {
				t.Fatalf("%s: e must quit so the tuner can take the terminal", tc.id)
			}
		}
	})
	t.Run("happy: e on a component opens the ASCII editor on its atlas", func(t *testing.T) {
		cases := []struct {
			id, wantBase string
		}{
			{"shotgun", "shotgun.json"},
			{"lander", "lm.json"},
			{"astronaut", "astronaut.json"},
		}
		for _, tc := range cases {
			m := sized(New(findItem(t, KindComponent, tc.id)), 80, 24)
			mm, cmd := m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
			m = mm.(Model)
			ed, ok := m.ChosenEdit()
			if !ok {
				t.Fatalf("%s: e must choose an edit", tc.id)
			}
			if ed.Kind != KindComponent {
				t.Fatalf("%s: edit kind %s, want component", tc.id, ed.Kind)
			}
			if filepath.Base(ed.Path) != tc.wantBase && !strings.HasSuffix(ed.Path, tc.wantBase) {
				t.Fatalf("%s: ascii editor path %q, want %s", tc.id, ed.Path, tc.wantBase)
			}
			if ed.Program != "" && ed.Program != editor.DefaultAssetsDir {
				// components open the editor, never a tuner package
				if strings.Contains(ed.Program, "adjust") {
					t.Fatalf("%s: a component must not open a particle tuner (%q)", tc.id, ed.Program)
				}
			}
			if cmd == nil {
				t.Fatalf("%s: e must quit so the ascii editor can take the terminal", tc.id)
			}
		}
	})
	t.Run("happy: closing a sub-edit resumes at the same catalog index", func(t *testing.T) {
		idx := findItem(t, KindParticle, "gunfire")
		m := sized(New(idx), 80, 24)
		mm, _ := m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
		m = mm.(Model)
		ed, ok := m.ChosenEdit()
		if !ok {
			t.Fatal("e must choose an edit")
		}
		resume := sized(New(m.Index()), 80, 24)
		if resume.Index() != idx {
			t.Fatalf("resume index %d, want %d", resume.Index(), idx)
		}
		if resume.Current().ID != "gunfire" {
			t.Fatalf("resume landed on %q, want gunfire", resume.Current().ID)
		}
		_, kind := headerLines(resume.View().Content)
		if kind != "particle" {
			t.Fatalf("resumed type %q, want particle", kind)
		}
		if ed.Program != "./cmd/adjustgunfire/main" {
			t.Fatalf("the edit we would close is %q", ed.Program)
		}
	})
	t.Run("unhappy: q quits with no edit", func(t *testing.T) {
		m := boot()
		mm, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
		m = mm.(Model)
		if _, ok := m.ChosenEdit(); ok {
			t.Fatal("q must not choose an edit")
		}
		if cmd == nil {
			t.Fatal("q must quit")
		}
	})
	t.Run("unhappy: e on an empty catalog is a refused no-op", func(t *testing.T) {
		m := sized(NewWith(nil, 0), 80, 24)
		mm, cmd := m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
		m = mm.(Model)
		if _, ok := m.ChosenEdit(); ok {
			t.Fatal("an empty catalog has nothing to edit")
		}
		if cmd != nil {
			if _, isQuit := cmd().(tea.QuitMsg); isQuit {
				t.Fatal("e on an empty catalog must not quit the viewer")
			}
		}
		if m.Index() != 0 {
			t.Fatalf("empty catalog cursor moved to %d", m.Index())
		}
	})
}
