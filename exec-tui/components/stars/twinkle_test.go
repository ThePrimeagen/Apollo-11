package stars

// Tests written FIRST: Twinkle is a new sky mode — the sky parks
// where it scattered and some of the stars breathe, fading in and out
// while the rest hold steady. Every twinkling star runs its own
// clock: a full cycle (fade in, hold bright, fade out, hold dark)
// whose duration is picked deterministically from the active config's
// [MinCycleSeconds, MaxCycleSeconds], with ramps picked from
// [MinFadeSeconds, MaxFadeSeconds] and clamped so a fade never
// outlasts its half of the cycle. Mid-fade a star wears a dimmed gray
// from its layer's ramp; at full brightness it wears its own tint; at
// zero it is not painted at all. UseTwinkle/ActiveTwinkle/ResetTwinkle
// follow the sky-config pattern, so a tuner can retune the breathing
// live while the catalog paints.

import (
	"math"
	"testing"
)

// pinned is a config with no ranges: every twinkling star cycles in
// exactly 4 s with 1 s ramps, so tests can reason about the clock.
func pinned() TwinkleConfig {
	return TwinkleConfig{
		MinCycleSeconds: 4,
		MaxCycleSeconds: 4,
		MinFadeSeconds:  1,
		MaxFadeSeconds:  1,
	}
}

// findStars scans a small sky for one breathing star and one steady
// star of any kind.
func findStars(t *testing.T) (twRow, twCol, twKind, stRow, stCol, stKind int) {
	t.Helper()
	twRow, stRow = -1, -1
	for row := 0; row < 24; row++ {
		for col := 0; col < 48; col++ {
			for kind := 0; kind < 4; kind++ {
				if Twinkles(row, col, kind) {
					if twRow < 0 {
						twRow, twCol, twKind = row, col, kind
					}
				} else if stRow < 0 {
					stRow, stCol, stKind = row, col, kind
				}
			}
		}
	}
	if twRow < 0 || stRow < 0 {
		t.Fatal("the sky must hold both breathing and steady stars")
	}
	return
}

func TestTwinkleStrategy(t *testing.T) {
	t.Run("happy: twinkle is a named mode on the catalog and the sky parks under it", func(t *testing.T) {
		if Twinkle.Name != "twinkle" {
			t.Fatalf("the mode is named %q, want twinkle", Twinkle.Name)
		}
		got, ok := Lookup("twinkle")
		if !ok || got.Name != Twinkle.Name {
			t.Fatalf("Lookup(twinkle) = %+v ok=%v — the mode must be on the catalog", got, ok)
		}
		found := false
		for _, s := range Strategies() {
			if s.Name == "twinkle" {
				found = true
			}
		}
		if !found {
			t.Fatal("Strategies() must list the twinkle mode")
		}
		if Twinkle.Delay != [4]int{} {
			t.Fatalf("twinkle parks every layer, delays %v", Twinkle.Delay)
		}
	})
	t.Run("unhappy: a twinkling sky never drifts — the home column holds at any tick", func(t *testing.T) {
		ResetTwinkle()
		cat := NewCatalog(40, 12, [4]int{})
		home := map[[2]int]rune{}
		cat.Paint(0, Still, func(row, col int, ch rune, fg int) {
			home[[2]int{row, col}] = ch
		})
		for _, tick := range []int{0, 33, 500, 3000} {
			cat.Paint(tick, Twinkle, func(row, col int, ch rune, fg int) {
				if home[[2]int{row, col}] != ch {
					t.Fatalf("tick %d painted %q at (%d,%d) — off its scattered home", tick, ch, row, col)
				}
			})
		}
	})
}

