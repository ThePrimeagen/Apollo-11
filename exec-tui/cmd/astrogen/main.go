// astrogen: regenerate the astronaut art. Compiles the pixel grids in
// components/astro into the editable atlas JSON the sprite editor
// opens, and — with -png — dumps magnified PNGs of every pose for art
// review outside a terminal.
//
//	go run ./cmd/astrogen                     # rewrite assets/astronaut.json
//	go run ./cmd/astrogen -png /tmp/astro     # also dump review PNGs
//	go run ./cmd/astrogen -o /tmp/a.json      # write somewhere else
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/theprimeagen/apollo-11/exec-tui/components/astro"
)

func main() {
	out := flag.String("o", "", "atlas output path (default: the shipped assets/astronaut.json)")
	pngDir := flag.String("png", "", "also write magnified pose PNGs into this directory")
	scale := flag.Int("scale", 24, "PNG magnification, screen pixels per art pixel")
	flag.Parse()
	path := *out
	if path == "" {
		path = astro.FindAtlas()
	}
	if path == "" {
		fmt.Fprintln(os.Stderr, "astrogen: no atlas path found; pass -o")
		os.Exit(1)
	}
	if err := astro.WriteAtlasFile(path); err != nil {
		fmt.Fprintln(os.Stderr, "astrogen:", err)
		os.Exit(1)
	}
	fmt.Println("wrote", path)
	if *pngDir != "" {
		if err := astro.WritePNGs(*pngDir, *scale); err != nil {
			fmt.Fprintln(os.Stderr, "astrogen:", err)
			os.Exit(1)
		}
		fmt.Println("pngs in", *pngDir)
	}
}
