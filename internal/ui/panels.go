package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"artemis/internal/dsn"
	"artemis/internal/horizons"
	"artemis/internal/mission"
	"artemis/internal/nasablog"
	"artemis/internal/spaceweather"
)

func (m Model) View() string {
	if m.width == 0 || m.layout == nil {
		return "Loading..."
	}

	// Minimum terminal size guard.
	if m.width < 60 || m.height < 14 {
		msg := fmt.Sprintf(
			"Terminal too small\n\nMinimum: 60 x 14\nCurrent: %d x %d\n\nPlease resize your terminal.",
			m.width, m.height,
		)
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			dimStyle.Render(msg))
	}

	w := m.layout[panelHeader].width

	// Clock + header render fresh every frame (time-sensitive)
	header := renderHeader(w)

	if m.blogReaderOpen {
		availableH := m.height - measureHeight(header) - 1
		if availableH < 6 {
			availableH = 6
		}
		reader := renderMissionLogReader(m, w, availableH)
		help := renderFooter(m, w)
		result := lipgloss.JoinVertical(lipgloss.Left, header, reader, help)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Top, result)
	}

	var sections []string
	sections = append(sections, header)

	if !m.visualizationFullscreen {
		topRow := renderTopRow(m, w)
		sections = append(sections, topRow)
	}

	if pl := m.layout[panelSpaceWeather]; pl.visible {
		sections = append(sections, m.cachedSW)
	}
	if pl := m.layout[panelDSN]; pl.visible {
		sections = append(sections, m.cachedDSN)
	}
	if pl := m.layout[panelTimeline]; pl.visible {
		sections = append(sections, m.cachedTimeline)
	}
	if pl := m.layout[panelMissionLog]; pl.visible {
		sections = append(sections, m.cachedBlog)
	}
	if pl := m.layout[panelTrajectory]; pl.visible {
		sections = append(sections, m.cachedTrajectory)
	}
	if pl := m.layout[panelCrew]; pl.visible {
		sections = append(sections, m.cachedCrew)
	}

	help := renderFooter(m, w)
	sections = append(sections, help)

	result := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Fill entire terminal: center horizontally, top-align vertically.
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Top, result)
}

func renderFooter(m Model, w int) string {
	if m.blogReaderOpen {
		return renderBlogReaderFooter(m, w)
	}

	theme := ThemeName()
	hiddenWide := hiddenPanelSummary(m, false)
	hiddenCompact := hiddenPanelSummary(m, true)
	notificationError := m.footerNotificationError()
	uptimeWide, uptimeCompact := m.footerUptimeParts()
	notifyState := "off"
	if m.notificationsEnabled {
		notifyState = "on"
	}
	fullscreenWide := "f fullscreen"
	fullscreenCompact := "f full"
	fullscreenTight := "f"
	if m.visualizationFullscreen {
		fullscreenWide = "f windowed"
		fullscreenCompact = "f win"
	}
	viewWide := "v view"
	viewCompact := "v view"
	viewTight := "v"
	unitWide := fmt.Sprintf("u units(%s)", m.units.name())
	unitCompact := fmt.Sprintf("u %s", m.units.name())
	unitTight := fmt.Sprintf("u(%s)", m.units.compactName())
	notifyWide := fmt.Sprintf("n notify(%s)", notifyState)
	notifyCompact := fmt.Sprintf("n ntfy(%s)", notifyState)
	notifyTight := fmt.Sprintf("n(%s)", notifyState)
	debugWide := ""
	debugCompact := ""
	debugTight := ""
	if m.debugKeysEnabled {
		debugWide = "N test-notify"
		debugCompact = "N test"
		debugTight = "N"
	}
	if m.layout != nil {
		if pl, ok := m.layout[panelTrajectory]; ok && !pl.visible {
			viewWide = ""
			viewCompact = ""
			viewTight = ""
		}
	}

	candidates := []string{
		joinFooterParts(
			"q/esc quit",
			"t timeline",
			viewWide,
			fullscreenWide,
			unitWide,
			notifyWide,
			debugWide,
			fmt.Sprintf("c theme(%s)", theme),
			"s stars",
			"r refresh",
			"j/k/enter log",
			notificationError,
			hiddenWide,
			uptimeWide,
			fmt.Sprintf("%dx%d", m.width, m.height),
		),
		joinFooterParts(
			"q quit",
			"t tl",
			viewCompact,
			fullscreenCompact,
			unitCompact,
			notifyCompact,
			debugCompact,
			fmt.Sprintf("c %s", theme),
			"s stars",
			"r",
			"log nav",
			notificationError,
			hiddenCompact,
			uptimeCompact,
			fmt.Sprintf("%dx%d", m.width, m.height),
		),
		joinFooterParts(
			joinFooterParts("q", "t", viewTight, fullscreenTight, unitTight, notifyTight, debugTight, "c", "s", "r", "log"),
			notificationError,
			hiddenCompact,
			uptimeCompact,
			fmt.Sprintf("%s %dx%d", theme, m.width, m.height),
		),
		joinFooterParts(hiddenCompact, fmt.Sprintf("%dx%d", m.width, m.height)),
		fmt.Sprintf("%dx%d", m.width, m.height),
	}

	for _, candidate := range candidates {
		if lipgloss.Width(candidate) <= w {
			return helpStyle.Width(w).Align(lipgloss.Center).Render(candidate)
		}
	}

	return helpStyle.Width(w).Align(lipgloss.Center).Render(candidates[len(candidates)-1])
}

