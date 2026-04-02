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

func TestRenderFooterKeepsViewShortcutWhenTrajectoryHidden(t *testing.T) {
	m := Model{
		width:  72,
		height: 24,
		layout: map[panelID]panelLayout{
			panelTrajectory: {visible: false},
		},
	}

	got := renderFooter(m, 16)
	if !strings.Contains(got, "v") {
		t.Fatalf("expected narrow footer to keep view shortcut when visualization is hidden, got %q", got)
	}
}