func TestTwinkles(t *testing.T) {
	t.Run("happy: some stars breathe and some hold steady — deterministically", func(t *testing.T) {
		breathing, steady := 0, 0
		for row := 0; row < 30; row++ {
			for col := 0; col < 60; col++ {
				kind := (row + col) % 4
				a := Twinkles(row, col, kind)
				if a != Twinkles(row, col, kind) {
					t.Fatalf("Twinkles(%d,%d,%d) flapped — it must be deterministic", row, col, kind)
				}
				if a {
					breathing++
				} else {
					steady++
				}
			}
		}
		total := breathing + steady
		if breathing == 0 || steady == 0 {
			t.Fatalf("breathing %d steady %d — the mode is SOME stars fading, not all and not none", breathing, steady)
		}
		if breathing < total/20 || breathing > total*3/4 {
			t.Fatalf("%d of %d stars breathe — a share worth watching, not a strobe and not a still", breathing, total)
		}
	})
	t.Run("unhappy: a steady star holds full brightness at every instant", func(t *testing.T) {
		_, _, _, row, col, kind := findStars(t)
		for _, tt := range []float64{0, 0.3, 1.7, 9.2, 100} {
			if lvl := TwinkleLevel(row, col, kind, tt, pinned()); lvl != 1 {
				t.Fatalf("steady star level %v at t=%.1f, want 1", lvl, tt)
			}
		}
	})
}

func TestTwinkleLevel(t *testing.T) {
	row, col, kind, _, _, _ := findStars(t)
	cfg := pinned()
	t.Run("happy: a breathing star runs a full cycle — dark, bright, and smooth ramps between", func(t *testing.T) {
		const dt = 0.05
		lo, hi := math.Inf(1), math.Inf(-1)
		prev := TwinkleLevel(row, col, kind, 0, cfg)
		for tt := dt; tt <= cfg.MinCycleSeconds+dt/2; tt += dt {
			lvl := TwinkleLevel(row, col, kind, tt, cfg)
			if lvl < 0 || lvl > 1 {
				t.Fatalf("level %v at t=%.2f — brightness lives in [0,1]", lvl, tt)
			}
			if math.Abs(lvl-prev) > dt/cfg.MinFadeSeconds+1e-6 {
				t.Fatalf("level jumped %v→%v across %.0fms — the fade is a ramp, not a cut", prev, lvl, dt*1000)
			}
			lo, hi = math.Min(lo, lvl), math.Max(hi, lvl)
			prev = lvl
		}
		if lo != 0 {
			t.Fatalf("the dimmest instant is %v — the star must fade all the way out", lo)
		}
		if hi != 1 {
			t.Fatalf("the brightest instant is %v — the star must fade all the way in", hi)
		}
	})
	t.Run("happy: the cycle repeats — one lap later the level is the same", func(t *testing.T) {
		for _, tt := range []float64{0.1, 0.9, 2.2, 3.7} {
			a := TwinkleLevel(row, col, kind, tt, cfg)
			b := TwinkleLevel(row, col, kind, tt+cfg.MinCycleSeconds, cfg)
			if math.Abs(a-b) > 1e-9 {
				t.Fatalf("level %v at t=%.1f but %v one cycle later", a, tt, b)
			}
		}
	})
	t.Run("unhappy: time before the curtain clamps to the start", func(t *testing.T) {
		a := TwinkleLevel(row, col, kind, 0, cfg)
		if b := TwinkleLevel(row, col, kind, -7, cfg); a != b {
			t.Fatalf("negative time moved the fade: %v vs %v", a, b)
		}
	})
	t.Run("unhappy: a fade wider than the cycle clamps to its half — the ramps still meet", func(t *testing.T) {
		wide := TwinkleConfig{MinCycleSeconds: 1, MaxCycleSeconds: 1, MinFadeSeconds: 10, MaxFadeSeconds: 10}
		lo, hi := math.Inf(1), math.Inf(-1)
		for tt := 0.0; tt <= 1.0; tt += 0.01 {
			lvl := TwinkleLevel(row, col, kind, tt, wide)
			if lvl < 0 || lvl > 1 {
				t.Fatalf("level %v at t=%.2f under a clamped fade", lvl, tt)
			}
			lo, hi = math.Min(lo, lvl), math.Max(hi, lvl)
		}
		if lo > 0.01 || hi < 0.99 {
			t.Fatalf("a clamped fade still breathes the full range, saw [%v, %v]", lo, hi)
		}
	})
}

