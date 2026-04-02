package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderOrbitalPanel renders the top-down Earth-Moon system map.
func renderOrbitalPanel(m Model, w, plotH int) string {
	plotW := w - 6
	if plotW < 30 {
		plotW = 30
	}

	plot := renderOrbitalMap(m, plotW, plotH)

	legend := earthGlyphStyle.Render("(E)") + dimStyle.Render("=Earth  ") +
		moonGlyphStyle.Render("{M}") + dimStyle.Render("=Moon  ") +
		spacecraftBright.Render("*") + dimStyle.Render("=Orion  ") +
		dimStyle.Render("v: switch view")

	return panelStyle.Width(w - 2).Render(
		panelTitleStyle.Render("ORBITAL CONTEXT") + "  " + legend + "\n" + plot,
	)
}

func renderOrbitalMap(m Model, plotW, plotH int) string {
	// String canvas.
	canvas := make([][]string, plotH)
	for i := range canvas {
		canvas[i] = make([]string, plotW)
		for j := range canvas[i] {
			canvas[i][j] = " "
		}
	}

	// Layer 1: Stars background.
	if m.showStars {
		placeStars(canvas, plotW, plotH, m.tickCount)
	} else {
		placeStars(canvas, plotW, plotH, 0)
	}

	// Canvas center = Earth position.
	cx := plotW / 2
	cy := plotH / 2

	// Scale: fit Moon's orbit (~384,400 km) within canvas.
	// Use the smaller dimension (accounting for aspect ratio) to determine scale.
	const moonOrbitKm = 384400.0
	const aspectRatio = 0.5 // terminal chars are ~2x taller than wide

	maxRadiusX := float64(plotW)/2.0 - 4.0
	maxRadiusY := (float64(plotH)/2.0 - 2.0) / aspectRatio
	maxRadius := maxRadiusX
	if maxRadiusY < maxRadius {
		maxRadius = maxRadiusY
	}
	if maxRadius < 5 {
		maxRadius = 5
	}

	// scale: km per canvas column
	scale := moonOrbitKm / (maxRadius * 0.85)

	// Layer 2: Distance scale rings at 100k, 200k, 300k, 400k km.
	scaleDistances := []float64{100000, 200000, 300000, 400000}
	for _, dist := range scaleDistances {
		r := dist / scale
		if r > 2 && r < maxRadius+2 {
			drawCircle(canvas, cx, cy, r, aspectRatio, ".", scaleRingStyle, plotW, plotH)
		}
	}

	// Scale labels.
	for _, dist := range scaleDistances {
		r := dist / scale
		labelX := cx + int(r) + 1
		label := fmt.Sprintf("%dk", int(dist/1000))
		if labelX+len(label) < plotW && labelX > 0 {
			for i, ch := range label {
				x := labelX + i
				if x >= 0 && x < plotW {
					canvas[cy][x] = scaleLabelStyle.Render(string(ch))
				}
			}
		}
	}

	// Layer 3: Moon's orbit ring (dotted).
	moonOrbitR := moonOrbitKm / scale
	if moonOrbitR > 2 {
		drawCircle(canvas, cx, cy, moonOrbitR, aspectRatio, "·", orbitRingStyle, plotW, plotH)
	}

	// Layer 4: Earth glyph at center.
	placeGlyph(canvas, cx, cy, plotW, "E", "(", ")", earthGlyphStyle)

	// Layer 5: Moon at actual angular position.
	moonAngle := 0.0
	moonDist := moonOrbitKm
	if m.hzState != nil {
		// Moon Earth-centered position = SC_earth - SC_moon (same as IsOccluded logic).
		moonEX := m.hzState.Position.X - m.hzState.MoonPosition.X
		moonEY := m.hzState.Position.Y - m.hzState.MoonPosition.Y
		moonAngle = math.Atan2(moonEY, moonEX)
		moonDist = math.Sqrt(moonEX*moonEX + moonEY*moonEY)
	}
	moonCX, moonCY := worldToCanvas(
		moonDist*math.Cos(moonAngle),
		moonDist*math.Sin(moonAngle),
		cx, cy, scale, aspectRatio,
	)
	if moonCX >= 0 && moonCX < plotW && moonCY >= 0 && moonCY < plotH {
		placeGlyph(canvas, moonCX, moonCY, plotW, "M", "{", "}", moonGlyphStyle)
	}

	// Layer 6: Spacecraft position trail.
	for i, pos := range m.positionTrail {
		tx, ty := worldToCanvas(pos.X, pos.Y, cx, cy, scale, aspectRatio)
		if tx >= 0 && tx < plotW && ty >= 0 && ty < plotH && canvas[ty][tx] == " " {
			if i < len(m.positionTrail)/2 {
				canvas[ty][tx] = trailDimStyle.Render("·")
			} else {
				canvas[ty][tx] = trailStyle.Render("·")
			}
		}
	}

	// Layer 7: Spacecraft at current position (pulsing).
	if m.hzState != nil {
		scX, scY := worldToCanvas(m.hzState.Position.X, m.hzState.Position.Y, cx, cy, scale, aspectRatio)
		if scX >= 0 && scX < plotW && scY >= 0 && scY < plotH {
			placeSpacecraft(canvas, scX, scY, plotW, plotH, m.tickCount, m.hzState.IsOccluded())
		}
	}

	// Layer 8: Info line at bottom.
	if plotH >= 3 {
		info := buildOrbitalInfo(m, moonAngle)
		infoY := plotH - 1
		startX := cx - len(info)/2
		if startX < 1 {
			startX = 1
		}
		for i, ch := range info {
			x := startX + i
			if x >= 0 && x < plotW {
				canvas[infoY][x] = scaleLabelStyle.Render(string(ch))
			}
		}
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

// drawCircle plots a styled character in a circle with aspect ratio correction.
func drawCircle(canvas [][]string, cx, cy int, radius, aspect float64, ch string, style lipgloss.Style, plotW, plotH int) {
	steps := int(radius * 4)
	if steps < 24 {
		steps = 24
	}
	if steps > 120 {
		steps = 120
	}
	for i := 0; i < steps; i++ {
		angle := 2.0 * math.Pi * float64(i) / float64(steps)
		x := cx + int(math.Round(radius*math.Cos(angle)))
		y := cy + int(math.Round(radius*math.Sin(angle)*aspect))
		if x >= 0 && x < plotW && y >= 0 && y < plotH {
			if canvas[y][x] == " " {
				canvas[y][x] = style.Render(ch)
			}
		}
	}
}

// worldToCanvas converts world coordinates (km) to canvas coordinates.
func worldToCanvas(wx, wy float64, cx, cy int, scale, aspect float64) (int, int) {
	x := cx + int(math.Round(wx/scale))
	y := cy + int(math.Round(wy/scale*aspect))
	return x, y
}

func buildOrbitalInfo(m Model, moonAngle float64) string {
	earthDist := "---"
	moonDist := "---"
	if m.hzState != nil {
		earthDist = formatCompactDist(m.hzState.EarthDist)
		moonDist = formatCompactDist(m.hzState.MoonDist)
	}

	phaseDeg := moonAngle * 180.0 / math.Pi
	if phaseDeg < 0 {
		phaseDeg += 360
	}
	phase := fmt.Sprintf("%.0f°", phaseDeg)

	return fmt.Sprintf("Earth: %s  Moon: %s  Phase: %s", earthDist, moonDist, phase)
}
