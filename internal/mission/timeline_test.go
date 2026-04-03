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
