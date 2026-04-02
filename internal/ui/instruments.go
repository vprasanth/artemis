package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Instrument-specific styles (theme-independent).
var (
	gaugeFilledStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4FC3F7"))
	gaugeEmptyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))
	sparklineStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#4DD0E1"))
	compassStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	compassLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#888888"))
	scopeRingStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
)

// renderInstrumentPanel renders the spacecraft instrument panel HUD.
func renderInstrumentPanel(m Model, w, plotH int) string {
	plotW := w - 6
	if plotW < 30 {
		plotW = 30
	}

	plot := renderInstruments(m, plotW, plotH)

	legend := dimStyle.Render("Velocity  Range  Bearing  Signal  Radiation  Proximity") +
		"  " + dimStyle.Render("v: switch view")

	return panelStyle.Width(w - 2).Render(
		panelTitleStyle.Render("INSTRUMENTS") + "  " + legend + "\n" + plot,
	)
}

func renderInstruments(m Model, plotW, plotH int) string {
	// Divide canvas into 2 rows x 3 columns of instrument groups.
	// Each group renders into a sub-canvas that we then blit onto the main canvas.

	canvas := make([][]string, plotH)
	for i := range canvas {
		canvas[i] = make([]string, plotW)
		for j := range canvas[i] {
			canvas[i][j] = " "
		}
	}

	// Column widths: velocity=~35%, range=~30%, bearing=~35%
	col1W := plotW * 35 / 100
	col2W := plotW * 30 / 100
	col3W := plotW - col1W - col2W

	topH := plotH / 2
	botH := plotH - topH

	// --- TOP ROW ---

	// Top-left: Velocity gauge
	velLines := renderVelocityGauge(m, col1W, topH)
	blitLines(canvas, velLines, 0, 0, plotW)

	// Top-center: Range finder
	rangeLines := renderRangeFinder(m, col2W, topH)
	blitLines(canvas, rangeLines, col1W, 0, plotW)

	// Top-right: Bearing compass
	bearingLines := renderBearingDisplay(m, col3W, topH)
	blitLines(canvas, bearingLines, col1W+col2W, 0, plotW)

	// --- BOTTOM ROW ---

	// Bottom-left: Signal health
	signalLines := renderSignalHealth(m, col1W, botH)
	blitLines(canvas, signalLines, 0, topH, plotW)

	// Bottom-center: Radiation
	radLines := renderRadiation(m, col2W, botH)
	blitLines(canvas, radLines, col1W, topH, plotW)

	// Bottom-right: Proximity scope
	proxLines := renderProximityScope(m, col3W, botH)
	blitLines(canvas, proxLines, col1W+col2W, topH, plotW)

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

// blitLines writes pre-rendered lines onto the canvas at the given offset.
func blitLines(canvas [][]string, lines []string, offX, offY int, plotW int) {
	for i, line := range lines {
		y := offY + i
		if y >= len(canvas) {
			break
		}
		// Each character in the line goes to one cell.
		// Since lines may contain ANSI sequences, we render cell-by-cell.
		x := offX
		for _, ch := range line {
			if x >= plotW {
				break
			}
			if x >= 0 && x < len(canvas[y]) {
				canvas[y][x] = string(ch)
			}
			x++
		}
	}
}

// --- Velocity Gauge ---

func renderVelocityGauge(m Model, w, h int) []string {
	lines := make([]string, h)
	if h < 3 || w < 10 {
		return lines
	}

	speed := 0.0
	if m.hzState != nil {
		speed = m.hzState.Speed
	}

	// Title
	lines[0] = gaugeFilledStyle.Render("VELOCITY") + " " + dimStyle.Render("km/s")

	// Horizontal bar gauge (0–2 km/s)
	if h > 1 {
		barW := w - 2
		if barW < 5 {
			barW = 5
		}
		lines[1] = renderHBar(speed, 2.0, barW, gaugeFilledStyle, gaugeEmptyStyle)
	}

	// Speed value
	if h > 2 {
		lines[2] = gaugeFilledStyle.Render(fmt.Sprintf("%.3f", speed)) + " " + dimStyle.Render("km/s") +
			"  " + dimStyle.Render(fmt.Sprintf("(%.0f km/h)", speed*3600))
	}

	// Sparkline from speed history
	if h > 4 && len(m.speedHistory) > 1 {
		sparkW := w - 2
		if sparkW > len(m.speedHistory) {
			sparkW = len(m.speedHistory)
		}
		lines[4] = sparklineStyle.Render(renderSparkline(m.speedHistory, sparkW))

		// Min/Avg/Max
		if h > 5 {
			mn, avg, mx := minAvgMax(m.speedHistory)
			lines[5] = dimStyle.Render(fmt.Sprintf("min:%.3f avg:%.3f max:%.3f", mn, avg, mx))
		}
	}

	return lines
}

// --- Range Finder ---

func renderRangeFinder(m Model, w, h int) []string {
	lines := make([]string, h)
	if h < 3 || w < 10 {
		return lines
	}

	earthDist := 0.0
	moonDist := 0.0
	if m.hzState != nil {
		earthDist = m.hzState.EarthDist
		moonDist = m.hzState.MoonDist
	}

	lines[0] = gaugeFilledStyle.Render("RANGE") + " " + dimStyle.Render("km")

	// Dual vertical bars rendered horizontally
	barH := h - 3
	if barH < 2 {
		barH = 2
	}

	// Earth bar (max ~450,000 km) and Moon bar side by side
	maxRange := 450000.0
	earthFill := earthDist / maxRange
	moonFill := moonDist / maxRange
	if earthFill > 1 {
		earthFill = 1
	}
	if moonFill > 1 {
		moonFill = 1
	}

	barW := (w - 6) / 2
	if barW < 3 {
		barW = 3
	}

	// Render as two horizontal bars
	if h > 1 {
		lines[1] = earthGlyphStyle.Render("E ") + renderHBar(earthDist, maxRange, barW, gaugeFilledStyle, gaugeEmptyStyle)
	}
	if h > 2 {
		lines[2] = dimStyle.Render("  ") + dimStyle.Render(formatCompactDist(earthDist))
	}
	if h > 3 {
		lines[3] = moonGlyphStyle.Render("M ") + renderHBar(moonDist, maxRange, barW, gaugeFilledStyle, gaugeEmptyStyle)
	}
	if h > 4 {
		lines[4] = dimStyle.Render("  ") + dimStyle.Render(formatCompactDist(moonDist))
	}

	// Closing rate indicator
	if h > 5 && len(m.speedHistory) >= 2 {
		latest := m.speedHistory[len(m.speedHistory)-1]
		prev := m.speedHistory[len(m.speedHistory)-2]
		arrow := "─"
		if latest > prev {
			arrow = gaugeFilledStyle.Render("▲")
		} else if latest < prev {
			arrow = gaugeFilledStyle.Render("▼")
		}
		lines[5] = dimStyle.Render("rate ") + arrow
	}

	return lines
}

// --- Bearing Display ---

func renderBearingDisplay(m Model, w, h int) []string {
	lines := make([]string, h)
	if h < 3 || w < 8 {
		return lines
	}

	bearing := 0.0
	if m.hzState != nil {
		bearing = math.Atan2(m.hzState.Position.Y, m.hzState.Position.X) * 180.0 / math.Pi
		if bearing < 0 {
			bearing += 360
		}
	}

	lines[0] = gaugeFilledStyle.Render("BEARING") + " " + dimStyle.Render("ecliptic")

	// Compass rose
	radius := h/2 - 1
	if radius < 2 {
		radius = 2
	}
	if radius > w/4-1 {
		radius = w/4 - 1
	}

	rose := renderCompassRose(bearing, radius)
	for i, row := range rose {
		y := 1 + i
		if y >= h {
			break
		}
		lines[y] = strings.Join(row, "")
	}

	// Heading value at bottom
	if h > 2+len(rose) {
		lines[2+len(rose)] = gaugeFilledStyle.Render(fmt.Sprintf("HDG %.0f°", bearing))
	}

	return lines
}

// --- Signal Health ---

func renderSignalHealth(m Model, w, h int) []string {
	lines := make([]string, h)
	if h < 2 || w < 10 {
		return lines
	}

	lines[0] = gaugeFilledStyle.Render("SIGNAL") + " " + dimStyle.Render("health")

	// AOS/LOS indicator
	isOccluded := false
	if m.hzState != nil {
		isOccluded = m.hzState.IsOccluded()
	}
	if h > 1 {
		if isOccluded {
			lines[1] = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF1744")).Render("LOS") +
				" " + dimStyle.Render("loss of signal")
		} else {
			lines[1] = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#4CAF50")).Render("AOS") +
				" " + dimStyle.Render("signal acquired")
		}
	}

	// Active DSN dish
	if h > 2 {
		dish := "---"
		if m.dsnStatus != nil && len(m.dsnStatus.Dishes) > 0 {
			dish = m.dsnStatus.Dishes[0].Name + " " + m.dsnStatus.Dishes[0].Station
		}
		lines[2] = dimStyle.Render("DSN: ") + dimStyle.Render(dish)
	}

	// RTLT bar
	if h > 3 {
		rtlt := 0.0
		if m.dsnStatus != nil && m.dsnStatus.RTLT > 0 {
			rtlt = m.dsnStatus.RTLT
		}
		barW := w - 12
		if barW < 5 {
			barW = 5
		}
		lines[3] = dimStyle.Render("RTLT ") + renderHBar(rtlt, 10.0, barW, gaugeFilledStyle, gaugeEmptyStyle) +
			" " + dimStyle.Render(fmt.Sprintf("%.1fs", rtlt))
	}

	// Data rate
	if h > 4 && m.dsnStatus != nil && len(m.dsnStatus.Dishes) > 0 {
		rate := 0.0
		for _, ds := range m.dsnStatus.Dishes[0].DownSignals {
			if ds.Active && ds.DataRate > 0 {
				rate = ds.DataRate
				break
			}
		}
		if rate > 0 {
			lines[4] = dimStyle.Render("Rate: ") + dimStyle.Render(formatDataRate(rate))
		} else {
			lines[4] = dimStyle.Render("Rate: ---")
		}
	}

	return lines
}

// --- Radiation ---

func renderRadiation(m Model, w, h int) []string {
	lines := make([]string, h)
	if h < 2 || w < 10 {
		return lines
	}

	lines[0] = gaugeFilledStyle.Render("RADIATION") + " " + dimStyle.Render("env")

	kp := 0.0
	protonFlux := 0.0
	bz := 0.0
	if m.swStatus != nil {
		kp = m.swStatus.Kp
		protonFlux = m.swStatus.ProtonFlux10MeV
		bz = m.swStatus.Bz
	}

	// Kp bar gauge 0–9
	if h > 1 {
		barW := w - 8
		if barW < 5 {
			barW = 5
		}
		kpStyle := gaugeFilledStyle
		if kp >= 7 {
			kpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF5350"))
		} else if kp >= 5 {
			kpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF176"))
		}
		lines[1] = dimStyle.Render("Kp ") + renderHBar(kp, 9.0, barW, kpStyle, gaugeEmptyStyle) +
			" " + kpStyle.Render(fmt.Sprintf("%.0f", kp))
	}

	// Proton flux level
	if h > 2 {
		pStyle := dimStyle
		if protonFlux >= 10 {
			pStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF5350"))
		} else if protonFlux >= 1 {
			pStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF176"))
		}
		lines[2] = dimStyle.Render("p+  ") + pStyle.Render(fmt.Sprintf("%.2f pfu", protonFlux))
	}

	// Bz with direction
	if h > 3 {
		bzDir := "─"
		bzStyle := dimStyle
		if bz < -5 {
			bzDir = "▼S"
			bzStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF176"))
		} else if bz < -10 {
			bzDir = "▼S"
			bzStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF5350"))
		} else if bz > 5 {
			bzDir = "▲N"
		}
		lines[3] = dimStyle.Render("Bz  ") + bzStyle.Render(fmt.Sprintf("%.1f nT ", bz)) + dimStyle.Render(bzDir)
	}

	return lines
}

// --- Proximity Scope ---

func renderProximityScope(m Model, w, h int) []string {
	lines := make([]string, h)
	if h < 3 || w < 8 {
		return lines
	}

	lines[0] = gaugeFilledStyle.Render("PROXIMITY")

	// Radar-style sub-canvas
	radius := h/2 - 1
	if radius < 2 {
		radius = 2
	}
	if radius > w/2-2 {
		radius = w/2 - 2
	}

	scope := renderProximityScopeRings(m, radius)
	for i, row := range scope {
		y := 1 + i
		if y >= h {
			break
		}
		lines[y] = strings.Join(row, "")
	}

	return lines
}

// --- Helper Functions ---

// renderSparkline generates a sparkline string from data using block characters.
func renderSparkline(data []float64, width int) string {
	if len(data) == 0 || width <= 0 {
		return ""
	}

	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// Use only the most recent 'width' entries.
	start := 0
	if len(data) > width {
		start = len(data) - width
	}
	subset := data[start:]

	mn, mx := subset[0], subset[0]
	for _, v := range subset {
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
	}

	rng := mx - mn
	if rng == 0 {
		rng = 1
	}

	var sb strings.Builder
	for _, v := range subset {
		idx := int((v - mn) / rng * float64(len(blocks)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		sb.WriteRune(blocks[idx])
	}
	return sb.String()
}

// renderHBar renders a horizontal bar gauge.
func renderHBar(value, max float64, width int, filled, empty lipgloss.Style) string {
	if max <= 0 || width <= 0 {
		return ""
	}
	ratio := value / max
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	fillW := int(ratio * float64(width))
	emptyW := width - fillW
	return filled.Render(strings.Repeat("▓", fillW)) + empty.Render(strings.Repeat("░", emptyW))
}

// renderCompassRose renders a small compass sub-canvas.
func renderCompassRose(bearing float64, radius int) [][]string {
	size := radius*2 + 1
	// Width doubled for aspect ratio
	widthSize := radius*4 + 1
	rose := make([][]string, size)
	for i := range rose {
		rose[i] = make([]string, widthSize)
		for j := range rose[i] {
			rose[i][j] = " "
		}
	}

	cx := widthSize / 2
	cy := radius

	// Draw circle
	for i := 0; i < 32; i++ {
		angle := 2.0 * math.Pi * float64(i) / 32.0
		x := cx + int(math.Round(float64(radius)*2.0*math.Cos(angle)))
		y := cy + int(math.Round(float64(radius)*math.Sin(angle)))
		if x >= 0 && x < widthSize && y >= 0 && y < size {
			rose[y][x] = compassStyle.Render("·")
		}
	}

	// Cardinal labels
	if cy-radius >= 0 {
		rose[cy-radius][cx] = compassLabelStyle.Render("N")
	}
	if cy+radius < size {
		rose[cy+radius][cx] = compassLabelStyle.Render("S")
	}
	if cx-radius*2 >= 0 {
		rose[cy][cx-radius*2] = compassLabelStyle.Render("W")
	}
	if cx+radius*2 < widthSize {
		rose[cy][cx+radius*2] = compassLabelStyle.Render("E")
	}

	// Heading indicator
	bearingRad := bearing * math.Pi / 180.0
	hx := cx + int(math.Round(float64(radius-1)*2.0*math.Cos(bearingRad-math.Pi/2)))
	hy := cy + int(math.Round(float64(radius-1)*math.Sin(bearingRad-math.Pi/2)))
	if hx >= 0 && hx < widthSize && hy >= 0 && hy < size {
		rose[hy][hx] = spacecraftBright.Render("*")
	}

	// Center crosshair
	rose[cy][cx] = compassStyle.Render("+")

	return rose
}

// renderProximityScopeRings renders radar-style rings centered on spacecraft with Moon plotted.
func renderProximityScopeRings(m Model, radius int) [][]string {
	size := radius*2 + 1
	widthSize := radius*4 + 1
	scope := make([][]string, size)
	for i := range scope {
		scope[i] = make([]string, widthSize)
		for j := range scope[i] {
			scope[i][j] = " "
		}
	}

	cx := widthSize / 2
	cy := radius

	// Draw concentric rings
	for r := 1; r <= radius; r++ {
		for i := 0; i < 24; i++ {
			angle := 2.0 * math.Pi * float64(i) / 24.0
			x := cx + int(math.Round(float64(r)*2.0*math.Cos(angle)))
			y := cy + int(math.Round(float64(r)*math.Sin(angle)))
			if x >= 0 && x < widthSize && y >= 0 && y < size {
				scope[y][x] = scopeRingStyle.Render("·")
			}
		}
	}

	// Crosshairs
	for x := 0; x < widthSize; x++ {
		if scope[cy][x] == " " {
			scope[cy][x] = scopeRingStyle.Render("─")
		}
	}
	for y := 0; y < size; y++ {
		if scope[y][cx] == " " {
			scope[y][cx] = scopeRingStyle.Render("│")
		}
	}

	// Center = spacecraft
	scope[cy][cx] = spacecraftBright.Render("+")

	// Plot Moon at relative bearing/distance
	if m.hzState != nil && m.hzState.MoonDist > 0 {
		moonBearing := math.Atan2(m.hzState.MoonPosition.Y, m.hzState.MoonPosition.X)
		// Normalize distance: full radius = 500,000 km
		normDist := m.hzState.MoonDist / 500000.0
		if normDist > 1 {
			normDist = 1
		}
		moonR := normDist * float64(radius)
		mx := cx + int(math.Round(moonR*2.0*math.Cos(moonBearing)))
		my := cy + int(math.Round(moonR*math.Sin(moonBearing)))
		if mx >= 0 && mx+2 < widthSize && my >= 0 && my < size {
			scope[my][mx] = moonGlyphStyle.Render("[")
			scope[my][mx+1] = moonGlyphStyle.Render("M")
			scope[my][mx+2] = moonGlyphStyle.Render("]")
		}
	}

	return scope
}

// minAvgMax returns the min, average, and max of a slice.
func minAvgMax(data []float64) (float64, float64, float64) {
	if len(data) == 0 {
		return 0, 0, 0
	}
	mn := data[0]
	mx := data[0]
	sum := 0.0
	for _, v := range data {
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
		sum += v
	}
	return mn, sum / float64(len(data)), mx
}
