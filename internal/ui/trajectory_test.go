package ui

import (
	"math"
	"strings"
	"testing"
	"time"

	"artemis/internal/dsn"
	"artemis/internal/horizons"
)

func TestEarthMoonVectorUsesEarthAndMoonFrames(t *testing.T) {
	state := &horizons.State{
		Position:     horizons.Vector3{X: 100000, Y: 0, Z: 0},
		MoonPosition: horizons.Vector3{X: -300000, Y: 0, Z: 0},
	}

	got, ok := earthMoonVector(state)
	if !ok {
		t.Fatal("expected earthMoonVector to be available")
	}
	if got.X != 400000 || got.Y != 0 {
		t.Fatalf("earthMoonVector() = %+v, want X=400000 Y=0", got)
	}
}

func TestProjectEarthMoonFrameAlignsToCurrentMoonVector(t *testing.T) {
	state := &horizons.State{
		Position:     horizons.Vector3{X: 100000, Y: 100000, Z: 0},
		MoonPosition: horizons.Vector3{X: -300000, Y: -300000, Z: 0},
	}

	frame := buildTrajectoryFrame(state, nil, 80, 20)
	point, ok := frame.project(state.Position)
	if !ok {
		t.Fatal("expected projected spacecraft point to be valid")
	}
	if point.y != frame.centerY {
		t.Fatalf("projected Y = %d, want %d on Earth-Moon line", point.y, frame.centerY)
	}
	if point.x <= frame.earthX || point.x >= frame.moonX {
		t.Fatalf("projected X = %d, want between earth=%d and moon=%d", point.x, frame.earthX, frame.moonX)
	}
}

func TestMoonRelativeVectorPointsFromSpacecraftToMoon(t *testing.T) {
	got := moonRelativeVector(horizons.Vector3{X: 12, Y: -5, Z: 2})
	want := horizons.Vector3{X: -12, Y: 5, Z: -2}
	if got != want {
		t.Fatalf("moonRelativeVector() = %+v, want %+v", got, want)
	}
}

func TestProjectEarthMoonFrameCrossTrack(t *testing.T) {
	sqrtHalf := math.Sqrt(0.5)
	axisX := horizons.Vector3{X: sqrtHalf, Y: sqrtHalf}
	axisY := horizons.Vector3{X: -sqrtHalf, Y: sqrtHalf}

	along, cross := projectEarthMoonFrame(horizons.Vector3{X: 100, Y: 100}, axisX, axisY)
	if math.Abs(along-141.421356) > 0.001 {
		t.Fatalf("along = %v, want about 141.421", along)
	}
	if math.Abs(cross) > 0.001 {
		t.Fatalf("cross = %v, want about 0", cross)
	}
}

func TestFormatStateAgePrefersEphemerisSampleTime(t *testing.T) {
	now := time.Date(2026, time.April, 2, 12, 0, 0, 0, time.UTC)
	m := Model{
		hzState: &horizons.State{
			Time:      now.Add(-95 * time.Second),
			Timestamp: now.Add(-5 * time.Second),
		},
	}

	got := formatStateAge(m, now)
	if !strings.Contains(got, "HZ 1m35s") {
		t.Fatalf("expected HZ age to use sample time, got %q", got)
	}
}

func TestEffectiveEarthDistPrefersDSNRange(t *testing.T) {
	m := Model{
		hzState:   &horizons.State{EarthDist: 65000},
		dsnStatus: &dsn.Status{Range: 62000},
	}

	if got := effectiveEarthDist(m); got != 62000 {
		t.Fatalf("effectiveEarthDist() = %v, want 62000", got)
	}
}

func TestFormatCompactDist(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{950, "950 km"},
		{65000, "65k km"},
		{1336000, "1.3M km"},
	}

	for _, tc := range cases {
		if got := formatCompactDist(tc.in); got != tc.want {
			t.Fatalf("formatCompactDist(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
