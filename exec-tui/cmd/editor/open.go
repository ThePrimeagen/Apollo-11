package editor

import (
	"os"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// DefaultAtlasPath is the LM art the editor opens when no path is
// given: the atlas that ships inside the lander component, relative to
// the module root the launcher and `go run` start from.
const DefaultAtlasPath = "components/lander/lm.json"

// Open reads the atlas at path into an editor. On the very first run —
// no file yet — it seeds the file with the hand-drawn default art, so
// there is always something on disk to edit and save. A corrupt file
// or an unwritable seed path is an error, never a silent fallback.
func Open(path string) (Model, error) {
	a, err := load(path)
	if err != nil {
		return Model{}, err
	}
	return New(a, path), nil
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
