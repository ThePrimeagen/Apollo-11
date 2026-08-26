package termreset

// Tests written first: after a Bubble Tea program leaves, the cursor
// must go back to column 0 and the leftover modes (hidden cursor,
// disabled wrap, raw SGR) must come off. Without that reset zsh paints
// a `%` on a partial line and the next command's output stair-steps.

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestWrite(t *testing.T) {
	t.Run("happy: the reset puts the cursor on column 0 and restores wrap", func(t *testing.T) {
		var buf bytes.Buffer
		if err := Write(&buf); err != nil {
			t.Fatalf("Write: %v", err)
		}
		got := buf.String()
		for _, want := range []string{
			"\r",        // carriage return — zsh's partial-line `%` check
			"\x1b[0m",   // SGR reset so a leftover color cannot leak
			"\x1b[?25h", // show the cursor
			"\x1b[?7h",  // re-enable autowrap (ultraviolet toggles it off)
			"\x1b[2K",   // wipe the current line so leftovers do not sit under the prompt
		} {
			if !strings.Contains(got, want) {
				t.Fatalf("reset missing %q, got %q", want, got)
			}
		}
	})
	t.Run("unhappy: a broken writer surfaces the write error", func(t *testing.T) {
		err := Write(errWriter{})
		if err == nil {
			t.Fatal("Write must return the writer's error")
		}
		if !errors.Is(err, errBoom) {
			t.Fatalf("got %v, want the writer's boom", err)
		}
	})
}

func TestEnableCooked(t *testing.T) {
	t.Run("happy: a plain writer is left alone — cooked restore is a tty job", func(t *testing.T) {
		var buf bytes.Buffer
		if err := EnableCooked(&buf); err != nil {
			t.Fatalf("EnableCooked on a buffer must be a no-op, got %v", err)
		}
		if buf.Len() != 0 {
			t.Fatal("EnableCooked must not write sequences to a non-tty")
		}
	})
	t.Run("unhappy: a non-tty fd reports that cooked restore cannot run", func(t *testing.T) {
		err := enableCooked(^uintptr(0))
		if err == nil {
			t.Fatal("enableCooked on a bogus fd must fail")
		}
	})
}

var errBoom = errors.New("boom")

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errBoom }

var _ io.Writer = errWriter{}
