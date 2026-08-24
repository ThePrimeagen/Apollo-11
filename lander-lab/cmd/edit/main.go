package main

// lander-edit: a vim-ish terminal editor for the LM ASCII atlas.
//
//	h j k l     move (canvas / palette / frames)
//	space       toggle-select the cell under the cursor
//	i           paint selected palette color (fg+bg); blank becomes █
//	f / b       paint foreground / background only
//	d           delete to transparent
//	ctrl-a / b  increment / decrement shade (░▒▓█)
//	ctrl-w h/l  focus canvas / palette (vim splits)
//	ctrl-w j/k  focus frames / palette
//	mouse       click a canvas cell to jump there
//	ctrl-s      save JSON
//	q           quit

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/lander-lab/editor"
	"github.com/theprimeagen/apollo-11/lander-lab/sprite"
)

func main() {
	path := "sprites/lm.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	a, err := load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "lander-edit:", err)
		os.Exit(1)
	}
	m := editor.New(a, path)
	p := tea.NewProgram(wrapSave{Model: m}, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lander-edit:", err)
		os.Exit(1)
	}
}

func load(path string) (*sprite.Atlas, error) {
	if _, err := os.Stat(path); err == nil {
		return sprite.LoadFile(path)
	}
	a := sprite.Default()
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0o755)
	}
	if err := a.WriteFile(path); err != nil {
		return a, nil // still edit in memory
	}
	return a, nil
}

// wrapSave adds ctrl-s → save on top of the editor model.
type wrapSave struct {
	editor.Model
}

func (w wrapSave) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && k.Type == tea.KeyCtrlS {
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
