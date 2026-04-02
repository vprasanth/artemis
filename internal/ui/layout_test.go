package ui

import (
	"strings"
	"testing"
	"time"

	"artemis/internal/dsn"
	"artemis/internal/horizons"
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

func TestRenderTopRowUsesSharedHeight(t *testing.T) {
	m := Model{
		hzState: &horizons.State{
			Position:     horizons.Vector3{X: 12310, Y: -45027, Z: 3771},
			MoonPosition: horizons.Vector3{X: 384400, Y: 0, Z: 0},
			EarthDist:    37900,
			MoonDist:     339579,
			Speed:        2.726,
		},
		dsnStatus: &dsn.Status{
			Range:     37900,
			RTLT:      0.25,
			Timestamp: time.Now(),
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
		want  string
	}{
		{200, "n notify(on)"},
		{72, "n(on)"},
	}

	for _, tc := range checks {
		got := renderFooter(m, tc.width)
		if !strings.Contains(got, tc.want) {
			t.Fatalf("expected footer width %d to include %q, got %q", tc.width, tc.want, got)
		}
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
	if got := renderFooter(enabled, 120); !strings.Contains(got, "N test") {
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
	if !strings.Contains(got, "up 02:03:04") {
		t.Fatalf("expected footer to show uptime, got %q", got)
	}
}