func renderBlogReaderFooter(m Model, w int) string {
	candidates := []string{
		joinFooterParts("esc close", "j/k scroll", "pgup/pgdn page", "g/G top/end", "o browser", "r reload", "q quit"),
		joinFooterParts("esc", "j/k", "pgup/pgdn", "g/G", "o", "r", "q"),
		joinFooterParts("esc", "scroll", "o", "q"),
	}

	for _, candidate := range candidates {
		if lipgloss.Width(candidate) <= w {
			return helpStyle.Width(w).Align(lipgloss.Center).Render(candidate)
		}
	}

	return helpStyle.Width(w).Align(lipgloss.Center).Render(candidates[len(candidates)-1])
}

func (m Model) footerNotificationError() string {
	if m.notificationError == "" {
		return ""
	}
	if time.Since(m.notificationErrorAt) > 5*time.Second {
		return ""
	}
	return m.notificationError
}

func (m Model) footerUptimeParts() (string, string) {
	if m.startedAt.IsZero() {
		return "", ""
	}
	uptime := formatFooterUptime(time.Since(m.startedAt))
	return "up " + uptime, uptime
}

func formatFooterUptime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int(d.Seconds())
	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if days > 0 {
		return fmt.Sprintf("%dd%02dh", days, hours)
	}
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func hiddenPanelSummary(m Model, compact bool) string {
	if m.visualizationFullscreen {
		return ""
	}
	if m.layout == nil {
		return ""
	}

	panelNames := []struct {
		id      panelID
		wide    string
		compact string
	}{
		{panelTrajectory, "visualization", "viz"},
		{panelTimeline, "timeline", "tl"},
		{panelSpaceWeather, "weather", "wx"},
		{panelDSN, "dsn", "dsn"},
		{panelMissionLog, "log", "log"},
		{panelCrew, "crew", "crew"},
	}

	var hidden []string
	for _, panel := range panelNames {
		pl, ok := m.layout[panel.id]
		if !ok || pl.visible {
			continue
		}
		if compact {
			hidden = append(hidden, panel.compact)
		} else {
			hidden = append(hidden, panel.wide)
		}
	}

	if len(hidden) == 0 {
		return ""
	}

	return "hidden: " + strings.Join(hidden, ",")
}

func joinFooterParts(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.Join(filtered, "  |  ")
}

func renderHeader(w int) string {
	return renderHeaderAt(w, mission.MET())
}

func renderHeaderAt(w int, met time.Duration) string {
	progress := mission.MissionProgressAt(met)
	barWidth := w - 4
	if barWidth < 0 {
		barWidth = 0
	}
	filled := int(progress * float64(barWidth))
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}

	bar := progressFullStyle.Render(strings.Repeat("━", filled)) +
		progressEmptyStyle.Render(strings.Repeat("─", barWidth-filled))

	title := titleStyle.Width(renderWidthFor(titleStyle, w)).Align(lipgloss.Center).
		Render("ARTEMIS II  ─  Orion \"Integrity\"  ─  Lunar Flyby Mission")

	return lipgloss.JoinVertical(lipgloss.Left, title, "  "+bar)
}

func renderTopRow(m Model, w int) string {
	clockW, spacecraftW := splitWidthEvenly(w)

	clockPanel := renderClockPanel(clockW, 0)
	spacecraftPanel := renderSpacecraftPanel(m, spacecraftW, 0)

	targetHeight := measureHeight(clockPanel)
	if spacecraftHeight := measureHeight(spacecraftPanel); spacecraftHeight > targetHeight {
		targetHeight = spacecraftHeight
	}

	clockPanel = renderClockPanel(clockW, targetHeight)
	spacecraftPanel = renderSpacecraftPanel(m, spacecraftW, targetHeight)

	return lipgloss.JoinHorizontal(lipgloss.Top, clockPanel, spacecraftPanel)
}

