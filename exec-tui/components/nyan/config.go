// Package nyan is the pop-tart cat: a hull sprite plus a rainbow
// particle trail driven by the shared particle engine. Trail knobs
// (life, spawn, speed, band width) live in this component's config
// file and become the active trail via UseTrail, so the editor and
// the cat stay on the same values.
package nyan

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
)

var (
	ErrBandWidth = errors.New("nyan: band width must be positive")
)

// TrailConfig is the JSON that tunes the rainbow plume: the particle
// engine knobs plus how tall each rainbow color is, in units.
type TrailConfig struct {
	BandWidth float64 `json:"bandWidth"`
	Count     int     `json:"count"`
	Period    float64 `json:"period"`
	MinLife   float64 `json:"minLife"`
	MaxLife   float64 `json:"maxLife"`
	MinSpeed  float64 `json:"minSpeed"`
	MaxSpeed  float64 `json:"maxSpeed"`
	Spread    float64 `json:"spread"`
	Nozzle    float64 `json:"nozzle"`
}

// DefaultTrail is a long, tight, six-stripe nyan plume.
func DefaultTrail() TrailConfig {
	return TrailConfig{
		BandWidth: 2.0,
		Count:     10,
		Period:    0.001,
		MinLife:   1.0,
		MaxLife:   1.6,
		MinSpeed:  16,
		MaxSpeed:  24,
		Spread:    0.04,
		Nozzle:    12.0,
	}
}

var activeTrail = DefaultTrail()

// UseTrail makes ActiveTrail — and every cat reading it — burn these
// settings. Invalid settings are rejected and the active trail holds.
func UseTrail(c TrailConfig) error {
	if err := c.Validate(); err != nil {
		return err
	}
	activeTrail = c
	return nil
}

// ResetTrail restores the stock plume.
func ResetTrail() {
	_ = UseTrail(DefaultTrail())
}

// ActiveTrail is the plume settings now in effect.
func ActiveTrail() TrailConfig { return activeTrail }

// Validate reports the first thing wrong with c.
func (c TrailConfig) Validate() error {
	if c.BandWidth <= 0 {
		return ErrBandWidth
	}
	return c.ParticleConfig().Validate()
}

// ParticleConfig is the particle.Engine world this trail describes:
// a leftward box so the rainbow hangs off the cat's rear.
func (c TrailConfig) ParticleConfig() particle.Config {
	w, h := c.box()
	return particle.Config{
		Width:     w,
		Height:    h,
		Origin:    particle.Vec2{X: w - 1.2, Y: h / 2},
		Direction: particle.Vec2{X: -1, Y: 0},
		Count:     c.Count,
		Period:    c.Period,
		MinLife:   c.MinLife,
		MaxLife:   c.MaxLife,
		MinSpeed:  c.MinSpeed,
		MaxSpeed:  c.MaxSpeed,
		Spread:    c.Spread,
		Nozzle:    c.Nozzle,
	}
}

func (c TrailConfig) box() (width, height float64) {
	bands := 6 * c.BandWidth
	h := bands
	if c.Nozzle > h {
		h = c.Nozzle
	}
	h += 2 * particle.CellHeightUnits
	reach := c.MaxLife * c.MaxSpeed
	w := reach
	if w < 20 {
		w = 20
	}
	return w - 0.01, h - 0.01
}

// LoadTrail reads a trail-config JSON file.
func LoadTrail(path string) (TrailConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return TrailConfig{}, err
	}
	var c TrailConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return TrailConfig{}, fmt.Errorf("nyan: %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return TrailConfig{}, err
	}
	return c, nil
}

// Save writes the trail settings as JSON.
func (c TrailConfig) Save(path string) error {
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
