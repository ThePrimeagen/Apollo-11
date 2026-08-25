package stars

import (
	"encoding/json"
	"errors"
	"os"
)

const (
	// LayerCount is the four star layers: dust, spark, mid, near.
	LayerCount = 4
	// MinDelay/MaxDelay bound a layer's ticks-per-cell in a sky config:
	// 1 streaks every tick, 60 barely crawls.
	MinDelay = 1
	MaxDelay = 60
	// MinDensity bounds a layer's stars per 1000 cells; MaxDensity is
	// the same flood cap the catalog enforces.
	MinDensity = 1
)

var (
	ErrLayerCount   = errors.New("stars: need 4 delays and 4 densities")
	ErrDelayRange   = errors.New("stars: each delay must be 1..60")
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
	return SkyConfig{}
}

// UseSky makes ActiveSky — and everything reading it — fly these
// settings. Invalid settings are rejected and the active sky holds.
func UseSky(c SkyConfig) error {
	return nil
}

// ResetSky restores the stock sky.
func ResetSky() {
}

// ActiveSky is the sky settings now in effect.
func ActiveSky() SkyConfig {
	return SkyConfig{}
}

// Validate reports the first thing wrong with c.
func (c SkyConfig) Validate() error {
	_ = json.Marshal
	_ = os.ReadFile
	return nil
}

// LoadSky reads a sky-config JSON file.
func LoadSky(path string) (SkyConfig, error) {
	return SkyConfig{}, nil
}

// Save writes the sky settings as JSON.
func (c SkyConfig) Save(path string) error {
	return nil
}

// FlyStrategy is the fly style these settings describe.
func (c SkyConfig) FlyStrategy() Strategy {
	return Strategy{}
}

// DensityLayers is the per-layer density these settings describe.
func (c SkyConfig) DensityLayers() [4]int {
	return [4]int{}
}
