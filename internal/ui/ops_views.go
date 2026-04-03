package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"artemis/internal/dsn"
)

func renderDSNSky(m Model, plotW, plotH int) string {
	if plotH < 8 || plotW < 32 {
		return renderDSNSkyList(m, plotW)
	}
	if m.dsnStatus == nil || len(m.dsnStatus.Dishes) == 0 {
		return dimStyle.Render("Awaiting DSN tracking geometry...")
	}

	scopeH := plotH * 65 / 100
	if scopeH < 6 {
		scopeH = 6
	}
	listH := plotH - scopeH
	scope := renderDSNSkyScope(m, plotW, scopeH)
	list := renderDSNSkyList(m, plotW)

	lines := strings.Split(list, "\n")
	if listH > 0 && len(lines) > listH {
		lines = lines[:listH]
	}
	return lipgloss.JoinVertical(lipgloss.Left, scope, strings.Join(lines, "\n"))
}

func renderDSNSkyScope(m Model, plotW, plotH int) string {
	canvas := make([][]string, plotH)
	for y := range canvas {
		canvas[y] = make([]string, plotW)
		for x := range canvas[y] {
			canvas[y][x] = " "
		}
	}

	cx := plotW / 2
	cy := plotH - 2
	radius := minInt(plotW/3, plotH-3)
	if radius < 3 {
		radius = 3
	}

	for _, elev := range []float64{30, 60} {
		r := int(math.Round((90 - elev) / 90 * float64(radius)))
		drawCircle(canvas, cx, cy, float64(r), 0.5, "·", scopeRingStyle, plotW, plotH)
	}
	drawCircle(canvas, cx, cy, float64(radius), 0.5, "·", scopeRingStyle, plotW, plotH)

	labels := map[string][2]int{
		"N": {cx, cy - int(float64(radius)*0.5) - 1},
		"E": {cx + radius, cy},
		"W": {cx - radius, cy},
	}
	for label, pos := range labels {
		x, y := pos[0], pos[1]
		if x >= 0 && x < plotW && y >= 0 && y < plotH {
			for i, ch := range label {
				if x+i < plotW {
					canvas[y][x+i] = compassLabelStyle.Render(string(ch))
				}
			}
		}
	}

	for _, dish := range m.dsnStatus.Dishes {
		point, ok := projectDishPoint(dish.Azimuth, dish.Elevation, cx, cy, radius)
		if !ok || point.x < 0 || point.x >= plotW || point.y < 0 || point.y >= plotH {
			continue
		}
		ch := dishMarker(dish)
		canvas[point.y][point.x] = activeStyle.Render(ch)
	}

	var rows []string
	rows = append(rows, instTitleStyle.Render("TRACKING SKY")+" "+dimStyle.Render("az/el"))
	for _, row := range canvas {
		rows = append(rows, joinCanvasRow(row))
	}
	return strings.Join(rows, "\n")
}

func renderDSNSkyList(m Model, plotW int) string {
	if m.dsnStatus == nil || len(m.dsnStatus.Dishes) == 0 {
		return dimStyle.Render("No DSN dishes currently tracking Artemis II")
	}

	lines := []string{instTitleStyle.Render("ACTIVE DISHES")}
	for _, dish := range m.dsnStatus.Dishes {
		rate := "---"
		band := "---"
		power := 0.0
		for _, signal := range dish.DownSignals {
			if signal.Active {
				if signal.DataRate > 0 {
					rate = formatDataRate(signal.DataRate)
				}
				if signal.Band != "" {
					band = signal.Band
				}
				power = signal.Power
				break
			}
		}
		line := fmt.Sprintf(
			"%s  %s  az %.0f  el %.0f  %s  %s  %.0f W",
			dishMarker(dish),
			dish.Name,
			dish.Azimuth,
			dish.Elevation,
			band,
			rate,
			power,
		)
		lines = append(lines, trimToWidth(line, plotW))
	}
	return strings.Join(lines, "\n")
}

type dishPoint struct {
	x int
	y int
}

func projectDishPoint(azimuth, elevation float64, cx, cy, radius int) (dishPoint, bool) {
	if elevation < 0 || elevation > 90 {
		return dishPoint{}, false
	}
	r := (90 - elevation) / 90 * float64(radius)
	az := azimuth * math.Pi / 180
	x := cx + int(math.Round(r*math.Sin(az)))
	y := cy - int(math.Round(r*math.Cos(az)*0.5))
	return dishPoint{x: x, y: y}, true
}

