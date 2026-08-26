package editor

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// DefaultAssetsDir is the folder of JSON atlases the editor opens when
// no path is given, relative to the module root the launcher and
// `go run` start from.
const DefaultAssetsDir = "assets"

// Asset is one JSON atlas in the assets folder.
type Asset struct {
	Name string
	Path string
}

// ListAssets returns every *.json atlas in dir, sorted by name.
// A missing directory is an empty list, not an error — the picker
// can still open and say there is nothing to switch to.
func ListAssets(dir string) ([]Asset, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Asset
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		out = append(out, Asset{
			Name: strings.TrimSuffix(e.Name(), ".json"),
			Path: filepath.Join(dir, e.Name()),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// LoadAsset reads a JSON atlas from path.
func LoadAsset(path string) (*sprite.Atlas, error) {
	return sprite.LoadFile(path)
}

// blankAtlas is the seed for brand-new files: the default palette and
// one empty size-4 frame — a fresh canvas with no project's art baked
// in. Editing other sizes and headings creates their frames on demand.
func blankAtlas() *sprite.Atlas {
	a := &sprite.Atlas{Palette: append([]sprite.PaletteEntry(nil), sprite.DefaultPalette...)}
	w, h := sprite.Size4.Dim()
	a.SetFrame(sprite.Size4, sprite.N, sprite.New(w, h))
	return a
}

// FindAssetsDir locates the shipped assets folder: first a nearby
// assets/ that already holds JSON, then any assets/ sitting next to a
// go.mod, then the relative default so callers always get a path.
func FindAssetsDir() string {
	seen := map[string]bool{}
	var cands []string
	addFrom := func(start string) {
		dir := start
		for i := 0; i < 8; i++ {
			cand := filepath.Join(dir, DefaultAssetsDir)
			if !seen[cand] {
				seen[cand] = true
				cands = append(cands, cand)
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				return
			}
			dir = parent
		}
	}
	if wd, err := os.Getwd(); err == nil {
		addFrom(wd)
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		addFrom(filepath.Dir(file))
	}

	var empty string
	for _, d := range cands {
		matches, err := filepath.Glob(filepath.Join(d, "*.json"))
		if err == nil && len(matches) > 0 {
			return d
		}
		if empty == "" {
			if st, err := os.Stat(d); err == nil && st.IsDir() {
				empty = d
			}
		}
	}
	if empty != "" {
		return empty
	}
	return DefaultAssetsDir
}

func (m Model) assetsDir() string {
	if m.AssetsDir != "" {
		return m.AssetsDir
	}
	return FindAssetsDir()
}

func (m *Model) snapToExistingFrame() {
	if m.Atlas == nil {
		return
	}
	if _, ok := m.Atlas.Frame(m.Size, m.Heading); ok {
		m.clampCursor()
		return
	}
	for _, sz := range sprite.Sizes {
		for _, h := range sprite.Headings {
			if _, ok := m.Atlas.Frame(sz, h); ok {
				m.Size = sz
				m.Heading = h
				m.clampCursor()
				return
			}
		}
	}
}

// openAsset switches the canvas to the atlas at path. The atlas being
// left stays warm in the cache, so its unsaved edits survive the
// switch and the next save flushes them to disk.
func (m *Model) openAsset(path string) error {
	if m.atlases == nil {
		m.atlases = map[string]*sprite.Atlas{}
	}
	if m.Atlas != nil && m.Path != "" {
		m.atlases[m.Path] = m.Atlas
	}
	a := m.atlases[path]
	if a == nil {
		var err error
		a, err = LoadAsset(path)
		if err != nil {
			return err
		}
		m.atlases[path] = a
	}
	m.Atlas = a
	m.Path = path
	m.sel = map[cellKey]bool{}
	m.snapToExistingFrame()
	m.SetStatus("opened " + path)
	return nil
}
