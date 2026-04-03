package mission

import (
	"fmt"
	"time"
)

var LaunchTime = time.Date(2026, time.April, 1, 22, 35, 12, 0, time.UTC)

type CrewMember struct {
	Role   string
	Name   string
	Agency string
}

var Crew = []CrewMember{
	{Role: "CDR", Name: "Reid Wiseman", Agency: "NASA"},
	{Role: "PLT", Name: "Victor Glover", Agency: "NASA"},
	{Role: "MS1", Name: "Christina Koch", Agency: "NASA"},
	{Role: "MS2", Name: "Jeremy Hansen", Agency: "CSA"},
}

type EventStatus int

const (
	EventPending EventStatus = iota
	EventActive
	EventCompleted
)

type Event struct {
	METOffset time.Duration
	Label     string
	Detail    string
}

var Timeline = []Event{
	{d(0, 0, 0), "Launch", "Liftoff from LC-39B, Kennedy Space Center"},
	{d(0, 0, 20), "Solar Array Deploy", "Orion deploys solar arrays in Earth orbit"},
	{d(0, 0, 50), "Perigee Raise", "Perigee raise maneuver (2223 x 185 km)"},
	{d(0, 1, 48), "Apogee Raise", "Apogee raise maneuver (70377 km)"},
	{d(0, 3, 24), "ICPS Separation", "Orion/ICPS separation"},
	{d(0, 4, 50), "Upper Stage Sep Burn", "Orion upper stage separation burn"},
	{d(0, 5, 4), "CubeSat Deploy", "4 CubeSats deploy at one minute intervals"},
	{d(0, 12, 55), "Perigee Raise Burn", "Perigee raise burn"},
	{d(1, 1, 8), "TLI Burn", "Translunar Injection burn (~8 min)"},
	{d(1, 1, 35), "Earth Shadow Entry", "Enter Earth shadow"},
	{d(1, 2, 41), "Earth Shadow Exit", "Exit Earth shadow"},
	{d(2, 0, 8), "Correction Burn #1", "Orbit trajectory correction burn #1"},
	{d(3, 1, 8), "Correction Burn #2", "Orbit trajectory correction burn #2"},
	{d(4, 4, 29), "Correction Burn #3", "Orbit trajectory correction burn #3"},
	{d(4, 6, 8), "Lunar SOI Entry", "Orion enters lunar sphere of influence"},
	{d(5, 0, 31), "Closest Moon Approach", "Closest approach to the Moon"},
	{d(5, 0, 34), "Max Earth Distance", "Maximum distance from Earth"},
	{d(5, 18, 52), "Lunar SOI Exit", "Orion exits lunar sphere of influence"},
	{d(6, 1, 29), "Return Correction #1", "Return trajectory correction burn #1"},
	{d(7, 4, 20), "Manual Piloting Demo", "Manual piloting demonstration"},
	{d(8, 4, 29), "Return Correction #2", "Return trajectory correction burn #2"},
	{d(8, 20, 29), "Return Correction #3", "Return trajectory correction burn #3"},
	{d(9, 1, 9), "Module Separation", "Orion crew and service module separation"},
	{d(9, 1, 29), "Entry Interface", "Entry interface (122 km above Earth)"},
	{d(9, 1, 42), "Splashdown", "Splashdown in Pacific Ocean near Baja California"},
}

func d(days, hours, minutes int) time.Duration {
	return time.Duration(days)*24*time.Hour +
		time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute
}

func MET() time.Duration {
	return time.Since(LaunchTime)
}

func MissionDay() int {
	return MissionDayAt(MET())
}

func MissionDayAt(met time.Duration) int {
	if met <= 0 {
		return 1
	}

	day := int(met.Hours()/24) + 1
	total := TotalMissionDays()
	if day > total {
		return total
	}
	return day
}

func TotalDuration() time.Duration {
	return Timeline[len(Timeline)-1].METOffset
}

func TotalMissionDays() int {
	return int(TotalDuration().Hours()/24) + 1
}

func FormatMET(met time.Duration) string {
	total := int(met.Seconds())
	if total < 0 {
		return "T-" + formatDuration(-total)
	}
	return "T+" + formatDuration(total)
}

func formatDuration(totalSec int) string {
	days := totalSec / 86400
	remainder := totalSec % 86400
	hours := remainder / 3600
	remainder = remainder % 3600
	minutes := remainder / 60
	seconds := remainder % 60
	return fmt.Sprintf("%03d:%02d:%02d:%02d", days, hours, minutes, seconds)
}

func FormatCountdown(d time.Duration) string {
	total := int(d.Seconds())
	if total < 0 {
		return "PASSED"
	}
	hours := total / 3600
	remainder := total % 3600
	minutes := remainder / 60
	seconds := remainder % 60
	if hours > 24 {
		days := hours / 24
		hours = hours % 24
		return fmt.Sprintf("%dd %02dh %02dm", days, hours, minutes)
	}
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func EventStatusAt(e Event, met time.Duration) EventStatus {
	if met >= e.METOffset {
		return EventCompleted
	}
	return EventPending
}

func CurrentEventIndex(met time.Duration) int {
	last := 0
	for i, e := range Timeline {
		if met >= e.METOffset {
			last = i
		}
	}
	return last
}

func NextEvent(met time.Duration) *Event {
	for i := range Timeline {
		if met < Timeline[i].METOffset {
			return &Timeline[i]
		}
	}
	return nil
}

func MissionProgress() float64 {
	return MissionProgressAt(MET())
}

func MissionProgressAt(met time.Duration) float64 {
	totalDuration := TotalDuration()
	if totalDuration <= 0 {
		return 0
	}
	progress := float64(met) / float64(totalDuration)
	if progress < 0 {
		return 0
	}
	if progress > 1 {
		return 1
	}
	return progress
}
