package ui

import "github.com/charmbracelet/lipgloss"

// ThemeID identifies a color theme.
type ThemeID int

const (
	ThemeDefault         ThemeID = iota
	ThemeRetro                   // Amber/green phosphor terminal
	ThemeHighContrast            // High-visibility
	ThemeMissionCritical         // Dark red
	themeCount                   // sentinel for cycling
)

// Theme holds the full color palette.
type Theme struct {
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color
	Dim       lipgloss.Color
	Bright    lipgloss.Color
	Muted     lipgloss.Color
	Green     lipgloss.Color
	Red       lipgloss.Color
	Yellow    lipgloss.Color
	Cyan      lipgloss.Color
}

var themes = map[ThemeID]Theme{
	ThemeDefault: {
		Primary:   lipgloss.Color("#4FC3F7"),
		Secondary: lipgloss.Color("#81C784"),
		Accent:    lipgloss.Color("#FFB74D"),
		Dim:       lipgloss.Color("#666666"),
		Bright:    lipgloss.Color("#FFFFFF"),
		Muted:     lipgloss.Color("#888888"),
		Green:     lipgloss.Color("#4CAF50"),
		Red:       lipgloss.Color("#EF5350"),
		Yellow:    lipgloss.Color("#FFF176"),
		Cyan:      lipgloss.Color("#4DD0E1"),
	},
	ThemeRetro: {
		Primary:   lipgloss.Color("#FFBF00"),
		Secondary: lipgloss.Color("#33FF33"),
		Accent:    lipgloss.Color("#FFD700"),
		Dim:       lipgloss.Color("#777711"),
		Bright:    lipgloss.Color("#FFFF66"),
		Muted:     lipgloss.Color("#AAAA44"),
		Green:     lipgloss.Color("#33FF33"),
		Red:       lipgloss.Color("#FF6600"),
		Yellow:    lipgloss.Color("#FFFF00"),
		Cyan:      lipgloss.Color("#CCFF00"),
	},
	ThemeHighContrast: {
		Primary:   lipgloss.Color("#FFFFFF"),
		Secondary: lipgloss.Color("#00FF00"),
		Accent:    lipgloss.Color("#FFFF00"),
		Dim:       lipgloss.Color("#808080"),
		Bright:    lipgloss.Color("#FFFFFF"),
		Muted:     lipgloss.Color("#C0C0C0"),
		Green:     lipgloss.Color("#00FF00"),
		Red:       lipgloss.Color("#FF0000"),
		Yellow:    lipgloss.Color("#FFFF00"),
		Cyan:      lipgloss.Color("#00FFFF"),
	},
	ThemeMissionCritical: {
		Primary:   lipgloss.Color("#CC3333"),
		Secondary: lipgloss.Color("#BB6655"),
		Accent:    lipgloss.Color("#FF6644"),
		Dim:       lipgloss.Color("#774444"),
		Bright:    lipgloss.Color("#FFCCCC"),
		Muted:     lipgloss.Color("#AA8888"),
		Green:     lipgloss.Color("#669966"),
		Red:       lipgloss.Color("#FF2222"),
		Yellow:    lipgloss.Color("#FFAA44"),
		Cyan:      lipgloss.Color("#DD8888"),
	},
}

var currentTheme ThemeID

// Color variables -- reassigned by applyTheme().
var (
	colorPrimary   lipgloss.Color
	colorSecondary lipgloss.Color
	colorAccent    lipgloss.Color
	colorDim       lipgloss.Color
	colorBright    lipgloss.Color
	colorMuted     lipgloss.Color
	colorGreen     lipgloss.Color
	colorRed       lipgloss.Color
	colorYellow    lipgloss.Color
	colorCyan      lipgloss.Color
)

// Style variables -- reassigned by applyTheme().
var (
	titleStyle         lipgloss.Style
	panelStyle         lipgloss.Style
	panelTitleStyle    lipgloss.Style
	labelStyle         lipgloss.Style
	valueStyle         lipgloss.Style
	metStyle           lipgloss.Style
	activeStyle        lipgloss.Style
	dimStyle           lipgloss.Style
	completedStyle     lipgloss.Style
	pendingStyle       lipgloss.Style
	currentStyle       lipgloss.Style
	signalUpStyle      lipgloss.Style
	signalDownStyle    lipgloss.Style
	errorStyle         lipgloss.Style
	progressFullStyle  lipgloss.Style
	progressEmptyStyle lipgloss.Style
	crewRoleStyle      lipgloss.Style
	crewNameStyle      lipgloss.Style
	crewAgencyStyle    lipgloss.Style
	helpStyle          lipgloss.Style

	ganttCompletedBar lipgloss.Style
	ganttActiveBar    lipgloss.Style
	ganttCursorBar    lipgloss.Style
	ganttPendingBar   lipgloss.Style
	ganttNowMarker    lipgloss.Style

	logTimeStyle  lipgloss.Style
	logTitleStyle lipgloss.Style

	logSelectedCursorStyle lipgloss.Style
	logSelectedTimeStyle   lipgloss.Style
	logSelectedTitleStyle  lipgloss.Style

	// Trajectory view styles.
	starDimStyle         lipgloss.Style
	starMedStyle         lipgloss.Style
	starBrightStyle      lipgloss.Style
	earthGlyphStyle      lipgloss.Style
	moonGlyphStyle       lipgloss.Style
	spacecraftBright     lipgloss.Style
	spacecraftDim        lipgloss.Style
	spacecraftLOS        lipgloss.Style
	spacecraftLOSDim     lipgloss.Style
	pathOutboundStyle    lipgloss.Style
	pathReturnStyle      lipgloss.Style
	sunDirectionStyle    lipgloss.Style
	trajectoryLabelStyle lipgloss.Style

	// Orbital view styles.
	orbitRingStyle  lipgloss.Style
	scaleRingStyle  lipgloss.Style
	scaleLabelStyle lipgloss.Style
	trailStyle      lipgloss.Style
	trailDimStyle   lipgloss.Style

	// Instrument panel styles.
	gaugeFilledStyle  lipgloss.Style
	gaugeEmptyStyle   lipgloss.Style
	sparklineStyle    lipgloss.Style
	compassStyle      lipgloss.Style
	compassLabelStyle lipgloss.Style
	scopeRingStyle    lipgloss.Style
	instTitleStyle    lipgloss.Style
)

