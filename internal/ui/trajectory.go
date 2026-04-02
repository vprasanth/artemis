package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"artemis/internal/mission"
)

// Trajectory-specific styles (not theme-dependent since they use fixed colors
// for visual consistency across themes).
var (
	starDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))
	starMedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	starBrightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))

	earthGlyphStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#4FC3F7"))
	moonGlyphStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E0E0E0"))
	spacecraftBright     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF6D00"))
	spacecraftDim        = lipgloss.NewStyle().Foreground(lipgloss.Color("#BF5600"))
	pathOutboundStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#4DD0E1"))
	pathReturnStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#81C784"))
	trajectoryLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
)

func renderTrajectory(earthDist, moonDist float64, plotW, plotH int, tickCount uint64, showStars bool) string {
	met := mission.MET()
	progress := mission.MissionProgress()

	// String canvas: each cell holds a plain " " or a styled character.
	canvas := make([][]string, plotH)
	for i := range canvas {
		canvas[i] = make([]string, plotW)
		for j := range canvas[i] {
			canvas[i][j] = " "
		}
	}

	// Layer 1: Stars (background). When animation is off, stars are static.
	if showStars {
		placeStars(canvas, plotW, plotH, tickCount)
	} else {
		placeStars(canvas, plotW, plotH, 0)
	}

	// Compute body positions.
	earthX := plotW * 6 / 50
	earthY := plotH / 2
	moonX := plotW - plotW*8/50
	moonY := plotH / 2

	// Layer 2: Trajectory path with outbound/return differentiation.
	pathPoints := generateTrajectoryPath(earthX, earthY, moonX, moonY, plotH)
	for _, p := range pathPoints {
		if p.x >= 0 && p.x < plotW && p.y >= 0 && p.y < plotH {
			if p.segment == 0 {
				canvas[p.y][p.x] = pathOutboundStyle.Render("·")
			} else {
				canvas[p.y][p.x] = pathReturnStyle.Render("∙")
			}
		}
	}

	// Layer 3: Earth and Moon glyphs (overwrite path).
	placeGlyph(canvas, earthX, earthY, plotW, "E", "(", ")", earthGlyphStyle)
	placeGlyph(canvas, moonX, moonY, plotW, "M", "[", "]", moonGlyphStyle)

	// Layer 4: Distance labels.
	if plotH >= 10 {
		placeLabel(canvas, earthX, earthY+2, earthDist, plotW, plotH)
		placeLabel(canvas, moonX, moonY-2, moonDist, plotW, plotH)
	}

	// Layer 5: Spacecraft (highest priority, pulsing).
	totalPoints := len(pathPoints)
	scIdx := int(progress * float64(totalPoints-1))
	if scIdx < 0 {
		scIdx = 0
	}
	if scIdx >= totalPoints {
		scIdx = totalPoints - 1
	}

	isTLIDone := met >= mission.Timeline[8].METOffset
	if !isTLIDone {
		placeSpacecraft(canvas, earthX+2, earthY-1, plotW, plotH, tickCount)
	} else if scIdx >= 0 && scIdx < totalPoints {
		sp := pathPoints[scIdx]
		placeSpacecraft(canvas, sp.x, sp.y, plotW, plotH, tickCount)
	}

	// Layer 6: Legend (bottom-right).
	if plotW >= 40 && plotH >= 8 {
		placeLegend(canvas, plotW, plotH)
	}

	// Render canvas to string.
	var sb strings.Builder
	sb.Grow(plotW * plotH * 10)
	for i, row := range canvas {
		if i > 0 {
			sb.WriteByte('\n')
		}
		for _, cell := range row {
			sb.WriteString(cell)
		}
	}
	return sb.String()
}

