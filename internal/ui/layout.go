package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type panelID int

const (
	panelHeader panelID = iota
	panelTopRow
	panelSpaceWeather
	panelDSN
	panelTimeline
	panelMissionLog
	panelOpsRow
	panelInfoRow
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
	if useWideTopQuad(w) {
		prioritized = []panelID{
			panelTimeline,
			panelCrew,
		}
	}
	if useWideDashboardPairs(w) {
		prioritized = []panelID{
			panelOpsRow,
			panelTimeline,
			panelInfoRow,
		}
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

func measureHeight(s string) int {
	if s == "" {
		return 0
	}
	return lipgloss.Height(s)
}

func renderWidthFor(style lipgloss.Style, totalWidth int) int {
	width := totalWidth - style.GetHorizontalBorderSize()
	if width < 0 {
		return 0
	}
	return width
}

func innerWidthFor(style lipgloss.Style, totalWidth int) int {
	width := totalWidth - style.GetHorizontalFrameSize()
	if width < 0 {
		return 0
	}
	return width
}

func splitWidthEvenly(totalWidth int) (int, int) {
	left := totalWidth / 2
	return left, totalWidth - left
}

func weightedSplitWidths(totalWidth int, weights []int) []int {
	if len(weights) == 0 {
		return nil
	}

	totalWeight := 0
	for _, weight := range weights {
		if weight > 0 {
			totalWeight += weight
		}
	}
	if totalWeight <= 0 {
		widths := make([]int, len(weights))
		base := totalWidth / len(weights)
		used := 0
		for i := range widths {
			widths[i] = base
			used += base
		}
		widths[len(widths)-1] += totalWidth - used
		return widths
	}

	widths := make([]int, len(weights))
	used := 0
	for i, weight := range weights {
		if i == len(weights)-1 {
			widths[i] = totalWidth - used
			break
		}
		if weight < 0 {
			weight = 0
		}
		widths[i] = (totalWidth * weight) / totalWeight
		used += widths[i]
	}
	return widths
}

func useWideDashboardPairs(width int) bool {
	return width >= 140 && !useWideTopQuad(width)
}

func useWideTopQuad(width int) bool {
	return width >= 180
}

func fitBlockHeight(s string, height int) string {
	if height <= 0 {
		return ""
	}

	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}
