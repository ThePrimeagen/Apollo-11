// adjustflame reads heat-threshold JSON, lets you edit the rungs in a
// TUI (hjkl + s), and writes the file back on save. The same page
// plays all eight headings. -tape writes that runner page as a video.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustflame"
	"github.com/theprimeagen/apollo-11/exec-tui/ui"
)

func main() {
	path := flag.String("config", adjustflame.DefaultConfigPath, "heat threshold JSON")
	tape := flag.String("tape", "", "write a 20s runner tape (dir) and encode mp4 next to it")
	flag.Parse()
	m, err := adjustflame.Open(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *tape != "" {
		if err := writeTape(*tape, m); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	// Same escape hatch as the main TUI: detached ptys and tape rigs
	// fail profile detection and would strip the plumes to monochrome.
	var opts []tea.ProgramOption
	if p, ok := ui.ForcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	p := tea.NewProgram(m, opts...)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writeTape(dir string, m adjustflame.Model) error {
	const (
		seconds = 20
		rate    = 20
		cellW   = 16
	)
	if _, err := m.WriteTape(dir, seconds*rate, cellW); err != nil {
		return err
	}
	mp4 := filepath.Clean(dir) + ".mp4"
	cmd := exec.Command("ffmpeg", "-y",
		"-framerate", fmt.Sprintf("%d", rate),
		"-i", filepath.Join(dir, "frame-%04d.png"),
		"-pix_fmt", "yuv420p",
		"-c:v", "libx264",
		"-movflags", "+faststart",
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
