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

	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustarmed"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustcloud"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustdust"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustflame"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustgunfire"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustparticle"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustsky"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/editor"
	"github.com/theprimeagen/apollo-11/exec-tui/components/ie"
	"github.com/theprimeagen/apollo-11/exec-tui/components/pools"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/america"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/bobble"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/coreset"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/landing"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/liftoff"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/moonwalk"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/skies"
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
			"shotgun":    KindComponent,
			"stars":      KindComponent,
			"sky":        KindComponent,
			"cloud":      KindComponent,
			"lander":     KindComponent,
			"flag":       KindComponent,
			"transition": KindComponent,
			"eagle":      KindComponent,
			"armed":      KindComponent,
			"moon":       KindComponent,
			"ie":         KindComponent,
			"dsky":       KindComponent,
			"coreset":    KindComponent,
			"coresets":   KindComponent,
			"vac":        KindComponent,
			"vacs":       KindComponent,
			"cpugraph":   KindComponent,
			"breakdown":  KindScene,
			"scan":       KindScene,
			"title":      KindComponent,
			"astronaut":  KindComponent,
			"rocket":     KindComponent,
			"gunfire":    KindParticle,
			"flame":      KindParticle,
			"dust":       KindParticle,
			"nyan":       KindParticle,
			"landing":    KindScene,
			"america":    KindScene,
			"moonwalk":   KindScene,
			"skies":      KindScene,
			"liftoff":    KindScene,
			"bobble":     KindScene,
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
	t.Run("happy: e on a scene opens that scene's tuner on its config", func(t *testing.T) {
		cases := []struct {
			id, wantPath, wantPkg string
		}{
			{"landing", landing.DefaultConfigPath, "./cmd/landing"},
			{"america", america.DefaultConfigPath, "./cmd/america"},
			{"moonwalk", moonwalk.DefaultConfigPath, "./cmd/astronaut"},
			{"skies", skies.DefaultConfigPath, "./cmd/skies"},
			{"breakdown", coreset.DefaultConfigPath, "./cmd/coreset"},
			{"scan", "scenes/coreset2", "./cmd/coreset2"},
			{"liftoff", liftoff.DefaultConfigPath, "./cmd/liftoff"},
			{"bobble", bobble.DefaultConfigPath, "./cmd/bobble"},
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
			if ed.Path != tc.wantPath {
				t.Fatalf("%s: edit path %q, want %q — the tuner opens on the scene's own knobs", tc.id, ed.Path, tc.wantPath)
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
			{"armed", adjustarmed.DefaultConfigPath, "./cmd/adjustarmed/main"},
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

// Tests written FIRST: the EXPLORER item is the ie component — the
// old Internet Explorer logo, the fixed 14×7 card of the bold blue e
// wearing its golden swoosh — listed as a component, staged centered,
// and edited like any other code-drawn card: e opens the assets
// editor, never a tuner.

func TestExplorerItem(t *testing.T) {
	t.Run("happy: the catalog lists EXPLORER and stages the blue e under the golden swoosh", func(t *testing.T) {
		idx := findItem(t, KindComponent, "ie")
		it := Catalog()[idx]
		if it.Title != "EXPLORER" {
			t.Fatalf("the item's banner is %q, want EXPLORER", it.Title)
		}
		comp := it.spawn()
		if _, ok := comp.(*ie.Logo); !ok {
			t.Fatalf("the ie item must stage the logo component, got %T", comp)
		}
		comp.Start(80, 19)
		sp := comp.Render()
		if sp.Width != 80 || sp.Height != 19 {
			t.Fatalf("the preview rendered %dx%d, want the 80x19 stage", sp.Width, sp.Height)
		}
		blue, gold := false, false
		for r := 0; r < sp.Height; r++ {
			for c := 0; c < sp.Width; c++ {
				cell := sp.At(r, c)
				if cell.FG == ie.BlueInk || cell.BG == ie.BlueInk {
					blue = true
				}
				if cell.FG == ie.GoldInk || cell.BG == ie.GoldInk {
					gold = true
				}
			}
		}
		if !blue || !gold {
			t.Fatalf("the preview must wear the blue e and the golden swoosh, blue %v gold %v", blue, gold)
		}
	})
	t.Run("unhappy: e on the logo opens the assets editor, never a tuner", func(t *testing.T) {
		m := sized(New(findItem(t, KindComponent, "ie")), 80, 24)
		mm, cmd := m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
		m = mm.(Model)
		ed, ok := m.ChosenEdit()
		if !ok {
			t.Fatal("e must choose an edit")
		}
		if ed.Kind != KindComponent {
			t.Fatalf("edit kind %s, want component", ed.Kind)
		}
		if ed.Path != editor.DefaultAssetsDir {
			t.Fatalf("edit path %q, want the assets folder %q", ed.Path, editor.DefaultAssetsDir)
		}
		if ed.Program != "" {
			t.Fatalf("a code-drawn card must not launch a tuner, got %q", ed.Program)
		}
		if cmd == nil {
			t.Fatal("e must quit so the editor can take the terminal")
		}
	})
}

// Tests written FIRST: the CORE SETS and VAC AREAS items are the two
// pool views wrapped in a play-button demo. Space (the viewer's
// trigger key) starts a scripted walk of the whole lifecycle — add
// three jobs, drop those three, add four, drop those four, then fill
// every slot to raise the pool's program alarm, hold the full pool on
// stage for poolAlarmHoldBeats so the chip can be read, and drain it
// back to empty — one step every poolStepSeconds, each job wearing
// its own ink so the colors read against the other graphs. The bottom
// row carries the hint while idle and the current action while
// playing. Firing mid-walk is refused; when the walk ends the trigger
// is live again.

func spawnDemo(t *testing.T, id string) *poolDemo {
	t.Helper()
	it := Catalog()[findItem(t, KindComponent, id)]
	comp := it.spawn()
	d, ok := comp.(*poolDemo)
	if !ok {
		t.Fatalf("%s must spawn the pool demo, got %T", id, comp)
	}
	d.Start(80, 19)
	return d
}

// playSteps ticks the demo across n script boundaries.
func playSteps(d *poolDemo, n int) {
	for i := 0; i < n; i++ {
		d.Update(poolStepSeconds + 0.001)
	}
}

func demoRow(sp sprite.Sprite, r int) string {
	rs := make([]rune, sp.Width)
	for c := 0; c < sp.Width; c++ {
		ch := sp.At(r, c).Ch
		if ch == 0 {
			ch = ' '
		}
		rs[c] = ch
	}
	return string(rs)
}

func demoText(sp sprite.Sprite, text string) bool {
	for r := 0; r < sp.Height; r++ {
		if strings.Contains(demoRow(sp, r), text) {
			return true
		}
	}
	return false
}

func distinctInks(t *testing.T, d *poolDemo, names []string) {
	t.Helper()
	seen := map[int]bool{}
	for i, name := range names {
		j, ok := d.view.JobAt(i)
		if !ok || j.Name != name {
			t.Fatalf("slot %d holds %q ok=%v, want %q", i, j.Name, ok, name)
		}
		if j.Ink <= 0 {
			t.Fatalf("demo job %q carries no ink — every job wears a color", name)
		}
		if seen[j.Ink] {
			t.Fatalf("ink %d repeats on %q — the demo colors must differ", j.Ink, name)
		}
		seen[j.Ink] = true
	}
}

func TestPoolDemo(t *testing.T) {
	t.Run("happy: space plays the core set lifecycle — add 3, drop 3, add 4, drop 4, fill to 1202, drain", func(t *testing.T) {
		d := spawnDemo(t, "coresets")
		if d.view.Cap() != 8 {
			t.Fatalf("the core set demo wraps %d slots, want 8", d.view.Cap())
		}
		if !d.Fire() {
			t.Fatal("the idle trigger must start the walk")
		}
		if d.view.Busy() != 1 {
			t.Fatalf("the trigger lands the first job at once, busy %d", d.view.Busy())
		}
		playSteps(d, 2)
		if d.view.Busy() != 3 {
			t.Fatalf("after the add-3 act busy is %d, want 3", d.view.Busy())
		}
		distinctInks(t, d, []string{"SERVICER", "CHARIN", "MONITOR"})
		playSteps(d, 3)
		if d.view.Busy() != 0 {
			t.Fatalf("after the drop-3 act busy is %d, want 0", d.view.Busy())
		}
		playSteps(d, 4)
		if d.view.Busy() != 4 {
			t.Fatalf("after the add-4 act busy is %d, want 4", d.view.Busy())
		}
		distinctInks(t, d, []string{"RR READ", "LR READ", "GYRO COMP", "DAP"})
		playSteps(d, 4)
		if d.view.Busy() != 0 {
			t.Fatalf("after the drop-4 act busy is %d, want 0", d.view.Busy())
		}
		playSteps(d, 8)
		if !d.view.Full() {
			t.Fatalf("the fill act must reach every slot, busy %d", d.view.Busy())
		}
		sp := d.Render()
		if !demoText(sp, "→ 1202") {
			t.Fatal("a full core set pool must raise the 1202 chip")
		}
		playSteps(d, poolAlarmHoldBeats)
		if !d.view.Full() || !d.playing {
			t.Fatalf("the alarm must hold the full pool on stage, busy %d playing %v", d.view.Busy(), d.playing)
		}
		if !demoText(d.Render(), "→ 1202") {
			t.Fatal("the chip must stay up through the hold")
		}
		playSteps(d, 8)
		if d.view.Busy() != 0 || d.playing {
			t.Fatalf("the drain act must empty the pool and end the walk, busy %d playing %v", d.view.Busy(), d.playing)
		}
		if !d.Fire() {
			t.Fatal("a finished walk must be replayable")
		}
		if d.view.Busy() != 1 {
			t.Fatalf("the replay lands the first job again, busy %d", d.view.Busy())
		}
	})
	t.Run("happy: the VAC demo walks the same script to 1201 on five slots", func(t *testing.T) {
		d := spawnDemo(t, "vacs")
		if d.view.Cap() != 5 {
			t.Fatalf("the VAC demo wraps %d slots, want 5", d.view.Cap())
		}
		if !d.Fire() {
			t.Fatal("the idle trigger must start the walk")
		}
		playSteps(d, 2)
		distinctInks(t, d, []string{"SERVICER", "CHARIN", "MONITOR"})
		playSteps(d, 3+4+4)
		playSteps(d, 5)
		if !d.view.Full() {
			t.Fatalf("the fill act must reach all five VACs, busy %d", d.view.Busy())
		}
		sp := d.Render()
		if !demoText(sp, "→ 1201") {
			t.Fatal("a full VAC pool must raise the 1201 chip")
		}
		if demoText(sp, "1202") {
			t.Fatal("the VAC pool never raises the core sets' 1202")
		}
		playSteps(d, poolAlarmHoldBeats)
		if !d.view.Full() || !demoText(d.Render(), "→ 1201") {
			t.Fatalf("the alarm must hold all five VACs on stage, busy %d", d.view.Busy())
		}
		playSteps(d, 5)
		if d.view.Busy() != 0 || d.playing {
			t.Fatalf("the drain act must empty the pool and end the walk, busy %d playing %v", d.view.Busy(), d.playing)
		}
	})
	t.Run("happy: the viewer's space key pulls the trigger", func(t *testing.T) {
		m := sized(New(findItem(t, KindComponent, "coresets")), 80, 24)
		m = key(m, ' ')
		d, ok := m.preview.(*poolDemo)
		if !ok {
			t.Fatalf("the coresets item must stage the pool demo, got %T", m.preview)
		}
		if d.view.Busy() != 1 {
			t.Fatalf("space must start the walk, busy %d", d.view.Busy())
		}
	})
	t.Run("happy: the bottom row hints the play button while idle and the action while playing", func(t *testing.T) {
		d := spawnDemo(t, "vacs")
		sp := d.Render()
		if !strings.Contains(demoRow(sp, sp.Height-1), "space plays") {
			t.Fatalf("the idle hint must name the button:\n%q", demoRow(sp, sp.Height-1))
		}
		d.Fire()
		sp = d.Render()
		bottom := demoRow(sp, sp.Height-1)
		if !strings.Contains(bottom, "+ SERVICER") {
			t.Fatalf("the playing hint must show the action, got %q", bottom)
		}
		if strings.Contains(bottom, "space plays") {
			t.Fatal("the idle hint must clear during the walk")
		}
		playSteps(d, len(d.script))
		sp = d.Render()
		if !strings.Contains(demoRow(sp, sp.Height-1), "space plays") {
			t.Fatal("the finished walk must bring the play hint back")
		}
	})
	t.Run("unhappy: firing mid-walk is refused and the script keeps its place", func(t *testing.T) {
		d := spawnDemo(t, "coresets")
		if !d.Fire() {
			t.Fatal("the first trigger must start the walk")
		}
		d.Update(0.1)
		if d.Fire() {
			t.Fatal("a walk already playing must refuse the trigger")
		}
		if d.view.Busy() != 1 {
			t.Fatalf("the refused trigger changed the pool, busy %d", d.view.Busy())
		}
		playSteps(d, 2)
		if d.view.Busy() != 3 {
			t.Fatalf("the script lost its place — busy %d, want 3", d.view.Busy())
		}
	})
	t.Run("unhappy: the alarm chip colors match the pools package inks", func(t *testing.T) {
		d := spawnDemo(t, "vacs")
		d.Fire()
		playSteps(d, 2+3+4+4+5)
		sp := d.Render()
		found := false
		for r := 0; r < sp.Height && !found; r++ {
			row := demoRow(sp, r)
			i := strings.Index(row, "→ 1201")
			if i < 0 {
				continue
			}
			found = true
			cell := sp.At(r, len([]rune(row[:i])))
			if cell.FG != pools.AlarmFG || cell.BG != pools.AlarmBG {
				t.Fatalf("the chip wears %d on %d, want %d on %d", cell.FG, cell.BG, pools.AlarmFG, pools.AlarmBG)
			}
		}
		if !found {
			t.Fatal("the full pool must show its chip")
		}
	})
}

// Tests written FIRST: the CORE SET and VAC items are single Box
// components wrapped in a toggle demo. Space turns the slot on
// through four different job inks, one per press, then off again —
// the little pill that lights up in different colors with a bit of
// text inside. The bottom row hints the button.

func spawnBox(t *testing.T, id string) *boxDemo {
	t.Helper()
	it := Catalog()[findItem(t, KindComponent, id)]
	comp := it.spawn()
	d, ok := comp.(*boxDemo)
	if !ok {
		t.Fatalf("%s must spawn the box demo, got %T", id, comp)
	}
	d.Start(80, 19)
	return d
}

func TestBoxDemo(t *testing.T) {
	t.Run("happy: space cycles the slot through four job inks, then off, then round again", func(t *testing.T) {
		d := spawnBox(t, "coreset")
		if d.view.Busy() {
			t.Fatal("the demo opens on a free slot")
		}
		seen := map[int]bool{}
		var names []string
		for i := 0; i < 4; i++ {
			if !d.Fire() {
				t.Fatal("the toggle must always take")
			}
			j, ok := d.view.Job()
			if !ok {
				t.Fatalf("press %d must light the slot", i+1)
			}
			if j.Ink <= 0 || seen[j.Ink] {
				t.Fatalf("press %d wears ink %d — four presses, four different inks", i+1, j.Ink)
			}
			seen[j.Ink] = true
			names = append(names, j.Name)
		}
		if !d.Fire() {
			t.Fatal("the fifth press must take")
		}
		if d.view.Busy() {
			t.Fatal("the fifth press turns the slot off")
		}
		if !d.Fire() {
			t.Fatal("the sixth press must take")
		}
		j, ok := d.view.Job()
		if !ok || j.Name != names[0] {
			t.Fatalf("the cycle must wrap to %q, got %+v ok=%v", names[0], j, ok)
		}
	})
	t.Run("happy: the box demo shows its unnumbered label and the hint", func(t *testing.T) {
		d := spawnBox(t, "coreset")
		sp := d.Render()
		found := false
		for r := 0; r < sp.Height; r++ {
			if strings.Contains(demoRow(sp, r), "CORE SET") {
				found = true
			}
		}
		if !found {
			t.Fatal("the core set box wears the plain CORE SET label — no number")
		}
		if !strings.Contains(demoRow(sp, sp.Height-1), "space toggles") {
			t.Fatalf("the hint must name the button, got %q", demoRow(sp, sp.Height-1))
		}
		v := spawnBox(t, "vac")
		sp = v.Render()
		if !demoText(sp, "VAC") {
			t.Fatal("the VAC box wears its plain VAC label")
		}
	})
	t.Run("unhappy: before Start the demo renders nothing and never panics", func(t *testing.T) {
		it := Catalog()[findItem(t, KindComponent, "vac")]
		d, ok := it.spawn().(*boxDemo)
		if !ok {
			t.Fatalf("vac must spawn the box demo, got %T", it.spawn())
		}
		if sp := d.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("before Start the demo renders %dx%d, want nothing", sp.Width, sp.Height)
		}
		d.Update(1)
		if !d.Fire() {
			t.Fatal("the toggle works even before the curtain — state is identity, the stage is not")
		}
	})
}

// Tests written FIRST: the CPU GRAPH item is the standalone cpugraph
// component — the portrait extracted from the graphs screen, shown by
// itself with no legend and no switch row — wrapped in a switch-walk
// demo. Space (the viewer's trigger key) steps the component's own
// switch API through the story one state per press: the healthy
// portrait (descent alone), the radar-steal knife edge, the 1668
// monitor crossing the line, P64 crossing it harder, everything off,
// and around again. The bottom row hints the button and names the
// state now on stage.

func spawnCPUGraph(t *testing.T) *cpuDemo {
	t.Helper()
	it := Catalog()[findItem(t, KindComponent, "cpugraph")]
	comp := it.spawn()
	d, ok := comp.(*cpuDemo)
	if !ok {
		t.Fatalf("cpugraph must spawn the switch-walk demo, got %T", comp)
	}
	d.Start(80, 19)
	return d
}

func TestCPUGraphDemo(t *testing.T) {
	t.Run("happy: space walks the switch story — healthy, knife edge, 1668, P64, idle, around", func(t *testing.T) {
		d := spawnCPUGraph(t)
		g := d.view
		if !g.Descent() || g.Monitor() || g.Radar() || g.Approach() {
			t.Fatalf("the demo opens on the healthy portrait: descent %v, monitor %v, radar %v, approach %v",
				g.Descent(), g.Monitor(), g.Radar(), g.Approach())
		}
		if !d.Fire() {
			t.Fatal("the trigger must always take")
		}
		if !g.Descent() || !g.Radar() || g.Monitor() || g.Approach() {
			t.Fatalf("press 1 is the knife edge — descent + radar, got monitor %v radar %v approach %v",
				g.Monitor(), g.Radar(), g.Approach())
		}
		d.Fire()
		if !g.Monitor() || !g.Radar() || g.Approach() {
			t.Fatalf("press 2 keys the 1668 monitor over the steal, got monitor %v radar %v approach %v",
				g.Monitor(), g.Radar(), g.Approach())
		}
		d.Fire()
		if !g.Approach() || g.Monitor() || !g.Radar() {
			t.Fatalf("press 3 swaps 1668 for P64, got monitor %v radar %v approach %v",
				g.Monitor(), g.Radar(), g.Approach())
		}
		d.Fire()
		if g.Descent() || g.Monitor() || g.Radar() || g.Approach() {
			t.Fatalf("press 4 switches everything off, got descent %v monitor %v radar %v approach %v",
				g.Descent(), g.Monitor(), g.Radar(), g.Approach())
		}
		d.Fire()
		if !g.Descent() || g.Monitor() || g.Radar() || g.Approach() {
			t.Fatalf("press 5 wraps back to the healthy portrait")
		}
	})
	t.Run("happy: the viewer's space key pulls the trigger", func(t *testing.T) {
		m := sized(New(findItem(t, KindComponent, "cpugraph")), 80, 24)
		m = key(m, ' ')
		d, ok := m.preview.(*cpuDemo)
		if !ok {
			t.Fatalf("the cpugraph item must stage the switch-walk demo, got %T", m.preview)
		}
		if !d.view.Radar() || d.view.Monitor() {
			t.Fatalf("space must step to the knife edge, got radar %v monitor %v", d.view.Radar(), d.view.Monitor())
		}
	})
	t.Run("happy: the graph stands alone and the bottom row names the state", func(t *testing.T) {
		d := spawnCPUGraph(t)
		sp := d.Render()
		if !demoText(sp, "VAC JOBS") || !demoText(sp, "SERVICER") {
			t.Fatalf("the demo must show the portrait's lanes")
		}
		for _, banned := range []string{"total ::", "DESCENT", "q quit"} {
			if demoText(sp, banned) {
				t.Fatalf("the standalone graph must carry no surrounding information, found %q", banned)
			}
		}
		bottom := demoRow(sp, sp.Height-1)
		if !strings.Contains(bottom, "space") || !strings.Contains(bottom, "healthy") {
			t.Fatalf("the idle hint must name the button and the healthy state, got %q", bottom)
		}
		d.Fire()
		bottom = demoRow(d.Render(), d.Render().Height-1)
		if !strings.Contains(bottom, "knife edge") {
			t.Fatalf("after one press the hint must name the knife edge, got %q", bottom)
		}
	})
	t.Run("unhappy: before Start the demo renders nothing and the switches still take", func(t *testing.T) {
		it := Catalog()[findItem(t, KindComponent, "cpugraph")]
		d, ok := it.spawn().(*cpuDemo)
		if !ok {
			t.Fatalf("cpugraph must spawn the switch-walk demo, got %T", it.spawn())
		}
		if sp := d.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("before Start the demo renders %dx%d, want nothing", sp.Width, sp.Height)
		}
		d.Update(1)
		if !d.Fire() {
			t.Fatal("the trigger works before the curtain — state is identity, the stage is not")
		}
		if !d.view.Radar() {
			t.Fatal("the unstaged press must still land on the switch API")
		}
	})
}
