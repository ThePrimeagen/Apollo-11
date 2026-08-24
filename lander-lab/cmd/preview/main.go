package main

// preview prints every size of heading N (the shrink animation frames)
// then the eight headings at size 4. Pass -shrink to play the shrink
// animation in the terminal.

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/theprimeagen/apollo-11/lander-lab/sprite"
)

func main() {
	shrink := flag.Bool("shrink", false, "play the 4→1 shrink animation")
	dump := flag.String("dump", "", "write the atlas JSON to this path")
	flag.Parse()
	a := sprite.Default()
	if *dump != "" {
		if err := a.WriteFile(*dump); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *shrink {
		playShrink(a)
		return
	}
	fmt.Println("=== shrink frames (heading N) ===")
	for i, sp := range sprite.ShrinkSequence(a, sprite.N) {
		fmt.Printf("\n-- frame %d  %dx%d --\n", i+1, sp.Width, sp.Height)
		fmt.Println(sprite.Render(sp))
	}
	fmt.Println("\n=== size 4, eight headings ===")
	for _, h := range sprite.Headings {
		sp, _ := a.Frame(sprite.Size4, h)
		fmt.Printf("\n-- %s --\n", h)
		fmt.Println(sprite.Render(sp))
	}
}

func playShrink(a *sprite.Atlas) {
	frames := sprite.ShrinkSequence(a, sprite.N)
	for i := 0; ; i++ {
		sp := frames[i%len(frames)]
		fmt.Print("\x1b[2J\x1b[H")
		fmt.Printf("shrink %d/%d  %dx%d  (ctrl-c quit)\n\n", i%len(frames)+1, len(frames), sp.Width, sp.Height)
		fmt.Println(center(sp, 40, 16))
		time.Sleep(400 * time.Millisecond)
	}
}

func center(sp sprite.Sprite, w, h int) string {
	raw := strings.Split(sprite.Render(sp), "\n")
	padTop := (h - sp.Height) / 2
	if padTop < 0 {
		padTop = 0
	}
	var b strings.Builder
	for i := 0; i < padTop; i++ {
		b.WriteByte('\n')
	}
	left := (w - sp.Width) / 2
	if left < 0 {
		left = 0
	}
	pad := strings.Repeat(" ", left)
	for _, line := range raw {
		b.WriteString(pad)
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
