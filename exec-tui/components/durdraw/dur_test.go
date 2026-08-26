package durdraw

// Tests written FIRST. Optional reader/writer for native durdraw .dur
// files (gzipped DurMovie JSON). The sprite editor saves JSON atlases.

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func sampleSprite() sprite.Sprite {
	sp := sprite.New(4, 2)
	sp.Set(0, 0, sprite.Cell{Ch: '█', FG: 178, BG: 94})
	sp.Set(0, 1, sprite.Cell{Ch: '░', FG: 24, BG: 232})
	sp.Set(1, 3, sprite.Cell{Ch: 'W', FG: 252, BG: -1})
	return sp
}

func TestDurRoundTrip(t *testing.T) {
	t.Run("happy: glyphs and 256 colors survive sprite → dur → sprite", func(t *testing.T) {
		want := sampleSprite()
		raw, err := Marshal(FromSprite("probe", want))
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		mov, err := Unmarshal(raw)
		if err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		got, err := mov.Sprite()
		if err != nil {
			t.Fatalf("sprite: %v", err)
		}
		if got.Width != want.Width || got.Height != want.Height {
			t.Fatalf("dims %dx%d, want %dx%d", got.Width, got.Height, want.Width, want.Height)
		}
		for r := 0; r < want.Height; r++ {
			for c := 0; c < want.Width; c++ {
				a, b := want.At(r, c), got.At(r, c)
				if a.Ch != b.Ch || a.FG != b.FG || a.BG != b.BG {
					t.Fatalf("cell (%d,%d): got %+v want %+v", r, c, b, a)
				}
			}
		}
	})
	t.Run("happy: bytes are gzipped DurMovie formatVersion 7", func(t *testing.T) {
		raw, err := Marshal(FromSprite("probe", sampleSprite()))
		if err != nil {
			t.Fatal(err)
		}
		zr, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("dur must be gzip: %v", err)
		}
		var doc map[string]json.RawMessage
		if err := json.NewDecoder(zr).Decode(&doc); err != nil {
			t.Fatalf("dur gzip body must be JSON: %v", err)
		}
		movie, ok := doc["DurMovie"]
		if !ok {
			t.Fatal("root object must be DurMovie")
		}
		var header struct {
			FormatVersion int    `json:"formatVersion"`
			ColorFormat   string `json:"colorFormat"`
			Encoding      string `json:"encoding"`
		}
		if err := json.Unmarshal(movie, &header); err != nil {
			t.Fatal(err)
		}
		if header.FormatVersion != 7 {
			t.Fatalf("formatVersion %d, want 7", header.FormatVersion)
		}
		if header.ColorFormat != "256" {
			t.Fatalf("colorFormat %q, want 256", header.ColorFormat)
		}
		if header.Encoding != "utf-8" {
			t.Fatalf("encoding %q, want utf-8", header.Encoding)
		}
	})
	t.Run("unhappy: corrupt bytes are an error, not a zero movie", func(t *testing.T) {
		if _, err := Unmarshal([]byte("not a dur file")); err == nil {
			t.Fatal("garbage must fail")
		}
	})
	t.Run("unhappy: empty gzip JSON is an error", func(t *testing.T) {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, _ = zw.Write([]byte(`{}`))
		_ = zw.Close()
		if _, err := Unmarshal(buf.Bytes()); err == nil {
			t.Fatal("a file with no DurMovie must fail")
		}
	})
}

func TestDurFile(t *testing.T) {
	t.Run("happy: WriteFile/LoadFile round-trip a heading movie", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "lm-4.dur")
		want := sampleSprite()
		if err := WriteFile(path, FromSprite("N", want)); err != nil {
			t.Fatal(err)
		}
		mov, err := LoadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got, err := mov.Sprite()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(got.GlyphRows(), "\n") != strings.Join(want.GlyphRows(), "\n") {
			t.Fatalf("glyphs drifted:\n%s\nwant\n%s", strings.Join(got.GlyphRows(), "\n"), strings.Join(want.GlyphRows(), "\n"))
		}
	})
	t.Run("unhappy: a missing file is an error", func(t *testing.T) {
		if _, err := LoadFile(filepath.Join(t.TempDir(), "nope.dur")); err == nil {
			t.Fatal("missing dur must fail")
		}
	})
}