func dishMarker(dish dsn.Dish) string {
	name := strings.TrimPrefix(strings.ToUpper(dish.Name), "DSS")
	if name == "" {
		return "•"
	}
	return string(name[0])
}

func renderWeatherTrends(m Model, plotW, plotH int) string {
	if plotH < 8 || plotW < 40 {
		return renderWeatherSnapshot(m, plotW)
	}

	leftW := plotW / 2
	rightW := plotW - leftW
	topH := plotH / 2
	botH := plotH - topH
	windUnit := "km/s"
	windHistory := m.windSpeedHistory
	if m.units == unitImperial {
		windUnit = "mi/s"
		windHistory = convertHistory(windHistory, func(v float64) float64 { return speedInUnits(v, m.units) })
	}

	leftTop := renderTrendMetric("KP", "idx", currentMetricValue(m.kpHistory), m.kpHistory, leftW, topH, valueStyle)
	rightTop := renderTrendMetric("BZ", "nT", currentMetricValue(m.bzHistory), m.bzHistory, rightW, topH, valueStyle)
	leftBot := renderTrendMetric("WIND", windUnit, currentMetricValue(windHistory), windHistory, leftW, botH, valueStyle)
	rightBot := renderTrendMetric("PROTON", "pfu", currentMetricValue(m.protonFluxHistory), m.protonFluxHistory, rightW, botH, valueStyle)

	grid := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, leftTop, rightTop),
		lipgloss.JoinHorizontal(lipgloss.Top, leftBot, rightBot),
	)

	if plotH >= 14 {
		snapshot := renderWeatherSnapshot(m, plotW)
		return lipgloss.JoinVertical(lipgloss.Left, grid, snapshot)
	}
	return grid
}

func renderTrendMetric(title, unit string, current float64, history []float64, w, h int, style lipgloss.Style) string {
	box := lipgloss.NewStyle().Width(w).Height(h)
	if h < 3 || w < 12 {
		return box.Render("")
	}

	lines := []string{instTitleStyle.Render(title) + " " + dimStyle.Render(unit)}
	if len(history) > 0 {
		lines = append(lines, style.Render(fmt.Sprintf("%.2f", current)))
		lines = append(lines, sparklineStyle.Render(renderSparkline(history, sparklineWidth(history, w-4))))
		mn, avg, mx := minAvgMax(history)
		lines = append(lines, dimStyle.Render(fmt.Sprintf("min %.2f  avg %.2f  max %.2f", mn, avg, mx)))
	} else {
		lines = append(lines, dimStyle.Render("awaiting data"))
	}
	return box.Render(strings.Join(lines, "\n"))
}

func renderWeatherSnapshot(m Model, plotW int) string {
	if m.swStatus == nil {
		return dimStyle.Render("Awaiting space weather sample...")
	}

	s := m.swStatus
	lines := []string{
		instTitleStyle.Render("CURRENT CONDITIONS"),
		trimToWidth(fmt.Sprintf("density %.1f n/cc  temp %.0f K  Bt %.1f nT", s.WindDensity, s.WindTemp, s.Bt), plotW),
		trimToWidth(fmt.Sprintf("wind %s  Kp %.1f  Bz %.1f  proton %.2f pfu", formatWindSpeedForUnits(s.WindSpeed, m.units), s.Kp, s.Bz, s.ProtonFlux10MeV), plotW),
	}
	if s.LatestAlert != "" {
		lines = append(lines, trimToWidth("alert "+s.LatestAlert, plotW))
	}
	return strings.Join(lines, "\n")
}

func currentMetricValue(history []float64) float64 {
	if len(history) == 0 {
		return 0
	}
	return history[len(history)-1]
}

func convertHistory(history []float64, fn func(float64) float64) []float64 {
	if len(history) == 0 {
		return nil
	}
	out := make([]float64, len(history))
	for i, v := range history {
		out[i] = fn(v)
	}
	return out
}

func trimToWidth(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	if width <= 3 {
		return string(runes[:width])
	}
	if len(runes) > width-3 {
		return string(runes[:width-3]) + "..."
	}
	return s
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
