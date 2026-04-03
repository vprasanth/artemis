package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"artemis/internal/dsn"
	"artemis/internal/horizons"
	"artemis/internal/mission"
	"artemis/internal/nasablog"
	"artemis/internal/spaceweather"
)

func TestRenderInstrumentsCompactHeightKeepsPrimaryTelemetry(t *testing.T) {
	m := Model{
		hzState: &horizons.State{
			Position:     horizons.Vector3{X: 1200, Y: -450, Z: 0},
			MoonPosition: horizons.Vector3{X: 384400, Y: 0, Z: 0},
			EarthDist:    1250,
			MoonDist:     383150,
			Speed:        2.75,
		},
		speedHistory: []float64{2.60, 2.68, 2.75},
		dsnStatus: &dsn.Status{
			RTLT: 0.3,
			Dishes: []dsn.Dish{
				{
					Name:    "DSS43",
					Station: "Canberra, AU",
					DownSignals: []dsn.Signal{
						{Active: true, DataRate: 2_000_000},
					},
				},
			},
		},
	}

	got := renderInstruments(m, 100, 6)
	if !strings.Contains(got, "VELOCITY") {
		t.Fatalf("expected compact instruments to keep velocity gauge, got:\n%s", got)
	}
	if !strings.Contains(got, "RANGE") {
		t.Fatalf("expected compact instruments to keep range finder, got:\n%s", got)
	}
}

func TestRenderInstrumentsShowsScopeExplanationsAndDerivedMetrics(t *testing.T) {
	m := Model{
		hzState: &horizons.State{
			Position:     horizons.Vector3{X: 21000, Y: 8000, Z: -500},
			Velocity:     horizons.Vector3{X: 2.1, Y: 0.5, Z: -0.1},
			MoonPosition: horizons.Vector3{X: 364000, Y: 1000, Z: 0},
			EarthDist:    22500,
			MoonDist:     342000,
			Speed:        2.16,
		},
		speedHistory:    []float64{2.01, 2.08, 2.16},
		radialHistory:   []float64{1.90, 2.02, 2.14},
		dsnRangeHistory: []float64{21000, 21800, 22500},
		rtltHistory:     []float64{0.2, 0.21, 0.22},
		dsnRateHistory:  []float64{1_200_000, 1_600_000, 2_100_000},
	}

	got := renderInstruments(m, 120, 30)
	for _, want := range []string{
		"radial",
		"radial trend",
		"vx ",
		"split E",
		"E-M baseline",
		"dsn range trend",
		"RTLT trend",
		"downlink trend",
		"Earth-centered heading in the ecliptic plane",
		"Moon relative to Orion; center = spacecraft",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected instruments view to include %q, got:\n%s", want, got)
		}
	}
}

func TestRenderTopRowUsesSharedHeight(t *testing.T) {
	m := Model{
		hzState: &horizons.State{
			Position:     horizons.Vector3{X: 12310, Y: -45027, Z: 3771},
			Velocity:     horizons.Vector3{X: 0.4, Y: 2.6, Z: -0.1},
			MoonPosition: horizons.Vector3{X: 384400, Y: 0, Z: 0},
			EarthDist:    37900,
			MoonDist:     339579,
			Speed:        2.726,
			Timestamp:    time.Now().Add(-8 * time.Second),
		},
		dsnStatus: &dsn.Status{
			Range:     37900,
			RTLT:      0.25,
			Timestamp: time.Now().Add(-3 * time.Second),
		},
	}

	w := 100
	clockW, spacecraftW := splitWidthEvenly(w)
	clockBase := renderClockPanel(clockW, 0)
	spacecraftBase := renderSpacecraftPanel(m, spacecraftW, 0)

	wantHeight := measureHeight(clockBase)
	if h := measureHeight(spacecraftBase); h > wantHeight {
		wantHeight = h
	}

	got := renderTopRow(m, w)
	if measureHeight(got) != wantHeight {
		t.Fatalf("top row height = %d, want %d", measureHeight(got), wantHeight)
	}
}

func TestRenderClockPanelUsesDerivedMissionDayTotal(t *testing.T) {
	met := mission.TotalDuration() + 6*time.Hour
	got := renderClockPanelAt(60, 0, met)
	want := fmt.Sprintf("%d / %d", mission.TotalMissionDays(), mission.TotalMissionDays())
	if !strings.Contains(got, want) {
		t.Fatalf("expected clock panel to show derived mission day total %q, got:\n%s", want, got)
	}
}

