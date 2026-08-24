package main

// preview prints every size of heading N (the shrink animation frames)
// then the eight headings at size 4. Pass -shrink to play the shrink
// animation in the terminal.

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/theprimeagen/apollo-11/lander-lab/components/fire"
	"github.com/theprimeagen/apollo-11/lander-lab/sprite"
)

func main() {
	shrink := flag.Bool("shrink", false, "play the 4→1 shrink animation")
	dump := flag.String("dump", "", "write the atlas JSON to this path")
	playFire := flag.Bool("fire", false, "play the left-to-right booster flame")
	tape := flag.String("tape", "", "write a 20s booster tape (dir) and encode mp4 next to it")
	flag.Parse()
	if *tape != "" {
		if err := writeFireTape(*tape); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *playFire {
		playFlame()
		return
	}
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

func playFlame() {
	fmt.Print(fire.Guide())
	f := fire.Booster(1)
	for {
		f.Update(1.0 / 20)
		fmt.Print("\x1b[2J\x1b[H")
		fmt.Printf("booster  left→right  4×2 units  100 particles  (ctrl-c quit)\n\n")
		fmt.Println(sprite.Render(f.View()))
		time.Sleep(50 * time.Millisecond)
	}
}

func writeFireTape(dir string) error {
	const (
		seconds = 20
		rate    = 20
		cellW   = 32
		roseW   = 16
	)
	fmt.Print(fire.Guide())
	frames := seconds * rate
	roseDir := filepath.Join(dir, "compass")
	if _, err := fire.WriteCompassTape(roseDir, fire.NewCompass(1), frames, roseW); err != nil {
		return err
	}
	warm := fire.NewCompass(1)
	for i := 0; i < 24; i++ {
		warm.Update(1.0 / 20)
	}
	if err := fire.WritePNG(filepath.Join(dir, "compass-still.png"), warm.View(), roseW); err != nil {
		return err
	}
	if err := encodeMP4(roseDir, filepath.Join(dir, "compass.mp4"), rate); err != nil {
		return err
	}
	for _, course := range fire.Courses() {
		name := strings.ToLower(course.Name)
		sub := filepath.Join(dir, name)
		f := fire.Toward(1, course.Dir)
		if _, err := fire.WriteTape(sub, f, frames, cellW); err != nil {
			return err
		}
		still := fire.Toward(1, course.Dir)
		for i := 0; i < 24; i++ {
			still.Update(1.0 / 20)
		}
		if err := fire.WritePNG(filepath.Join(dir, name+".png"), still.View(), cellW); err != nil {
			return err
		}
		if err := encodeMP4(sub, filepath.Join(dir, name+".mp4"), rate); err != nil {
			return err
		}
	}
	return nil
}

func encodeMP4(framesDir, mp4 string, rate int) error {
	cmd := exec.Command("ffmpeg", "-y",
		"-framerate", fmt.Sprintf("%d", rate),
		"-i", filepath.Join(framesDir, "frame-%04d.png"),
		"-pix_fmt", "yuv420p",
		"-c:v", "libx264",
		mp4,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg %s: %w", mp4, err)
	}
	fmt.Println(mp4)
	return nil
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
