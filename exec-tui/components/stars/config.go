package stars

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const (
	// LayerCount is the four star layers: dust, spark, mid, near.
	LayerCount = 4
	// MinDelay/MaxDelay bound a layer's ticks-per-cell in a sky config:
	// 0 parks the layer still, 1 streaks every tick, 60 barely crawls.
	MinDelay = 0
	MaxDelay = 60
	// MinDensity bounds a layer's stars per 1000 cells; MaxDensity is
	// the same flood cap the catalog enforces.
	MinDensity = 1
)

var (
	ErrLayerCount   = errors.New("stars: need 4 delays and 4 densities")
	ErrDelayRange   = errors.New("stars: each delay must be 0..60 (0 parks the layer)")
	ErrDensityRange = errors.New("stars: each density must be 1..400")
)

// SkyConfig is the JSON that tunes the sky — the same shape of config
// file the fire's heat ladder uses. One fly delay (ticks per cell) and
// one density (stars per 1000 cells) per layer, dust to near.
type SkyConfig struct {
	Delay   []int `json:"delay"`
	Density []int `json:"density"`
}

// DefaultSky is the stock sky: the drift delays over DefaultDensity.
func DefaultSky() SkyConfig {
	return SkyConfig{
		Delay:   Drift.Delay[:],
		Density: DefaultDensity[:],
	}
}

var activeSky = DefaultSky()

// UseSky makes ActiveSky — and everything reading it — fly these
// settings. Invalid settings are rejected and the active sky holds.
func UseSky(c SkyConfig) error {
	if err := c.Validate(); err != nil {
		return err
	}
	activeSky = SkyConfig{
		Delay:   append([]int(nil), c.Delay...),
		Density: append([]int(nil), c.Density...),
	}
	return nil
}

// ResetSky restores the stock sky.
func ResetSky() {
	_ = UseSky(DefaultSky())
}

// ActiveSky is the sky settings now in effect.
func ActiveSky() SkyConfig {
	return SkyConfig{
		Delay:   append([]int(nil), activeSky.Delay...),
		Density: append([]int(nil), activeSky.Density...),
	}
}

// Validate reports the first thing wrong with c.
func (c SkyConfig) Validate() error {
	if len(c.Delay) != LayerCount || len(c.Density) != LayerCount {
		return ErrLayerCount
	}
	for _, d := range c.Delay {
		if d < MinDelay || d > MaxDelay {
			return ErrDelayRange
		}
	}
	for _, d := range c.Density {
		if d < MinDensity || d > MaxDensity {
			return ErrDensityRange
		}
	}
	return nil
}

// LoadSky reads a sky-config JSON file.
func LoadSky(path string) (SkyConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return SkyConfig{}, err
	}
	var c SkyConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return SkyConfig{}, fmt.Errorf("stars: %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return SkyConfig{}, err
	}
	return c, nil
}

// Save writes the sky settings as JSON.
func (c SkyConfig) Save(path string) error {
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

// FlyStrategy is the fly style these settings describe.
func (c SkyConfig) FlyStrategy() Strategy {
	s := Strategy{Name: "tuned"}
	copy(s.Delay[:], c.Delay)
	return s
}

// DensityLayers is the per-layer density these settings describe.
func (c SkyConfig) DensityLayers() [4]int {
	var out [4]int
	copy(out[:], c.Density)
	return out
}