func TestRenderSpacecraftPanelShowsDerivedTelemetry(t *testing.T) {
	now := time.Now()
	m := Model{
		hzState: &horizons.State{
			Position:     horizons.Vector3{X: 1000, Y: 1000, Z: 1000},
			Velocity:     horizons.Vector3{X: 0.4, Y: 0.3, Z: 0.2},
			MoonPosition: horizons.Vector3{X: 384400, Y: 0, Z: 0},
			EarthDist:    1732,
			MoonDist:     382668,
			Speed:        0.539,
			Timestamp:    now.Add(-8 * time.Second),
		},
		dsnStatus: &dsn.Status{
			RTLT:      0.25,
			Timestamp: now.Add(-3 * time.Second),
		},
	}

	got := renderSpacecraftPanel(m, 60, 0)
	for _, want := range []string{"Earth Rate:", "Ecl Lon/Lat:", "Data Age:", "HZ 8s", "DSN 3s"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected spacecraft panel to include %q, got:\n%s", want, got)
		}
	}
}

func TestRenderSpacecraftPanelSupportsImperialUnits(t *testing.T) {
	now := time.Now()
	m := Model{
		units: unitImperial,
		hzState: &horizons.State{
			Position:     horizons.Vector3{X: 1000, Y: -2000, Z: 500},
			Velocity:     horizons.Vector3{X: 0.4, Y: 0.3, Z: 0.2},
			MoonPosition: horizons.Vector3{X: 384400, Y: 0, Z: 0},
			EarthDist:    1732,
			MoonDist:     382668,
			Speed:        0.539,
			Timestamp:    now.Add(-8 * time.Second),
		},
		dsnStatus: &dsn.Status{
			Range:     1732,
			RTLT:      0.25,
			Timestamp: now.Add(-3 * time.Second),
		},
	}

	got := renderSpacecraftPanel(m, 70, 0)
	for _, want := range []string{"mi/s", "mph", " mi", "X:621  Y:-1243  Z:311 mi"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected imperial spacecraft panel to include %q, got:\n%s", want, got)
		}
	}
}

func TestRadialVelocity(t *testing.T) {
	got, ok := radialVelocity(
		horizons.Vector3{X: 1000, Y: 0, Z: 0},
		horizons.Vector3{X: -2.5, Y: 1, Z: 0},
	)
	if !ok {
		t.Fatal("expected radial velocity to be computable")
	}
	if got != -2.5 {
		t.Fatalf("radialVelocity() = %v, want -2.5", got)
	}
}

func TestEclipticCoords(t *testing.T) {
	lon, lat, ok := eclipticCoords(horizons.Vector3{X: 1, Y: 1, Z: 1})
	if !ok {
		t.Fatal("expected ecliptic coordinates to be computable")
	}
	if lon < 44.9 || lon > 45.1 {
		t.Fatalf("longitude = %v, want about 45", lon)
	}
	if lat < 35.2 || lat > 35.3 {
		t.Fatalf("latitude = %v, want about 35.26", lat)
	}
}

func TestDayAxisMarksUseMissionEnd(t *testing.T) {
	marks := dayAxisMarks(mission.TotalDuration())
	if len(marks) == 0 {
		t.Fatal("expected day axis marks")
	}

	last := marks[len(marks)-1]
	if last.label != "E" {
		t.Fatalf("expected final day axis mark to label mission end, got %#v", last)
	}
	if last.offset != mission.TotalDuration() {
		t.Fatalf("expected final day axis mark at total mission duration, got %v want %v", last.offset, mission.TotalDuration())
	}

	for _, mark := range marks {
		if mark.label == "10" {
			t.Fatalf("did not expect synthetic day-10 axis mark for mission duration %v", mission.TotalDuration())
		}
	}
}

