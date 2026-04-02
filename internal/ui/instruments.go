package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderInstrumentPanel renders the spacecraft instrument panel HUD.
func renderInstrumentPanel(m Model, w, plotH int) string {
	plotW := w - 6
	if plotW < 30 {
		plotW = 30
	}

	plot := renderInstruments(m, plotW, plotH)

	legend := dimStyle.Render("v: switch view")

	return panelStyle.Width(w - 2).Render(
		panelTitleStyle.Render("INSTRUMENTS") + "  " + legend + "\n" + plot,
	)
}

func renderInstruments(m Model, plotW, plotH int) string {
	// Column widths for the 2x3 grid.
	col1W := plotW * 38 / 100
	col2W := plotW * 30 / 100
	col3W := plotW - col1W - col2W

	topH := plotH / 2
	botH := plotH - topH

	// Each instrument returns a multi-line string at a fixed size.
	// Use lipgloss to compose into a grid.

	vel := renderVelocityGauge(m, col1W, topH)
	rng := renderRangeFinder(m, col2W, topH)
	bearing := renderBearingDisplay(m, col3W, topH)

	sig := renderSignalHealth(m, col1W, botH)
	rad := renderRadiation(m, col2W, botH)
	prox := renderProximityScope(m, col3W, botH)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, vel, rng, bearing)
	botRow := lipgloss.JoinHorizontal(lipgloss.Top, sig, rad, prox)

	return lipgloss.JoinVertical(lipgloss.Left, topRow, botRow)
}

// --- Velocity Gauge ---

func renderVelocityGauge(m Model, w, h int) string {
	style := lipgloss.NewStyle().Width(w).Height(h)
	if h < 3 || w < 10 {
		return style.Render("")
	}

	speed := 0.0
	if m.hzState != nil {
		speed = m.hzState.Speed
	}

	var lines []string

	// Title
	lines = append(lines, instTitleStyle.Render("VELOCITY")+" "+dimStyle.Render("km/s"))

	// Horizontal bar gauge (0–2 km/s)
	barW := w - 4
	if barW < 5 {
		barW = 5
	}
	lines = append(lines, renderHBar(speed, 2.0, barW, gaugeFilledStyle, gaugeEmptyStyle))

	// Speed value
	lines = append(lines, gaugeFilledStyle.Render(fmt.Sprintf("%.3f", speed))+" "+dimStyle.Render("km/s")+
		"  "+dimStyle.Render(fmt.Sprintf("(%.0f km/h)", speed*3600)))

	// Blank separator
	lines = append(lines, "")

	// Sparkline from speed history
	if len(m.speedHistory) > 1 {
		sparkW := w - 4
		if sparkW > len(m.speedHistory) {
			sparkW = len(m.speedHistory)
		}
		lines = append(lines, sparklineStyle.Render(renderSparkline(m.speedHistory, sparkW)))

		mn, avg, mx := minAvgMax(m.speedHistory)
		lines = append(lines, dimStyle.Render(fmt.Sprintf("min:%.3f avg:%.3f max:%.3f", mn, avg, mx)))
	}

	return style.Render(strings.Join(lines, "\n"))
}

// --- Range Finder ---

func renderRangeFinder(m Model, w, h int) string {
	style := lipgloss.NewStyle().Width(w).Height(h)
	if h < 3 || w < 10 {
		return style.Render("")
	}

	earthDist := 0.0
	moonDist := 0.0
	if m.hzState != nil {
		earthDist = m.hzState.EarthDist
		moonDist = m.hzState.MoonDist
	}

	var lines []string

	lines = append(lines, instTitleStyle.Render("RANGE")+" "+dimStyle.Render("km"))

	maxRange := 450000.0
	barW := w - 6
	if barW < 3 {
		barW = 3
	}

	// Earth distance bar
	lines = append(lines, earthGlyphStyle.Render("E ")+renderHBar(earthDist, maxRange, barW, gaugeFilledStyle, gaugeEmptyStyle))
	lines = append(lines, "  "+dimStyle.Render(formatCompactDist(earthDist)))

	// Moon distance bar
	lines = append(lines, moonGlyphStyle.Render("M ")+renderHBar(moonDist, maxRange, barW, gaugeFilledStyle, gaugeEmptyStyle))
	lines = append(lines, "  "+dimStyle.Render(formatCompactDist(moonDist)))

	// Closing rate indicator
	if len(m.speedHistory) >= 2 {
		latest := m.speedHistory[len(m.speedHistory)-1]
		prev := m.speedHistory[len(m.speedHistory)-2]
		arrow := dimStyle.Render("─")
		if latest > prev {
			arrow = gaugeFilledStyle.Render("▲")
		} else if latest < prev {
			arrow = gaugeFilledStyle.Render("▼")
		}
		lines = append(lines, dimStyle.Render("rate ")+arrow)
	}

	return style.Render(strings.Join(lines, "\n"))
}

