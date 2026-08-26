package astro

import (
	"fmt"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// CompileGrid folds a pixel grid of palette letters into half-block
// terminal cells: two pixel rows per cell row.
func CompileGrid(rows []string) (sprite.Sprite, error) {
	return sprite.Sprite{}, fmt.Errorf("astro: CompileGrid not implemented")
}
