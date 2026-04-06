package ui

import (
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"artemis/internal/mission"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const ganttLabelWidth = 15

const (
	focusPastSpan   = 12 * time.Hour
	focusFutureSpan = 18 * time.Hour
)

var timelineZoomSpans = []struct {
	past   time.Duration
	future time.Duration
}{
	{36 * time.Hour, 54 * time.Hour},
	{24 * time.Hour, 36 * time.Hour},
	{12 * time.Hour, 18 * time.Hour},
	{8 * time.Hour, 12 * time.Hour},
	{4 * time.Hour, 6 * time.Hour},
	{2 * time.Hour, 3 * time.Hour},
}

const defaultTimelineZoomLevel = 2

type activityPaletteColor struct {
	bg string
	fg string
}

var crewActivityPalette = []activityPaletteColor{
	{bg: "#1D4ED8", fg: "#F8FAFC"},
	{bg: "#0F766E", fg: "#F8FAFC"},
	{bg: "#B45309", fg: "#FFF7ED"},
	{bg: "#BE123C", fg: "#FFF1F2"},
	{bg: "#6D28D9", fg: "#F5F3FF"},
	{bg: "#15803D", fg: "#F0FDF4"},
	{bg: "#C2410C", fg: "#FFF7ED"},
	{bg: "#334155", fg: "#F8FAFC"},
}

func renderGanttPanel(w int) string {
	return renderGanttPanelAt(w, mission.MET())
}

func renderGanttPanelAt(w int, met time.Duration) string {
	return renderGanttPanelZoomAt(w, met, defaultTimelineZoomLevel)
}

func renderGanttPanelZoomAt(w int, met time.Duration, zoom int) string {
	totalDur := mission.TotalDuration()

	barW := innerWidthFor(panelStyle, w) - ganttLabelWidth
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

	// Focused event strip
	lines = append(lines, "")
	lines = append(lines, renderFocusedEventStrip(met, barW, totalDur, zoom)...)

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
	return panelStyle.Width(renderWidthFor(panelStyle, w)).Render(
		panelTitleStyle.Render("MISSION TIMELINE") +
			"  " + dimStyle.Render(fmt.Sprintf("t: event list  +/-: zoom x%d", zoom+1)) + "\n" + content,
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

func renderFocusedEventStrip(met time.Duration, barWidth int, totalDur time.Duration, zoom int) []string {
	windowStart, windowEnd := focusedWindow(met, totalDur, zoom)
	pad := strings.Repeat(" ", ganttLabelWidth)
	lineWidth := ganttLabelWidth + barWidth
	nowCol := metToFocusCol(met, windowStart, windowEnd, barWidth)

	labels, ticks := renderFocusedAxis(barWidth, windowStart, windowEnd, met)
	blocks, blockLabel := renderFocusedScheduleBlocks(barWidth, windowStart, windowEnd, met, totalDur)
	nowMarker := renderFocusedNowMarker(barWidth, nowCol, "▼")
	currentSummary, nextSummary := renderFocusedScheduleSummary(met)

	lines := []string{
		panelTitleStyle.Render("  FOCUSED ACTIVITY"),
		clipStyledLine(pad+dimStyle.Render(labels), lineWidth),
		clipStyledLine(pad+dimStyle.Render(ticks), lineWidth),
		clipStyledLine(ganttNowMarker.Render(fmt.Sprintf("  %-13s", "NOW"))+nowMarker, lineWidth),
		clipStyledLine(labelStyle.Render(fmt.Sprintf("  %-13s", blockLabel))+blocks, lineWidth),
		"  " + currentSummary,
		"  " + nextSummary,
	}
	return lines
}

func focusedWindow(met, totalDur time.Duration, zoom int) (time.Duration, time.Duration) {
	spans := timelineZoomWindow(zoom)
	span := spans.past + spans.future
	if totalDur <= span {
		return 0, totalDur
	}

	start := met - spans.past
	if start < 0 {
		start = 0
	}
	end := start + span
	if end > totalDur {
		end = totalDur
		start = end - span
		if start < 0 {
			start = 0
		}
	}

	return start, end
}

func timelineZoomWindow(zoom int) struct {
	past   time.Duration
	future time.Duration
} {
	if zoom < 0 {
		zoom = 0
	}
	if zoom >= len(timelineZoomSpans) {
		zoom = len(timelineZoomSpans) - 1
	}
	return timelineZoomSpans[zoom]
}

func maxTimelineZoomLevel() int {
	return len(timelineZoomSpans) - 1
}

func renderFocusedAxis(barWidth int, windowStart, windowEnd, now time.Duration) (string, string) {
	labelLine := make([]rune, barWidth)
	tickLine := make([]rune, barWidth)
	for i := range labelLine {
		labelLine[i] = ' '
		tickLine[i] = '─'
	}

	window := windowEnd - windowStart
	if window <= 0 {
		return string(labelLine), string(tickLine)
	}

	marks := []struct {
		offset time.Duration
		label  string
	}{
		{windowStart, mission.FormatMET(windowStart)},
		{now, mission.FormatMET(now)},
		{windowEnd, mission.FormatMET(windowEnd)},
	}

	for _, mark := range marks {
		col := metToFocusCol(mark.offset, windowStart, windowEnd, barWidth)
		if col < 0 {
			col = 0
		}
		if col >= barWidth {
			col = barWidth - 1
		}
		tickLine[col] = '┼'
		placeAxisLabel(labelLine, col, mark.label)
	}

	step := 6 * time.Hour
	for tick := windowStart.Truncate(step); tick <= windowEnd; tick += step {
		if tick <= windowStart || tick >= windowEnd {
			continue
		}
		col := metToFocusCol(tick, windowStart, windowEnd, barWidth)
		if col < 0 || col >= barWidth {
			continue
		}
		if tick == now {
			continue
		}
		tickLine[col] = '┬'
	}

	return string(labelLine), string(tickLine)
}

func placeAxisLabel(line []rune, col int, label string) {
	if len(line) == 0 || label == "" {
		return
	}

	labelRunes := []rune(label)
	start := col - len(labelRunes)/2
	if start < 0 {
		start = 0
	}
	if start+len(labelRunes) > len(line) {
		start = len(line) - len(labelRunes)
	}
	if start < 0 {
		return
	}
	copy(line[start:], labelRunes)
}

func metToFocusCol(target, windowStart, windowEnd time.Duration, barWidth int) int {
	if barWidth <= 1 || windowEnd <= windowStart {
		return 0
	}
	if target <= windowStart {
		return 0
	}
	if target >= windowEnd {
		return barWidth - 1
	}
	progress := float64(target-windowStart) / float64(windowEnd-windowStart)
	col := int(progress * float64(barWidth-1))
	if col < 0 {
		return 0
	}
	if col >= barWidth {
		return barWidth - 1
	}
	return col
}

func renderFocusedEventBlocks(barWidth int, windowStart, windowEnd, met, totalDur time.Duration) string {
	currentIdx := mission.CurrentEventIndex(met)
	var b strings.Builder
	cursor := 0

	for i, event := range mission.Timeline {
		segStart := event.METOffset
		segEnd := totalDur
		if i+1 < len(mission.Timeline) {
			segEnd = mission.Timeline[i+1].METOffset
		}
		if segEnd <= windowStart || segStart >= windowEnd {
			continue
		}

		startCol := metToFocusCol(segStart, windowStart, windowEnd, barWidth)
		endCol := metToFocusCol(segEnd, windowStart, windowEnd, barWidth) + 1
		if segStart <= windowStart {
			startCol = 0
		}
		if segEnd >= windowEnd {
			endCol = barWidth
		}
		if endCol <= startCol {
			endCol = startCol + 1
		}
		if endCol > barWidth {
			endCol = barWidth
		}

		if startCol > cursor {
			b.WriteString(strings.Repeat(" ", startCol-cursor))
		}

		width := endCol - startCol
		statusStyle := ganttFutureBlock
		switch {
		case i < currentIdx:
			statusStyle = ganttPastBlock
		case i == currentIdx && met >= event.METOffset:
			statusStyle = ganttCurrentBlock
		}

		b.WriteString(renderEventBlock(event.Label, width, statusStyle))
		cursor = endCol
	}

	if cursor < barWidth {
		b.WriteString(strings.Repeat(" ", barWidth-cursor))
	}

	return b.String()
}

func renderFocusedScheduleBlocks(barWidth int, windowStart, windowEnd, met, totalDur time.Duration) (string, string) {
	if blocks, ok := renderFocusedCrewBlocks(barWidth, windowStart, windowEnd, met); ok {
		return blocks, "CREW"
	}
	return renderFocusedEventBlocks(barWidth, windowStart, windowEnd, met, totalDur), "EVENTS"
}

func renderFocusedCrewBlocks(barWidth int, windowStart, windowEnd, met time.Duration) (string, bool) {
	current := mission.CurrentCrewActivity(met)

	var b strings.Builder
	cursor := 0
	found := false
	paintGapAtNow := current == nil

	for _, activity := range mission.CrewActivities {
		if activity.EndMET <= windowStart || activity.StartMET >= windowEnd {
			continue
		}
		if current != nil && activity.StartMET > current.EndMET && activity.Label == current.Label {
			continue
		}
		found = true
		startCol := metToFocusCol(activity.StartMET, windowStart, windowEnd, barWidth)
		endCol := metToFocusCol(activity.EndMET, windowStart, windowEnd, barWidth) + 1
		if activity.StartMET <= windowStart {
			startCol = 0
		}
		if activity.EndMET >= windowEnd {
			endCol = barWidth
		}
		if endCol <= startCol {
			endCol = startCol + 1
		}
		if endCol > barWidth {
			endCol = barWidth
		}

		if startCol > cursor {
			if paintGapAtNow && met >= windowStart && met < activity.StartMET {
				b.WriteString(renderSyntheticCrewGapBlock(startCol - cursor))
				cursor = startCol
				paintGapAtNow = false
			}
			if startCol > cursor {
				b.WriteString(strings.Repeat(" ", startCol-cursor))
			}
		}

		width := endCol - startCol
		b.WriteString(renderCrewActivityBlock(activity, width, current, met))
		cursor = endCol
	}

	if cursor < barWidth {
		if paintGapAtNow && met >= windowStart && met < windowEnd {
			b.WriteString(renderSyntheticCrewGapBlock(barWidth - cursor))
		} else {
			b.WriteString(strings.Repeat(" ", barWidth-cursor))
		}
	}
	return clipStyledLine(b.String(), barWidth), found
}

func renderEventBlock(label string, width int, style lipgloss.Style) string {
	if width <= 0 {
		return ""
	}

	switch {
	case width == 1:
		return style.Inline(true).Width(width).MaxWidth(width).Render("│")
	case width == 2:
		return style.Inline(true).Width(width).MaxWidth(width).Render("│ ")
	default:
		text := truncateLabel(label, width-2)
		content := ganttBlockDivider.Render("│") + text
		if pad := width - 1 - lipgloss.Width(text); pad > 0 {
			content += strings.Repeat(" ", pad)
		}
		return style.Inline(true).Width(width).MaxWidth(width).Render(content)
	}
}

func renderCrewActivityBlock(activity mission.CrewActivity, width int, current *mission.CrewActivity, met time.Duration) string {
	if width <= 0 {
		return ""
	}

	palette := crewColorForLabel(activity.Label)
	style := lipgloss.NewStyle().
		Inline(true).
		Width(width).
		MaxWidth(width).
		Foreground(lipgloss.Color(palette.fg)).
		Background(lipgloss.Color(palette.bg))

	if current != nil && activity.StartMET == current.StartMET && activity.EndMET == current.EndMET {
		style = style.Bold(true)
	}
	if activity.StartMET < met && (current == nil || activity.StartMET != current.StartMET || activity.EndMET != current.EndMET) {
		style = style.Faint(true)
	}

	switch {
	case width == 1:
		return style.Render("▏")
	case width == 2:
		return style.Render("▏▏")
	default:
		text := truncateLabel(activity.Label, width-2)
		content := "▏" + text
		if pad := width - 2 - lipgloss.Width(text); pad > 0 {
			content += strings.Repeat(" ", pad)
		}
		content += "▏"
		return style.Render(content)
	}
}

func renderSyntheticCrewGapBlock(width int) string {
	if width <= 0 {
		return ""
	}

	style := lipgloss.NewStyle().
		Inline(true).
		Width(width).
		MaxWidth(width).
		Foreground(colorBright).
		Background(colorDim)

	switch {
	case width == 1:
		return style.Render("▏")
	case width == 2:
		return style.Render("▏▏")
	default:
		text := truncateLabel("Unstructured", width-2)
		content := "▏" + text
		if pad := width - 2 - lipgloss.Width(text); pad > 0 {
			content += strings.Repeat(" ", pad)
		}
		content += "▏"
		return style.Render(content)
	}
}

func crewColorForLabel(label string) activityPaletteColor {
	h := fnv.New32a()
	_, _ = h.Write([]byte(label))
	return crewActivityPalette[int(h.Sum32())%len(crewActivityPalette)]
}

func renderFocusedNowMarker(barWidth, nowCol int, marker string) string {
	if barWidth <= 0 {
		return ""
	}
	if nowCol < 0 {
		nowCol = 0
	}
	if nowCol >= barWidth {
		nowCol = barWidth - 1
	}
	return strings.Repeat(" ", nowCol) + ganttNowMarker.Render(marker) + strings.Repeat(" ", barWidth-nowCol-1)
}

func truncateLabel(label string, width int) string {
	if width <= 0 {
		return ""
	}

	runes := []rune(label)
	if len(runes) <= width {
		return label
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}

func clipStyledLine(s string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Cut(s, 0, width)
}

func renderFocusedScheduleSummary(met time.Duration) (string, string) {
	current := mission.CurrentCrewActivity(met)
	nextCrew := mission.NextCrewActivity(met)

	if current != nil {
		currentSummary := labelStyle.Render("Current: ") +
			currentStyle.Render(current.Label) +
			dimStyle.Render("  from ") +
			metStyle.Render(mission.FormatMET(current.StartMET)) +
			dimStyle.Render(" to ") +
			metStyle.Render(mission.FormatMET(current.EndMET))

		if nextCrew != nil {
			nextSummary := labelStyle.Render("Next: ") +
				valueStyle.Render(nextCrew.Label) +
				dimStyle.Render("  at ") +
				metStyle.Render(mission.FormatMET(nextCrew.StartMET)) +
				dimStyle.Render("  in ") +
				metStyle.Render(mission.FormatCountdown(nextCrew.StartMET-met))
			return currentSummary, nextSummary
		}

		return currentSummary, labelStyle.Render("Next: ") + dimStyle.Render("No later crew activity scheduled")
	}

	if nextCrew != nil {
		currentSummary := labelStyle.Render("Current: ") + dimStyle.Render("Unstructured crew ops")
		nextSummary := labelStyle.Render("Next: ") +
			valueStyle.Render(nextCrew.Label) +
			dimStyle.Render("  at ") +
			metStyle.Render(mission.FormatMET(nextCrew.StartMET)) +
			dimStyle.Render("  in ") +
			metStyle.Render(mission.FormatCountdown(nextCrew.StartMET-met))
		return currentSummary, nextSummary
	}

	currentIdx := mission.CurrentEventIndex(met)
	currentEvent := mission.Timeline[currentIdx]
	next := mission.NextEvent(met)

	currentEndsAt := mission.TotalDuration()
	if currentIdx+1 < len(mission.Timeline) {
		currentEndsAt = mission.Timeline[currentIdx+1].METOffset
	}

	currentSummary := labelStyle.Render("Current: ") +
		currentStyle.Render(currentEvent.Label) +
		dimStyle.Render("  from ") +
		metStyle.Render(mission.FormatMET(currentEvent.METOffset)) +
		dimStyle.Render(" to ") +
		metStyle.Render(mission.FormatMET(currentEndsAt))

	if next == nil {
		return currentSummary, labelStyle.Render("Next: ") + dimStyle.Render("Final milestone reached")
	}

	countdown := mission.FormatCountdown(next.METOffset - met)
	nextSummary := labelStyle.Render("Next: ") +
		valueStyle.Render(next.Label) +
		dimStyle.Render("  at ") +
		metStyle.Render(mission.FormatMET(next.METOffset)) +
		dimStyle.Render("  in ") +
		metStyle.Render(countdown)

	return currentSummary, nextSummary
}