// --- Bearing Display ---

func renderBearingDisplay(m Model, w, h int) string {
	style := lipgloss.NewStyle().Width(w).Height(h)
	if h < 3 || w < 8 {
		return style.Render("")
	}

	bearing := 0.0
	if m.hzState != nil {
		bearing = math.Atan2(m.hzState.Position.Y, m.hzState.Position.X) * 180.0 / math.Pi
		if bearing < 0 {
			bearing += 360
		}
	}

	var lines []string
	lines = append(lines, instTitleStyle.Render("BEARING")+" "+dimStyle.Render("ecliptic"))

	// Compass rose rendered into a [][]string canvas, then joined.
	radius := (h - 3) / 2
	if radius < 2 {
		radius = 2
	}
	maxR := (w - 2) / 4
	if maxR < 2 {
		maxR = 2
	}
	if radius > maxR {
		radius = maxR
	}

	rose := renderCompassRose(bearing, radius)
	for _, row := range rose {
		lines = append(lines, joinCanvasRow(row))
	}

	lines = append(lines, gaugeFilledStyle.Render(fmt.Sprintf("HDG %.0f°", bearing)))

	return style.Render(strings.Join(lines, "\n"))
}

// --- Signal Health ---

func renderSignalHealth(m Model, w, h int) string {
	style := lipgloss.NewStyle().Width(w).Height(h)
	if h < 2 || w < 10 {
		return style.Render("")
	}

	var lines []string

	lines = append(lines, instTitleStyle.Render("SIGNAL")+" "+dimStyle.Render("health"))

	// AOS/LOS indicator
	isOccluded := false
	if m.hzState != nil {
		isOccluded = m.hzState.IsOccluded()
	}
	if isOccluded {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF1744")).Render("LOS")+
			" "+dimStyle.Render("loss of signal"))
	} else {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#4CAF50")).Render("AOS")+
			" "+dimStyle.Render("signal acquired"))
	}

	// Active DSN dish
	dish := "---"
	if m.dsnStatus != nil && len(m.dsnStatus.Dishes) > 0 {
		dish = m.dsnStatus.Dishes[0].Name + " " + m.dsnStatus.Dishes[0].Station
	}
	lines = append(lines, dimStyle.Render("DSN: ")+dimStyle.Render(dish))

	// RTLT bar
	rtlt := 0.0
	if m.dsnStatus != nil && m.dsnStatus.RTLT > 0 {
		rtlt = m.dsnStatus.RTLT
	}
	barW := w - 14
	if barW < 5 {
		barW = 5
	}
	lines = append(lines, dimStyle.Render("RTLT ")+renderHBar(rtlt, 10.0, barW, gaugeFilledStyle, gaugeEmptyStyle)+
		" "+dimStyle.Render(fmt.Sprintf("%.1fs", rtlt)))

	// Data rate
	if m.dsnStatus != nil && len(m.dsnStatus.Dishes) > 0 {
		rate := 0.0
		for _, ds := range m.dsnStatus.Dishes[0].DownSignals {
			if ds.Active && ds.DataRate > 0 {
				rate = ds.DataRate
				break
			}
		}
		if rate > 0 {
			lines = append(lines, dimStyle.Render("Rate: ")+dimStyle.Render(formatDataRate(rate)))
		} else {
			lines = append(lines, dimStyle.Render("Rate: ---"))
		}
	} else {
		lines = append(lines, dimStyle.Render("Rate: ---"))
	}

	return style.Render(strings.Join(lines, "\n"))
}