func TestTwinkleInk(t *testing.T) {
	row, col, kind, _, _, _ := findStars(t)
	cfg := pinned()
	t.Run("happy: dark is unpainted, bright is the star's own tint, mid-fade is a dimmed gray", func(t *testing.T) {
		sawDark, sawBright, sawDim := false, false, false
		base := -2
		cat := NewCatalog(48, 24, [4]int{})
		cat.Paint(0, Still, func(r, c int, ch rune, fg int) {
			if r == row && c == col && ch == Glyphs[kind] {
				base = fg
			}
		})
		for tt := 0.0; tt <= cfg.MinCycleSeconds; tt += 0.02 {
			lvl := TwinkleLevel(row, col, kind, tt, cfg)
			ink := TwinkleInk(row, col, kind, tt, cfg)
			switch {
			case lvl <= 0:
				if ink != -1 {
					t.Fatalf("a star faded out must not paint, got ink %d at t=%.2f", ink, tt)
				}
				sawDark = true
			case lvl >= 1:
				if base >= 0 && ink != base {
					t.Fatalf("full brightness wears the star's own tint %d, got %d at t=%.2f", base, ink, tt)
				}
				sawBright = true
			default:
				if ink < 0 || ink > 255 {
					t.Fatalf("mid-fade ink %d at t=%.2f is not an xterm color", ink, tt)
				}
				if base >= 0 && ink == base {
					t.Fatalf("mid-fade must dim the star, still wearing the full tint %d at t=%.2f", base, tt)
				}
				sawDim = true
			}
		}
		if !sawDark || !sawBright || !sawDim {
			t.Fatalf("one cycle must visit dark, dim and bright: dark %v dim %v bright %v", sawDark, sawDim, sawBright)
		}
	})
	t.Run("unhappy: a steady star's ink never moves", func(t *testing.T) {
		_, _, _, r, c, k := findStars(t)
		first := TwinkleInk(r, c, k, 0, cfg)
		for _, tt := range []float64{0.5, 2, 33.3} {
			if got := TwinkleInk(r, c, k, tt, cfg); got != first {
				t.Fatalf("a steady star changed ink %d→%d at t=%.1f", first, got, tt)
			}
		}
	})
}

func TestTwinkleConfig(t *testing.T) {
	t.Cleanup(ResetTwinkle)
	t.Run("happy: the stock breathing is valid and active by default", func(t *testing.T) {
		ResetTwinkle()
		d := DefaultTwinkle()
		if err := d.Validate(); err != nil {
			t.Fatalf("the stock twinkle must validate: %v", err)
		}
		if got := ActiveTwinkle(); got != d {
			t.Fatalf("ActiveTwinkle %+v, want the stock %+v", got, d)
		}
		if d.MinCycleSeconds > d.MaxCycleSeconds || d.MinFadeSeconds > d.MaxFadeSeconds {
			t.Fatalf("the stock ranges must be ordered: %+v", d)
		}
	})
	t.Run("happy: UseTwinkle retunes the active breathing", func(t *testing.T) {
		c := pinned()
		if err := UseTwinkle(c); err != nil {
			t.Fatalf("UseTwinkle(pinned): %v", err)
		}
		if got := ActiveTwinkle(); got != c {
			t.Fatalf("ActiveTwinkle %+v, want %+v", got, c)
		}
		ResetTwinkle()
		if got := ActiveTwinkle(); got != DefaultTwinkle() {
			t.Fatalf("ResetTwinkle left %+v", got)
		}
	})
	t.Run("unhappy: broken ranges are rejected and the active twinkle holds", func(t *testing.T) {
		ResetTwinkle()
		before := ActiveTwinkle()
		bad := []TwinkleConfig{
			{MinCycleSeconds: 5, MaxCycleSeconds: 2, MinFadeSeconds: 0.5, MaxFadeSeconds: 1},   // min above max
			{MinCycleSeconds: 2, MaxCycleSeconds: 5, MinFadeSeconds: 1, MaxFadeSeconds: 0.2},   // fades reversed
			{MinCycleSeconds: 0, MaxCycleSeconds: 5, MinFadeSeconds: 0.5, MaxFadeSeconds: 1},   // under the cycle floor
			{MinCycleSeconds: 2, MaxCycleSeconds: 1e6, MinFadeSeconds: 0.5, MaxFadeSeconds: 1}, // over the cycle ceiling
			{MinCycleSeconds: 2, MaxCycleSeconds: 5, MinFadeSeconds: 0, MaxFadeSeconds: 1},     // under the fade floor
			{MinCycleSeconds: 2, MaxCycleSeconds: 5, MinFadeSeconds: 0.5, MaxFadeSeconds: 1e6}, // over the fade ceiling
			{MinCycleSeconds: math.NaN(), MaxCycleSeconds: 5, MinFadeSeconds: 0.5, MaxFadeSeconds: 1},
		}
		for _, c := range bad {
			if err := UseTwinkle(c); err == nil {
				t.Fatalf("UseTwinkle(%+v) must be refused", c)
			}
			if got := ActiveTwinkle(); got != before {
				t.Fatalf("a refused config moved the active twinkle to %+v", got)
			}
		}
	})
	t.Run("happy: the rails are real numbers a knob can live between", func(t *testing.T) {
		if MinTwinkleCycle <= 0 || MaxTwinkleCycle <= MinTwinkleCycle {
			t.Fatalf("cycle rails [%v, %v] must be an ordered positive range", MinTwinkleCycle, MaxTwinkleCycle)
		}
		if MinTwinkleFade <= 0 || MaxTwinkleFade <= MinTwinkleFade {
			t.Fatalf("fade rails [%v, %v] must be an ordered positive range", MinTwinkleFade, MaxTwinkleFade)
		}
	})
}