func TestAtlasMovie(t *testing.T) {
	t.Run("happy: size frames encode as named extra.headings", func(t *testing.T) {
		a := &sprite.Atlas{Palette: append([]sprite.PaletteEntry(nil), sprite.DefaultPalette...)}
		n := sampleSprite()
		s := sampleSprite()
		s.Set(0, 0, sprite.Cell{Ch: 'S', FG: 208, BG: 52})
		a.SetFrame(sprite.Size4, sprite.N, n)
		a.SetFrame(sprite.Size4, sprite.S, s)
		a.SetFrame(sprite.Size4, sprite.W, sampleSprite())
		mov, err := EncodeSize(a, sprite.Size4)
		if err != nil {
			t.Fatal(err)
		}
		if mov.Extra == nil || mov.Extra.Size != 4 {
			t.Fatalf("extra.size must be 4, got %+v", mov.Extra)
		}
		if strings.Join(mov.Extra.Headings, ",") != "N,S,W" {
			t.Fatalf("size-4 headings %v, want N,S,W", mov.Extra.Headings)
		}
		if len(mov.Frames) != 3 {
			t.Fatalf("want 3 frames, got %d", len(mov.Frames))
		}
		got, sz, err := DecodeAtlas(mov)
		if err != nil {
			t.Fatal(err)
		}
		if sz != sprite.Size4 {
			t.Fatalf("size %d, want 4", sz)
		}
		for _, h := range []sprite.Heading{sprite.N, sprite.S, sprite.W} {
			if _, ok := got.Frame(sprite.Size4, h); !ok {
				t.Fatalf("decoded atlas missing %s", h)
			}
		}
		if _, ok := got.Frame(sprite.Size4, sprite.E); ok {
			t.Fatal("size-4 E must not be invented")
		}
		if got.MustFrame(sprite.Size4, sprite.S).At(0, 0).Ch != 'S' {
			t.Fatal("heading S must keep its own glyphs")
		}
	})
	t.Run("unhappy: a movie with no frames is an error", func(t *testing.T) {
		if _, _, err := DecodeAtlas(Movie{SizeX: 1, SizeY: 1}); err == nil {
			t.Fatal("empty movie must fail")
		}
	})
}

func TestColorMapIsColumnMajor(t *testing.T) {
	t.Run("happy: colorMap[col][row] matches durdraw v7", func(t *testing.T) {
		sp := sprite.New(2, 1)
		sp.Set(0, 1, sprite.Cell{Ch: 'X', FG: 208, BG: 52})
		mov := FromSprite("N", sp)
		if len(mov.Frames) != 1 {
			t.Fatal("one sprite is one frame")
		}
		cm := mov.Frames[0].ColorMap
		if len(cm) != 2 || len(cm[1]) != 1 {
			t.Fatalf("colorMap shape %v, want 2 cols × 1 row", cm)
		}
		if cm[1][0][0] != 208 || cm[1][0][1] != 52 {
			t.Fatalf("col 1 row 0 color %v, want [208 52]", cm[1][0])
		}
	})
}

func TestWriteFileCreatesParent(t *testing.T) {
	t.Run("unhappy: WriteFile into a missing directory errors", func(t *testing.T) {
		err := WriteFile(filepath.Join(t.TempDir(), "no", "lm.dur"), FromSprite("N", sampleSprite()))
		if err == nil {
			t.Fatal("missing parent dir must fail")
		}
	})
}

func TestLoadFileRejectsJSONAtlas(t *testing.T) {
	t.Run("unhappy: the old sprite JSON atlas is not a dur file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "lm-4.json")
		if err := os.WriteFile(path, []byte(`{"palette":[],"frames":{}}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadFile(path); err == nil {
			t.Fatal("bespoke JSON atlas must not load as durdraw")
		}
	})
}
