// Package bigstar is the larger star component: a sparkle that occupies
// one cell at size 1 and grows into a multi-cell burst (span 2*size-1)
// at sizes 2..5. Size and heading can be set, or rolled random at
// Start. Place pins the center; a parked star sits at stage center.
// The package does not move — motion is the shooting-star scene's.
package bigstar

import (
	"errors"
	"math"
	"math/rand"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	MinSize   = 1
	MaxSize   = 5
	CoreGlyph = '★'
)

var ErrSize = errors.New("bigstar: size must be 1..5")

// ValidateSize reports whether n is a playable star size.
func ValidateSize(n int) error {
	if n < MinSize || n > MaxSize {
		return ErrSize
	}
	return nil
}

// Art paints a size-n burst. Heading, when non-zero, stretches a
// trailing spark opposite the flight so a meteor reads as moving.
// A rejected size is an empty sprite.
func Art(size int, heading particle.Vec2) sprite.Sprite {
	if ValidateSize(size) != nil {
		return sprite.Sprite{}
	}
	n := 2*size - 1
	sp := sprite.New(n, n)
	cx, cy := size-1, size-1
	wake := [2]int{}
	if heading != (particle.Vec2{}) {
		h := heading.Normalize()
		wake = [2]int{-sign(h.X), -sign(h.Y)}
	}
	dirs := [][2]int{
		{0, -1}, {1, -1}, {1, 0}, {1, 1},
		{0, 1}, {-1, 1}, {-1, 0}, {-1, -1},
	}
	glyphs := []rune{'*', '·', '˚', '·'}
	colors := []int{255, 229, 195, 245}
	for _, d := range dirs {
		extra := 0
		if d == wake {
			extra = 1
		}
		for s := 1; s < size+extra; s++ {
			gi := s - 1
			if gi >= len(glyphs) {
				gi = len(glyphs) - 1
			}
			sp.Set(cy+d[1]*s, cx+d[0]*s, sprite.Cell{Ch: glyphs[gi], FG: colors[gi%len(colors)], BG: -1})
		}
	}
	sp.Set(cy, cx, sprite.Cell{Ch: CoreGlyph, FG: 255, BG: -1})
	return sp
}

func sign(v float64) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	default:
		return 0
	}
}

// Star is the larger star as a scene component.
type Star struct {
	Size       int
	Heading    particle.Vec2
	RandomSize bool
	RandomDir  bool

	Seed   int64
	col    int
	row    int
	pinned bool
	body   sprite.Sprite
	w, h   int
	staged bool
}

// New binds a star to its seed. Size starts at MinSize; set Size,
// RandomSize, RandomDir before Start.
func New(seed int64) *Star {
	return &Star{Size: MinSize, Seed: seed}
}

// NewSized binds a star of the given size. A rejected size falls
// back to MinSize so the caller still has a performer.
func NewSized(size int) *Star {
	if ValidateSize(size) != nil {
		size = MinSize
	}
	return &Star{Size: size}
}

// Span is the burst's width and height in cells: 2*size-1.
func (s *Star) Span() int {
	if s == nil {
		return 0
	}
	n := s.Size
	if ValidateSize(n) != nil {
		n = MinSize
	}
	return 2*n - 1
}

// Place pins the core at (col, row). Call before or after Start.
func (s *Star) Place(col, row int) {
	if s == nil {
		return
	}
	s.col, s.row = col, row
	s.pinned = true
}

// Center is the core's cell.
func (s *Star) Center() (col, row int) {
	if s == nil {
		return 0, 0
	}
	return s.col, s.row
}

func (s *Star) Start(w, h int) {
	if s == nil {
		return
	}
	s.w, s.h = w, h
	rng := rand.New(rand.NewSource(s.Seed))
	if s.RandomSize {
		s.Size = MinSize + rng.Intn(MaxSize-MinSize+1)
	}
	if ValidateSize(s.Size) != nil {
		s.Size = MinSize
	}
	if s.RandomDir {
		ang := rng.Float64() * 2 * math.Pi
		sin, cos := math.Sincos(ang)
		s.Heading = particle.Vec2{X: cos, Y: sin}
	}
	if !s.pinned {
		s.col, s.row = w/2, h/2
	}
	s.body = Art(s.Size, s.Heading)
	s.staged = true
}

func (s *Star) Update(dt float64) {
	if s == nil || dt <= 0 {
		return
	}
}

func (s *Star) Render() sprite.Sprite {
	if s == nil || !s.staged || s.w < 1 || s.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(s.w, s.h)
	s.body = Art(s.Size, s.Heading)
	off := s.Size - 1
	if off < 0 {
		off = 0
	}
	sprite.Blit(stage, s.col-off, s.row-off, s.body)
	return stage
}

func (s *Star) Stop() {
	if s == nil {
		return
	}
	s.body = sprite.Sprite{}
	s.staged = false
}
