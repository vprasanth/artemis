package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"artemis/internal/dsn"
	"artemis/internal/mission"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	w := m.width
	if w > 120 {
		w = 120
	}

	// Clock, header, and timeline always render fresh (time-sensitive, cheap)
	header := renderHeader(w)
	clockPanel := renderClockPanel(w / 3)
	timelinePanel := renderTimelinePanel(w / 3)

	// Everything else uses pre-built cache from Update()
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, clockPanel, m.cachedSpacecraft)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, timelinePanel, m.cachedTrajectory)
	help := helpStyle.Render("  q/esc: quit  |  DSN ~5s  Horizons ~30s  Weather ~60s")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		topRow,
		m.cachedSW,
		m.cachedDSN,
		bottomRow,
		m.cachedCrew,
		help,
	)
}

func renderHeader(w int) string {
	progress := mission.MissionProgress()
	barWidth := w - 4
	filled := int(progress * float64(barWidth))
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}

	bar := progressFullStyle.Render(strings.Repeat("━", filled)) +
		progressEmptyStyle.Render(strings.Repeat("─", barWidth-filled))

	title := titleStyle.Width(w - 2).Align(lipgloss.Center).
		Render("ARTEMIS II  ─  Orion \"Integrity\"  ─  Lunar Flyby Mission")

	return lipgloss.JoinVertical(lipgloss.Left, title, "  "+bar)
}

func renderClockPanel(w int) string {
	met := mission.MET()
	day := mission.MissionDay()
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

	content := fmt.Sprintf("%s  %s\n%s  %s\n%s  %s / 10\n\n%s",
		labelStyle.Render("MET:"),
		metStyle.Render(metStr),
		labelStyle.Render("UTC:"),
		valueStyle.Render(fmt.Sprintf("%s", mission.LaunchTime.Add(met).UTC().Format("2006-01-02 15:04:05"))),
		labelStyle.Render("Day:"),
		valueStyle.Render(fmt.Sprintf("%d", day)),
		nextLine,
	)

	return panelStyle.Width(w - 2).Render(
		panelTitleStyle.Render("MISSION CLOCK") + "\n" + content,
	)
}

func renderSpacecraftPanel(m Model, w int) string {
	var content string

	if m.hzErr != nil && m.hzState == nil {
		content = errorStyle.Render("Waiting for Horizons data...")
	} else if m.hzState != nil {
		s := m.hzState
		earthDist := s.EarthDist
		moonDist := s.MoonDist

		// Use DSN range if available (more real-time)
		if m.dsnStatus != nil && m.dsnStatus.Range > 0 {
			earthDist = m.dsnStatus.Range
		}

		content = fmt.Sprintf(
			"%s  %s\n%s  %s\n%s  %s\n%s  %s\n\n%s  %s",
			labelStyle.Render("Earth Dist:"),
			valueStyle.Render(formatDist(earthDist)),
			labelStyle.Render("Moon Dist: "),
			valueStyle.Render(formatMoonDist(moonDist)),
			labelStyle.Render("Speed:     "),
			valueStyle.Render(fmt.Sprintf("%.3f km/s  (%.0f km/h)", s.Speed, s.Speed*3600)),
			labelStyle.Render("Position:  "),
			dimStyle.Render(fmt.Sprintf("X:%.0f  Y:%.0f  Z:%.0f km", s.Position.X, s.Position.Y, s.Position.Z)),
			labelStyle.Render("RTLT:      "),
			formatRTLT(m),
		)
	} else {
		content = dimStyle.Render("Fetching spacecraft data...")
	}

	return panelStyle.Width(w - 2).Render(
		panelTitleStyle.Render("SPACECRAFT STATE") + "\n" + content,
	)
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
					rangeTxt = formatDist(t.DownlegRange)
				}
			}

			line := fmt.Sprintf("  %s %s %-16s %s %-8s %-12s %s",
				upArrow, downArrow,
				valueStyle.Render(dish.Name)+" "+dimStyle.Render(dish.Station),
				dimStyle.Render(band),
				dimStyle.Render(rate),
				dimStyle.Render(rangeTxt),
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
	met := mission.MET()
	currentIdx := mission.CurrentEventIndex(met)
	events := mission.Timeline

	// Show a window of events around the current one
	startIdx := currentIdx - 3
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + 12
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
		line := fmt.Sprintf("%s%s %s", prefix, style.Render(e.Label), dimStyle.Render(metLabel))
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	return panelStyle.Width(w - 2).Render(
		panelTitleStyle.Render("MISSION TIMELINE") + "\n" + content,
	)
}

func renderTrajectoryPanel(m Model, w int) string {
	earthDist := 0.0
	moonDist := 0.0
	if m.hzState != nil {
		earthDist = m.hzState.EarthDist
		moonDist = m.hzState.MoonDist
	}

	plot := renderTrajectory(earthDist, moonDist)

	return panelStyle.Width(w - 2).Render(
		panelTitleStyle.Render("TRAJECTORY") +
			"  " + dimStyle.Render("E=Earth  M=Moon  *=Orion") + "\n" + plot,
	)
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

func formatDist(km float64) string {
	if km >= 1e6 {
		return fmt.Sprintf("%.1f M km", km/1e6)
	}
	if km >= 1000 {
		return fmt.Sprintf("%.0f km", km)
	}
	return fmt.Sprintf("%.1f km", km)
}

func formatMoonDist(km float64) string {
	if km < 0 {
		return dimStyle.Render("calculating...")
	}
	return formatDist(km)
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

	// R/S/G scale indicators
	rScale := formatScaleIndicator("R", s.RadioBlackout.Scale, "Radio")
	sScale := formatScaleIndicator("S", s.SolarRadiation.Scale, "Solar Rad")
	gScale := formatScaleIndicator("G", s.GeomagStorm.Scale, "Geomag")

	scales := fmt.Sprintf("  %s    %s    %s", rScale, sScale, gScale)

	// Solar wind + magnetic field + Kp
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

	bzColor := colorGreen
	if s.Bz < -5 {
		bzColor = colorYellow
	}
	if s.Bz < -10 {
		bzColor = colorRed
	}

	details := fmt.Sprintf(
		"  %s %s  %s %s  %s %s  %s %s  %s %s",
		labelStyle.Render("Kp:"),
		lipgloss.NewStyle().Bold(true).Foreground(kpColor).Render(fmt.Sprintf("%.0f %s", s.Kp, kpLabel)),
		labelStyle.Render("Wind:"),
		valueStyle.Render(fmt.Sprintf("%.0f km/s", s.WindSpeed)),
		labelStyle.Render("Bz:"),
		lipgloss.NewStyle().Bold(true).Foreground(bzColor).Render(fmt.Sprintf("%.1f nT", s.Bz)),
		labelStyle.Render("Protons:"),
		formatProtonFlux(s.ProtonFlux10MeV),
		labelStyle.Render("Flare:"),
		valueStyle.Render(s.CurrentFlareClass),
	)

	content := scales + "\n" + details
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
