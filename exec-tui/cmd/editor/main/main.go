package main

// lander-edit: a vim-ish terminal editor for the LM ASCII atlas.
//
//	ctrl-h / ctrl-l  cycle canvas layers: ascii outline ↔ foreground ↔ background
//	h j k l / arrows  move the canvas cursor
//	1-0         clutch last 10 colors (applies to the active layer)
//	!@#$%^&*()  jump to ░▒▓█ ▀▄▌▐ ▖▗ on the symbol list
//	p / P       paste glyph on outline; paint FG on fg; paint BG on bg
//	i           one-shot insert: next character typed lands on the cell
//	c           8-bit color dropdown (greys + cube; space picks, esc closes)
//	space       toggle-select the cell under the cursor
//	f / b       paint foreground / background only
//	d           delete: ASCII on outline; color only on fg/bg (glyph stays)
//	x           cut: pick up + delete glyph on outline; color only on fg/bg
//	ctrl-a / b  increment / decrement shade (░▒▓█)
//	ctrl-w h/l  close popup / open symbols
//	ctrl-w j/k  open palette / frames (popups over centered art)
//	mouse       click a canvas cell or a symbol to jump there
//	s / ctrl-s  save JSON (3-height toast, 5s)
//	ctrl-p      size×heading gallery — full composite (outline+fg+bg)
//	ctrl-e      26×10 glyph grid (hjkl + enter, or 1a–0z)
//	ctrl-k      named color palette (jk move, enter pick, esc close)
//	q           quit

import (
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/cmd/editor"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

func main() {
	path := editor.DefaultAtlasPath
	if len(os.Args) > 1 {
		path = os.Args[1]
	} else if cand := filepath.Join(editor.FindAssetsDir(), "lm-4.json"); fileExists(cand) {
		path = cand
	}
	m, err := editor.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "editor:", err)
		os.Exit(1)
	}
	if _, err := termreset.Run(wrapSave{Model: m}); err != nil {
		fmt.Fprintln(os.Stderr, "editor:", err)
		os.Exit(1)
	}
}

// wrapSave adds ctrl-s → save on top of the editor model, even when a
// popup would otherwise swallow the key.
type wrapSave struct {
	editor.Model
}

func (w wrapSave) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "ctrl+s" {
		return w, w.Model.SaveWithToast()
	}
	got, cmd := w.Model.Update(msg)
	w.Model = got.(editor.Model)
	return w, cmd
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
