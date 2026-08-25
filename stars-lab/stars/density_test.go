package stars

// Tests written FIRST: Density is the per-layer frequency knob — stars
// per 1000 cells for dust, spark, mid, near. The zero value keeps the
// stock sky (DefaultDensity), raising one layer thickens that layer
// without touching the ones scattered before it, and absurd values
// clamp instead of hanging the catalog.

import "testing"

func TestDensity(t *testing.T) {
	t.Run("happy: raising one layer multiplies that layer's stars", func(t *testing.T) {
		base := countKinds(Field{Width: 60, Height: 30, Strategy: Still})
		thick := countKinds(Field{Width: 60, Height: 30, Strategy: Still, Density: [4]int{0, 0, 0, 120}})
		if thick[3] < base[3]*5 {
			t.Fatalf("near density 120 painted ✦%d, want at least 5x the stock %d", thick[3], base[3])
		}
		for kind := 0; kind < 3; kind++ {
			if thick[kind] != base[kind] {
				t.Fatalf("layer %d changed %d -> %d; only the raised layer may move", kind, base[kind], thick[kind])
			}
		}
	})
	t.Run("happy: a low density thins the scatter to the anchors", func(t *testing.T) {
		base := countKinds(Field{Width: 60, Height: 30, Strategy: Still})
		thin := countKinds(Field{Width: 60, Height: 30, Strategy: Still, Density: [4]int{1, 0, 0, 0}})
		if thin[0]*2 >= base[0] {
			t.Fatalf("dust density 1 painted ·%d, want under half the stock %d", thin[0], base[0])
		}
		if thin[0] < 3 {
			t.Fatalf("dust can thin but never vanish, got %d", thin[0])
		}
	})
	t.Run("happy: DefaultDensity spelled out is exactly the zero value", func(t *testing.T) {
		zero := Field{Width: 48, Height: 20, Strategy: Still}
		spelled := Field{Width: 48, Height: 20, Strategy: Still, Density: DefaultDensity}
		if zero.Render() != spelled.Render() {
			t.Fatal("Density == DefaultDensity must paint the stock sky, cell for cell")
		}
	})
	t.Run("unhappy: negative densities fall back to the stock layer", func(t *testing.T) {
		zero := Field{Width: 48, Height: 20, Strategy: Still}
		neg := Field{Width: 48, Height: 20, Strategy: Still, Density: [4]int{-9, -1, -400, -7}}
		if zero.Render() != neg.Render() {
			t.Fatal("negative densities must paint the stock sky, cell for cell")
		}
	})
	t.Run("unhappy: an absurd density clamps instead of flooding", func(t *testing.T) {
		n := countKinds(Field{Width: 60, Height: 30, Strategy: Still, Density: [4]int{0, 0, 0, 99999}})
		// the scatter is capped; the mid-row anchors ride on top of it
		if cap := 60*30*MaxDensity/1000 + 4; n[3] > cap {
			t.Fatalf("near painted %d stars, cap is %d", n[3], cap)
		}
		if n[3] < 100 {
			t.Fatalf("a clamped flood is still thick, got %d", n[3])
		}
	})
	t.Run("unhappy: densities on a zero-size field stay safe and empty", func(t *testing.T) {
		f := Field{Density: [4]int{300, 300, 300, 300}}
		if got := f.Render(); got != "" {
			t.Fatalf("empty field must render empty, got %q", got)
		}
	})
}
