package mission

import (
	"testing"
	"time"
)

func TestMissionDayAtClampsToMissionSpan(t *testing.T) {
	if got := MissionDayAt(-6 * time.Hour); got != 1 {
		t.Fatalf("MissionDayAt(prelaunch) = %d, want 1", got)
	}

	wantLastDay := TotalMissionDays()
	if got := MissionDayAt(TotalDuration() + 48*time.Hour); got != wantLastDay {
		t.Fatalf("MissionDayAt(post-mission) = %d, want %d", got, wantLastDay)
	}
}

func TestTotalMissionDaysMatchesTimelineDuration(t *testing.T) {
	if got := TotalMissionDays(); got != 10 {
		t.Fatalf("TotalMissionDays() = %d, want 10", got)
	}
}

func TestMissionProgressAtTracksTimelineSpan(t *testing.T) {
	if got := MissionProgressAt(-6 * time.Hour); got != 0 {
		t.Fatalf("MissionProgressAt(prelaunch) = %v, want 0", got)
	}

	if got := MissionProgressAt(TotalDuration() / 2); got != 0.5 {
		t.Fatalf("MissionProgressAt(halfway) = %v, want 0.5", got)
	}

	if got := MissionProgressAt(TotalDuration() + 48*time.Hour); got != 1 {
		t.Fatalf("MissionProgressAt(post-mission) = %v, want 1", got)
	}
}

func TestTimelineUsesUpdatedNASAOverviewMilestones(t *testing.T) {
	tests := map[string]time.Duration{
		"TLI Burn":              d(1, 1, 37),
		"Lunar SOI Entry":       d(4, 6, 59),
		"Closest Moon Approach": ds(5, 1, 23, 20),
		"Max Earth Distance":    ds(5, 1, 26, 57),
		"RTC-1":                 d(6, 4, 23),
		"RTC-2":                 d(8, 4, 33),
		"RTC-3":                 d(8, 20, 33),
		"Splashdown":            d(9, 1, 46),
	}

	for label, want := range tests {
		found := false
		for _, event := range Timeline {
			if event.Label != label {
				continue
			}
			found = true
			if event.METOffset != want {
				t.Fatalf("%s offset = %v, want %v", label, event.METOffset, want)
			}
		}
		if !found {
			t.Fatalf("timeline missing %q", label)
		}
	}
}

func TestCrewActivitySchedulePrefersMealAfterOTC1(t *testing.T) {
	met := d(2, 0, 6)

	current := CurrentCrewActivity(met)
	if current == nil || current.Label != "OTC-1" {
		t.Fatalf("CurrentCrewActivity(%v) = %+v, want OTC-1", met, current)
	}

	next := NextCrewActivity(met)
	if next == nil || next.Label != "Meal" {
		t.Fatalf("NextCrewActivity(%v) = %+v, want Meal", met, next)
	}
}
