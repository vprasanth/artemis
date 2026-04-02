package mission

import "time"

type Phase struct {
	Name      string
	ShortName string
	StartMET  time.Duration
	EndMET    time.Duration
}

var Phases = []Phase{
	{"Earth Orbit Operations", "Earth Orbit", d(0, 0, 0), d(1, 1, 8)},
	{"Trans-Lunar Coast", "TL Coast", d(1, 1, 8), d(4, 6, 8)},
	{"Lunar Flyby", "Lunar Flyby", d(4, 6, 8), d(5, 18, 52)},
	{"Trans-Earth Coast", "TE Coast", d(5, 18, 52), d(9, 1, 9)},
	{"Entry & Splashdown", "Entry", d(9, 1, 9), d(9, 1, 42)},
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