func renderClockPanel(w, totalHeight int) string {
	return renderClockPanelAt(w, totalHeight, mission.MET())
}

func renderClockPanelAt(w, totalHeight int, met time.Duration) string {
	day := mission.MissionDayAt(met)
	totalDays := mission.TotalMissionDays()
	metStr := mission.FormatMET(met)

	var nextLine string
	next := mission.NextEvent(met)
	if next != nil {
		countdown := next.METOffset - met
		nextLine = fmt.Sprintf("%s  %s\n%s  %s",
			labelStyle.Render("Next:"),
			valueStyle.Render(next.Label),
			labelStyle.Render("In:"),
			metStyle.Render(mission.FormatCountdown(countdown)),
		)
	} else {
		nextLine = activeStyle.Render("Mission Complete")
	}

	content := fmt.Sprintf("%s  %s\n%s  %s\n%s  %s\n\n%s",
		labelStyle.Render("MET:"),
		metStyle.Render(metStr),
		labelStyle.Render("UTC:"),
		valueStyle.Render(fmt.Sprintf("%s", mission.LaunchTime.Add(met).UTC().Format("2006-01-02 15:04:05"))),
		labelStyle.Render("Day:"),
		valueStyle.Render(fmt.Sprintf("%d / %d", day, totalDays)),
		nextLine,
	)

	style := panelStyle.Width(renderWidthFor(panelStyle, w))
	if totalHeight > 0 {
		contentHeight := totalHeight - panelStyle.GetVerticalBorderSize()
		if contentHeight < 0 {
			contentHeight = 0
		}
		style = style.Height(contentHeight)
	}

	return style.Render(panelTitleStyle.Render("MISSION CLOCK") + "\n" + content)
}

func renderSpacecraftPanel(m Model, w, totalHeight int) string {
	var content string

	if m.hzErr != nil && m.hzState == nil {
		content = errorStyle.Render("Waiting for Horizons data...")
	} else if m.hzState != nil {
		s := m.hzState
		earthDist := effectiveEarthDist(m)
		moonDist := s.MoonDist

		var signalStr string
		if s.IsOccluded() {
			signalStr = lipgloss.NewStyle().Bold(true).Foreground(colorRed).Render("LOS") +
				"  " + dimStyle.Render("loss of signal — Moon blocking Earth contact")
		} else {
			signalStr = lipgloss.NewStyle().Bold(true).Foreground(colorGreen).Render("AOS") +
				"  " + dimStyle.Render("acquisition of signal — Earth contact nominal")
		}

		content = fmt.Sprintf(
			"%s  %s\n%s  %s\n%s  %s\n%s  %s\n%s  %s\n%s  %s\n\n%s  %s\n%s  %s\n%s  %s",
			labelStyle.Render("Earth Dist:"),
			valueStyle.Render(formatDist(earthDist, m.units)),
			labelStyle.Render("Moon Dist: "),
			valueStyle.Render(formatMoonDist(moonDist, m.units)),
			labelStyle.Render("Speed:     "),
			valueStyle.Render(formatSpeedForUnits(s.Speed, m.units)),
			labelStyle.Render("Earth Rate:"),
			formatEarthRateForUnits(s, m.units),
			labelStyle.Render("Ecl Lon/Lat:"),
			formatEclipticCoords(s.Position),
			labelStyle.Render("Position:  "),
			dimStyle.Render(formatPositionVector(s.Position, m.units)),
			labelStyle.Render("Data Age:  "),
			formatStateAge(m, time.Now()),
			labelStyle.Render("RTLT:      "),
			formatRTLT(m),
			labelStyle.Render("Signal:    "),
			signalStr,
		)
	} else {
		content = dimStyle.Render("Fetching spacecraft data...")
	}

	style := panelStyle.Width(renderWidthFor(panelStyle, w))
	if totalHeight > 0 {
		contentHeight := totalHeight - panelStyle.GetVerticalBorderSize()
		if contentHeight < 0 {
			contentHeight = 0
		}
		style = style.Height(contentHeight)
	}

	return style.Render(panelTitleStyle.Render("SPACECRAFT STATE") + "\n" + content)
}

