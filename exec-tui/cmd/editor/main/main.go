package main

// lander-edit: a vim-ish terminal editor for the LM ASCII atlas.
//
//	h j k l     move (canvas / symbols / palette / frames)
//	1-0         clutch last 10 colors
//	!@#$%^&*()  jump to ░▒▓█ ▀▄▌▐ ▖▗ on the symbol list
//	p           cycle the symbol list (full / half / quarter / shade)
//	P           paste the selected symbol in the current color
//	i           one-shot insert: next character typed lands on the cell
//	c           8-bit color dropdown (greys + cube; space picks, esc closes)
//	space       toggle-select the cell under the cursor
//	f / b       paint foreground / background only
//	d           delete to transparent
//	ctrl-a / b  increment / decrement shade (░▒▓█)
//	ctrl-w h/l  focus canvas / symbols (vim splits)
//	ctrl-w j/k  focus palette+frames / symbols
//	mouse       click a canvas cell or a symbol to jump there
//	ctrl-s      save JSON
//	q           quit

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/cmd/editor"
)

func main() {
	path := editor.DefaultAtlasPath
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	m, err := editor.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "editor:", err)
		os.Exit(1)
	}
	p := tea.NewProgram(wrapSave{Model: m})
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "editor:", err)
		os.Exit(1)
	}
}

// wrapSave adds ctrl-s → save on top of the editor model.
type wrapSave struct {
	editor.Model
}

func (w wrapSave) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "ctrl+s" {
		if err := w.Save(); err != nil {
			w.Model.SetErr(err.Error())
		} else {
			w.Model.SetStatus("wrote " + w.Path)
		}
		return w, nil
	}
	got, cmd := w.Model.Update(msg)
	w.Model = got.(editor.Model)
	return w, cmd
}
