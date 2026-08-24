package font

// The one-cell alpha style needs a real font: Unicode has no segmented
// letters. This TTF puts 14-segment A–Z at U+E000–U+E019.

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFontFile(t *testing.T) {
	t.Run("happy: SegmentedAlpha.ttf exists and is a real font", func(t *testing.T) {
		p := filepath.Join("SegmentedAlpha.ttf")
		st, err := os.Stat(p)
		if err != nil {
			t.Fatalf("font missing: %v", err)
		}
		if st.Size() < 1024 {
			t.Fatalf("font too small: %d", st.Size())
		}
	})
	t.Run("unhappy: a missing regen script is a build break, not silent", func(t *testing.T) {
		if _, err := os.Stat("genfont.py"); err != nil {
			t.Fatal("genfont.py must ship next to the TTF")
		}
	})
}

func TestFontCmap(t *testing.T) {
	t.Run("happy: cmap has A–Z in the PUA and official segmented digits", func(t *testing.T) {
		cmd := exec.Command("python3", "-c", `
from fontTools.ttLib import TTFont
f = TTFont("SegmentedAlpha.ttf")
cmap = f.getBestCmap()
missing = [hex(cp) for cp in list(range(0xE000, 0xE01A)) + list(range(0x1FBF0, 0x1FBFA)) if cp not in cmap]
if missing:
    raise SystemExit("missing " + ",".join(missing))
print("ok", len(cmap))
`)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("cmap check failed: %v\n%s", err, out)
		}
	})
	t.Run("unhappy: Latin A–Z are not stolen from the UI font", func(t *testing.T) {
		cmd := exec.Command("python3", "-c", `
from fontTools.ttLib import TTFont
f = TTFont("SegmentedAlpha.ttf")
cmap = f.getBestCmap()
stolen = [chr(cp) for cp in range(0x41, 0x5B) if cp in cmap]
if stolen:
    raise SystemExit("must not map ASCII " + "".join(stolen))
`)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("ASCII cmap leaked: %v\n%s", err, out)
		}
	})
}