func renderDSNPanel(m Model, w int) string {
	var content string

	if m.dsnErr != nil && m.dsnStatus == nil {
		content = errorStyle.Render("Waiting for DSN data...")
	} else if m.dsnStatus != nil && len(m.dsnStatus.Dishes) > 0 {
		var lines []string
		for _, dish := range m.dsnStatus.Dishes {
			upArrow := dimStyle.Render("·")
			downArrow := dimStyle.Render("·")

			for _, us := range dish.UpSignals {
				if us.Active {
					upArrow = signalUpStyle.Render("▲")
					break
				}
			}
			for _, ds := range dish.DownSignals {
				if ds.Active {
					downArrow = signalDownStyle.Render("▼")
					break
				}
			}

			band := ""
			rate := ""
			for _, ds := range dish.DownSignals {
				if ds.Active {
					band = ds.Band + "-band"
					if ds.DataRate > 0 {
						rate = formatDataRate(ds.DataRate)
					}
					break
				}
			}

			rangeTxt := ""
			for _, t := range dish.Targets {
				if t.DownlegRange > 0 {
					rangeTxt = formatDist(t.DownlegRange, m.units)
				}
			}

			// Pad plain text to fixed widths, then style each column
			dishCol := fmt.Sprintf("%-5s", dish.Name)
			stationCol := fmt.Sprintf("%-14s", dish.Station)
			bandCol := fmt.Sprintf("%-8s", band)
			rateCol := fmt.Sprintf("%-10s", rate)
			rangeCol := fmt.Sprintf("%-10s", rangeTxt)

			line := fmt.Sprintf("  %s %s %s %s %s %s %s %s",
				upArrow, downArrow,
				valueStyle.Render(dishCol),
				dimStyle.Render(stationCol),
				dimStyle.Render(bandCol),
				dimStyle.Render(rateCol),
				dimStyle.Render(rangeCol),
				formatDishActivity(dish),
			)
			lines = append(lines, line)
		}
		content = strings.Join(lines, "\n")
	} else if m.dsnStatus != nil {
		content = dimStyle.Render("No DSN dishes currently tracking Artemis II")
	} else {
		content = dimStyle.Render("Fetching DSN status...")
	}

	return panelStyle.Width(w - 2).Render(
		panelTitleStyle.Render("DEEP SPACE NETWORK") +
			"  " + dimStyle.Render("▲ uplink  ▼ downlink") + "\n" + content,
	)
}

func renderTimelinePanel(w int) string {
	return renderTimelinePanelAt(w, mission.MET())
}

