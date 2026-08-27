package shotgun

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"unicode/utf8"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const atlasFile = "shotgun.json"

func pixelColor(px rune) (int, error) {
	if px == '.' {
		return -1, nil
	}
	for _, p := range Palette {
		if p.ID == string(px) {
			return p.FG, nil
		}
	}
	return -1, fmt.Errorf("shotgun: unknown pixel letter %q", string(px))
}

// compileGrid folds a pixel grid of palette letters into half-block
// terminal cells. Pair-duplicated rows compile to solid █ so the
// compass mirrors stay honest.
func compileGrid(rows []string) (sprite.Sprite, error) {
	if len(rows) == 0 {
		return sprite.Sprite{}, fmt.Errorf("shotgun: empty pixel grid")
	}
	if len(rows)%2 != 0 {
		return sprite.Sprite{}, fmt.Errorf("shotgun: %d pixel rows — half-block cells need an even count", len(rows))
	}
	w := utf8.RuneCountInString(rows[0])
	if w == 0 {
		return sprite.Sprite{}, fmt.Errorf("shotgun: empty pixel row")
	}
	for i, row := range rows {
		if utf8.RuneCountInString(row) != w {
			return sprite.Sprite{}, fmt.Errorf("shotgun: pixel row %d has %d pixels, want %d", i, utf8.RuneCountInString(row), w)
		}
	}
	sp := sprite.New(w, len(rows)/2)
	for cr := 0; cr < len(rows)/2; cr++ {
		top := []rune(rows[2*cr])
		bot := []rune(rows[2*cr+1])
		for c := 0; c < w; c++ {
			tc, err := pixelColor(top[c])
			if err != nil {
				return sprite.Sprite{}, fmt.Errorf("%w (row %d col %d)", err, 2*cr, c)
			}
			bc, err := pixelColor(bot[c])
			if err != nil {
				return sprite.Sprite{}, fmt.Errorf("%w (row %d col %d)", err, 2*cr+1, c)
			}
			switch {
			case tc < 0 && bc < 0:
			case tc >= 0 && bc < 0:
				sp.Set(cr, c, sprite.Cell{Ch: '▀', FG: tc, BG: -1})
			case tc < 0 && bc >= 0:
				sp.Set(cr, c, sprite.Cell{Ch: '▄', FG: bc, BG: -1})
			case tc == bc:
				sp.Set(cr, c, sprite.Cell{Ch: '█', FG: tc, BG: -1})
			default:
				sp.Set(cr, c, sprite.Cell{Ch: '▀', FG: tc, BG: bc})
			}
		}
	}
	return sp, nil
}

// BuildAtlas compiles the one 2D east gun and spins it onto every
// compass heading around the Y-axis coming out of the screen.
func BuildAtlas() (*sprite.Atlas, error) {
	a := &sprite.Atlas{Palette: append([]sprite.PaletteEntry(nil), Palette...)}
	for _, heading := range sprite.Headings {
		sp, err := compileGrid(dup(rotateGrid(east, headingDeg(heading))...))
		if err != nil {
			return nil, fmt.Errorf("shotgun: %s: %w", heading, err)
		}
		a.SetFrame(Size, heading, sp)
	}
	return a, nil
}

// FindAtlas locates the shipped assets/shotgun.json.
func FindAtlas() string {
	seen := map[string]bool{}
	var cands []string
	addFrom := func(start string) {
		dir := start
		for i := 0; i < 8; i++ {
			cand := filepath.Join(dir, "assets")
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
	var existingDir string
	for _, d := range cands {
		path := filepath.Join(d, atlasFile)
		if _, err := os.Stat(path); err == nil {
			return path
		}
		if existingDir == "" {
			if st, err := os.Stat(d); err == nil && st.IsDir() {
				existingDir = d
			}
		}
	}
	if existingDir != "" {
		return filepath.Join(existingDir, atlasFile)
	}
	return filepath.Join("assets", atlasFile)
}

// Load reads the shipped atlas.
func Load() (*sprite.Atlas, error) {
	return LoadPath(FindAtlas())
}

// LoadPath reads a shotgun atlas from path. A missing or corrupt
// file is an error, never a blank gun.
func LoadPath(path string) (*sprite.Atlas, error) {
	a, err := sprite.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("shotgun: %w", err)
	}
	return a, nil
}
