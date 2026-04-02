package ui

import "strings"

type panelID int

const (
	panelHeader panelID = iota
	panelTopRow
	panelSpaceWeather
	panelDSN
	panelTimeline
	panelMissionLog
	panelTrajectory
	panelCrew
	panelHelp
)

type panelLayout struct {
	visible bool
	height  int
	width   int
}

// computeLayout decides which optional panels fit based on measured heights.
// Trajectory is treated as a flex panel: it gets whatever space remains after
// all fixed-height panels are placed. Other panels are placed greedily by priority.
// Returns the layout map and the height available for the trajectory plot.
func computeLayout(w, termHeight, fixedHeight int, measured map[panelID]int) (map[panelID]panelLayout, int) {
	layout := make(map[panelID]panelLayout)

	remaining := termHeight - fixedHeight
	if remaining < 0 {
		remaining = 0
	}

	// Fixed-height panels in priority order (trajectory excluded -- it's flex).
	prioritized := []panelID{
		panelDSN,
		panelTimeline,
		panelSpaceWeather,
		panelMissionLog,
		panelCrew,
	}

	used := 0
	for _, id := range prioritized {
		h := measured[id]
		if h == 0 {
			layout[id] = panelLayout{visible: false, height: 0, width: w}
			continue
		}
		if used+h <= remaining {
			layout[id] = panelLayout{visible: true, height: h, width: w}
			used += h
		} else {
			layout[id] = panelLayout{visible: false, height: 0, width: w}
		}
	}

	// Trajectory gets remaining space. It needs at least 3 lines for the panel
	// (2 border + 1 title) plus a minimum plot height of 6.
	trajectoryAvail := remaining - used
	minTrajectoryH := 9 // 2 border + 1 title + 6 min plot
	if trajectoryAvail >= minTrajectoryH {
		layout[panelTrajectory] = panelLayout{visible: true, height: trajectoryAvail, width: w}
	} else {
		layout[panelTrajectory] = panelLayout{visible: false, height: 0, width: w}
	}

	return layout, trajectoryAvail
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n") + 1
	if strings.HasSuffix(s, "\n") {
		n--
	}
	return n
}
