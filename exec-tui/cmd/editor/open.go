package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// Open reads path into an editor: one .json atlas, or a folder of
// them. On the very first run — a .json path with no file yet — it
// seeds the file with a blank canvas, so there is always something on
// disk to edit and save. A corrupt file, an unwritable seed path, or
// a missing folder is an error, never a silent fallback.
func Open(path string) (Model, error) {
	st, err := os.Stat(path)
	if err == nil && st.IsDir() {
		return OpenDir(path)
	}
	if err != nil && !strings.HasSuffix(path, ".json") {
		return Model{}, err
	}
	a, err := load(path)
	if err != nil {
		return Model{}, err
	}
	m := New(a, path)
	m.atlases = map[string]*sprite.Atlas{path: a}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		m.AssetsDir = dir
	}
	m.snapToExistingFrame()
	return m, nil
}

// OpenDir loads every *.json atlas in dir up front — the whole folder
// warm in one command. The first file (sorted by name) lands on the
// canvas; ctrl-p switches between the rest. An empty folder seeds
// untitled.json so there is something to edit; a corrupt atlas fails
// loudly by name.
func OpenDir(dir string) (Model, error) {
	files, err := ListAssets(dir)
	if err != nil {
		return Model{}, err
	}
	if len(files) == 0 {
		seed := filepath.Join(dir, "untitled.json")
		if err := blankAtlas().WriteFile(seed); err != nil {
			return Model{}, err
		}
		files = []Asset{{Name: "untitled", Path: seed}}
	}
	atlases := make(map[string]*sprite.Atlas, len(files))
	for _, f := range files {
		a, err := LoadAsset(f.Path)
		if err != nil {
			return Model{}, fmt.Errorf("%s: %w", filepath.Base(f.Path), err)
		}
		atlases[f.Path] = a
	}
	m := New(atlases[files[0].Path], files[0].Path)
	m.AssetsDir = dir
	m.Files = files
	m.atlases = atlases
	m.snapToExistingFrame()
	return m, nil
}

func load(path string) (*sprite.Atlas, error) {
	if _, err := os.Stat(path); err == nil {
		return sprite.LoadFile(path)
	}
	a := blankAtlas()
	if err := a.WriteFile(path); err != nil {
		return nil, err
	}
	return a, nil
}