func placeStars(canvas [][]string, plotW, plotH int, tickCount uint64) {
	starChars := []string{".", "\u00b7", "+", "\u02d9", "*"}
	starStyles := []lipgloss.Style{starDimStyle, starMedStyle, starBrightStyle}

	seed := uint64(plotW*1000 + plotH)
	numStars := (plotW * plotH) / 25

	for i := 0; i < numStars; i++ {
		seed = seed*6364136223846793005 + 1442695040888963407
		x := int(seed>>33) % plotW
		seed = seed*6364136223846793005 + 1442695040888963407
		y := int(seed>>33) % plotH

		if canvas[y][x] != " " {
			continue
		}

		twinkleHash := tickCount + uint64(x*31+y*97)
		charIdx := int(twinkleHash/2) % len(starChars)
		styleIdx := int(twinkleHash/3) % len(starStyles)
		canvas[y][x] = starStyles[styleIdx].Render(starChars[charIdx])
	}
}

func placeGlyph(canvas [][]string, x, y, plotW int, center, left, right string, style lipgloss.Style) {
	if y < 0 || y >= len(canvas) {
		return
	}
	if x-1 >= 0 {
		canvas[y][x-1] = style.Render(left)
	}
	if x >= 0 && x < plotW {
		canvas[y][x] = style.Render(center)
	}
	if x+1 < plotW {
		canvas[y][x+1] = style.Render(right)
	}
}

func placeSpacecraft(canvas [][]string, x, y, plotW, plotH int, tickCount uint64) {
	if x < 0 || x >= plotW || y < 0 || y >= plotH {
		return
	}
	if tickCount%4 < 2 {
		canvas[y][x] = spacecraftBright.Render("*")
	} else {
		canvas[y][x] = spacecraftDim.Render("+")
	}
}

func placeLabel(canvas [][]string, cx, cy int, dist float64, plotW, plotH int) {
	if dist <= 0 || cy < 0 || cy >= plotH {
		return
	}
	label := formatCompactDist(dist)
	startX := cx - len(label)/2
	for i, ch := range label {
		x := startX + i
		if x >= 0 && x < plotW {
			canvas[cy][x] = trajectoryLabelStyle.Render(string(ch))
		}
	}
}

func formatCompactDist(km float64) string {
	if km >= 1e6 {
		return fmt.Sprintf("%.1fMkm", km/1e6)
	}
	if km >= 1e4 {
		return fmt.Sprintf("%.0fkkm", km/1e3)
	}
	if km >= 1000 {
		return fmt.Sprintf("%.0fkm", km)
	}
	return fmt.Sprintf("%.0fkm", km)
}

func placeLegend(canvas [][]string, plotW, plotH int) {
	items := []struct {
		ch    string
		label string
	}{
		{pathOutboundStyle.Render("·"), " out"},
		{pathReturnStyle.Render("∙"), " ret"},
	}
	startY := plotH - len(items)
	startX := plotW - 6
	if startX < 0 || startY < 0 {
		return
	}
	for i, item := range items {
		y := startY + i
		if y >= plotH {
			break
		}
		canvas[y][startX] = item.ch
		for j, ch := range item.label {
			x := startX + 1 + j
			if x < plotW {
				canvas[y][x] = trajectoryLabelStyle.Render(string(ch))
			}
		}
	}
}

type pathPoint struct {
	x, y    int
	segment int // 0 = outbound, 1 = return
}

func generateTrajectoryPath(ex, ey, mx, my, plotH int) []pathPoint {
	var points []pathPoint

	steps := 80
	outboundArc := -6.0 * float64(plotH) / 16.0
	returnArc := 5.0 * float64(plotH) / 16.0

	for i := 0; i <= steps/2; i++ {
		t := float64(i) / float64(steps/2)
		x := float64(ex) + t*float64(mx-ex)
		yOffset := outboundArc * 4.0 * t * (1.0 - t)
		y := float64(ey) + yOffset
		points = append(points, pathPoint{int(math.Round(x)), int(math.Round(y)), 0})
	}

	for i := 0; i <= steps/2; i++ {
		t := float64(i) / float64(steps/2)
		x := float64(mx) - t*float64(mx-ex)
		yOffset := returnArc * 4.0 * t * (1.0 - t)
		y := float64(my) + yOffset
		points = append(points, pathPoint{int(math.Round(x)), int(math.Round(y)), 1})
	}

	return points
}