func TestFormatDataAge(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{8 * time.Second, "8s"},
		{2*time.Minute + 5*time.Second, "2m05s"},
		{90*time.Minute + 4*time.Second, "1h30m"},
	}

	for _, tc := range cases {
		if got := formatDataAge(tc.in); got != tc.want {
			t.Fatalf("formatDataAge(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRenderFooterOmitsViewShortcutWhenTrajectoryHidden(t *testing.T) {
	m := Model{
		width:  72,
		height: 24,
		layout: map[panelID]panelLayout{
			panelTrajectory: {visible: false},
		},
	}

	for _, width := range []int{16, 72, 200} {
		got := renderFooter(m, width)
		if strings.Contains(got, "v view") {
			t.Fatalf("expected footer width %d to omit view shortcut when visualization is hidden, got %q", width, got)
		}
	}
}

func TestRenderFooterIncludesNotificationShortcut(t *testing.T) {
	m := Model{
		width:                72,
		height:               24,
		notificationsEnabled: true,
		layout: map[panelID]panelLayout{
			panelTrajectory: {visible: true},
		},
	}

	checks := []struct {
		width int
		want  []string
	}{
		{200, []string{"n notify(on)", "n ntfy(on)", "n(on)"}},
		{96, []string{"n(on)", "n ntfy(on)"}},
	}

	for _, tc := range checks {
		got := renderFooter(m, tc.width)
		matched := false
		for _, want := range tc.want {
			if strings.Contains(got, want) {
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("expected footer width %d to include one of %q, got %q", tc.width, tc.want, got)
		}
	}
}

func TestRenderFooterIncludesUnitsShortcut(t *testing.T) {
	m := Model{
		width:  120,
		height: 24,
		units:  unitImperial,
		layout: map[panelID]panelLayout{
			panelTrajectory: {visible: true},
		},
	}

	got := renderFooter(m, 120)
	if !strings.Contains(got, "u imperial") && !strings.Contains(got, "u(imp)") {
		t.Fatalf("expected footer to include units shortcut, got %q", got)
	}
}

func TestRenderFooterIncludesScreenProtectionShortcut(t *testing.T) {
	m := Model{
		width:             160,
		height:            24,
		screenProtectMode: screenProtectDriftIdle,
		layout: map[panelID]panelLayout{
			panelTrajectory: {visible: true},
		},
	}

	got := renderFooter(m, 160)
	if !strings.Contains(got, "p guard(drift+idle)") && !strings.Contains(got, "p d+i") && !strings.Contains(got, " |  p  | ") {
		t.Fatalf("expected footer to include screen protection shortcut, got %q", got)
	}
}

func TestRenderFooterIncludesVisualEffectsShortcutState(t *testing.T) {
	m := Model{
		width:         120,
		height:        24,
		visualEffects: effectsStarsSprite,
		layout: map[panelID]panelLayout{
			panelTrajectory: {visible: true},
		},
	}

	got := renderFooter(m, 120)
	if !strings.Contains(got, "s ship") && !strings.Contains(got, "s fx(ship)") && !strings.Contains(got, " |  s  | ") {
		t.Fatalf("expected footer to include visual effects shortcut state, got %q", got)
	}
}

func TestRenderFooterIncludesFullscreenShortcut(t *testing.T) {
	m := Model{
		width:  120,
		height: 24,
		layout: map[panelID]panelLayout{
			panelTrajectory: {visible: true},
		},
	}

	if got := renderFooter(m, 120); !strings.Contains(got, "f full") && !strings.Contains(got, " |  f  | ") {
		t.Fatalf("expected footer to include fullscreen shortcut, got %q", got)
	}

	m.visualizationFullscreen = true
	if got := renderFooter(m, 120); !strings.Contains(got, "f win") && !strings.Contains(got, " |  f  | ") {
		t.Fatalf("expected footer to include windowed shortcut in fullscreen mode, got %q", got)
	}
}

func TestRenderFooterShowsDebugShortcutOnlyWhenEnabled(t *testing.T) {
	disabled := Model{
		width:                120,
		height:               24,
		notificationsEnabled: true,
		layout: map[panelID]panelLayout{
			panelTrajectory: {visible: true},
		},
	}
	if got := renderFooter(disabled, 120); strings.Contains(got, "N test-notify") {
		t.Fatalf("expected footer without debug mode to omit debug shortcut, got %q", got)
	}

	enabled := disabled
	enabled.debugKeysEnabled = true
	if got := renderFooter(enabled, 120); !strings.Contains(got, "N test") && !strings.Contains(got, " |  N  | ") {
		t.Fatalf("expected footer with debug mode to include debug shortcut, got %q", got)
	}
}

func TestRenderFooterShowsNotificationFailure(t *testing.T) {
	m := Model{
		width:               140,
		height:              24,
		startedAt:           time.Now().Add(-10 * time.Minute),
		notificationError:   "notify failed",
		notificationErrorAt: time.Now(),
		layout: map[panelID]panelLayout{
			panelTrajectory: {visible: true},
		},
	}

	got := renderFooter(m, 140)
	if !strings.Contains(got, "notify failed") {
		t.Fatalf("expected footer to show notification failure, got %q", got)
	}
}

func TestFormatFooterUptime(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{0, "00:00:00"},
		{2*time.Hour + 3*time.Minute + 4*time.Second, "02:03:04"},
		{27*time.Hour + 15*time.Minute, "1d03h"},
	}

	for _, tc := range cases {
		if got := formatFooterUptime(tc.in); got != tc.want {
			t.Fatalf("formatFooterUptime(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRenderFooterShowsUptime(t *testing.T) {
	m := Model{
		width:     160,
		height:    24,
		startedAt: time.Now().Add(-(2*time.Hour + 3*time.Minute + 4*time.Second)),
		layout: map[panelID]panelLayout{
			panelTrajectory: {visible: true},
		},
	}

	got := renderFooter(m, 160)
	if !strings.Contains(got, "02:03:04") {
		t.Fatalf("expected footer to show uptime, got %q", got)
	}
}

func TestHiddenPanelSummaryOmitsFullscreenMode(t *testing.T) {
	m := Model{
		visualizationFullscreen: true,
		layout: map[panelID]panelLayout{
			panelTrajectory: {visible: true},
			panelTimeline:   {visible: false},
			panelDSN:        {visible: false},
		},
	}

	if got := hiddenPanelSummary(m, false); got != "" {
		t.Fatalf("expected fullscreen hidden summary to be suppressed, got %q", got)
	}
}

func TestRenderVisualizationPanelFullscreenEmbedsTopRow(t *testing.T) {
	now := time.Now()
	m := Model{
		trajectoryView: 0,
		hzState: &horizons.State{
			Position:     horizons.Vector3{X: 1000, Y: 1000, Z: 1000},
			Velocity:     horizons.Vector3{X: 0.4, Y: 0.3, Z: 0.2},
			MoonPosition: horizons.Vector3{X: 384400, Y: 0, Z: 0},
			EarthDist:    1732,
			MoonDist:     382668,
			Speed:        0.539,
			Timestamp:    now.Add(-8 * time.Second),
		},
		dsnStatus: &dsn.Status{
			RTLT:      0.25,
			Timestamp: now.Add(-3 * time.Second),
		},
	}

	got := renderVisualizationPanel(m, 120, 10, true)
	for _, want := range []string{"MISSION CLOCK", "SPACECRAFT STATE", "f: windowed"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected fullscreen visualization to include %q, got:\n%s", want, got)
		}
	}
}

func TestRenderVisualizationPanelSupportsNewViews(t *testing.T) {
	m := Model{
		dsnStatus: &dsn.Status{
			Dishes: []dsn.Dish{{Name: "DSS43", Azimuth: 120, Elevation: 45, Station: "Canberra, AU"}},
		},
		swStatus: &spaceweather.Status{
			Kp:              4,
			Bz:              -5.2,
			Bt:              8.1,
			WindSpeed:       520,
			WindDensity:     7.4,
			WindTemp:        110000,
			ProtonFlux10MeV: 1.2,
			LatestAlert:     "ALERT: Geomagnetic activity observed",
		},
		kpHistory:          []float64{2, 3, 4},
		bzHistory:          []float64{-2, -3.5, -5.2},
		windSpeedHistory:   []float64{480, 500, 520},
		protonFluxHistory:  []float64{0.4, 0.8, 1.2},
		windDensityHistory: []float64{5.2, 6.1, 7.4},
	}

	m.trajectoryView = 3
	if got := renderVisualizationPanel(m, 100, 16, false); !strings.Contains(got, "DSN SKY") {
		t.Fatalf("expected DSN SKY view title, got:\n%s", got)
	}

	m.trajectoryView = 4
	if got := renderVisualizationPanel(m, 100, 16, false); !strings.Contains(got, "WEATHER TRENDS") {
		t.Fatalf("expected WEATHER TRENDS view title, got:\n%s", got)
	}
}

func TestRenderVisualizationPanelFullscreenFillsRequestedHeightForOpsViews(t *testing.T) {
	m := Model{
		dsnStatus: &dsn.Status{
			Dishes: []dsn.Dish{{Name: "DSS43", Azimuth: 120, Elevation: 45, Station: "Canberra, AU"}},
		},
		swStatus: &spaceweather.Status{
			Kp:              4,
			Bz:              -5.2,
			Bt:              8.1,
			WindSpeed:       520,
			WindDensity:     7.4,
			WindTemp:        110000,
			ProtonFlux10MeV: 1.2,
		},
		kpHistory:         []float64{2, 3, 4},
		bzHistory:         []float64{-2, -3.5, -5.2},
		windSpeedHistory:  []float64{480, 500, 520},
		protonFluxHistory: []float64{0.4, 0.8, 1.2},
		hzState: &horizons.State{
			Position:     horizons.Vector3{X: 1000, Y: 1000, Z: 1000},
			Velocity:     horizons.Vector3{X: 0.4, Y: 0.3, Z: 0.2},
			MoonPosition: horizons.Vector3{X: 384400, Y: 0, Z: 0},
			EarthDist:    1732,
			MoonDist:     382668,
			Speed:        0.539,
			Timestamp:    time.Now().Add(-8 * time.Second),
		},
	}

	const (
		panelWidth = 100
		plotHeight = 16
	)

	topRowHeight := measureHeight(renderTopRow(m, innerWidthFor(panelStyle, panelWidth)))
	wantHeight := plotHeight + topRowHeight + 1 + panelStyle.GetVerticalBorderSize()

	for _, view := range []int{3, 4} {
		m.trajectoryView = view
		if got := measureHeight(renderVisualizationPanel(m, panelWidth, plotHeight, true)); got != wantHeight {
			t.Fatalf("fullscreen view %d height = %d, want %d", view, got, wantHeight)
		}
	}
}

func TestRenderCachedTrajectoryPanelFillsFullscreenHeightForOpsViews(t *testing.T) {
	m := Model{
		width:                   100,
		height:                  40,
		visualizationFullscreen: true,
		dsnStatus: &dsn.Status{
			Dishes: []dsn.Dish{{Name: "DSS43", Azimuth: 120, Elevation: 45, Station: "Canberra, AU"}},
		},
		swStatus: &spaceweather.Status{
			Kp:              4,
			Bz:              -5.2,
			Bt:              8.1,
			WindSpeed:       520,
			WindDensity:     7.4,
			WindTemp:        110000,
			ProtonFlux10MeV: 1.2,
		},
		kpHistory:         []float64{2, 3, 4},
		bzHistory:         []float64{-2, -3.5, -5.2},
		windSpeedHistory:  []float64{480, 500, 520},
		protonFluxHistory: []float64{0.4, 0.8, 1.2},
		hzState: &horizons.State{
			Position:     horizons.Vector3{X: 1000, Y: 1000, Z: 1000},
			Velocity:     horizons.Vector3{X: 0.4, Y: 0.3, Z: 0.2},
			MoonPosition: horizons.Vector3{X: 384400, Y: 0, Z: 0},
			EarthDist:    1732,
			MoonDist:     382668,
			Speed:        0.539,
			Timestamp:    time.Now().Add(-8 * time.Second),
		},
	}

	for _, view := range []int{3, 4} {
		m.trajectoryView = view
		if got := measureHeight(m.renderCachedTrajectoryPanel(24)); got != 24 {
			t.Fatalf("cached fullscreen view %d height = %d, want 24", view, got)
		}
	}
}

func TestRenderMissionLogReaderUsesExcerptFallback(t *testing.T) {
	m := Model{
		width:          100,
		height:         30,
		blogReaderOpen: true,
		blogStatus: &nasablog.Status{
			Entries: []nasablog.Entry{{
				ID:      7,
				Title:   "Crew Completes TLI Burn",
				Excerpt: "Orion completed the translunar injection burn and configured for coast.",
			}},
		},
		blogPostCache: make(map[int]*nasablog.Post),
	}

	got := renderMissionLogReader(m, 100, 20)
	for _, want := range []string{"MISSION LOG READER", "Crew Completes TLI Burn", "translunar injection burn"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected mission log reader to include %q, got:\n%s", want, got)
		}
	}
}

func TestBuildCacheHidesTopRowInFullscreen(t *testing.T) {
	m := Model{
		width:                   120,
		height:                  40,
		visualizationFullscreen: true,
	}

	m.buildCache()

	if m.layout[panelTopRow].visible {
		t.Fatalf("expected top row to be hidden in fullscreen mode")
	}
	if !m.layout[panelTrajectory].visible {
		t.Fatalf("expected trajectory panel to remain visible in fullscreen mode")
	}
}

func TestShiftScreenFrameAppliesRightAndDownOffsets(t *testing.T) {
	got := shiftScreenFrame("ABCD\nEFGH\nIJKL", 4, 3, 1, 1)
	want := "    \n ABC\n EFG"
	if got != want {
		t.Fatalf("shiftScreenFrame() = %q, want %q", got, want)
	}
}

func TestViewRendersIdleScreenProtectionScreen(t *testing.T) {
	now := time.Now()
	m := Model{
		width:                  80,
		height:                 24,
		screenProtectMode:      screenProtectDriftIdle,
		screenProtectIdleAfter: time.Minute,
		lastActivityAt:         now.Add(-2 * time.Minute),
		screenProtectNow:       now,
		layout: map[panelID]panelLayout{
			panelHeader: {visible: true, width: 80},
		},
	}

	got := m.View()
	for _, want := range []string{"screen protection active", "press any key to wake"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected idle screen to include %q, got:\n%s", want, got)
		}
	}
}
