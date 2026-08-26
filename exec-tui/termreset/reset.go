// Package termreset puts the cursor and line discipline back the way
// the shell expects after a Bubble Tea program leaves. The v2 renderer
// hides the cursor and toggles autowrap on the alternate screen; if
// those are still in effect — or the cursor is mid-line — zsh paints a
// `%` on a partial line and the next command's output stair-steps.
package termreset

import (
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
)

// sequences is the teardown Bubble Tea v1 used to emit on renderer
// stop and v2's cursed renderer still skips after leaving the alt
// screen: SGR off, cursor shown, wrap back on, current line wiped,
// cursor to column 0.
const sequences = "\x1b[0m\x1b[?25h\x1b[?7h\x1b[2K\r"

// Write emits the cursor-reset sequences to w.
func Write(w io.Writer) error {
	_, err := io.WriteString(w, sequences)
	return err
}

// EnableCooked turns ONLCR/OPOST back on when w is a tty. A leftover
// raw-mode output discipline is what makes `git status` walk across
// the screen after exit. Non-ttys are a no-op.
func EnableCooked(w io.Writer) error {
	f, ok := w.(*os.File)
	if !ok {
		return nil
	}
	return enableCooked(f.Fd())
}

// Restore writes the cursor reset to stdout and re-enables cooked
// output on the tty. Safe to call more than once.
func Restore() {
	_ = Write(os.Stdout)
	_ = EnableCooked(os.Stdout)
}

// Run starts a Bubble Tea program and restores the cursor when it
// returns, success or failure.
func Run(m tea.Model, opts ...tea.ProgramOption) (tea.Model, error) {
	got, err := tea.NewProgram(m, opts...).Run()
	Restore()
	return got, err
}
