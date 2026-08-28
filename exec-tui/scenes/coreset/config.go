package coreset

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"
)

// Config is every timing of the lesson as a live knob: how long the
// memory unit holds, the drain's stagger and per-box burn, the
// survivor's glide to the top, the settle it rests there before the
// first word (the beat that keeps the landing out of the layout
// reveal), the word bar's cadence and hold, and the priority zoom's
// two beats — the fade that burns the rest away and the glide that
// carries PRIO to center stage. The standalone runner nudges each
// knob 50ms at a time and s writes this JSON next to the scene; the
// act boundaries are derived clock marks, so a retimed show still
// knows where its acts begin.
type Config struct {
	UnitSeconds      float64 `json:"unitSeconds"`
	FadeBeat         float64 `json:"fadeBeat"`
	DissolveSeconds  float64 `json:"dissolveSeconds"`
	MoveSeconds      float64 `json:"moveSeconds"`
	SettleSeconds    float64 `json:"settleSeconds"`
	WordBeat         float64 `json:"wordBeat"`
	WordHold         float64 `json:"wordHold"`
	ZoomFadeSeconds  float64 `json:"zoomFadeSeconds"`
	ZoomGlideSeconds float64 `json:"zoomGlideSeconds"`
}

const (
	// StepSeconds is one tick of every knob: 50ms. It is also the
	// floor under the two glides — a move with no duration has no
	// clock to ease along.
	StepSeconds = 0.050

	// DefaultConfigPath is the scene's own JSON, relative to the
	// module root.
	DefaultConfigPath = "scenes/coreset/config.json"
)

// DefaultConfig is the scene as staged: the stock timings, plus the
// settle that parks the survivor before the words begin.
func DefaultConfig() Config {
	return Config{
		UnitSeconds:      UnitSeconds,
		FadeBeat:         FadeBeat,
		DissolveSeconds:  DissolveSeconds,
		MoveSeconds:      MoveSeconds,
		SettleSeconds:    SettleSeconds,
		WordBeat:         WordBeat,
		WordHold:         WordHold,
		ZoomFadeSeconds:  ZoomFadeSeconds,
		ZoomGlideSeconds: ZoomGlideSeconds,
	}
}

// The act boundaries, cumulative — the same marks the old constants
// spelled, now derived from whatever the knobs hold.

// FadeStart is when the drain begins: the unit hold's end.
func (c Config) FadeStart() float64 { return c.UnitSeconds }

// FadeSeconds covers the whole drain: fourteen dissolves — five VAC
// boxes, the VAC title, seven core sets, the core title.
func (c Config) FadeSeconds() float64 { return 14*c.FadeBeat + c.DissolveSeconds }

// MoveStart is when the survivor's glide to the top center begins.
func (c Config) MoveStart() float64 { return c.FadeStart() + c.FadeSeconds() }

// SettleStart is the landing: the glide is over, the box is parked
// on its exact spot, and it rests.
func (c Config) SettleStart() float64 { return c.MoveStart() + c.MoveSeconds }

// WordsStart is when the twelve-word bar begins — only after the
// settle, so the transition finishes before the layout displays.
func (c Config) WordsStart() float64 { return c.SettleStart() + c.SettleSeconds }

// WordsSeconds covers the reveal and the hold on the finished anatomy.
func (c Config) WordsSeconds() float64 { return 12*c.WordBeat + c.WordHold }

// ZoomStart is when everything but PRIORITY begins to burn away.
func (c Config) ZoomStart() float64 { return c.WordsStart() + c.WordsSeconds() }

// ZoomSeconds is the whole priority zoom: the fade, then the glide.
func (c Config) ZoomSeconds() float64 { return c.ZoomFadeSeconds + c.ZoomGlideSeconds }

// BitsStart is when the 15-bit word breaks open — PRIO already parked
// on its seat.
func (c Config) BitsStart() float64 { return c.ZoomStart() + c.ZoomSeconds() }

// Knob is which knob the cursor is on.
type Knob int

const (
	KnobUnit Knob = iota
	KnobFadeBeat
	KnobDissolve
	KnobMove
	KnobSettle
	KnobWordBeat
	KnobWordHold
	KnobZoomFade
	KnobZoomGlide
	KnobCount
)

// KnobLabel is the panel name of knob k.
func KnobLabel(k Knob) string {
	switch k {
	case KnobUnit:
		return "unit hold"
	case KnobFadeBeat:
		return "fade beat"
	case KnobDissolve:
		return "dissolve"
	case KnobMove:
		return "move"
	case KnobSettle:
		return "settle"
	case KnobWordBeat:
		return "word beat"
	case KnobWordHold:
		return "word hold"
	case KnobZoomFade:
		return "zoom fade"
	case KnobZoomGlide:
		return "zoom glide"
	default:
		return ""
	}
}

// Value reads one knob's seconds for the panel and the tests.
func (c Config) Value(k Knob) float64 {
	switch k {
	case KnobUnit:
		return c.UnitSeconds
	case KnobFadeBeat:
		return c.FadeBeat
	case KnobDissolve:
		return c.DissolveSeconds
	case KnobMove:
		return c.MoveSeconds
	case KnobSettle:
		return c.SettleSeconds
	case KnobWordBeat:
		return c.WordBeat
	case KnobWordHold:
		return c.WordHold
	case KnobZoomFade:
		return c.ZoomFadeSeconds
	case KnobZoomGlide:
		return c.ZoomGlideSeconds
	default:
		return 0
	}
}

