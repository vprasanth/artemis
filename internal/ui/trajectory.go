package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"artemis/internal/horizons"
)

// renderTrajectory fits the Earth-centered Horizons mission arc into the panel.
// Earth stays at the origin, while the Moon, Orion, and the sampled path are
// projected from the same ecliptic XY coordinates so the visible trace matches
// the fetched trajectory data.
func renderTrajectory(m Model, plotW, plotH int) string {
	// String canvas: each cell holds a plain " " or a styled character.
	canvas := make([][]string, plotH)
	for i := range canvas {
		canvas[i] = make([]string, plotW)
		for j := range canvas[i] {
			canvas[i][j] = " "
		}
	}

	// Layer 1: Stars (background). When animation is off, stars are static.
	if m.showStars {
		placeStars(canvas, plotW, plotH, m.tickCount)
	} else {
		placeStars(canvas, plotW, plotH, 0)
	}

	path := trajectoryPositions(m)
	frame := buildTrajectoryFrame(m.hzState, path, plotW, plotH)

	// Layer 2: Sampled mission path using Earth-centered Horizons positions.
	plotPath(canvas, frame, path, plotW, plotH)

	// Layer 3: Earth and Moon glyphs (overwrite path).
	earthPoint := frame.project(horizons.Vector3{})
	placeGlyph(canvas, earthPoint.x, earthPoint.y, plotW, "E", "(", ")", earthGlyphStyle)
	if moonVec, ok := earthMoonVector(m.hzState); ok {
		moonPoint := frame.project(moonVec)
		placeGlyph(canvas, moonPoint.x, moonPoint.y, plotW, "M", "[", "]", moonGlyphStyle)
	}

	// Layer 4: Distance labels.
	if plotH >= 10 {
		earthDist := effectiveEarthDist(m)
		moonDist := 0.0
		if m.hzState != nil {
			moonDist = m.hzState.MoonDist
		}
		placeLabel(canvas, earthPoint.x, earthPoint.y+2, earthDist, plotW, plotH)
		if moonVec, ok := earthMoonVector(m.hzState); ok {
			moonPoint := frame.project(moonVec)
			placeLabel(canvas, moonPoint.x, moonPoint.y-2, moonDist, plotW, plotH)
		}
	}

	// Layer 5: Spacecraft (highest priority, pulsing).
	if m.hzState != nil {
		scPoint := frame.project(m.hzState.Position)
		placeSpacecraft(canvas, scPoint.x, scPoint.y, plotW, plotH, m.tickCount, m.hzState.IsOccluded())
	}

	// Layer 6: Legend (bottom-right).
	if plotW >= 40 && plotH >= 8 {
		placeLegend(canvas, plotW, plotH)
	}
	if plotH >= 2 {
		placeTrajectoryStatus(canvas, trajectoryPathStatus(m), plotW, plotH)
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

func trajectoryPositions(m Model) []horizons.Vector3 {
	if len(m.trajectoryPath) > 1 {
		return append([]horizons.Vector3(nil), m.trajectoryPath...)
	}

	positions := append([]horizons.Vector3{}, m.positionTrail...)
	if m.hzState != nil {
		if len(positions) == 0 || positions[len(positions)-1] != m.hzState.Position {
			positions = append(positions, m.hzState.Position)
		}
	}
	return positions
}

func trajectoryPathStatus(m Model) string {
	switch {
	case len(m.trajectoryPath) > 1:
		return fmt.Sprintf("arc %d samples", len(m.trajectoryPath))
	case m.hzPathLoading:
		return "arc loading"
	case m.hzPathErr != nil:
		return "arc unavailable"
	case len(trajectoryPositions(m)) > 1:
		return "arc live trail"
	default:
		return "arc waiting"
	}
}

type trajectoryFrame struct {
	centerX      int
	centerY      int
	worldCenterX float64
	worldCenterY float64
	scale        float64
	aspect       float64
}

func buildTrajectoryFrame(state *horizons.State, trail []horizons.Vector3, plotW, plotH int) trajectoryFrame {
	const (
		aspectRatio = 0.5
		defaultSpan = 384400.0
	)

	minX, maxX := 0.0, 0.0
	minY, maxY := 0.0, 0.0
	initialized := false
	include := func(pos horizons.Vector3) {
		if !initialized {
			minX, maxX = pos.X, pos.X
			minY, maxY = pos.Y, pos.Y
			initialized = true
			return
		}
		minX = math.Min(minX, pos.X)
		maxX = math.Max(maxX, pos.X)
		minY = math.Min(minY, pos.Y)
		maxY = math.Max(maxY, pos.Y)
	}

	include(horizons.Vector3{})
	for _, pos := range trail {
		include(pos)
	}
	if state != nil {
		include(state.Position)
		if moonVec, ok := earthMoonVector(state); ok {
			include(moonVec)
		}
	}

	if !initialized {
		maxX = defaultSpan
	}

	spanX := maxX - minX
	spanY := maxY - minY
	if spanX < defaultSpan*0.1 {
		pad := defaultSpan * 0.05
		minX -= pad
		maxX += pad
		spanX = maxX - minX
	}
	if spanY < defaultSpan*0.1 {
		pad := defaultSpan * 0.05
		minY -= pad
		maxY += pad
		spanY = maxY - minY
	}

	padX := math.Max(defaultSpan*0.03, spanX*0.08)
	padY := math.Max(defaultSpan*0.03, spanY*0.08)
	minX -= padX
	maxX += padX
	minY -= padY
	maxY += padY

	usableW := float64(maxInt(8, plotW-6))
	usableH := float64(maxInt(6, plotH-4))
	scaleX := (maxX - minX) / usableW
	scaleY := (maxY - minY) / (usableH / aspectRatio)
	scale := math.Max(scaleX, scaleY)
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}

	return trajectoryFrame{
		centerX:      plotW / 2,
		centerY:      plotH / 2,
		worldCenterX: (minX + maxX) / 2,
		worldCenterY: (minY + maxY) / 2,
		scale:        scale,
		aspect:       aspectRatio,
	}
}

func (f trajectoryFrame) project(pos horizons.Vector3) pathPoint {
	x := f.centerX + int(math.Round((pos.X-f.worldCenterX)/f.scale))
	y := f.centerY - int(math.Round((pos.Y-f.worldCenterY)/f.scale*f.aspect))
	return pathPoint{x: x, y: y}
}

func plotPath(canvas [][]string, frame trajectoryFrame, positions []horizons.Vector3, plotW, plotH int) {
	if len(positions) < 2 {
		return
	}

	for i := 1; i < len(positions); i++ {
		start := frame.project(positions[i-1])
		end := frame.project(positions[i])

		style := pathOutboundStyle
		ch := "·"
		if positions[i].Magnitude() < positions[i-1].Magnitude() {
			style = pathReturnStyle
			ch = "∙"
		}
		drawTrailSegment(canvas, start.x, start.y, end.x, end.y, plotW, plotH, style, ch)
	}
}

func drawTrailSegment(canvas [][]string, x0, y0, x1, y1, plotW, plotH int, style lipgloss.Style, ch string) {
	steps := maxInt(absInt(x1-x0), absInt(y1-y0))
	lineCh := segmentGlyph(x0, y0, x1, y1, ch)
	if steps == 0 {
		if x0 >= 0 && x0 < plotW && y0 >= 0 && y0 < plotH {
			canvas[y0][x0] = style.Render(lineCh)
		}
		return
	}

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Round(float64(x0) + (float64(x1-x0) * t)))
		y := int(math.Round(float64(y0) + (float64(y1-y0) * t)))
		if x >= 0 && x < plotW && y >= 0 && y < plotH {
			canvas[y][x] = style.Render(lineCh)
		}
	}
}