// --- Radiation ---

func renderRadiation(m Model, w, h int) string {
	style := lipgloss.NewStyle().Width(w).Height(h)
	if h < 2 || w < 10 {
		return style.Render("")
	}

	var lines []string

	lines = append(lines, instTitleStyle.Render("RADIATION")+" "+dimStyle.Render("env"))

	kp := 0.0
	protonFlux := 0.0
	bz := 0.0
	if m.swStatus != nil {
		kp = m.swStatus.Kp
		protonFlux = m.swStatus.ProtonFlux10MeV
		bz = m.swStatus.Bz
	}

	// Kp bar gauge 0–9
	barW := w - 10
	if barW < 5 {
		barW = 5
	}
	kpStyle := gaugeFilledStyle
	if kp >= 7 {
		kpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF5350"))
	} else if kp >= 5 {
		kpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF176"))
	}
	lines = append(lines, dimStyle.Render("Kp ")+renderHBar(kp, 9.0, barW, kpStyle, gaugeEmptyStyle)+
		" "+kpStyle.Render(fmt.Sprintf("%.0f", kp)))

	// Proton flux level
	pStyle := dimStyle
	if protonFlux >= 10 {
		pStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF5350"))
	} else if protonFlux >= 1 {
		pStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF176"))
	}
	lines = append(lines, dimStyle.Render("p+  ")+pStyle.Render(fmt.Sprintf("%.2f pfu", protonFlux)))

	// Bz with direction — check more severe threshold first
	bzDir := "─"
	bzStyle := dimStyle
	if bz < -10 {
		bzDir = "▼S"
		bzStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF5350"))
	} else if bz < -5 {
		bzDir = "▼S"
		bzStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF176"))
	} else if bz > 5 {
		bzDir = "▲N"
	}
	lines = append(lines, dimStyle.Render("Bz  ")+bzStyle.Render(fmt.Sprintf("%.1f nT ", bz))+dimStyle.Render(bzDir))

	return style.Render(strings.Join(lines, "\n"))
}

// --- Proximity Scope ---

func renderProximityScope(m Model, w, h int) string {
	style := lipgloss.NewStyle().Width(w).Height(h)
	if h < 3 || w < 8 {
		return style.Render("")
	}

	var lines []string
	lines = append(lines, instTitleStyle.Render("PROXIMITY"))

	// Radar-style sub-canvas
	radius := (h - 2) / 2
	if radius < 2 {
		radius = 2
	}
	maxR := (w - 2) / 4
	if maxR < 2 {
		maxR = 2
	}
	if radius > maxR {
		radius = maxR
	}

	scope := renderProximityScopeCanvas(m, radius)
	for _, row := range scope {
		lines = append(lines, joinCanvasRow(row))
	}

	return style.Render(strings.Join(lines, "\n"))
}

// --- Helper Functions ---

// joinCanvasRow joins a []string canvas row where each cell is already styled.
func joinCanvasRow(row []string) string {
	var sb strings.Builder
	for _, cell := range row {
		sb.WriteString(cell)
	}
	return sb.String()
}

// renderSparkline generates a sparkline string from data using block characters.
func renderSparkline(data []float64, width int) string {
	if len(data) == 0 || width <= 0 {
		return ""
	}

	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

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

// renderCompassRose renders a small compass into a [][]string canvas.
func renderCompassRose(bearing float64, radius int) [][]string {
	size := radius*2 + 1
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
	steps := 32
	if radius > 3 {
		steps = 48
	}
	for i := 0; i < steps; i++ {
		angle := 2.0 * math.Pi * float64(i) / float64(steps)
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

	// Heading indicator (bearing measured from N clockwise, so offset by -π/2)
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

// renderProximityScopeCanvas renders radar-style rings with Moon plotted.
func renderProximityScopeCanvas(m Model, radius int) [][]string {
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

	// Draw concentric rings (every other radius for less clutter)
	for r := 1; r <= radius; r++ {
		steps := 24
		if r > 2 {
			steps = 36
		}
		for i := 0; i < steps; i++ {
			angle := 2.0 * math.Pi * float64(i) / float64(steps)
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
