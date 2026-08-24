// Terminal Fonts demo — prints the A-Z banner catalog and the
// seven-segment digits one through zero at heights 1 through 5.
//
//	go run .                     # both catalogs, every height
//	go run . -height 3           # one height only
//	go run . -seven              # seven-segment digits only
//	go run . -text "APOLLO 11"   # custom banner text
//	go run . -text 1969 -seven   # custom seven-segment text
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/theprimeagen/apollo-11/terminal-fonts/termfont"
)

const (
	abc       = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits    = "1234567890"
	headerFG  = "\x1b[38;5;39m"
	resetANSI = "\x1b[0m"
)

// renderABC returns the A-Z catalog banner for one height. Heights 4 and
// 5 split the alphabet across two blocks (A-M, N-Z) separated by a blank
// line so wide banners stay readable.
func renderABC(height int) (string, error) {
	chunks := []string{abc}
	if height >= 4 {
		chunks = []string{abc[:13], abc[13:]}
	}
	blocks := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		lines, err := termfont.Lines(height, chunk)
		if err != nil {
			return "", err
		}
		blocks = append(blocks, strings.Join(lines, "\n"))
	}
	return strings.Join(blocks, "\n\n"), nil
}

// renderDigits returns one through zero as a seven-segment banner.
func renderDigits(height int) (string, error) {
	lines, err := termfont.LinesSeven(height, digits)
	if err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}

// banner renders the requested catalog or caller-supplied text.
func banner(height int, text string, seven bool) (string, error) {
	switch {
	case seven && text != "":
		lines, err := termfont.LinesSeven(height, text)
		if err != nil {
			return "", err
		}
		return strings.Join(lines, "\n"), nil
	case seven:
		return renderDigits(height)
	case text != "":
		lines, err := termfont.Lines(height, text)
		if err != nil {
			return "", err
		}
		return strings.Join(lines, "\n"), nil
	default:
		letters, err := renderABC(height)
		if err != nil {
			return "", err
		}
		numbers, err := renderDigits(height)
		if err != nil {
			return "", err
		}
		return letters + "\n\n" + numbers, nil
	}
}

func main() {
	height := flag.Int("height", 0, "render one height 1..5 (0 = all)")
	text := flag.String("text", "", "custom text instead of the catalogs")
	seven := flag.Bool("seven", false, "use the seven-segment number font")
	flag.Parse()

	heights := []int{1, 2, 3, 4, 5}
	if *height != 0 {
		heights = []int{*height}
	}
	for i, h := range heights {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("%sHEIGHT %d%s\n", headerFG, h, resetANSI)
		out, err := banner(h, *text, *seven)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Println(out)
	}
}
