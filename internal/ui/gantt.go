package ui

import (
	"fmt"
	"strings"
	"time"

	"artemis/internal/mission"
)

const ganttLabelWidth = 15

func renderGanttPanel(w int) string {
	return renderGanttPanelAt(w, mission.MET())
}

func renderGanttPanelAt(w int, met time.Duration) string {
	totalDur := mission.TotalDuration()

	barW := w - 4 - ganttLabelWidth
	if barW < 20 {
		barW = 20
	}

	dayLabels, dayTicks := renderDayAxis(barW, totalDur)
	pad := strings.Repeat(" ", ganttLabelWidth)

	var lines []string

	// Day axis header
	lines = append(lines,
		pad+dimStyle.Render(dayLabels),
		pad+dimStyle.Render(dayTicks),
	)

	// Phase rows
	for _, phase := range mission.Phases {
		status := mission.GetPhaseStatus(phase, met)
		var label string
		switch status {
		case mission.PhaseCompleted:
			label = completedStyle.Render(fmt.Sprintf("  %-13s", phase.ShortName))
		case mission.PhaseActive:
			label = currentStyle.Render(fmt.Sprintf("  %-13s", phase.ShortName))
		default:
			label = pendingStyle.Render(fmt.Sprintf("  %-13s", phase.ShortName))
		}
		bar := renderPhaseBar(phase, met, barW, totalDur)
		lines = append(lines, label+bar)
	}

	// Day axis footer
	lines = append(lines, pad+dimStyle.Render(dayTicks))

	// NOW marker
	nowLabel := ganttNowMarker.Render(fmt.Sprintf("  %-13s", "NOW"))
	nowLine := renderNowMarker(met, barW, totalDur)
	lines = append(lines, nowLabel+nowLine)

	// Status line
	phaseIdx := mission.CurrentPhase(met)
	phaseName := mission.Phases[phaseIdx].Name
	day := mission.MissionDayAt(met)
	totalDays := mission.TotalMissionDays()
	statusLine := fmt.Sprintf("  %s  %s  %s %s",
		metStyle.Render(mission.FormatMET(met)),
		dimStyle.Render(fmt.Sprintf("Day %d/%d", day, totalDays)),
		labelStyle.Render("Phase:"),
		currentStyle.Render(phaseName),
	)
	lines = append(lines, statusLine)

	content := strings.Join(lines, "\n")
	return panelStyle.Width(w - 2).Render(
		panelTitleStyle.Render("MISSION GANTT CHART") + "\n" + content,
	)
}

type dayAxisMark struct {
	offset time.Duration
	label  string
}

func dayAxisMarks(totalDur time.Duration) []dayAxisMark {
	fullDays := int(totalDur / (24 * time.Hour))
	marks := make([]dayAxisMark, 0, fullDays+2)
	for day := 0; day <= fullDays; day++ {
		marks = append(marks, dayAxisMark{
			offset: time.Duration(day) * 24 * time.Hour,
			label:  fmt.Sprintf("%d", day),
		})
	}

	if totalDur%(24*time.Hour) != 0 {
		marks = append(marks, dayAxisMark{
			offset: totalDur,
			label:  "E",
		})
	}

	return marks
}

func metToCol(met time.Duration, barWidth int, totalDur time.Duration) int {
	if met <= 0 {
		return 0
	}
	if met >= totalDur {
		return barWidth - 1
	}
	col := int(float64(met) / float64(totalDur) * float64(barWidth))
	if col >= barWidth {
		col = barWidth - 1
	}
	return col
}

func renderDayAxis(barWidth int, totalDur time.Duration) (string, string) {
	dayLine := make([]byte, barWidth)
	tickLine := make([]byte, barWidth)
	for i := range dayLine {
		dayLine[i] = ' '
		tickLine[i] = ' '
	}

	for _, mark := range dayAxisMarks(totalDur) {
		col := metToCol(mark.offset, barWidth, totalDur)
		if col >= barWidth {
			col = barWidth - 1
		}
		tickLine[col] = '|'
		if col+len(mark.label) <= barWidth {
			copy(dayLine[col:], mark.label)
		}
	}

	return string(dayLine), string(tickLine)
}

func renderPhaseBar(phase mission.Phase, met time.Duration, barWidth int, totalDur time.Duration) string {
	startCol := metToCol(phase.StartMET, barWidth, totalDur)
	endCol := metToCol(phase.EndMET, barWidth, totalDur)
	if endCol <= startCol {
		endCol = startCol + 1
	}
	if endCol > barWidth {
		endCol = barWidth
	}

	status := mission.GetPhaseStatus(phase, met)

	prefix := strings.Repeat(" ", startCol)
	phaseWidth := endCol - startCol
	suffix := ""
	if barWidth-endCol > 0 {
		suffix = strings.Repeat(" ", barWidth-endCol)
	}

	switch status {
	case mission.PhaseCompleted:
		return prefix + ganttCompletedBar.Render(strings.Repeat("━", phaseWidth)) + suffix
	case mission.PhasePending:
		return prefix + ganttPendingBar.Render(strings.Repeat("─", phaseWidth)) + suffix
	case mission.PhaseActive:
		nowCol := metToCol(met, barWidth, totalDur)
		if nowCol < startCol {
			nowCol = startCol
		}
		if nowCol >= endCol {
			nowCol = endCol - 1
		}

		doneCols := nowCol - startCol
		pendCols := endCol - nowCol - 1

		var bar string
		if doneCols > 0 {
			bar += ganttActiveBar.Render(strings.Repeat("━", doneCols))
		}
		bar += ganttCursorBar.Render("▎")
		if pendCols > 0 {
			bar += ganttPendingBar.Render(strings.Repeat("─", pendCols))
		}
		return prefix + bar + suffix
	}

	return strings.Repeat(" ", barWidth)
}

func renderNowMarker(met time.Duration, barWidth int, totalDur time.Duration) string {
	nowCol := metToCol(met, barWidth, totalDur)
	if nowCol < 1 {
		return ganttNowMarker.Render("▼") + strings.Repeat(" ", barWidth-1)
	}
	return ganttNowMarker.Render(strings.Repeat("─", nowCol)+"▼") +
		strings.Repeat(" ", barWidth-nowCol-1)
}
