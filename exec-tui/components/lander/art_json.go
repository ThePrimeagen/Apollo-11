package lander

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const artAssets = "assets"

// FindArtDir locates the folder of shipped lm-*.json atlases: a nearby
// assets/ that already holds them, then any assets/ next to a go.mod,
// then any folder carrying the legacy single-file lm.json. All the
// lunar art lives together there — lm.json included.
func FindArtDir() string {
	seen := map[string]bool{}
	var cands []string
	addFrom := func(start string) {
		dir := start
		for i := 0; i < 8; i++ {
			cand := filepath.Join(dir, artAssets)
			if !seen[cand] {
				seen[cand] = true
				cands = append(cands, cand)
			}
			if !seen[dir] {
				seen[dir] = true
				cands = append(cands, dir)
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
	var jsonDir, empty string
	for _, d := range cands {
		if _, err := os.Stat(filepath.Join(d, "lm-4.json")); err == nil {
			return d
		}
		matches, err := filepath.Glob(filepath.Join(d, "lm-*.json"))
		if err == nil && len(matches) > 0 {
			return d
		}
		if jsonDir == "" {
			if _, err := os.Stat(filepath.Join(d, "lm.json")); err == nil {
				jsonDir = d
			}
		}
		if empty == "" {
			if st, err := os.Stat(d); err == nil && st.IsDir() {
				empty = d
			}
		}
	}
	if jsonDir != "" {
		return jsonDir
	}
	if empty != "" {
		return empty
	}
	return artAssets
}

// LoadJSONDir reads lm-1.json … lm-4.json from dir and merges them
// into one atlas. A directory with only lm.json uses that file. A
// directory with no atlas JSON is an error.
func LoadJSONDir(dir string) (*sprite.Atlas, error) {
	merged := &sprite.Atlas{Palette: append([]sprite.PaletteEntry(nil), sprite.DefaultPalette...)}
	found := 0
	for _, sz := range sprite.Sizes {
		path := filepath.Join(dir, fmt.Sprintf("lm-%d.json", int(sz)))
		part, err := sprite.LoadFile(path)
		if err != nil {
			continue
		}
		if len(part.Palette) > 0 {
			merged.Palette = append([]sprite.PaletteEntry(nil), part.Palette...)
		}
		for _, h := range HeadingsFor(sz) {
			if sp, ok := part.Frame(sz, h); ok {
				merged.SetFrame(sz, h, sp)
			}
		}
		found++
	}
	if found == 0 {
		a, err := sprite.LoadFile(filepath.Join(dir, "lm.json"))
		if err != nil {
			return nil, fmt.Errorf("lander: no lm-*.json in %s", dir)
		}
		return a, nil
	}
	return merged, nil
}