func earthMoonVector(state *horizons.State) (horizons.Vector3, bool) {
	if state == nil {
		return horizons.Vector3{}, false
	}

	moonVec := horizons.Vector3{
		X: state.Position.X - state.MoonPosition.X,
		Y: state.Position.Y - state.MoonPosition.Y,
		Z: state.Position.Z - state.MoonPosition.Z,
	}
	if math.Hypot(moonVec.X, moonVec.Y) == 0 {
		return horizons.Vector3{}, false
	}
	return moonVec, true
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

func placeSpacecraft(canvas [][]string, x, y, plotW, plotH int, tickCount uint64, occluded bool) {
	if x < 0 || x >= plotW || y < 0 || y >= plotH {
		return
	}
	if occluded {
		if tickCount%4 < 2 {
			canvas[y][x] = spacecraftLOS.Render("*")
		} else {
			canvas[y][x] = spacecraftLOSDim.Render("+")
		}
	} else {
		if tickCount%4 < 2 {
			canvas[y][x] = spacecraftBright.Render("*")
		} else {
			canvas[y][x] = spacecraftDim.Render("+")
		}
	}
}

func segmentGlyph(x0, y0, x1, y1 int, fallback string) string {
	dx := x1 - x0
	dy := y1 - y0

	switch {
	case dx == 0 && dy == 0:
		return fallback
	case absInt(dy) > absInt(dx):
		return "│"
	case dy == 0:
		return "─"
	case (dx > 0 && dy > 0) || (dx < 0 && dy < 0):
		return "╲"
	default:
		return "╱"
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
		return fmt.Sprintf("%.1fM km", km/1e6)
	}
	if km >= 1000 {
		return fmt.Sprintf("%.0fk km", km/1e3)
	}
	return fmt.Sprintf("%.0f km", km)
}

func placeLegend(canvas [][]string, plotW, plotH int) {
	items := []struct {
		ch    string
		label string
	}{
		{pathOutboundStyle.Render("─"), " out"},
		{pathReturnStyle.Render("─"), " ret"},
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

func placeTrajectoryStatus(canvas [][]string, status string, plotW, plotH int) {
	if status == "" || plotH <= 0 {
		return
	}
	y := plotH - 1
	if y < 0 || y >= len(canvas) {
		return
	}
	startX := 1
	maxWidth := plotW - 10
	if maxWidth <= startX {
		return
	}
	for _, ch := range status {
		if startX >= maxWidth {
			break
		}
		canvas[y][startX] = trajectoryLabelStyle.Render(string(ch))
		startX++
	}
}

type pathPoint struct {
	x, y int
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
