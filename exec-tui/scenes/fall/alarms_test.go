package fall

// Tests written FIRST: the MAIN fall overlay pauses on the first three
// flight alarms — 1202, 1202, 1201 — at the altitudes this repo already
// treats as canonical, and elevation lerps with the hull row so the
// HUD matches each alarm as it fires.
//
// Sources (verified, not invented):
//   - exec-tui/cmd/lander/main.go script: first P63 1202 at 33,500 ft
//     (+316 s), second P63 1202 at 30,900 ft (+358 s), P64 1201 at
//     3,000 ft (+552 s).
//   - exec-tui/components/lander/descent/descent_test.go markers:
//     33500, 30900, 3000 (plus later low-altitude 1202s we do not use).
//   - exec-tui/RESEARCH.md / website_spec.md: first 1202 ~33,500 ft,
//     1201 ~3,000 ft. website_spec.md quotes ~29,000 ft for the second
//     P63 1202; lander/descent keep 30,900 ft — that is the number we
//     show. This is the prog / first-three-alarms story, not the later
//     P64 1202s at 2,000 ft and 770 ft.

import (
	"math"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
)

func TestAlarmAltitudes(t *testing.T) {
	t.Run("happy: the first three flight alarms are 33500, 30900, then 3000 ft", func(t *testing.T) {
		if Alarm1AltFt != 33500 {
			t.Fatalf("first P63 1202 is %v ft, want 33500 (cmd/lander + descent)", Alarm1AltFt)
		}
		if Alarm2AltFt != 30900 {
			t.Fatalf("second P63 1202 is %v ft, want 30900 (cmd/lander + descent, not website_spec's ~29000)", Alarm2AltFt)
		}
		if Alarm3AltFt != 3000 {
			t.Fatalf("P64 1201 is %v ft, want 3000 (cmd/lander + descent + RESEARCH.md)", Alarm3AltFt)
		}
		if OpenAltFt != 49971 {
			t.Fatalf("opening altitude %v, want PDI 49971 ft", OpenAltFt)
		}
		if CloseAltFt != 0 {
			t.Fatalf("closing altitude %v, want contact 0 ft", CloseAltFt)
		}
		if AlarmRowStep != 2 {
			t.Fatalf("row step %d, want 2", AlarmRowStep)
		}
		if Codes()[0] != "1202" || Codes()[1] != "1202" || Codes()[2] != "1201" {
			t.Fatalf("codes %v, want 1202, 1202, 1201", Codes())
		}
		if FormatElevation(33500) != "ALT  33500ft" {
			t.Fatalf("HUD %q, want the descent face ALT  33500ft", FormatElevation(33500))
		}
	})
	t.Run("unhappy: a missing code list is still three long, and a bad format stays finite", func(t *testing.T) {
		got := Codes()
		if len(got) != 3 {
			t.Fatalf("Codes len %d, want 3 — never a short list that 1201 could lead", len(got))
		}
		if math.IsNaN(ElevationAt(0, 0)) || math.IsInf(ElevationAt(-99, -1), 0) {
			t.Fatal("ElevationAt on a nonsense stage must stay finite")
		}
		if FormatElevation(math.NaN()) == "" {
			t.Fatal("FormatElevation must still return a string for NaN")
		}
	})
}

