package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"artemis/internal/horizons"
)

// renderTrajectory projects live Earth-centered Horizons vectors into the
// current Earth-Moon frame. Earth and Moon remain anchored for readability,
// while Orion and its recent trail are plotted from the actual state vectors.
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

	frame := buildTrajectoryFrame(m.hzState, m.positionTrail, plotW, plotH)

	// Layer 2: Earth-Moon reference line.
	drawReferenceLine(canvas, frame.earthX+2, frame.moonX-2, frame.centerY, plotW)

	// Layer 3: Recent trail using live Horizons positions.
	plotTrail(canvas, frame, m.positionTrail, m.hzState, plotW, plotH)

	// Layer 4: Earth and Moon glyphs (overwrite path).
	placeGlyph(canvas, frame.earthX, frame.centerY, plotW, "E", "(", ")", earthGlyphStyle)
	placeGlyph(canvas, frame.moonX, frame.centerY, plotW, "M", "[", "]", moonGlyphStyle)

	// Layer 5: Distance labels.
	if plotH >= 10 {
		earthDist := 0.0
		moonDist := 0.0
		if m.hzState != nil {
			earthDist = m.hzState.EarthDist
			moonDist = m.hzState.MoonDist
		}
		placeLabel(canvas, frame.earthX, frame.centerY+2, earthDist, plotW, plotH)
		placeLabel(canvas, frame.moonX, frame.centerY-2, moonDist, plotW, plotH)
	}

	// Layer 6: Spacecraft (highest priority, pulsing).
	if m.hzState != nil {
		scPoint, ok := frame.project(m.hzState.Position)
		if ok {
			placeSpacecraft(canvas, scPoint.x, scPoint.y, plotW, plotH, m.tickCount, m.hzState.IsOccluded())
		}
	}

	// Layer 7: Legend (bottom-right).
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

type trajectoryFrame struct {
	axisX   horizons.Vector3
	axisY   horizons.Vector3
	earthX  int
	moonX   int
	centerY int
	scaleX  float64
	scaleY  float64
}

func buildTrajectoryFrame(state *horizons.State, trail []horizons.Vector3, plotW, plotH int) trajectoryFrame {
	earthX := plotW * 6 / 50
	moonX := plotW - plotW*8/50
	centerY := plotH / 2

	axisX := horizons.Vector3{X: 1, Y: 0}
	axisY := horizons.Vector3{X: 0, Y: 1}
	moonDist := 384400.0
	if moonVec, ok := earthMoonVector(state); ok {
		moonDist = math.Hypot(moonVec.X, moonVec.Y)
		if moonDist > 0 {
			axisX = horizons.Vector3{X: moonVec.X / moonDist, Y: moonVec.Y / moonDist}
			axisY = horizons.Vector3{X: -axisX.Y, Y: axisX.X}
		}
	}
	if moonDist <= 0 {
		moonDist = 384400.0
	}

	scaleX := moonDist / float64(maxInt(1, moonX-earthX))
	maxCross := moonDist * 0.05
	for _, pos := range trail {
		_, cross := projectEarthMoonFrame(pos, axisX, axisY)
		if absFloat(cross) > maxCross {
			maxCross = absFloat(cross)
		}
	}
	if state != nil {
		_, cross := projectEarthMoonFrame(state.Position, axisX, axisY)
		if absFloat(cross) > maxCross {
			maxCross = absFloat(cross)
		}
	}
	halfHeight := maxInt(2, plotH/2-2)
	scaleY := maxCross * 1.15 / float64(halfHeight)
	if scaleY <= 0 {
		scaleY = 1
	}

	return trajectoryFrame{
		axisX:   axisX,
		axisY:   axisY,
		earthX:  earthX,
		moonX:   moonX,
		centerY: centerY,
		scaleX:  scaleX,
		scaleY:  scaleY,
	}
}

func (f trajectoryFrame) project(pos horizons.Vector3) (pathPoint, bool) {
	along, cross := projectEarthMoonFrame(pos, f.axisX, f.axisY)
	x := f.earthX + int(math.Round(along/f.scaleX))
	y := f.centerY - int(math.Round(cross/f.scaleY))
	return pathPoint{x: x, y: y}, x >= 0 && y >= 0
}

func plotTrail(canvas [][]string, frame trajectoryFrame, trail []horizons.Vector3, state *horizons.State, plotW, plotH int) {
	positions := append([]horizons.Vector3{}, trail...)
	if state != nil {
		if len(positions) == 0 || positions[len(positions)-1] != state.Position {
			positions = append(positions, state.Position)
		}
	}
	if len(positions) < 2 {
		return
	}

	for i := 1; i < len(positions); i++ {
		start, startOK := frame.project(positions[i-1])
		end, endOK := frame.project(positions[i])
		if !startOK || !endOK {
			continue
		}
		if start.x < 0 || start.x >= plotW || start.y < 0 || start.y >= plotH {
			continue
		}
		if end.x < 0 || end.x >= plotW || end.y < 0 || end.y >= plotH {
			continue
		}

		style := pathOutboundStyle
		ch := "·"
		if positions[i].Magnitude() < positions[i-1].Magnitude() {
			style = pathReturnStyle
			ch = "∙"
		}
		drawTrailSegment(canvas, start.x, start.y, end.x, end.y, plotW, plotH, style, ch)
	}
}

func drawReferenceLine(canvas [][]string, startX, endX, y, plotW int) {
	if y < 0 || y >= len(canvas) {
		return
	}
	if startX < 0 {
		startX = 0
	}
	if endX > plotW {
		endX = plotW
	}
	for x := startX; x < endX; x++ {
		if canvas[y][x] == " " {
			canvas[y][x] = scaleRingStyle.Render("─")
		}
	}
}

func drawTrailSegment(canvas [][]string, x0, y0, x1, y1, plotW, plotH int, style lipgloss.Style, ch string) {
	steps := maxInt(absInt(x1-x0), absInt(y1-y0))
	if steps == 0 {
		if x0 >= 0 && x0 < plotW && y0 >= 0 && y0 < plotH {
			canvas[y0][x0] = style.Render(ch)
		}
		return
	}

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Round(float64(x0) + (float64(x1-x0) * t)))
		y := int(math.Round(float64(y0) + (float64(y1-y0) * t)))
		if x >= 0 && x < plotW && y >= 0 && y < plotH {
			canvas[y][x] = style.Render(ch)
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

func projectEarthMoonFrame(pos, axisX, axisY horizons.Vector3) (float64, float64) {
	along := pos.X*axisX.X + pos.Y*axisX.Y
	cross := pos.X*axisY.X + pos.Y*axisY.Y
	return along, cross
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