// Display is knob k's panel reading, seconds to the millisecond.
func (c Config) Display(k Knob) string {
	if k < 0 || k >= KnobCount {
		return ""
	}
	return fmt.Sprintf("%7.3fs", c.Value(k))
}

// floor is the lowest a knob may go: the two glides keep one step of
// duration, every hold may reach zero.
func floor(k Knob) float64 {
	if k == KnobMove || k == KnobZoomGlide {
		return StepSeconds
	}
	return 0
}

func (c *Config) set(k Knob, v float64) {
	switch k {
	case KnobUnit:
		c.UnitSeconds = v
	case KnobFadeBeat:
		c.FadeBeat = v
	case KnobDissolve:
		c.DissolveSeconds = v
	case KnobMove:
		c.MoveSeconds = v
	case KnobSettle:
		c.SettleSeconds = v
	case KnobWordBeat:
		c.WordBeat = v
	case KnobWordHold:
		c.WordHold = v
	case KnobZoomFade:
		c.ZoomFadeSeconds = v
	case KnobZoomGlide:
		c.ZoomGlideSeconds = v
	}
}

// Nudge walks the selected knob by dir steps of the 50ms grid,
// rounded to the millisecond so a nudge up and back down always
// lands on the exact same value, clamped at the knob's floor. A bad
// cursor is a no-op.
func (c *Config) Nudge(k Knob, dir int) {
	if c == nil || dir == 0 || k < 0 || k >= KnobCount {
		return
	}
	v := math.Round((c.Value(k)+StepSeconds*float64(dir))*1000) / 1000
	if lo := floor(k); v < lo {
		v = lo
	}
	c.set(k, v)
}

var (
	errUnit      = errors.New("coreset: unit hold must not be negative")
	errFadeBeat  = errors.New("coreset: fade beat must not be negative")
	errDissolve  = errors.New("coreset: dissolve must not be negative")
	errMove      = errors.New("coreset: move must be at least 50ms")
	errSettle    = errors.New("coreset: settle must not be negative")
	errWordBeat  = errors.New("coreset: word beat must not be negative")
	errWordHold  = errors.New("coreset: word hold must not be negative")
	errZoomFade  = errors.New("coreset: zoom fade must not be negative")
	errZoomGlide = errors.New("coreset: zoom glide must be at least 50ms")

	activeMu sync.Mutex
	active   = DefaultConfig()
)

// finite is a knob that is a real duration, not a NaN or an infinity.
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// Validate reports whether the knobs are playable. Every hold may be
// zero — a skipped act is a choice — but the two glides keep at least
// one step of duration, because each one divides its clock.
func (c Config) Validate() error {
	if c.UnitSeconds < 0 || !finite(c.UnitSeconds) {
		return errUnit
	}
	if c.FadeBeat < 0 || !finite(c.FadeBeat) {
		return errFadeBeat
	}
	if c.DissolveSeconds < 0 || !finite(c.DissolveSeconds) {
		return errDissolve
	}
	if c.MoveSeconds < StepSeconds || !finite(c.MoveSeconds) {
		return errMove
	}
	if c.SettleSeconds < 0 || !finite(c.SettleSeconds) {
		return errSettle
	}
	if c.WordBeat < 0 || !finite(c.WordBeat) {
		return errWordBeat
	}
	if c.WordHold < 0 || !finite(c.WordHold) {
		return errWordHold
	}
	if c.ZoomFadeSeconds < 0 || !finite(c.ZoomFadeSeconds) {
		return errZoomFade
	}
	if c.ZoomGlideSeconds < StepSeconds || !finite(c.ZoomGlideSeconds) {
		return errZoomGlide
	}
	return nil
}

// Active is the timing New copies onto a Core Set scene: the last
// successful Use, or stock after Reset.
func Active() Config {
	activeMu.Lock()
	defer activeMu.Unlock()
	return active
}

// Use makes cfg the timing New plays. A bad cfg is rejected and
// Active is unchanged.
func Use(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	activeMu.Lock()
	active = cfg
	activeMu.Unlock()
	return nil
}

// Reset restores stock timing. Tests call this so a Use cannot leak.
func Reset() {
	activeMu.Lock()
	active = DefaultConfig()
	activeMu.Unlock()
}

// Load reads a Core Set config JSON file. Missing keys keep their
// stock values; broken knobs are refused loudly.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	c := DefaultConfig()
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("coreset: %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return Config{}, fmt.Errorf("coreset: %s: %w", path, err)
	}
	return c, nil
}

// LoadOrDefault is Load, except a missing file is stock timing, not
// an error — the same courtesy the other scene configs get.
func LoadOrDefault(path string) (Config, error) {
	c, err := Load(path)
	if err == nil {
		return c, nil
	}
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	return Config{}, err
}

// Save writes the knobs as JSON. A broken config refuses to save.
func (c Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
