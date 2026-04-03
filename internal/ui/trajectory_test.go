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

func TestBuildTrajectoryFrameFitsPathBounds(t *testing.T) {
	path := []horizons.Vector3{
		{X: 0, Y: 0},
		{X: 180000, Y: 70000},
		{X: 360000, Y: -40000},
	}
	state := &horizons.State{
		Position:     horizons.Vector3{X: 210000, Y: 30000},
		MoonPosition: horizons.Vector3{X: -140000, Y: 20000},
	}

	frame := buildTrajectoryFrame(state, path, 80, 20)
	for _, pos := range path {
		point := frame.project(pos)
		if point.x < 0 || point.x >= 80 || point.y < 0 || point.y >= 20 {
			t.Fatalf("projected point %+v = (%d,%d), want inside viewport", pos, point.x, point.y)
		}
	}
}

func TestMoonRelativeVectorPointsFromSpacecraftToMoon(t *testing.T) {
	got := moonRelativeVector(horizons.Vector3{X: 12, Y: -5, Z: 2})
	want := horizons.Vector3{X: -12, Y: 5, Z: -2}
	if got != want {
		t.Fatalf("moonRelativeVector() = %+v, want %+v", got, want)
	}
}

func TestPlotPathDrawsVisibleSegmentWhenEndpointsProjectOffscreen(t *testing.T) {
	canvas := make([][]string, 6)
	for i := range canvas {
		canvas[i] = make([]string, 12)
		for j := range canvas[i] {
			canvas[i][j] = " "
		}
	}

	frame := trajectoryFrame{
		centerX:      6,
		centerY:      3,
		worldCenterX: 300000,
		worldCenterY: 0,
		scale:        25000,
		aspect:       0.5,
	}
	path := []horizons.Vector3{
		{X: 0, Y: 0},
		{X: 600000, Y: 0},
	}

	plotPath(canvas, frame, path, 12, 6)

	found := false
	for _, row := range canvas {
		for _, cell := range row {
			if cell != " " {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("expected visible path segment inside viewport")
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
		name  string
		units unitSystem
		in    float64
		want  string
	}{
		{"metric-short", unitMetric, 950, "950 km"},
		{"metric-kilo", unitMetric, 65000, "65k km"},
		{"metric-mega", unitMetric, 1336000, "1.3M km"},
		{"imperial-kilo", unitImperial, 65000, "40k mi"},
	}

	for _, tc := range cases {
		if got := formatCompactDist(tc.in, tc.units); got != tc.want {
			t.Fatalf("%s: formatCompactDist(%v) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestTrajectoryPositionsPrefersMissionArc(t *testing.T) {
	arc := []horizons.Vector3{{X: 10}, {X: 20}, {X: 30}}
	m := Model{
		trajectoryPath: arc,
		positionTrail:  []horizons.Vector3{{X: 1}, {X: 2}},
		hzState:        &horizons.State{Position: horizons.Vector3{X: 3}},
	}

	got := trajectoryPositions(m)
	if len(got) != len(arc) {
		t.Fatalf("len(trajectoryPositions()) = %d, want %d", len(got), len(arc))
	}
	if got[0].X != 10 || got[len(got)-1].X != 30 {
		t.Fatalf("trajectoryPositions() = %+v, want mission arc", got)
	}
}

func TestTrajectoryPathStatus(t *testing.T) {
	tests := []struct {
		name string
		m    Model
		want string
	}{
		{
			name: "sampled arc",
			m: Model{
				trajectoryPath: []horizons.Vector3{{X: 1}, {X: 2}},
			},
			want: "arc 2 samples",
		},
		{
			name: "loading",
			m: Model{
				hzPathLoading: true,
			},
			want: "arc loading",
		},
		{
			name: "unavailable",
			m: Model{
				hzPathErr: assertErr("boom"),
			},
			want: "arc unavailable",
		},
		{
			name: "live trail fallback",
			m: Model{
				positionTrail: []horizons.Vector3{{X: 1}},
				hzState:       &horizons.State{Position: horizons.Vector3{X: 2}},
			},
			want: "arc live trail",
		},
		{
			name: "waiting",
			m:    Model{},
			want: "arc waiting",
		},
	}

	for _, tc := range tests {
		if got := trajectoryPathStatus(tc.m); got != tc.want {
			t.Fatalf("%s: trajectoryPathStatus() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestRenderTrajectoryShowsArcStatus(t *testing.T) {
	m := Model{
		hzPathErr: assertErr("path fetch failed"),
	}

	got := renderTrajectory(m, 60, 12)
	if !strings.Contains(got, "arc unavailable") {
		t.Fatalf("expected trajectory render to show arc status, got:\n%s", got)
	}
}

func TestTrajectoryLegendUsesClearerDirectionLabels(t *testing.T) {
	canvas := make([][]string, 10)
	for i := range canvas {
		canvas[i] = make([]string, 20)
		for j := range canvas[i] {
			canvas[i][j] = " "
		}
	}

	placeLegend(canvas, 20, 10)

	var rows []string
	for _, row := range canvas {
		rows = append(rows, strings.Join(row, ""))
	}
	got := strings.Join(rows, "\n")
	if !strings.Contains(got, "away") || !strings.Contains(got, "return") {
		t.Fatalf("expected legend to include clearer direction labels, got:\n%s", got)
	}
}

func TestRayEdgePointProjectsToViewportBoundary(t *testing.T) {
	edge, ok := rayEdgePoint(pathPoint{x: 10, y: 5}, pathPoint{x: 30, y: 2}, 40, 20)
	if !ok {
		t.Fatal("expected ray to intersect viewport boundary")
	}
	if edge.x < 1 || edge.x > 38 || edge.y < 1 || edge.y > 17 {
		t.Fatalf("edge point %+v outside expected viewport bounds", edge)
	}
}

func TestApproximateSunVectorIsNormalized(t *testing.T) {
	v := approximateSunVector(time.Date(2026, time.April, 3, 0, 0, 0, 0, time.UTC))
	mag := math.Hypot(v.X, v.Y)
	if mag < 0.999 || mag > 1.001 {
		t.Fatalf("approximateSunVector magnitude = %v, want about 1", mag)
	}
}

func TestPlaceSpacecraftUsesSingleCellGlyph(t *testing.T) {
	canvas := [][]string{{" ", " ", " ", " ", " "}}

	placeSpacecraft(canvas, 2, 0, 5, 1, 0, false)

	if canvas[0][2] == " " {
		t.Fatalf("expected spacecraft glyph at center cell, got row=%q", strings.Join(canvas[0], ""))
	}
	if canvas[0][1] != " " || canvas[0][3] != " " {
		t.Fatalf("expected single-cell spacecraft glyph, got row=%q", strings.Join(canvas[0], ""))
	}
}

func TestSegmentGlyph(t *testing.T) {
	cases := []struct {
		x0, y0 int
		x1, y1 int
		want   string
	}{
		{0, 0, 3, 0, "─"},
		{0, 0, 0, 3, "│"},
		{0, 0, 3, 3, "╲"},
		{0, 3, 3, 0, "╱"},
	}

	for _, tc := range cases {
		if got := segmentGlyph(tc.x0, tc.y0, tc.x1, tc.y1, "·"); got != tc.want {
			t.Fatalf("segmentGlyph((%d,%d)->(%d,%d)) = %q, want %q", tc.x0, tc.y0, tc.x1, tc.y1, got, tc.want)
		}
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