func TestTwinklePaint(t *testing.T) {
	t.Cleanup(ResetTwinkle)
	t.Run("happy: the twinkling sky is the still sky with some stars missing — never a stranger", func(t *testing.T) {
		if err := UseTwinkle(pinned()); err != nil {
			t.Fatal(err)
		}
		cat := NewCatalog(48, 20, [4]int{})
		still := map[[2]int]int{}
		cat.Paint(0, Still, func(row, col int, ch rune, fg int) {
			still[[2]int{row, col}] = fg
		})
		if len(still) == 0 {
			t.Fatal("test premise: the still sky holds stars")
		}
		sawFewer, sawSome := false, false
		for _, sec := range []float64{0, 0.5, 1, 1.5, 2, 2.5, 3, 3.5} {
			painted := 0
			cat.Paint(int(sec*StarFPS), Twinkle, func(row, col int, ch rune, fg int) {
				painted++
				if _, ok := still[[2]int{row, col}]; !ok {
					t.Fatalf("t=%.1f painted a star at (%d,%d) the still sky never held", sec, row, col)
				}
			})
			if painted == 0 {
				t.Fatalf("t=%.1f painted nothing — the steady stars always show", sec)
			}
			sawSome = true
			if painted < len(still) {
				sawFewer = true
			}
		}
		if !sawSome || !sawFewer {
			t.Fatal("across a cycle some instant must catch a star faded out")
		}
	})
	t.Run("happy: every painted ink is the star's twinkle ink at that instant", func(t *testing.T) {
		if err := UseTwinkle(pinned()); err != nil {
			t.Fatal(err)
		}
		cat := NewCatalog(40, 16, [4]int{})
		kinds := map[rune]int{}
		for k, g := range Glyphs {
			kinds[g] = k
		}
		const sec = 1.3
		cat.Paint(int(sec*StarFPS), Twinkle, func(row, col int, ch rune, fg int) {
			want := TwinkleInk(row, col, kinds[ch], sec, ActiveTwinkle())
			if fg != want {
				t.Fatalf("star (%d,%d) painted ink %d, TwinkleInk says %d", row, col, fg, want)
			}
		})
	})
	t.Run("unhappy: a negative tick freezes the breath at its opening frame", func(t *testing.T) {
		ResetTwinkle()
		cat := NewCatalog(40, 16, [4]int{})
		opening := map[[2]int]int{}
		cat.Paint(0, Twinkle, func(row, col int, ch rune, fg int) {
			opening[[2]int{row, col}] = fg
		})
		n := 0
		cat.Paint(-500, Twinkle, func(row, col int, ch rune, fg int) {
			n++
			if opening[[2]int{row, col}] != fg {
				t.Fatalf("a negative tick repainted (%d,%d) with ink %d", row, col, fg)
			}
		})
		if n != len(opening) {
			t.Fatalf("a negative tick painted %d stars, the opening frame holds %d", n, len(opening))
		}
	})
}
