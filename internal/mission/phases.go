package mission

import "time"

type Phase struct {
	Name      string
	ShortName string
	StartMET  time.Duration
	EndMET    time.Duration
}

var Phases = []Phase{
	{"Earth Orbit Operations", "Earth Orbit", d(0, 0, 0), d(1, 1, 37)},
	{"Trans-Lunar Coast", "TL Coast", d(1, 1, 37), d(4, 6, 59)},
	{"Lunar Flyby", "Lunar Flyby", d(4, 6, 59), d(5, 19, 47)},
	{"Trans-Earth Coast", "TE Coast", d(5, 19, 47), d(9, 1, 13)},
	{"Entry & Splashdown", "Entry", d(9, 1, 13), d(9, 1, 46)},
}

type PhaseStatus int

const (
	PhaseCompleted PhaseStatus = iota
	PhaseActive
	PhasePending
)

func CurrentPhase(met time.Duration) int {
	for i := len(Phases) - 1; i >= 0; i-- {
		if met >= Phases[i].StartMET {
			return i
		}
	}
	return 0
}

func GetPhaseStatus(phase Phase, met time.Duration) PhaseStatus {
	if met >= phase.EndMET {
		return PhaseCompleted
	}
	if met >= phase.StartMET {
		return PhaseActive
	}
	return PhasePending
}