func TestElevationAt(t *testing.T) {
	t.Run("happy: elevation matches each alarm row and lerps between them", func(t *testing.T) {
		rows := AlarmRows(stageH)
		if rows[0] != stageH/3 {
			t.Fatalf("first alarm row %d, want 1/3 of %d", rows[0], stageH)
		}
		if rows[1] != rows[0]+AlarmRowStep || rows[2] != rows[1]+AlarmRowStep {
			t.Fatalf("alarm rows %v, want +%d then +%d", rows, AlarmRowStep, AlarmRowStep)
		}
		if math.Abs(ElevationAt(rows[0], stageH)-Alarm1AltFt) > 1e-9 {
			t.Fatalf("at the first pause row elevation %v, want %v", ElevationAt(rows[0], stageH), Alarm1AltFt)
		}
		if math.Abs(ElevationAt(rows[1], stageH)-Alarm2AltFt) > 1e-9 {
			t.Fatalf("at the second pause row elevation %v, want %v", ElevationAt(rows[1], stageH), Alarm2AltFt)
		}
		if math.Abs(ElevationAt(rows[2], stageH)-Alarm3AltFt) > 1e-9 {
			t.Fatalf("at the 1201 row elevation %v, want %v", ElevationAt(rows[2], stageH), Alarm3AltFt)
		}
		mid := ElevationAt((rows[0]+rows[1])/2, stageH)
		if mid >= Alarm1AltFt || mid <= Alarm2AltFt {
			t.Fatalf("midway 33500→30900 is %v — must sit strictly between", mid)
		}
		at2 := ElevationAt(rows[1], stageH)
		next := ElevationAt(rows[1]+1, stageH)
		if math.Abs(at2-next) < 100 {
			t.Fatalf("one row past the second 1202 must move off %v, still %v", at2, next)
		}
		if math.Abs(next-Alarm2AltFt) > math.Abs(Alarm2AltFt-Alarm3AltFt) {
			t.Fatal("a one-row step must not jump the full 27,900 ft")
		}
	})
	t.Run("unhappy: off-stage rows stay on the rail and a zero stage does not panic", func(t *testing.T) {
		top := ElevationAt(-lander.BodyRows, stageH)
		if math.Abs(top-OpenAltFt) > 1e-9 {
			t.Fatalf("off the top elevation %v, want PDI %v", top, OpenAltFt)
		}
		bottom := ElevationAt(stageH, stageH)
		if math.Abs(bottom-CloseAltFt) > 1e-9 {
			t.Fatalf("off the bottom elevation %v, want contact %v", bottom, CloseAltFt)
		}
		_ = ElevationAt(0, 0)
		_ = ElevationAt(4, 1)
		_ = AlarmRows(0)
		_ = AlarmRows(-3)
	})
}

func TestAlarmBeats(t *testing.T) {
	t.Run("happy: MAIN beats park at the three alarm rows", func(t *testing.T) {
		c := mainFallKnobs()
		beats := AlarmBeats(stageH, c)
		if len(beats) != 4 {
			t.Fatalf("beats %d, want 4 (three pauses and the last drop)", len(beats))
		}
		rows := AlarmRows(stageH)
		t0 := beats[0].Drop
		r0, _ := lander.DropBeatPath(stageW, stageH, t0, beats)
		if r0 != rows[0] {
			t.Fatalf("first hold row %d, want alarm row %d", r0, rows[0])
		}
		t1 := t0 + beats[0].Hold + beats[1].Drop
		r1, _ := lander.DropBeatPath(stageW, stageH, t1, beats)
		if r1 != rows[1] {
			t.Fatalf("second hold row %d, want alarm row %d", r1, rows[1])
		}
		t2 := t1 + beats[1].Hold + beats[2].Drop
		r2, _ := lander.DropBeatPath(stageW, stageH, t2, beats)
		if r2 != rows[2] {
			t.Fatalf("third hold row %d, want alarm row %d", r2, rows[2])
		}
	})
	t.Run("unhappy: zero holds still build beats, and a tiny stage never panics", func(t *testing.T) {
		c := DefaultConfig()
		beats := AlarmBeats(stageH, c)
		if lander.DropBeatHold(beats[0].Drop+0.1, beats) != -1 {
			t.Fatal("stock zero holds must not park")
		}
		_ = AlarmBeats(1, c)
		_ = AlarmBeats(0, c)
		neg := mainFallKnobs()
		neg.Hold1 = -2
		_ = AlarmBeats(stageH, neg)
	})
}