func init() {
	applyTheme(ThemeDefault)
}

func applyTheme(id ThemeID) {
	currentTheme = id
	t := themes[id]

	colorPrimary = t.Primary
	colorSecondary = t.Secondary
	colorAccent = t.Accent
	colorDim = t.Dim
	colorBright = t.Bright
	colorMuted = t.Muted
	colorGreen = t.Green
	colorRed = t.Red
	colorYellow = t.Yellow
	colorCyan = t.Cyan

	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorDim).
		Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorAccent)

	labelStyle = lipgloss.NewStyle().
		Foreground(colorMuted)

	valueStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorBright)

	metStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorCyan)

	activeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorGreen)

	dimStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	completedStyle = lipgloss.NewStyle().
		Foreground(colorGreen)

	pendingStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	currentStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorYellow)

	signalUpStyle = lipgloss.NewStyle().
		Foreground(colorGreen)

	signalDownStyle = lipgloss.NewStyle().
		Foreground(colorCyan)

	errorStyle = lipgloss.NewStyle().
		Foreground(colorRed)

	progressFullStyle = lipgloss.NewStyle().
		Foreground(colorPrimary)

	progressEmptyStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	crewRoleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorAccent)

	crewNameStyle = lipgloss.NewStyle().
		Foreground(colorBright)

	crewAgencyStyle = lipgloss.NewStyle().
		Foreground(colorMuted)

	helpStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	ganttCompletedBar = lipgloss.NewStyle().
		Foreground(colorGreen)

	ganttActiveBar = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true)

	ganttCursorBar = lipgloss.NewStyle().
		Foreground(colorYellow).
		Bold(true)

	ganttPendingBar = lipgloss.NewStyle().
		Foreground(colorDim)

	ganttNowMarker = lipgloss.NewStyle().
		Foreground(colorYellow).
		Bold(true)

	logTimeStyle = lipgloss.NewStyle().
		Foreground(colorCyan)

	logTitleStyle = lipgloss.NewStyle().
		Foreground(colorBright)

	logSelectedCursorStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorAccent)

	logSelectedTimeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorAccent)

	logSelectedTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorAccent)

	// Trajectory view.
	starDimStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	starMedStyle = lipgloss.NewStyle().
		Foreground(colorMuted)

	starBrightStyle = lipgloss.NewStyle().
		Foreground(colorMuted)

	earthGlyphStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary)

	moonGlyphStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorBright)

	spacecraftBright = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorAccent)

	spacecraftDim = lipgloss.NewStyle().
		Foreground(colorAccent)

	spacecraftLOS = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorRed)

	spacecraftLOSDim = lipgloss.NewStyle().
		Foreground(colorRed)

	pathOutboundStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorAccent)

	pathReturnStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorYellow)

	sunDirectionStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorYellow)

	trajectoryLabelStyle = lipgloss.NewStyle().
		Foreground(colorMuted)

	// Orbital view.
	orbitRingStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	scaleRingStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	scaleLabelStyle = lipgloss.NewStyle().
		Foreground(colorMuted)

	trailStyle = lipgloss.NewStyle().
		Foreground(colorCyan)

	trailDimStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	// Instrument panel.
	gaugeFilledStyle = lipgloss.NewStyle().
		Foreground(colorPrimary)

	gaugeEmptyStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	sparklineStyle = lipgloss.NewStyle().
		Foreground(colorCyan)

	compassStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	compassLabelStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorMuted)

	scopeRingStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	instTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary)
}

// NextTheme cycles to the next theme and applies it.
func NextTheme() {
	applyTheme((currentTheme + 1) % themeCount)
}

// ThemeName returns the current theme's display name.
func ThemeName() string {
	switch currentTheme {
	case ThemeDefault:
		return "Default"
	case ThemeRetro:
		return "Retro"
	case ThemeHighContrast:
		return "Hi-Con"
	case ThemeMissionCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}
