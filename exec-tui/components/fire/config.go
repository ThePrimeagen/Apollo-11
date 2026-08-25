package fire

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const (
	MinThreshold = 0
	MaxThreshold = 500
	RungCount    = 8
)

var (
	ErrThresholdCount = errors.New("fire: need 8 heat thresholds")
	ErrThresholdRange = errors.New("fire: each threshold must be 0..500")
)

// HeatConfig is the JSON that drives the glyph ladder: one entry heat
// per rung, 0..500. Max of a rung is the next entry minus one.
type HeatConfig struct {
	Thresholds []int `json:"thresholds"`
}

// DefaultHeat is the 15% ladder shipped with the flame.
func DefaultHeat() HeatConfig {
	return HeatConfig{Thresholds: []int{1, 7, 13, 24, 47, 82, 139, 230}}
}

var activeHeat = DefaultHeat()

// UseHeat makes Bands and Style read these thresholds.
func UseHeat(c HeatConfig) error {
	if err := c.Validate(); err != nil {
		return err
	}
	activeHeat = HeatConfig{Thresholds: append([]int(nil), c.Thresholds...)}
	return nil
}

// ResetHeat restores the 15% default ladder.
func ResetHeat() {
	_ = UseHeat(DefaultHeat())
}

// Validate reports the first thing wrong with c.
func (c HeatConfig) Validate() error {
	if len(c.Thresholds) != RungCount {
		return ErrThresholdCount
	}
	for _, n := range c.Thresholds {
		if n < MinThreshold || n > MaxThreshold {
			return ErrThresholdRange
		}
	}
	return nil
}

// LoadHeat reads a heat-threshold JSON file.
func LoadHeat(path string) (HeatConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return HeatConfig{}, err
	}
	var c HeatConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return HeatConfig{}, err
	}
	if err := c.Validate(); err != nil {
		return HeatConfig{}, err
	}
	return c, nil
}

// Save writes the thresholds as JSON.
func (c HeatConfig) Save(path string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

// Bands paints the static glyphs onto these thresholds.
func (c HeatConfig) Bands() []Band {
	styles := rungStyles()
	out := make([]Band, len(styles))
	for i, s := range styles {
		min := 0
		if i < len(c.Thresholds) {
			min = c.Thresholds[i]
		}
		max := 1 << 30
		if i+1 < len(c.Thresholds) {
			max = c.Thresholds[i+1] - 1
		}
		eq := fmt.Sprintf("%d <= H <= %d", min, max)
		if max > 1000 {
			eq = fmt.Sprintf("H >= %d", min)
		}
		out[i] = Band{
			Min: min, Max: max,
			Glyph: s.glyph, Name: s.name,
			FG: s.fg, BG: s.bg, Eq: eq,
		}
	}
	return out
}

type rungStyle struct {
	glyph  rune
	name   string
	fg, bg int
}

func rungStyles() []rungStyle {
	return []rungStyle{
		{'⠁', "single braille", 88, -1},
		{'⠒', "two-dot braille", 124, -1},
		{'⠶', "four-dot braille", 160, -1},
		{'░', "quarter shade", 166, -1},
		{'▒', "half shade", 202, -1},
		{'▄', "half square", 208, 52},
		{'▓', "heavy shade", 220, 166},
		{'█', "solid bright yellow", 226, 220},
	}
}
