package fall

import (
	"fmt"
	"math"
	"regexp"
	"strconv"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
)

// Flight altitudes this repo already treats as canonical for the
// first three program alarms (two 1202s in P63, then the 1201 in
// P64). cmd/lander script and descent markers agree; website_spec.md
// quotes ~29,000 ft for the second P63 1202 — we keep 30,900.
const (
	Alarm1AltFt = 33500 // first P63 1202 — cmd/lander +316 s; RESEARCH.md 102:38:22
	Alarm2AltFt = 30900 // second P63 1202 — cmd/lander +358 s (not website_spec's ~29,000)
	Alarm3AltFt = 3000  // P64 1201 — cmd/lander +552 s; RESEARCH.md 102:42:18
	OpenAltFt   = 49971 // PDI — cmd/lander t=0; descent ceiling
	CloseAltFt  = 0     // contact light

	// AlarmRowStep is how many hull-top rows separate the pauses.
	AlarmRowStep = 2

	// AlarmBlink is the caption on/off half-period while a hold freezes the world.
	AlarmBlink = 0.25

	// ElevInk is the descent telemetry green (descent colGreen).
	ElevInk = 48
)

var elevRE = regexp.MustCompile(`ALT\s+(-?[0-9]+)ft`)

// Codes is the historical first-three-alarm order: two 1202s in P63,
// then the 1201 in P64. Not a knob.
func Codes() []string {
	return []string{"1202", "1202", "1201"}
}

// AlarmRows is the hull top-left rows of the three pauses on a stage
// of height h: about one third down, then two rows, then two more.
func AlarmRows(h int) [3]int {
	r1 := h / 3
	return [3]int{r1, r1 + AlarmRowStep, r1 + 2*AlarmRowStep}
}

// AlarmBeats is the pausing drop that parks at the three alarm rows.
// Drop distances share the top-to-bottom span so the hull sits on
// each alarm row when that hold begins. Holds come from cfg verbatim.
func AlarmBeats(stageH int, cfg Config) []lander.DropBeat {
	start, finish := -lander.BodyRows, stageH
	span := float64(finish - start)
	rows := AlarmRows(stageH)
	targets := []int{rows[0], rows[1], rows[2], finish}
	from := start
	drops := make([]float64, 4)
	var used float64
	if span != 0 && cfg.DropSeconds != 0 {
		for i, to := range targets {
			frac := float64(to-from) / span
			drops[i] = cfg.DropSeconds * frac
			used += drops[i]
			from = to
		}
		drops[3] += cfg.DropSeconds - used
	} else if cfg.DropSeconds != 0 {
		drops[3] = cfg.DropSeconds
	}
	return []lander.DropBeat{
		{Drop: drops[0], Hold: cfg.Hold1},
		{Drop: drops[1], Hold: cfg.Hold2},
		{Drop: drops[2], Hold: cfg.Hold3},
		{Drop: drops[3], Hold: 0},
	}
}

// ElevationAt is the flight altitude at a hull top-left row: PDI off
// the top, the three alarm altitudes at their pause rows, contact off
// the bottom, lerp in between so a one-row step cannot jump 30,000 ft.
func ElevationAt(row, stageH int) float64 {
	start := -lander.BodyRows
	rows := AlarmRows(stageH)
	points := [][2]float64{
		{float64(start), OpenAltFt},
		{float64(rows[0]), Alarm1AltFt},
		{float64(rows[1]), Alarm2AltFt},
		{float64(rows[2]), Alarm3AltFt},
		{float64(stageH), CloseAltFt},
	}
	if row <= int(points[0][0]) {
		return points[0][1]
	}
	for i := 1; i < len(points); i++ {
		r0, a0 := int(points[i-1][0]), points[i-1][1]
		r1, a1 := int(points[i][0]), points[i][1]
		if row > r1 {
			continue
		}
		if r1 == r0 {
			return a1
		}
		f := float64(row-r0) / float64(r1-r0)
		return a0 + f*(a1-a0)
	}
	return points[len(points)-1][1]
}

// FormatElevation is the descent HUD face: "ALT  33500ft".
func FormatElevation(alt float64) string {
	if math.IsNaN(alt) || math.IsInf(alt, 0) {
		return "ALT      ?ft"
	}
	return fmt.Sprintf("ALT %6.0fft", alt)
}

// ParseElevation reads FormatElevation (and the painted top-left HUD).
func ParseElevation(s string) (float64, bool) {
	m := elevRE.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	v, err := strconv.ParseFloat(m[1], 64)
	return v, err == nil
}
