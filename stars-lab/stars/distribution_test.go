package stars

// Tests written FIRST: the scatter must be uniform across the whole
// sky. No row may carry a visible stripe of stars (the old mid-row
// comb read as "the stars concentrate at the center"), and the spread
// must be flat — a uniform hash, never a normal distribution bunching
// stars around the middle. Happy + unhappy throughout.

import "testing"

// rowCounts paints f once and tallies stars per row. Tick 0 places
// every star on its own cell, so put fires exactly once per star.
func rowCounts(f Field) []int {
	rows := make([]int, f.Height)
	f.Paint(func(row, col int, ch rune, fg int) {
		rows[row]++
	})
	return rows
}

func colCounts(f Field) []int {
	cols := make([]int, f.Width)
	f.Paint(func(row, col int, ch rune, fg int) {
		cols[col]++
	})
	return cols
}

func sum(ns []int) int {
	t := 0
	for _, n := range ns {
		t += n
	}
	return t
}

func TestScatterDistribution(t *testing.T) {
	t.Run("happy: no row is a stripe — every row stays near the average", func(t *testing.T) {
		for _, w := range []int{60, 128, 200} {
			for _, h := range []int{24, 30, 31} {
				f := Field{Width: w, Height: h, Strategy: Still}
				rows := rowCounts(f)
				avg := sum(rows) / h
				for r, n := range rows {
					// the four glyph anchors ride on the mid row;
					// beyond them every row must look ordinary
					if limit := avg*3/2 + 4; n > limit {
						t.Fatalf("%dx%d: row %d holds %d stars, average is %d (limit %d) — the sky is striped",
							w, h, r, n, avg, limit)
					}
				}
			}
		}
	})
	t.Run("happy: rows spread uniformly — no normal-distribution bunching", func(t *testing.T) {
		f := Field{Width: 128, Height: 30, Strategy: Still}
		rows := rowCounts(f)
		third := f.Height / 3
		middle := sum(rows[third : f.Height-third])
		total := sum(rows)
		// uniform ⇒ the middle third holds about a third of the
		// stars; a normal distribution would hoard well over half
		if middle*100 > total*47 || middle*100 < total*20 {
			t.Fatalf("middle third of rows holds %d of %d stars (%d%%), want roughly a third",
				middle, total, middle*100/total)
		}
	})
	t.Run("happy: columns spread uniformly — no bunching at the center", func(t *testing.T) {
		f := Field{Width: 128, Height: 30, Strategy: Still}
		cols := colCounts(f)
		third := f.Width / 3
		middle := sum(cols[third : f.Width-third])
		total := sum(cols)
		if middle*100 > total*47 || middle*100 < total*20 {
			t.Fatalf("middle third of columns holds %d of %d stars (%d%%), want roughly a third",
				middle, total, middle*100/total)
		}
	})
	t.Run("unhappy: a field too narrow for anchors still paints a safe sky", func(t *testing.T) {
		f := Field{Width: 8, Height: 20, Strategy: Still}
		rows := rowCounts(f)
		if sum(rows) < 4 {
			t.Fatalf("an 8-wide sky must still hold stars, got %d", sum(rows))
		}
		f.Paint(func(row, col int, ch rune, fg int) {
			if row < 0 || row >= f.Height || col < 0 || col >= f.Width {
				t.Fatalf("star out of bounds at %d,%d", row, col)
			}
		})
	})
	t.Run("unhappy: a one-row sky scatters onto its only row without hanging", func(t *testing.T) {
		f := Field{Width: 50, Height: 1, Strategy: Still}
		rows := rowCounts(f)
		if rows[0] < 4 {
			t.Fatalf("a one-row sky must still hold stars, got %d", rows[0])
		}
		if rows[0] > 50 {
			t.Fatalf("a one-row sky cannot hold more stars than cells, got %d", rows[0])
		}
	})
}
