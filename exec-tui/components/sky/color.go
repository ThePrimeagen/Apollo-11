package sky

import "math"

// cube is the xterm-256 6×6×6 colour cube's one-axis lookup.
var cube = [6]int{0, 95, 135, 175, 215, 255}

// rgbOf is the 8-bit RGB of xterm-256 index n. Out-of-range inks
// collapse to black so a bad knob never panics the painter.
func rgbOf(n int) (r, g, b int) {
	if n < 0 {
		return 0, 0, 0
	}
	if n < 16 {
		return ansi16[n][0], ansi16[n][1], ansi16[n][2]
	}
	if n < 232 {
		n -= 16
		return cube[n/36], cube[(n%36)/6], cube[n%6]
	}
	if n > 255 {
		n = 255
	}
	v := 8 + (n-232)*10
	return v, v, v
}

var ansi16 = [16][3]int{
	{0, 0, 0}, {128, 0, 0}, {0, 128, 0}, {128, 128, 0},
	{0, 0, 128}, {128, 0, 128}, {0, 128, 128}, {192, 192, 192},
	{128, 128, 128}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0},
	{0, 0, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
}

// nearestInk is the xterm-256 index closest to r,g,b.
func nearestInk(r, g, b int) int {
	best, bestD := 16, math.MaxInt
	for n := 16; n < 256; n++ {
		rr, gg, bb := rgbOf(n)
		dr, dg, db := rr-r, gg-g, bb-b
		d := dr*dr + dg*dg + db*db
		if d < bestD {
			best, bestD = n, d
		}
	}
	return best
}

// lerpInk walks from a to b by t in 0..1 through RGB, then snaps
// back onto the xterm cube so a gradient between two knobs stays on
// the palette the rest of the show already speaks.
func lerpInk(a, b int, t float64) int {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	ar, ag, ab := rgbOf(a)
	br, bg, bb := rgbOf(b)
	r := int(math.Round(float64(ar) + t*float64(br-ar)))
	g := int(math.Round(float64(ag) + t*float64(bg-ag)))
	bl := int(math.Round(float64(ab) + t*float64(bb-ab)))
	return nearestInk(r, g, bl)
}

// Lum is the perceptual brightness of xterm-256 index n, 0..255.
// Tests and scenes use it to tell a darker blue from a lighter one
// without depending on the index numbers themselves.
func Lum(n int) int {
	r, g, b := rgbOf(n)
	return (r*3 + g*6 + b) / 10
}

// lum is the package-local spelling the sky tests already call.
func lum(n int) int { return Lum(n) }
