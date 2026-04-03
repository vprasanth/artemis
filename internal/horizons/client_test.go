package horizons

import (
	"testing"
	"time"
)

func TestParseVectorsChoosesClosestSample(t *testing.T) {
	text := `header
$$SOE
2461132.916666667 = A.D. 2026-Apr-02 10:00:00.0000 TDB
 X = 1.000000000000000E+03 Y = 0.000000000000000E+00 Z = 0.000000000000000E+00
 VX= 1.000000000000000E+00 VY= 0.000000000000000E+00 VZ= 0.000000000000000E+00
2461132.958333333 = A.D. 2026-Apr-02 10:01:00.0000 TDB
 X = 2.000000000000000E+03 Y = 0.000000000000000E+00 Z = 0.000000000000000E+00
 VX= 2.000000000000000E+00 VY= 0.000000000000000E+00 VZ= 0.000000000000000E+00
2461133.000000000 = A.D. 2026-Apr-02 10:02:00.0000 TDB
 X = 3.000000000000000E+03 Y = 0.000000000000000E+00 Z = 0.000000000000000E+00
 VX= 3.000000000000000E+00 VY= 0.000000000000000E+00 VZ= 0.000000000000000E+00
$$EOE`

	target := time.Date(2026, time.April, 2, 10, 1, 10, 0, time.UTC)
	got, err := parseVectors(text, target)
	if err != nil {
		t.Fatalf("parseVectors() error = %v", err)
	}

	wantTime := time.Date(2026, time.April, 2, 10, 1, 0, 0, time.UTC)
	if !got.Time.Equal(wantTime) {
		t.Fatalf("sample time = %v, want %v", got.Time, wantTime)
	}
	if got.Position.X != 2000 {
		t.Fatalf("position X = %v, want 2000", got.Position.X)
	}
	if got.Speed != 2 {
		t.Fatalf("speed = %v, want 2", got.Speed)
	}
}
