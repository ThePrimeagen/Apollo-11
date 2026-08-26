package editor

import (
	"os"
	"path/filepath"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// Open reads the atlas at path into an editor. On the very first run —
// no file yet — it seeds the file with the hand-drawn default art, so
// there is always something on disk to edit and save. A corrupt file
// or an unwritable seed path is an error, never a silent fallback.
func Open(path string) (Model, error) {
	a, err := load(path)
	if err != nil {
		return Model{}, err
	}
	m := New(a, path)
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		m.AssetsDir = dir
	}
	m.snapToExistingFrame()
	return m, nil
}

func load(path string) (*sprite.Atlas, error) {
	if _, err := os.Stat(path); err == nil {
		return sprite.LoadFile(path)
	}
	a := lander.DefaultAtlas()
	if err := a.WriteFile(path); err != nil {
		return nil, err
	}
	return a, nil
}