func renderTimelinePanelAt(w int, met time.Duration) string {
	currentIdx := mission.CurrentEventIndex(met)
	events := mission.Timeline

	startIdx := currentIdx - 4
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + 14
	if endIdx > len(events) {
		endIdx = len(events)
	}

	var lines []string
	for i := startIdx; i < endIdx; i++ {
		e := events[i]
		status := mission.EventStatusAt(e, met)
		var prefix string
		var style lipgloss.Style

		switch {
		case i == currentIdx && status == mission.EventCompleted:
			prefix = " > "
			style = currentStyle
		case status == mission.EventCompleted:
			prefix = " + "
			style = completedStyle
		default:
			prefix = "   "
			style = pendingStyle
		}

		metLabel := mission.FormatMET(e.METOffset)
		line := fmt.Sprintf("%s%-24s %s", prefix, style.Render(e.Label), dimStyle.Render(metLabel))
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	return panelStyle.Width(w - 2).Render(
		panelTitleStyle.Render("MISSION TIMELINE") +
			"  " + dimStyle.Render("t: switch to gantt") + "\n" + content,
	)
}

func renderTrajectoryPanel(m Model, w int, plotH int) string {
	return renderVisualizationPanel(m, w, plotH, false)
}

func renderVisualizationPanel(m Model, w, plotH int, fullscreen bool) string {
	plotW := innerWidthFor(panelStyle, w)
	if plotW < 30 {
		plotW = 30
	}
	if plotH < 0 {
		plotH = 0
	}

	title, legend := visualizationMeta(m, fullscreen)
	plot := fitBlockHeight(renderVisualizationBody(m, plotW, plotH), plotH)
	body := plot
	bodyHeight := plotH

	if fullscreen {
		topRow := renderTopRow(m, plotW)
		body = lipgloss.JoinVertical(lipgloss.Left, plot, topRow)
		bodyHeight += measureHeight(topRow)
	}

	return panelStyle.Width(renderWidthFor(panelStyle, w)).Height(1 + bodyHeight).Render(
		panelTitleStyle.Render(title) + "  " + legend + "\n" + body,
	)
}

func renderVisualizationBody(m Model, plotW, plotH int) string {
	switch m.trajectoryView {
	case 1:
		return renderOrbitalMap(m, plotW, plotH)
	case 2:
		return renderInstruments(m, plotW, plotH)
	case 3:
		return renderDSNSky(m, plotW, plotH)
	case 4:
		return renderWeatherTrends(m, plotW, plotH)
	default:
		return renderTrajectory(m, plotW, plotH)
	}
}

func visualizationMeta(m Model, fullscreen bool) (string, string) {
	fullscreenHint := "  " + dimStyle.Render("f: fullscreen")
	if fullscreen {
		fullscreenHint = "  " + dimStyle.Render("f: windowed")
	}

	switch m.trajectoryView {
	case 1:
		legend := earthGlyphStyle.Render("(E)") + dimStyle.Render("=Earth  ") +
			moonGlyphStyle.Render("{M}") + dimStyle.Render("=Moon  ") +
			spacecraftBright.Render("*") + dimStyle.Render("=Orion  ") +
			dimStyle.Render("s: stars  v: switch view") + fullscreenHint
		return "ORBITAL CONTEXT", legend
	case 2:
		legend := dimStyle.Render("v: switch view") + fullscreenHint
		return "INSTRUMENTS", legend
	case 3:
		legend := dimStyle.Render("dish tracks  v: switch view") + fullscreenHint
		return "DSN SKY", legend
	case 4:
		legend := dimStyle.Render("solar wind + geomag trends  v: switch view") + fullscreenHint
		return "WEATHER TRENDS", legend
	default:
		legend := earthGlyphStyle.Render("(E)") + dimStyle.Render("=Earth  ") +
			moonGlyphStyle.Render("[M]") + dimStyle.Render("=Moon  ") +
			spacecraftBright.Render("*") + dimStyle.Render("=Orion  ") +
			sunDirectionStyle.Render("SUN") + dimStyle.Render("=Sun dir  ") +
			dimStyle.Render("s: stars  v: switch view") + fullscreenHint
		return "TRAJECTORY", legend
	}
}

func renderCrewPanel(w int) string {
	var parts []string
	for _, c := range mission.Crew {
		part := fmt.Sprintf("%s %s %s",
			crewRoleStyle.Render(c.Role),
			crewNameStyle.Render(c.Name),
			crewAgencyStyle.Render("("+c.Agency+")"),
		)
		parts = append(parts, part)
	}
	content := strings.Join(parts, "   ")

	return panelStyle.Width(w - 2).Render(
		panelTitleStyle.Render("CREW") + "  " + content,
	)
}

func formatDist(km float64, units unitSystem) string {
	return formatDistForUnits(km, units)
}

func formatMoonDist(km float64, units unitSystem) string {
	if km < 0 {
		return dimStyle.Render("calculating...")
	}
	return formatDist(km, units)
}

func effectiveEarthDist(m Model) float64 {
	if m.dsnStatus != nil && m.dsnStatus.Range > 0 {
		return m.dsnStatus.Range
	}
	if m.hzState != nil {
		return m.hzState.EarthDist
	}
	return 0
}

func formatEarthRateForUnits(s *horizons.State, units unitSystem) string {
	rate, ok := radialVelocity(s.Position, s.Velocity)
	if !ok {
		return dimStyle.Render("n/a")
	}

	direction := "steady"
	switch {
	case rate > 0.005:
		direction = "outbound"
	case rate < -0.005:
		direction = "inbound"
	}

	return valueStyle.Render(formatRateForUnits(rate, units)) + dimStyle.Render(" "+direction)
}

func radialVelocity(position, velocity horizons.Vector3) (float64, bool) {
	mag := position.Magnitude()
	if mag == 0 {
		return 0, false
	}

	return (position.X*velocity.X + position.Y*velocity.Y + position.Z*velocity.Z) / mag, true
}

func formatEclipticCoords(position horizons.Vector3) string {
	lon, lat, ok := eclipticCoords(position)
	if !ok {
		return dimStyle.Render("n/a")
	}

	return valueStyle.Render(fmt.Sprintf("%.1f° / %+.1f°", lon, lat))
}

func formatPositionVector(position horizons.Vector3, units unitSystem) string {
	return fmt.Sprintf(
		"X:%.0f  Y:%.0f  Z:%.0f %s",
		distanceInUnits(position.X, units),
		distanceInUnits(position.Y, units),
		distanceInUnits(position.Z, units),
		units.distanceUnit(),
	)
}

func eclipticCoords(position horizons.Vector3) (float64, float64, bool) {
	r := position.Magnitude()
	if r == 0 {
		return 0, 0, false
	}

	lon := math.Atan2(position.Y, position.X) * 180 / math.Pi
	if lon < 0 {
		lon += 360
	}

	lat := math.Atan2(position.Z, math.Hypot(position.X, position.Y)) * 180 / math.Pi
	return lon, lat, true
}

func formatStateAge(m Model, now time.Time) string {
	parts := make([]string, 0, 2)

	if m.hzState != nil {
		hzTime := m.hzState.Time
		if hzTime.IsZero() {
			hzTime = m.hzState.Timestamp
		}
		if !hzTime.IsZero() {
			parts = append(parts, valueStyle.Render("HZ "+formatDataAge(now.Sub(hzTime))))
		}
	}
	if m.dsnStatus != nil && !m.dsnStatus.Timestamp.IsZero() {
		parts = append(parts, valueStyle.Render("DSN "+formatDataAge(now.Sub(m.dsnStatus.Timestamp))))
	}

	if len(parts) == 0 {
		return dimStyle.Render("n/a")
	}

	return strings.Join(parts, dimStyle.Render("  "))
}

func formatDataAge(age time.Duration) string {
	if age < 0 {
		age = 0
	}

	age = age.Round(time.Second)
	if age < time.Minute {
		return fmt.Sprintf("%ds", int(age/time.Second))
	}
	if age < time.Hour {
		minutes := int(age / time.Minute)
		seconds := int((age % time.Minute) / time.Second)
		return fmt.Sprintf("%dm%02ds", minutes, seconds)
	}

	hours := int(age / time.Hour)
	minutes := int((age % time.Hour) / time.Minute)
	return fmt.Sprintf("%dh%02dm", hours, minutes)
}

func formatRTLT(m Model) string {
	if m.dsnStatus != nil && m.dsnStatus.RTLT > 0 {
		return valueStyle.Render(fmt.Sprintf("%.2f sec", m.dsnStatus.RTLT))
	}
	return dimStyle.Render("n/a")
}

func formatDataRate(bps float64) string {
	if bps >= 1e6 {
		return fmt.Sprintf("%.1f Mbps", bps/1e6)
	}
	if bps >= 1e3 {
		return fmt.Sprintf("%.0f kbps", bps/1e3)
	}
	return fmt.Sprintf("%.0f bps", bps)
}

func formatDishActivity(dish dsn.Dish) string {
	hasUp := false
	hasDown := false
	for _, s := range dish.UpSignals {
		if s.Active {
			hasUp = true
			break
		}
	}
	for _, s := range dish.DownSignals {
		if s.Active {
			hasDown = true
			break
		}
	}

	if hasUp && hasDown {
		return activeStyle.Render("TX/RX")
	}
	if hasUp {
		return signalUpStyle.Render("TX")
	}
	if hasDown {
		return signalDownStyle.Render("RX")
	}
	return dimStyle.Render("IDLE")
}

func renderSpaceWeatherPanel(m Model, w int) string {
	if m.swStatus == nil {
		var content string
		if m.swErr != nil {
			content = errorStyle.Render("Waiting for space weather data...")
		} else {
			content = dimStyle.Render("Fetching space weather...")
		}
		return panelStyle.Width(w - 2).Render(
			panelTitleStyle.Render("SPACE WEATHER") + "\n" + content,
		)
	}

	s := m.swStatus

	// R/S/G scale indicators with fixed-width columns
	rScale := formatScaleIndicator("R", s.RadioBlackout.Scale, "Radio Blackout")
	sScale := formatScaleIndicator("S", s.SolarRadiation.Scale, "Solar Radiation")
	gScale := formatScaleIndicator("G", s.GeomagStorm.Scale, "Geomag Storm")

	scales := fmt.Sprintf("  %s     %s     %s", rScale, sScale, gScale)

	// Kp index
	kpColor := colorGreen
	kpLabel := "Quiet"
	switch {
	case s.Kp >= 7:
		kpColor = colorRed
		kpLabel = "Severe"
	case s.Kp >= 5:
		kpColor = colorYellow
		kpLabel = "Storm"
	case s.Kp >= 4:
		kpColor = colorAccent
		kpLabel = "Active"
	}

	// Bz color
	bzColor := colorGreen
	if s.Bz < -5 {
		bzColor = colorYellow
	}
	if s.Bz < -10 {
		bzColor = colorRed
	}

	details := fmt.Sprintf("  %s %-10s %s %-12s %s %-9s %s %-10s %s %s",
		labelStyle.Render("Kp:"),
		lipgloss.NewStyle().Bold(true).Foreground(kpColor).Render(fmt.Sprintf("%.0f %s", s.Kp, kpLabel)),
		labelStyle.Render("Wind:"),
		valueStyle.Render(formatWindSpeedForUnits(s.WindSpeed, m.units)),
		labelStyle.Render("Bz:"),
		lipgloss.NewStyle().Bold(true).Foreground(bzColor).Render(fmt.Sprintf("%.1f nT", s.Bz)),
		labelStyle.Render("Protons:"),
		formatProtonFlux(s.ProtonFlux10MeV),
		labelStyle.Render("Flare:"),
		valueStyle.Render(s.CurrentFlareClass),
	)

	summary := swSummary(s)

	content := scales + "\n" + details + "\n  " + summary
	return panelStyle.Width(w - 2).Render(
		panelTitleStyle.Render("SPACE WEATHER") +
			"  " + dimStyle.Render("NOAA/SWPC") + "\n" + content,
	)
}

func formatScaleIndicator(prefix string, level int, label string) string {
	var style lipgloss.Style
	switch {
	case level >= 4:
		style = lipgloss.NewStyle().Bold(true).Foreground(colorRed)
	case level >= 3:
		style = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	case level >= 2:
		style = lipgloss.NewStyle().Bold(true).Foreground(colorYellow)
	case level >= 1:
		style = lipgloss.NewStyle().Foreground(colorYellow)
	default:
		style = lipgloss.NewStyle().Foreground(colorGreen)
	}
	indicator := style.Render(fmt.Sprintf("%s%d", prefix, level))
	return fmt.Sprintf("%s %s", indicator, dimStyle.Render(label))
}

// swSummary returns a plain-language one-liner about current conditions.
func swSummary(s *spaceweather.Status) string {
	maxScale := s.RadioBlackout.Scale
	if s.SolarRadiation.Scale > maxScale {
		maxScale = s.SolarRadiation.Scale
	}
	if s.GeomagStorm.Scale > maxScale {
		maxScale = s.GeomagStorm.Scale
	}

	var msg string
	switch {
	case maxScale >= 4 || s.Kp >= 7:
		msg = "Severe space weather — possible comm disruptions and increased radiation exposure"
	case maxScale >= 3 || s.Kp >= 5:
		msg = "Elevated activity — minor comm interference possible, crew radiation dose monitored"
	case maxScale >= 1 || s.Kp >= 4:
		msg = "Mildly active — no impact to mission operations expected"
	default:
		msg = "All quiet — nominal conditions for crew and spacecraft"
	}
	return dimStyle.Render(msg)
}

func formatProtonFlux(flux float64) string {
	if flux >= 10 {
		return lipgloss.NewStyle().Bold(true).Foreground(colorRed).
			Render(fmt.Sprintf("%.1f pfu", flux))
	}
	if flux >= 1 {
		return lipgloss.NewStyle().Foreground(colorYellow).
			Render(fmt.Sprintf("%.1f pfu", flux))
	}
	return valueStyle.Render(fmt.Sprintf("%.2f pfu", flux))
}

func renderMissionLogPanel(m Model, w int, maxEntries int, selectedIdx int) string {
	if m.blogStatus == nil {
		var content string
		if m.blogErr != nil {
			content = errorStyle.Render("Waiting for mission log...")
		} else {
			content = dimStyle.Render("Fetching mission log...")
		}
		return panelStyle.Width(w - 2).Render(
			panelTitleStyle.Render("MISSION LOG") + "\n" + content,
		)
	}

	maxTitle := w - 20
	if maxTitle < 30 {
		maxTitle = 30
	}

	entries := m.blogStatus.Entries
	if maxEntries > 0 && len(entries) > maxEntries {
		entries = entries[:maxEntries]
	}

	var lines []string
	for i, entry := range entries {
		timeStr := entry.Time.Format("15:04Z")
		title := entry.Title
		if len(title) > maxTitle {
			title = title[:maxTitle-3] + "..."
		}
		if i == selectedIdx {
			line := fmt.Sprintf("  %s %s  %s",
				logSelectedCursorStyle.Render("▸"),
				logSelectedTimeStyle.Render(timeStr),
				logSelectedTitleStyle.Render(title),
			)
			lines = append(lines, line)
		} else {
			line := fmt.Sprintf("    %s  %s",
				logTimeStyle.Render(timeStr),
				logTitleStyle.Render(title),
			)
			lines = append(lines, line)
		}
	}

	content := strings.Join(lines, "\n")
	return panelStyle.Width(w - 2).Render(
		panelTitleStyle.Render("MISSION LOG") +
			"  " + dimStyle.Render("j/k: select  enter: read  o: browser") + "\n" + content,
	)
}

func renderMissionLogReader(m Model, w, totalHeight int) string {
	entry, ok := m.selectedBlogEntry()
	if !ok {
		return panelStyle.Width(w - 2).Height(maxInt(0, totalHeight-2)).Render(
			panelTitleStyle.Render("MISSION LOG READER") + "\n" + dimStyle.Render("No mission log entry selected."),
		)
	}

	post := m.blogPostCache[entry.ID]
	contentLines := buildMissionLogReaderLines(entry, post, m.blogPostLoading, m.blogPostErr, innerWidthFor(panelStyle, w))
	bodyH := totalHeight - panelStyle.GetVerticalBorderSize() - 1
	if bodyH < 1 {
		bodyH = 1
	}
	maxScroll := 0
	if len(contentLines) > bodyH {
		maxScroll = len(contentLines) - bodyH
	}
	scroll := m.blogReaderScroll
	if scroll < 0 {
		scroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	end := scroll + bodyH
	if end > len(contentLines) {
		end = len(contentLines)
	}
	visible := contentLines[scroll:end]
	for len(visible) < bodyH {
		visible = append(visible, "")
	}

	title := panelTitleStyle.Render("MISSION LOG READER")
	status := dimStyle.Render(fmt.Sprintf("%s  %s", entry.Time.UTC().Format("2006-01-02 15:04Z"), blogReaderProgress(scroll, maxScroll)))

	return panelStyle.Width(w - 2).Height(maxInt(0, totalHeight-2)).Render(
		title + "  " + status + "\n" + strings.Join(visible, "\n"),
	)
}

func buildMissionLogReaderLines(entry nasablog.Entry, post *nasablog.Post, loading bool, fetchErr error, width int) []string {
	if width < 20 {
		width = 20
	}

	var sections []string
	sections = append(sections, wrapText(entry.Title, width)...)
	sections = append(sections, "")
	if post != nil && post.Content != "" {
		sections = append(sections, wrapText(post.Content, width)...)
	} else {
		if entry.Excerpt != "" {
			sections = append(sections, wrapText(entry.Excerpt, width)...)
			sections = append(sections, "")
		}
		switch {
		case loading:
			sections = append(sections, "Loading full post...")
		case fetchErr != nil:
			sections = append(sections, "Unable to load full post; showing excerpt only.")
			sections = append(sections, fetchErr.Error())
		default:
			sections = append(sections, "Press r to fetch the full post body or o to open the canonical page.")
		}
	}

	styled := make([]string, 0, len(sections))
	for i, line := range sections {
		switch {
		case i == 0:
			styled = append(styled, logSelectedTitleStyle.Render(line))
		case line == "":
			styled = append(styled, "")
		case strings.HasPrefix(line, "Unable to load"):
			styled = append(styled, errorStyle.Render(line))
		case strings.HasPrefix(line, "Loading full") || strings.HasPrefix(line, "Press r"):
			styled = append(styled, dimStyle.Render(line))
		case fetchErr != nil && line == fetchErr.Error():
			styled = append(styled, dimStyle.Render(line))
		default:
			styled = append(styled, valueStyle.Render(line))
		}
	}
	return styled
}

func wrapText(text string, width int) []string {
	if width < 1 {
		return []string{text}
	}
	paragraphs := strings.Split(text, "\n")
	var lines []string
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		current := words[0]
		for _, word := range words[1:] {
			if lipgloss.Width(current)+1+lipgloss.Width(word) > width {
				lines = append(lines, current)
				current = word
				continue
			}
			current += " " + word
		}
		for lipgloss.Width(current) > width {
			runes := []rune(current)
			cut := width
			if cut > len(runes) {
				cut = len(runes)
			}
			lines = append(lines, string(runes[:cut]))
			current = strings.TrimSpace(string(runes[cut:]))
		}
		if current != "" {
			lines = append(lines, current)
		}
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func blogReaderProgress(scroll, maxScroll int) string {
	if maxScroll <= 0 {
		return "all"
	}
	return fmt.Sprintf("%d%%", int(math.Round(float64(scroll)*100/float64(maxScroll))))
}
